package api

import (
	"net/http"
	"strconv"
	"strings"

	"opsone/internal/store"
)

// handleAdminChatPatternsList — GET /api/v1/admin/chat-patterns
// Query: ?status=candidate|approved|deprecated (default: candidate)
func (s *Server) handleAdminChatPatternsList(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r, s.Config.DevAuthBypass) {
		return
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status == "" {
		status = "candidate"
	}
	patterns, err := s.DB.ListCommandPatterns(r.Context(), status, 200)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	items := make([]map[string]any, 0, len(patterns))
	for _, p := range patterns {
		items = append(items, patternJSON(p))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "status": status})
}

// handleAdminChatPatternPromote — POST /api/v1/admin/chat-patterns/:id/promote
func (s *Server) handleAdminChatPatternPromote(w http.ResponseWriter, r *http.Request, id int64) {
	if !requireAdmin(w, r, s.Config.DevAuthBypass) {
		return
	}
	by := actorFromRequest(r, s.Config.DevAuthBypass)
	if err := s.DB.PromoteCommandPattern(r.Context(), id, by); err != nil {
		writeError(w, http.StatusConflict, "promote_failed", "Pattern không tồn tại hoặc đã được xử lý")
		return
	}
	p, err := s.DB.GetCommandPatternByID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"id": id, "status": "approved"})
		return
	}
	writeJSON(w, http.StatusOK, patternJSON(p))
}

// handleAdminChatPatternDeprecate — POST /api/v1/admin/chat-patterns/:id/deprecate
func (s *Server) handleAdminChatPatternDeprecate(w http.ResponseWriter, r *http.Request, id int64) {
	if !requireAdmin(w, r, s.Config.DevAuthBypass) {
		return
	}
	if err := s.DB.DeprecateCommandPattern(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "status": "deprecated"})
}

// handleAdminChatMinePatterns — POST /api/v1/admin/chat-patterns/mine
// Triggers the daily mining job on-demand.
func (s *Server) handleAdminChatMinePatterns(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r, s.Config.DevAuthBypass) {
		return
	}
	results, err := s.DB.MineCommandPatterns(r.Context(), 3, 168)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "mine_error", err.Error())
		return
	}
	total := 0
	for _, res := range results {
		total += res.Upserted
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"mined":   total,
		"by_key":  results,
	})
}

// handleAdminMineFewShot — POST /api/v1/admin/chat-patterns/mine-few-shot
func (s *Server) handleAdminMineFewShot(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r, s.Config.DevAuthBypass) {
		return
	}
	n, err := s.DB.MineFewShotExamples(r.Context(), 0.8, 168)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "mine_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"inserted": n})
}

// handleAdminFewShotList — GET /api/v1/admin/few-shot
func (s *Server) handleAdminFewShotList(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r, s.Config.DevAuthBypass) {
		return
	}
	cmdKey := r.URL.Query().Get("command_key")
	status := r.URL.Query().Get("status")
	examples, err := s.DB.ListFewShotExamples(r.Context(), cmdKey, status, 100)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	items := make([]map[string]any, 0, len(examples))
	for _, e := range examples {
		items = append(items, fewShotJSON(e))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// handleAdminFewShotPromote — POST /api/v1/admin/few-shot/:id/promote
func (s *Server) handleAdminFewShotPromote(w http.ResponseWriter, r *http.Request, id int64) {
	if !requireAdmin(w, r, s.Config.DevAuthBypass) {
		return
	}
	if err := s.DB.PromoteFewShotExample(r.Context(), id); err != nil {
		writeError(w, http.StatusConflict, "promote_failed", "Example không tồn tại hoặc đã được xử lý")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "status": "approved"})
}

// handleAdminFewShotDeprecate — POST /api/v1/admin/few-shot/:id/deprecate
func (s *Server) handleAdminFewShotDeprecate(w http.ResponseWriter, r *http.Request, id int64) {
	if !requireAdmin(w, r, s.Config.DevAuthBypass) {
		return
	}
	if err := s.DB.DeprecateFewShotExample(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "status": "deprecated"})
}

