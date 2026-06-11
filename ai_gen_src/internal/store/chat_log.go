package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

// ChatInputSource is how the user message was entered.
type ChatInputSource string

const (
	ChatInputText  ChatInputSource = "text"
	ChatInputVoice ChatInputSource = "voice"
)

// ChatActionResult labels the outcome of a chat turn for learning stats.
type ChatActionResult string

const (
	ChatActionSuccess ChatActionResult = "success"
	ChatActionError   ChatActionResult = "error"
	ChatActionNoOp    ChatActionResult = "no_op"
	ChatActionWrong   ChatActionResult = "wrong_route"
)

// ChatInteractionLogInput is one row for chat_interaction_log (§7.6.5.5 P1).
type ChatInteractionLogInput struct {
	SessionUUID  string
	UserID       string
	UserMessage  string
	InputSource  ChatInputSource
	STTRaw       string
	Route        string
	IntentKey    string
	SlotsJSON    json.RawMessage
	ToolsCalled  json.RawMessage
	ActionResult ChatActionResult
	ReplyPreview string
	LatencyMS    int
	IsAdmin      bool
}

// UpsertChatSession returns internal session id for session_uuid + user_id.
func (db *DB) UpsertChatSession(ctx context.Context, sessionUUID, userID string) (int64, error) {
	sessionUUID = strings.TrimSpace(sessionUUID)
	userID = strings.TrimSpace(userID)
	if sessionUUID == "" || userID == "" {
		return 0, fmt.Errorf("chat session: missing session_uuid or user_id")
	}
	const lookup = `SELECT id FROM chat_sessions WHERE session_uuid = ? LIMIT 1`
	var id int64
	err := db.QueryRowContext(ctx, lookup, sessionUUID).Scan(&id)
	if err == nil {
		const touch = `UPDATE chat_sessions SET user_id = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
		if _, err := db.ExecContext(ctx, touch, userID, id); err != nil {
			return 0, err
		}
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}
	const insert = `INSERT INTO chat_sessions (session_uuid, user_id) VALUES (?, ?)`
	res, err := db.ExecContext(ctx, insert, sessionUUID, userID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// InsertChatMessage appends one message to chat_messages.
func (db *DB) InsertChatMessage(ctx context.Context, sessionID int64, role string, content string, src ChatInputSource, sttRaw string) error {
	if sessionID <= 0 {
		return fmt.Errorf("chat message: invalid session_id")
	}
	role = strings.TrimSpace(role)
	if role != "user" && role != "assistant" {
		return fmt.Errorf("chat message: role must be user or assistant")
	}
	if src == "" {
		src = ChatInputText
	}
	var stt sql.NullString
	if strings.TrimSpace(sttRaw) != "" {
		stt = sql.NullString{String: truncateChatField(sttRaw, 1024), Valid: true}
	}
	const query = `
		INSERT INTO chat_messages (session_id, role, content, input_source, stt_raw)
		VALUES (?, ?, ?, ?, ?)`
	_, err := db.ExecContext(ctx, query, sessionID, role, content, string(src), stt)
	return err
}

// InsertChatInteractionLog records one routed chat turn.
func (db *DB) InsertChatInteractionLog(ctx context.Context, in ChatInteractionLogInput) (int64, error) {
	sessionUUID := strings.TrimSpace(in.SessionUUID)
	if sessionUUID == "" {
		return 0, fmt.Errorf("chat log: missing session_uuid")
	}
	userMsg := truncateChatField(strings.TrimSpace(in.UserMessage), 1024)
	if userMsg == "" {
		return 0, fmt.Errorf("chat log: missing user_message")
	}
	route := strings.TrimSpace(in.Route)
	if route == "" {
		route = "unknown"
	}
	src := in.InputSource
	if src == "" {
		src = ChatInputText
	}
	action := in.ActionResult
	if action == "" {
		action = ChatActionNoOp
	}
	norm := NormalizeChatMessage(userMsg)
	var userID sql.NullString
	if strings.TrimSpace(in.UserID) != "" {
		userID = sql.NullString{String: strings.TrimSpace(in.UserID), Valid: true}
	}
	var intent sql.NullString
	if strings.TrimSpace(in.IntentKey) != "" {
		intent = sql.NullString{String: strings.TrimSpace(in.IntentKey), Valid: true}
	}
	var stt sql.NullString
	if strings.TrimSpace(in.STTRaw) != "" {
		stt = sql.NullString{String: truncateChatField(in.STTRaw, 1024), Valid: true}
	}
	var slots any
	if len(in.SlotsJSON) > 0 {
		slots = in.SlotsJSON
	}
	var tools any
	if len(in.ToolsCalled) > 0 {
		tools = in.ToolsCalled
	}
	var preview sql.NullString
	if p := truncateChatField(strings.TrimSpace(in.ReplyPreview), 512); p != "" {
		preview = sql.NullString{String: p, Valid: true}
	}
	var latency sql.NullInt64
	if in.LatencyMS > 0 {
		latency = sql.NullInt64{Int64: int64(in.LatencyMS), Valid: true}
	}
	const query = `
		INSERT INTO chat_interaction_log (
			session_uuid, user_id, user_message, message_norm, input_source, stt_raw,
			route, intent_key, slots_json, tools_called, action_result, reply_preview,
			latency_ms, is_admin
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	res, err := db.ExecContext(ctx, query,
		sessionUUID, userID, userMsg, norm, string(src), stt,
		route, intent, slots, tools, string(action), preview,
		latency, boolToTinyInt(in.IsAdmin),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// NormalizeChatMessage lowercases and collapses whitespace for pattern mining.
func NormalizeChatMessage(msg string) string {
	return normalizeChatPattern(msg)
}

func truncateChatField(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max]
}

func boolToTinyInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
