package api

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"sync"
	"time"

	"opsone/internal/chatresolve"
	"opsone/internal/store"
)

// voiceCorrectionsCache is a simple in-memory cache for voice corrections (P5).
// Refreshed lazily when TTL expires.
var voiceCorrectionsCache = struct {
	mu          sync.RWMutex
	corrections map[string]string
	loadedAt    time.Time
	ttl         time.Duration
}{
	corrections: make(map[string]string),
	ttl:         5 * time.Minute,
}

// ChatTurnInput is metadata from POST /chat for persistence (§7.6.5.5 P1).
type ChatTurnInput struct {
	SessionID   string
	UserID      string
	Message     string
	STTRaw      string
	InputSource store.ChatInputSource
	IsAdmin     bool
}

// ChatTurnOutcome captures routing metadata for one chat reply.
type ChatTurnOutcome struct {
	Route        string
	IntentKey    string
	Slots        map[string]any
	ToolsCalled  []string
	ActionResult store.ChatActionResult
}

func parseChatInputSource(v string) store.ChatInputSource {
	if strings.EqualFold(strings.TrimSpace(v), "voice") {
		return store.ChatInputVoice
	}
	return store.ChatInputText
}

func chatSlotsFromMessage(userMsg string) map[string]any {
	slots := map[string]any{}
	if p := chatresolve.ExtractProductFromText(userMsg); p != "" {
		slots["product"] = p
	}
	if sku := chatresolve.ExtractSKUFromText(userMsg); sku != "" {
		slots["sku"] = sku
	}
	if mode := chatresolve.ParseScopeAutoMode(userMsg); mode != "" {
		slots["auto_action"] = mode
	}
	if d := chatresolve.ExtractDurationMinutes(userMsg); d > 0 {
		slots["duration_min"] = d
	}
	if len(slots) == 0 {
		return nil
	}
	return slots
}

func chatCommandRouteKey(userMsg string) string {
	if action, _ := chatresolve.DetectUIAction(userMsg); action != "" {
		return "ui_" + string(action)
	}
	return "command"
}

func chatTurnOutcomeInit(intent chatresolve.ChatIntent) *ChatTurnOutcome {
	out := &ChatTurnOutcome{ActionResult: store.ChatActionSuccess}
	if intent != chatresolve.IntentUnknown {
		out.IntentKey = string(intent)
	}
	return out
}

func (out *ChatTurnOutcome) setDirectRoute(route string, userMsg string) {
	if out == nil {
		return
	}
	out.Route = route
	if out.Slots == nil {
		out.Slots = chatSlotsFromMessage(userMsg)
	}
	out.ActionResult = store.ChatActionSuccess
}

func (out *ChatTurnOutcome) appendTool(name string) {
	if out == nil || strings.TrimSpace(name) == "" {
		return
	}
	out.ToolsCalled = append(out.ToolsCalled, name)
}

func (s *Server) recordChatIntentHit(intent chatresolve.ChatIntent, userMsg, route string, result store.ChatActionResult) {
	if intent == chatresolve.IntentUnknown {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = s.DB.BumpChatIntentStat(ctx, string(intent), userMsg, route, string(result))
	}()
}

// getVoiceCorrections returns cached voice corrections, refreshing if stale (P5).
func (s *Server) getVoiceCorrections(ctx context.Context) map[string]string {
	voiceCorrectionsCache.mu.RLock()
	if time.Since(voiceCorrectionsCache.loadedAt) < voiceCorrectionsCache.ttl {
		out := voiceCorrectionsCache.corrections
		voiceCorrectionsCache.mu.RUnlock()
		return out
	}
	voiceCorrectionsCache.mu.RUnlock()

	// Refresh
	voiceCorrectionsCache.mu.Lock()
	defer voiceCorrectionsCache.mu.Unlock()
	if time.Since(voiceCorrectionsCache.loadedAt) < voiceCorrectionsCache.ttl {
		return voiceCorrectionsCache.corrections
	}
	if m, err := s.DB.GetVoiceCorrections(ctx); err == nil {
		voiceCorrectionsCache.corrections = m
		voiceCorrectionsCache.loadedAt = time.Now()
	}
	return voiceCorrectionsCache.corrections
}

