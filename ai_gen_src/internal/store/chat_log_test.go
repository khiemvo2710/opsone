package store_test

import (
	"context"
	"encoding/json"
	"testing"

	"opsone/internal/store"
)

func TestChatLogPersist(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	sessionUUID := "test-session-" + t.Name()
	userID := "test.user"

	sessionID, err := db.UpsertChatSession(ctx, sessionUUID, userID)
	if err != nil {
		t.Fatalf("UpsertChatSession: %v", err)
	}
	if sessionID <= 0 {
		t.Fatalf("expected session id > 0, got %d", sessionID)
	}

	if err := db.InsertChatMessage(ctx, sessionID, "user", "thẻ garena có bảo trì không", store.ChatInputText, ""); err != nil {
		t.Fatalf("InsertChatMessage user: %v", err)
	}
	if err := db.InsertChatMessage(ctx, sessionID, "assistant", "Không có bảo trì active.", store.ChatInputText, ""); err != nil {
		t.Fatalf("InsertChatMessage assistant: %v", err)
	}

	slots, _ := json.Marshal(map[string]string{"product": "GARENA"})
	logID, err := db.InsertChatInteractionLog(ctx, store.ChatInteractionLogInput{
		SessionUUID:  sessionUUID,
		UserID:       userID,
		UserMessage:  "thẻ garena có bảo trì không",
		InputSource:  store.ChatInputText,
		Route:        "direct_maintenance",
		IntentKey:    "maintenance",
		SlotsJSON:    slots,
		ActionResult: store.ChatActionSuccess,
		ReplyPreview: "Không có bảo trì active.",
		LatencyMS:    12,
	})
	if err != nil {
		t.Fatalf("InsertChatInteractionLog: %v", err)
	}
	if logID <= 0 {
		t.Fatalf("expected log id > 0, got %d", logID)
	}

	var count int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM chat_interaction_log WHERE session_uuid = ? AND route = ?`,
		sessionUUID, "direct_maintenance",
	).Scan(&count); err != nil {
		t.Fatalf("count log: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 interaction log, got %d", count)
	}
}

func TestNormalizeChatMessage(t *testing.T) {
	got := store.NormalizeChatMessage("  Thẻ   Garena   ")
	want := "thẻ garena"
	if got != want {
		t.Errorf("NormalizeChatMessage = %q, want %q", got, want)
	}
}
