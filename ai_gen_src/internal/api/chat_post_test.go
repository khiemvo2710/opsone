package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestChatPostPersistsInteractionLog(t *testing.T) {
	srv := testServer(t)
	sessionID := "api-test-session-" + t.Name()
	body := strings.NewReader(`{
		"message": "thẻ garena có đang bảo trì không",
		"session_id": "` + sessionID + `",
		"input_source": "text"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-OpsOne-Actor", "test.chat.user")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /chat status %d body %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Reply string `json:"reply"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(resp.Reply) == "" {
		t.Fatal("empty reply")
	}

	var count int
	err := srv.DB.QueryRowContext(req.Context(),
		`SELECT COUNT(*) FROM chat_interaction_log WHERE session_uuid = ? AND user_id = ?`,
		sessionID, "test.chat.user",
	).Scan(&count)
	if err != nil {
		t.Fatalf("query log: %v", err)
	}
	if count < 1 {
		t.Errorf("expected at least 1 interaction log row, got %d", count)
	}
}