// applyVoiceCorrectionToInput applies known STT corrections to a voice message (P5).
// Returns the corrected message.
func (s *Server) applyVoiceCorrectionToInput(ctx context.Context, msg string) string {
	norm := store.NormalizeChatMessage(msg)
	// 1. Apply static + DB-backed domain phonetic map first
	norm = store.ApplyDomainPhonetic(norm)
	if entries := s.DB.GetPhoneticEntries(ctx); len(entries) > 0 {
		norm = store.ApplyPhoneticEntries(norm, entries)
	}
	// 2. Apply learned STT corrections
	corrections := s.getVoiceCorrections(ctx)
	if len(corrections) > 0 {
		norm = store.ApplyVoiceCorrections(norm, corrections)
	}
	if norm == store.NormalizeChatMessage(msg) {
		return msg
	}
	return norm
}

// recordVoiceCorrection stores a STT correction pair when stt_raw differs from final message (P5).
func (s *Server) recordVoiceCorrection(ctx context.Context, sttRaw, finalMsg string) {
	if strings.TrimSpace(sttRaw) == "" || strings.TrimSpace(finalMsg) == "" {
		return
	}
	heardNorm := store.NormalizeChatMessage(sttRaw)
	correctedNorm := store.NormalizeChatMessage(finalMsg)
	if heardNorm == correctedNorm {
		return // no correction needed
	}
	go func() {
		ctx2, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := s.DB.UpsertVoiceCorrection(ctx2, heardNorm, correctedNorm); err != nil {
			log.Printf("voice correction upsert: %v", err)
		}
		// Invalidate cache
		voiceCorrectionsCache.mu.Lock()
		voiceCorrectionsCache.loadedAt = time.Time{}
		voiceCorrectionsCache.mu.Unlock()
	}()
}

// detectAndMarkRetry checks if the current message is a retry of a recent one,
// and marks the previous interaction as wrong_route (P5).
func (s *Server) detectAndMarkRetry(ctx context.Context, sessionUUID, currentMsg string) {
	if sessionUUID == "" {
		return
	}
	prevID, _, found := s.DB.FindRecentInteraction(ctx, sessionUUID, 60)
	if !found || prevID <= 0 {
		return
	}
	// Get the previous message norm to compare
	const q = `SELECT message_norm FROM chat_interaction_log WHERE id = ?`
	var prevNorm string
	if err := s.DB.QueryRowContext(ctx, q, prevID).Scan(&prevNorm); err != nil {
		return
	}
	currentNorm := store.NormalizeChatMessage(currentMsg)
	if store.DetectRetrySignal(currentNorm, prevNorm) {
		go func() {
			ctx2, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = s.DB.MarkInteractionWrongRoute(ctx2, prevID)
		}()
	}
}

func (s *Server) persistChatTurn(input ChatTurnInput, out *ChatTurnOutcome, reply string, latencyMS int) {
	if input.SessionID == "" || input.UserID == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	route := "unknown"
	intentKey := ""
	action := store.ChatActionSuccess
	var slotsJSON, toolsJSON json.RawMessage
	if out != nil {
		if out.Route != "" {
			route = out.Route
		}
		intentKey = out.IntentKey
		if out.ActionResult != "" {
			action = out.ActionResult
		}
		if len(out.Slots) > 0 {
			slotsJSON, _ = json.Marshal(out.Slots)
		}
		if len(out.ToolsCalled) > 0 {
			toolsJSON, _ = json.Marshal(out.ToolsCalled)
		}
	}

	sessionID, err := s.DB.UpsertChatSession(ctx, input.SessionID, input.UserID)
	if err != nil {
		log.Printf("chat persist session: %v", err)
		return
	}
	src := input.InputSource
	if src == "" {
		src = store.ChatInputText
	}
	if err := s.DB.InsertChatMessage(ctx, sessionID, "user", input.Message, src, input.STTRaw); err != nil {
		log.Printf("chat persist user message: %v", err)
	}
	if reply != "" {
		if err := s.DB.InsertChatMessage(ctx, sessionID, "assistant", reply, store.ChatInputText, ""); err != nil {
			log.Printf("chat persist assistant message: %v", err)
		}
	}
	if _, err := s.DB.InsertChatInteractionLog(ctx, store.ChatInteractionLogInput{
		SessionUUID:  input.SessionID,
		UserID:       input.UserID,
		UserMessage:  input.Message,
		InputSource:  src,
		STTRaw:       input.STTRaw,
		Route:        route,
		IntentKey:    intentKey,
		SlotsJSON:    slotsJSON,
		ToolsCalled:  toolsJSON,
		ActionResult: action,
		ReplyPreview: reply,
		LatencyMS:    latencyMS,
		IsAdmin:      input.IsAdmin,
	}); err != nil {
		log.Printf("chat persist interaction log: %v", err)
	}
}
