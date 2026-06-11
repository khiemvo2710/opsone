package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"opsone/internal/mock"
	"opsone/internal/output"
	"opsone/internal/store"
	"opsone/internal/tools"
)

func (s *Server) handleHealthStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cycle, ok, err := s.DB.GetLatestCycle(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", "Không đọc được chu kỳ phân tích")
		return
	}
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{
			"health_status":  "green",
			"health_label":   output.HealthLabel("green"),
			"health_summary": "Chưa có chu kỳ phân tích",
			"products":       []any{},
		})
		return
	}
	rows, err := s.DB.ListProductHealthByCycle(ctx, cycle.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	products := make([]map[string]any, 0, len(rows))
	for _, p := range rows {
		item := map[string]any{
			"product_code":  p.ProductCode,
			"health_status": p.HealthStatus,
		}
		if lbl, err := s.DB.GetProductByCode(ctx, p.ProductCode); err == nil {
			item["product_label"] = lbl.Label
		}
		if p.HealthSummary.Valid {
			item["health_summary"] = p.HealthSummary.String
		}
		products = append(products, item)
	}
	summary := ""
	if cycle.HealthSummary.Valid {
		summary = cycle.HealthSummary.String
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"cycle_id":       cycle.ID,
		"cycle_started":  cycle.CycleStarted,
		"health_status":  cycle.HealthStatus,
		"health_label":   output.HealthLabel(cycle.HealthStatus),
		"health_summary": summary,
		"products":       products,
	})
}

func (s *Server) handleConfigGet(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.DB.GetAgentSettingsFull(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, configJSON(cfg))
}

func (s *Server) handleConfigPut(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r, s.Config.DevAuthBypass) {
		return
	}
	var body store.ConfigUpdatePatch
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "JSON không hợp lệ")
		return
	}
	if body.MaintenanceDefaultDurationMin != nil {
		v := *body.MaintenanceDefaultDurationMin
		if v < 1 || v > 255 {
			writeError(w, http.StatusBadRequest, "invalid_input", "maintenance_default_duration_min phải từ 1 đến 255")
			return
		}
	}
	if err := s.DB.ApplyConfigUpdate(r.Context(), body); err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	cfg, _ := s.DB.GetAgentSettingsFull(r.Context())
	writeJSON(w, http.StatusOK, configJSON(cfg))
}

func (s *Server) handleIncidentsList(w http.ResponseWriter, r *http.Request) {
	var since *time.Time
	if v := r.URL.Query().Get("since"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			since = &t
		}
	}
	page := 1
	pageSize := 10
	if v := r.URL.Query().Get("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			page = n
		}
	}
	if v := r.URL.Query().Get("page_size"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 50 {
			pageSize = n
		}
	}
	offset := (page - 1) * pageSize

	total, err := s.DB.CountIncidents(r.Context(), since)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	rows, err := s.DB.ListIncidents(r.Context(), since, pageSize, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	items := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		items = append(items, incidentJSON(row))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (s *Server) handleIncidentGet(w http.ResponseWriter, r *http.Request, id string) {
	row, err := s.DB.GetIncidentByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Không tìm thấy sự cố")
		return
	}
	writeJSON(w, http.StatusOK, incidentJSON(row))
}