// handleAdminVoiceCorrectionsList — GET /api/v1/admin/voice-corrections
func (s *Server) handleAdminVoiceCorrectionsList(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r, s.Config.DevAuthBypass) {
		return
	}
	corrections, err := s.DB.ListVoiceCorrections(r.Context(), 100)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	items := make([]map[string]any, 0, len(corrections))
	for _, c := range corrections {
		items = append(items, map[string]any{
			"id":             c.ID,
			"heard_norm":     c.HeardNorm,
			"corrected_norm": c.CorrectedNorm,
			"hit_count":      c.HitCount,
			"last_seen_at":   c.LastSeenAt,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// handleChatFeedback — POST /api/v1/chat/feedback
// Body: { "interaction_id": int, "rating": "up"|"down"|"corrected", "correction"?: string, "expected_command"?: string }
func (s *Server) handleChatFeedback(w http.ResponseWriter, r *http.Request) {
	var body struct {
		InteractionID   int64  `json:"interaction_id"`
		Rating          string `json:"rating"`
		UserCorrection  string `json:"correction"`
		ExpectedCommand string `json:"expected_command"`
	}
	if err := decodeJSON(r, &body); err != nil || body.InteractionID <= 0 || body.Rating == "" {
		writeError(w, http.StatusBadRequest, "invalid_input", "Thiếu interaction_id hoặc rating")
		return
	}
	if err := s.DB.InsertChatFeedback(r.Context(), body.InteractionID, body.Rating, body.UserCorrection, body.ExpectedCommand); err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	// If "down" feedback, mark the interaction as wrong_route signal
	if body.Rating == "down" {
		_ = s.DB.MarkInteractionWrongRoute(r.Context(), body.InteractionID)
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// routeAdminChatPatterns dispatches /api/v1/admin/chat-patterns/*
func (s *Server) routeAdminChatPatterns(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/chat-patterns")
	path = strings.Trim(path, "/")

	if path == "" {
		if r.Method == http.MethodGet {
			s.handleAdminChatPatternsList(w, r)
			return
		}
		writeError(w, http.StatusMethodNotAllowed, "method", "Method not allowed")
		return
	}
	if path == "mine" && r.Method == http.MethodPost {
		s.handleAdminChatMinePatterns(w, r)
		return
	}
	if path == "mine-few-shot" && r.Method == http.MethodPost {
		s.handleAdminMineFewShot(w, r)
		return
	}
	parts := strings.Split(path, "/")
	if len(parts) == 2 && r.Method == http.MethodPost {
		id, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_id", "ID không hợp lệ")
			return
		}
		switch parts[1] {
		case "promote":
			s.handleAdminChatPatternPromote(w, r, id)
			return
		case "deprecate":
			s.handleAdminChatPatternDeprecate(w, r, id)
			return
		}
	}
	writeError(w, http.StatusNotFound, "not_found", "Not found")
}

// routeAdminFewShot dispatches /api/v1/admin/few-shot/*
func (s *Server) routeAdminFewShot(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/few-shot")
	path = strings.Trim(path, "/")

	if path == "" {
		if r.Method == http.MethodGet {
			s.handleAdminFewShotList(w, r)
			return
		}
		writeError(w, http.StatusMethodNotAllowed, "method", "Method not allowed")
		return
	}
	parts := strings.Split(path, "/")
	if len(parts) == 2 && r.Method == http.MethodPost {
		id, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_id", "ID không hợp lệ")
			return
		}
		switch parts[1] {
		case "promote":
			s.handleAdminFewShotPromote(w, r, id)
			return
		case "deprecate":
			s.handleAdminFewShotDeprecate(w, r, id)
			return
		}
	}
	writeError(w, http.StatusNotFound, "not_found", "Not found")
}

// patternJSON serializes a CommandPattern for API responses.
func patternJSON(p store.CommandPattern) map[string]any {
	m := map[string]any{
		"id":            p.ID,
		"command_key":   p.CommandKey,
		"pattern_type":  p.PatternType,
		"pattern_def":   p.PatternDef,
		"hit_count":     p.HitCount,
		"success_count": p.SuccessCount,
		"fail_count":    p.FailCount,
		"status":        string(p.Status),
		"min_role":      p.MinRole,
		"created_at":    p.CreatedAt,
	}
	if p.ApprovedBy != "" {
		m["approved_by"] = p.ApprovedBy
	}
	if p.ApprovedAt != nil {
		m["approved_at"] = p.ApprovedAt
	}
	if len(p.DefaultSlots) > 0 {
		m["default_slots"] = p.DefaultSlots
	}
	return m
}

// fewShotJSON serializes a FewShotExample for API responses.
func fewShotJSON(e store.FewShotExample) map[string]any {
	m := map[string]any{
		"id":                e.ID,
		"command_key":       e.CommandKey,
		"user_example":      e.UserExample,
		"assistant_example": e.AssistantExample,
		"priority":          e.Priority,
		"status":            string(e.Status),
		"created_at":        e.CreatedAt,
	}
	if e.SuccessRate != nil {
		m["success_rate"] = *e.SuccessRate
	}
	return m
}
