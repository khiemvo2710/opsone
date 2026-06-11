package api

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"opsone/internal/chatresolve"
	"opsone/internal/store"
)

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