func (s *Server) handleRoutingPlansLatest(w http.ResponseWriter, r *http.Request) {
	rows, err := s.DB.ListLatestRoutingPlans(r.Context(), 10)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	items := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		items = append(items, routingPlanJSON(row))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleRoutingPlanApprove(w http.ResponseWriter, r *http.Request, id uint64) {
	if !requireAdmin(w, r, s.Config.DevAuthBypass) {
		return
	}
	ctx := r.Context()
	plan, err := s.DB.GetRoutingPlan(ctx, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Không tìm thấy kế hoạch routing")
		return
	}
	if plan.Status != "pending_approve" && plan.Status != "draft" {
		writeError(w, http.StatusConflict, "invalid_status", "Kế hoạch không ở trạng thái chờ duyệt")
		return
	}
	var parsed struct {
		Scope       string             `json:"scope"`
		SKU         string             `json:"sku"`
		ProposedPct map[string]float64 `json:"proposed_pct"`
	}
	if err := json.Unmarshal(plan.PlanJSON, &parsed); err != nil {
		writeError(w, http.StatusInternalServerError, "parse_error", "plan_json không hợp lệ")
		return
	}
	if parsed.ProposedPct == nil {
		writeError(w, http.StatusInternalServerError, "parse_error", "Thiếu proposed_pct")
		return
	}
	routing := parsed.ProposedPct
	var reqBody struct {
		ProposedPct map[string]float64 `json:"proposed_pct"`
	}
	if err := decodeJSON(r, &reqBody); err == nil && len(reqBody.ProposedPct) > 0 {
		routing = reqBody.ProposedPct
	}
	by := actorFromRequest(r, s.Config.DevAuthBypass)
	planID := id
	var cycleID *uint64
	if plan.CycleID.Valid {
		c := uint64(plan.CycleID.Int64)
		cycleID = &c
	}
	incidentID, _ := s.DB.FindOpenIncidentForScope(ctx, plan.ProductCode, plan.SKUCode, cycleID)
	var incPtr *string
	if incidentID != "" {
		incPtr = &incidentID
	}
	out, err := s.Tools.UpdateRouting(ctx, tools.UpdateRoutingInput{
		Product:     plan.ProductCode,
		Scope:       plan.Scope,
		SKU:         plan.SKUCode,
		Routing:     routing,
		TriggerType: "admin_approve",
		ExecutedBy:  by,
		Reason:      fmt.Sprintf("approve routing plan #%d", id),
		PlanID:      &planID,
		CycleID:     cycleID,
		IncidentID:  incPtr,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, "routing_failed", err.Error())
		return
	}
	if err := s.DB.UpdateRoutingPlanStatus(ctx, id, "executed", by); err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"plan_id": id, "applied": out.Applied, "change_log_ids": out.ChangeLogIDs})
}

