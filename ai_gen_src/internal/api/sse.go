package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// handleSuggestions is a REST fallback for clients that cannot use SSE
// (e.g., when a reverse proxy buffers the event stream).
// GET /api/v1/suggestions → same payload as the SSE pending_suggestions event.
func (s *Server) handleSuggestions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "GET only")
		return
	}
	data, err := s.getPendingSuggestionsForSSE(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "sse_unsupported", "SSE không được hỗ trợ")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx proxy buffering for SSE

	var lastCycle uint64
	var lastSuggestionCheck int64 // Track if we already sent suggestions in this second
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	send := func(event string, data any) {
		raw, _ := json.Marshal(data)
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, raw)
		flusher.Flush()
	}

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			cycle, ok, err := s.DB.GetLatestCycle(r.Context())
			if err != nil || !ok {
				continue
			}
			if cycle.ID > lastCycle {
				lastCycle = cycle.ID
				send("cycle_finished", map[string]any{
					"cycle_id":      cycle.ID,
					"health_status": cycle.HealthStatus,
				})
				send("health_status", map[string]any{
					"health_status": cycle.HealthStatus,
					"health_label":  cycle.HealthStatus,
				})

				// Send pending suggestions when cycle finishes.
				// Always send (even when empty) so the frontend can clear stale proposal messages.
				now := time.Now().Unix()
				if now != lastSuggestionCheck { // Only check once per second
					lastSuggestionCheck = now
					suggestions, err := s.getPendingSuggestionsForSSE(r.Context())
					if err == nil && suggestions != nil {
						send("pending_suggestions", suggestions)
					}
				}
			}
		}
	}
}
