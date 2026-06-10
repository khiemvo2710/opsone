package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "sse_unsupported", "SSE không được hỗ trợ")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	var lastCycle uint64
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
			}
		}
	}
}