func (s *Server) handleRoutingPlanReject(w http.ResponseWriter, r *http.Request, id uint64) {
	if !requireAdmin(w, r, s.Config.DevAuthBypass) {
		return
	}
	ctx := r.Context()
	plan, err := s.DB.GetRoutingPlan(ctx, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Không tìm thấy kế hoạch routing")
		return
	}
	if err := s.DB.UpdateRoutingPlanStatus(ctx, id, "rejected", actorFromRequest(r, s.Config.DevAuthBypass)); err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	var cycleID *uint64
	if plan.CycleID.Valid {
		c := uint64(plan.CycleID.Int64)
		cycleID = &c
	}
	if incID, findErr := s.DB.FindOpenIncidentForScope(ctx, plan.ProductCode, plan.SKUCode, cycleID); findErr == nil && incID != "" {
		_ = s.DB.UpdateIncidentHandled(ctx, incID, store.IncidentHandleUpdate{
			Status:           "acknowledged",
			HandledBy:        actorFromRequest(r, s.Config.DevAuthBypass),
			ResolutionAction: "admin_reject",
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"plan_id": id, "status": "rejected"})
}

func (s *Server) handleMaintenanceList(w http.ResponseWriter, r *http.Request) {
	rows, err := s.DB.ListMaintenanceWindows(r.Context(), r.URL.Query().Get("product"), r.URL.Query().Get("status"), 50)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	items := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		items = append(items, maintenanceJSON(row))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleMaintenanceGet(w http.ResponseWriter, r *http.Request, id string) {
	row, err := s.DB.GetMaintenanceByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Không tìm thấy cửa sổ bảo trì")
		return
	}
	writeJSON(w, http.StatusOK, maintenanceJSON(row))
}

func (s *Server) handleNotificationsList(w http.ResponseWriter, r *http.Request) {
	rows, err := s.DB.ListNotifications(r.Context(), 50)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	items := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		items = append(items, notificationJSON(row))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleEscalationList(w http.ResponseWriter, r *http.Request) {
	rows, err := s.DB.ListChatEscalations(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": rows})
}

func (s *Server) handleProductsList(w http.ResponseWriter, r *http.Request) {
	products, err := s.DB.ListProducts(r.Context(), true)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	items := make([]map[string]any, 0, len(products))
	for _, p := range products {
		items = append(items, productJSON(p))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleProductRoutingGet(w http.ResponseWriter, r *http.Request, code string) {
	rows, err := s.DB.GetAllRoutingForProduct(r.Context(), code)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"product_code": code, "routing": rows})
}

func (s *Server) handleProductThresholdsGet(w http.ResponseWriter, r *http.Request, code string) {
	th, err := s.DB.GetProductThreshold(r.Context(), code)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Không tìm thấy ngưỡng sản phẩm")
		return
	}
	writeJSON(w, http.StatusOK, thresholdJSON(th))
}

func (s *Server) handleProductThresholdsPut(w http.ResponseWriter, r *http.Request, code string) {
	if !requireAdmin(w, r, s.Config.DevAuthBypass) {
		return
	}
	var body struct {
		SuccessRateMinPct  float64 `json:"success_rate_min_pct"`
		PendingRateMaxPct  float64 `json:"pending_rate_max_pct"`
		FailRateMaxPct     float64 `json:"fail_rate_max_pct"`
		FailTxnCountMax    uint    `json:"fail_txn_count_max"`
		PendingTxnCountMax uint    `json:"pending_txn_count_max"`
		ErrorEventCountMax uint    `json:"error_event_count_max"`
		AlertEmailEnabled  bool    `json:"alert_email_enabled"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "JSON không hợp lệ")
		return
	}
	by := actorFromRequest(r, s.Config.DevAuthBypass)
	updated := store.ProductThreshold{
		ProductCode:               code,
		SuccessRateMinPct:         body.SuccessRateMinPct,
		PendingRateMaxPct:         body.PendingRateMaxPct,
		FailRateMaxPct:            body.FailRateMaxPct,
		FailTxnCountMax:           body.FailTxnCountMax,
		PendingTxnCountMax:        body.PendingTxnCountMax,
		ErrorEventCountMax:        body.ErrorEventCountMax,
		AlertEmailEnabled:         body.AlertEmailEnabled,
	}
	if err := s.DB.UpdateProductThreshold(r.Context(), updated, by); err != nil {
		writeError(w, http.StatusBadRequest, "update_failed", err.Error())
		return
	}
	th, err := s.DB.GetProductThreshold(r.Context(), code)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, thresholdJSON(th))
}

func (s *Server) handleScopeAutoPut(w http.ResponseWriter, r *http.Request, product, sku string) {
	if !requireAdmin(w, r, s.Config.DevAuthBypass) {
		return
	}
	var body struct {
		AutoAction  string `json:"auto_action"`
		WindowStart string `json:"window_start"`
		WindowEnd   string `json:"window_end"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "JSON không hợp lệ")
		return
	}
	cfg := store.ScopeAutoConfig{
		ProductCode: product,
		SKUCode:     sku,
		AutoAction:  body.AutoAction,
		WindowStart: body.WindowStart,
		WindowEnd:   body.WindowEnd,
	}
	if err := s.DB.UpsertScopeAutoConfig(r.Context(), cfg); err != nil {
		writeError(w, http.StatusBadRequest, "update_failed", err.Error())
		return
	}
	saved, _ := s.DB.GetScopeAutoConfig(r.Context(), product, sku)
	if store.ShouldAutoApplyScope(saved, time.Now()) {
		if sku == "" {
			_ = s.DB.CancelPendingRoutingPlansForProduct(r.Context(), product)
		} else {
			_ = s.DB.CancelPendingRoutingPlansForScope(r.Context(), product, sku)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"product_code":  saved.ProductCode,
		"sku_code":      saved.SKUCode,
		"auto_action":   saved.AutoAction,
		"window_start":  saved.WindowStart,
		"window_end":    saved.WindowEnd,
	})
}

func (s *Server) handleMetricsQuery(w http.ResponseWriter, r *http.Request) {
	product := r.URL.Query().Get("product")
	provider := r.URL.Query().Get("provider")
	sku := r.URL.Query().Get("sku")
	window := r.URL.Query().Get("window")
	if product == "" || provider == "" {
		writeError(w, http.StatusBadRequest, "invalid_input", "Thiếu product hoặc provider")
		return
	}
	out, err := s.Tools.GetMetrics(r.Context(), tools.GetMetricsInput{
		Product: product, Provider: provider, SKU: sku, Window: window,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, "metrics_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleMockStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	count, _ := s.DB.CountMockMetrics(ctx)
	scenario, started, rows, ok, _ := s.DB.GetLastMockRun(ctx)
	settings, _ := s.DB.GetAgentSettings(ctx)
	writeJSON(w, http.StatusOK, map[string]any{
		"enabled":       settings.MockEnabled,
		"scenario":      settings.MockScenario,
		"metric_count":  count,
		"last_run":      started,
		"last_rows":     rows,
		"last_scenario": scenario,
		"last_ok":       ok,
	})
}

func (s *Server) handleMockGenerate(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r, s.Config.DevAuthBypass) {
		return
	}
	gen := mock.NewGenerator(s.DB, time.Now().UnixNano())
	n, err := gen.RunOnce(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "mock_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rows": n})
}

func (s *Server) handleChatPost(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Message   string `json:"message"`
		SessionID string `json:"session_id"`
	}
	if err := decodeJSON(r, &body); err != nil || body.Message == "" {
		writeError(w, http.StatusBadRequest, "invalid_input", "Thiếu message")
		return
	}
	isAdmin := requireAdminSilent(r, s.Config.DevAuthBypass)
	actor := actorFromRequest(r, s.Config.DevAuthBypass)
	reply, err := s.chatAgentReply(r.Context(), body.SessionID, body.Message, actor, isAdmin)
	if err != nil {
		writeError(w, http.StatusBadGateway, "llm_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"session_id": body.SessionID,
		"reply":      reply,
	})
}

func (s *Server) chatReply(ctx context.Context, msg string) string {
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "health") || strings.Contains(lower, "trạng thái") {
		cycle, ok, _ := s.DB.GetLatestCycle(ctx)
		if ok {
			return fmt.Sprintf("Trạng thái hệ thống: %s — %s", cycle.HealthStatus, output.HealthLabel(cycle.HealthStatus))
		}
		return "Hệ thống OK — chưa có chu kỳ phân tích."
	}
	if strings.Contains(lower, "incident") || strings.Contains(lower, "sự cố") {
		rows, _ := s.DB.ListIncidents(ctx, nil, 3, 0)
		if len(rows) == 0 {
			return "Không có sự cố mở gần đây."
		}
		return fmt.Sprintf("Sự cố gần nhất: %s (%s) — %s", rows[0].IncidentID, rows[0].ProductCode, rows[0].Severity)
	}
	return "OpsOne: dùng Dashboard hoặc hỏi về trạng thái / sự cố / routing."
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleRecommendationApprove(w http.ResponseWriter, r *http.Request, id uint64) {
	if !requireAdmin(w, r, s.Config.DevAuthBypass) {
		return
	}
	ctx := r.Context()
	rec, err := s.DB.GetRecommendation(ctx, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Không tìm thấy đề xuất")
		return
	}
	if rec.ActionType != "maintenance" {
		writeError(w, http.StatusConflict, "invalid_type", "Chỉ hỗ trợ duyệt đề xuất bảo trì")
		return
	}
	var reqBody struct {
		StartsAt    string `json:"starts_at"`
		EndsAt      string `json:"ends_at"`
		DurationMin int    `json:"duration_min"`
	}
	_ = decodeJSON(r, &reqBody)
	defaultMin := maintenanceDefaultDurationMin(ctx, s.DB)
	startsAt, endsAt, err := parseMaintenanceWindow(reqBody.StartsAt, reqBody.EndsAt, reqBody.DurationMin, defaultMin)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_window", err.Error())
		return
	}
	by := actorFromRequest(r, s.Config.DevAuthBypass)

	providers, err := s.DB.GetRoutingForScope(ctx, rec.ProductCode, rec.SKUCode)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	targets := maintenanceTargetsFromRecommendation(rec, providers)
	if len(targets) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_input", "Không xác định được provider bảo trì")
		return
	}

	applied, err := applyMaintenanceTargets(
		ctx, s.Tools, rec.ProductCode, rec.SKUCode, by,
		fmt.Sprintf("approve maintenance recommendation #%d by %s", id, by),
		targets, startsAt, endsAt,
	)
	if err != nil {
		writeError(w, http.StatusBadRequest, "maintenance_failed", err.Error())
		return
	}
	if err := s.DB.DeleteRecommendation(ctx, id); err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"recommendation_id": id,
		"applied":           applied,
	})
}

func (s *Server) handleRecommendationReject(w http.ResponseWriter, r *http.Request, id uint64) {
	if !requireAdmin(w, r, s.Config.DevAuthBypass) {
		return
	}
	ctx := r.Context()
	rec, err := s.DB.GetRecommendation(ctx, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Không tìm thấy đề xuất")
		return
	}
	if rec.ActionType == "maintenance" {
		by := actorFromRequest(r, s.Config.DevAuthBypass)
		if err := s.dismissMaintenanceSuggestion(ctx, rec.ProductCode, rec.SKUCode, rec.Detail, by); err != nil {
			writeError(w, http.StatusInternalServerError, "db_error", err.Error())
			return
		}
	}
	if err := s.DB.DeleteRecommendation(ctx, id); err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"recommendation_id": id,
		"action_type":       rec.ActionType,
		"status":            "rejected",
	})
}

func parseUintPath(seg string) (uint64, bool) {
	id, err := strconv.ParseUint(seg, 10, 64)
	return id, err == nil
}
