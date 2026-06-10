package api

import (
	"fmt"
	"net/http"
	"strings"

	"opsone/internal/agent"
	"opsone/internal/store"
	"opsone/internal/tools"
)

func (s *Server) handleScopeRoutingApprove(w http.ResponseWriter, r *http.Request, product, sku string) {
	if !requireAdmin(w, r, s.Config.DevAuthBypass) {
		return
	}
	ctx := r.Context()
	var body struct {
		ProposedPct map[string]float64    `json:"proposed_pct"`
		Plan        agent.RoutingPlanJSON `json:"plan"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "JSON không hợp lệ")
		return
	}
	routing := body.ProposedPct
	scope := body.Plan.Scope
	if scope == "" {
		scope = "sku"
		if sku == "" {
			scope = "provider"
		}
	}
	if len(routing) == 0 && len(body.Plan.Proposed) > 0 {
		routing = body.Plan.Proposed
	}
	if len(routing) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_input", "Thiếu proposed_pct")
		return
	}
	prod, err := s.DB.GetProductByCode(ctx, product)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Không tìm thấy sản phẩm")
		return
	}
	by := actorFromRequest(r, s.Config.DevAuthBypass)
	incidentID, _ := s.DB.FindOpenIncidentForScope(ctx, product, sku, nil)
	var incPtr *string
	if incidentID != "" {
		incPtr = &incidentID
	}
	planID, err := s.DB.InsertRoutingPlan(ctx, nil, product, scope, sku, body.Plan, "pending_approve")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	out, err := s.Tools.UpdateRouting(ctx, tools.UpdateRoutingInput{
		Product:     product,
		ServiceType: string(prod.ServiceType),
		Scope:       scope,
		SKU:         sku,
		Routing:     routing,
		TriggerType: "admin_approve",
		ExecutedBy:  by,
		Reason:      fmt.Sprintf("approve routing scope %s/%s", product, sku),
		PlanID:      &planID,
		IncidentID:  incPtr,
	})
	if err != nil {
		_ = s.DB.UpdateRoutingPlanStatus(ctx, planID, "cancelled", by)
		writeError(w, http.StatusBadRequest, "routing_failed", err.Error())
		return
	}
	if err := s.DB.UpdateRoutingPlanStatus(ctx, planID, "executed", by); err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"plan_id": planID, "applied": out.Applied, "change_log_ids": out.ChangeLogIDs})
}

func (s *Server) handleScopeRoutingApply(w http.ResponseWriter, r *http.Request, product, sku string) {
	if !requireAdmin(w, r, s.Config.DevAuthBypass) {
		return
	}
	ctx := r.Context()
	var body struct {
		ProposedPct map[string]float64 `json:"proposed_pct"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "JSON không hợp lệ")
		return
	}
	if len(body.ProposedPct) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_input", "Thiếu proposed_pct")
		return
	}
	prod, err := s.DB.GetProductByCode(ctx, product)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Không tìm thấy sản phẩm")
		return
	}
	scope := "sku"
	if sku == "" {
		scope = "provider"
	}
	by := actorFromRequest(r, s.Config.DevAuthBypass)
	out, err := s.Tools.UpdateRouting(ctx, tools.UpdateRoutingInput{
		Product:     product,
		ServiceType: string(prod.ServiceType),
		Scope:       scope,
		SKU:         sku,
		Routing:     body.ProposedPct,
		TriggerType: "manual_temp",
		ExecutedBy:  by,
		Reason:      fmt.Sprintf("manual routing restore %s/%s", product, sku),
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, "routing_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"applied": out.Applied, "change_log_ids": out.ChangeLogIDs})
}

func (s *Server) handleScopeRoutingReject(w http.ResponseWriter, r *http.Request, product, sku string) {
	if !requireAdmin(w, r, s.Config.DevAuthBypass) {
		return
	}
	ctx := r.Context()
	var body struct {
		Plan agent.RoutingPlanJSON `json:"plan"`
	}
	_ = decodeJSON(r, &body)
	scope := body.Plan.Scope
	if scope == "" {
		scope = "sku"
		if sku == "" {
			scope = "provider"
		}
	}
	if len(body.Plan.Proposed) == 0 {
		body.Plan.Product = product
		body.Plan.SKU = sku
		body.Plan.Scope = scope
	}
	planID, err := s.DB.InsertRoutingPlan(ctx, nil, product, scope, sku, body.Plan, "rejected")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	by := actorFromRequest(r, s.Config.DevAuthBypass)
	if incID, findErr := s.DB.FindOpenIncidentForScope(ctx, product, sku, nil); findErr == nil && incID != "" {
		_ = s.DB.UpdateIncidentHandled(ctx, incID, store.IncidentHandleUpdate{
			Status:           "acknowledged",
			HandledBy:        by,
			ResolutionAction: "admin_reject",
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"plan_id": planID, "status": "rejected"})
}

func (s *Server) handleScopeMaintenanceApprove(w http.ResponseWriter, r *http.Request, product, sku string) {
	if !requireAdmin(w, r, s.Config.DevAuthBypass) {
		return
	}
	ctx := r.Context()
	var body struct {
		Reason       string `json:"reason"`
		ProviderCode string `json:"provider_code"`
		StartsAt     string `json:"starts_at"`
		EndsAt       string `json:"ends_at"`
		DurationMin  int    `json:"duration_min"`
	}
	_ = decodeJSON(r, &body)
	detail := body.Reason
	if detail == "" {
		detail = "Đề xuất bảo trì — admin duyệt"
	}
	if body.ProviderCode != "" && !isSkuWideMaintenanceReason(detail) &&
		!strings.Contains(strings.ToUpper(detail), strings.ToUpper(body.ProviderCode)) {
		detail = body.ProviderCode + " — " + detail
	}
	id, err := s.DB.InsertRecommendationReturningID(ctx, nil, nil, product, sku, "maintenance", detail)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	rec, err := s.DB.GetRecommendation(ctx, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Không tìm thấy đề xuất")
		return
	}
	startsAt, endsAt, err := parseMaintenanceWindow(body.StartsAt, body.EndsAt, body.DurationMin)
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
	targets := maintenanceTargetsForScope(rec.Detail, rec.ProviderCode, providers)
	if len(targets) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_input", "Không xác định được provider bảo trì")
		return
	}
	applied, err := applyMaintenanceTargets(
		ctx, s.Tools, rec.ProductCode, rec.SKUCode, by,
		fmt.Sprintf("approve maintenance scope %s/%s by %s", product, sku, by),
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

func (s *Server) handleScopeMaintenanceReject(w http.ResponseWriter, r *http.Request, product, sku string) {
	if !requireAdmin(w, r, s.Config.DevAuthBypass) {
		return
	}
	ctx := r.Context()
	var body struct {
		Reason string `json:"reason"`
	}
	_ = decodeJSON(r, &body)
	detail := body.Reason
	if detail == "" {
		detail = "Đề xuất bảo trì"
	}
	if err := s.DB.InsertRecommendation(ctx, nil, nil, product, sku, "maintenance", "DISMISSED: "+detail); err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"product_code": product, "sku_code": sku, "status": "rejected"})
}

func (s *Server) handleScopeMaintenanceCancel(w http.ResponseWriter, r *http.Request, product, sku string) {
	if !requireAdmin(w, r, s.Config.DevAuthBypass) {
		return
	}
	ctx := r.Context()
	var body struct {
		MaintenanceIDs []string `json:"maintenance_ids"`
	}
	_ = decodeJSON(r, &body)
	by := actorFromRequest(r, s.Config.DevAuthBypass)
	var n int64
	var err error
	if len(body.MaintenanceIDs) > 0 {
		n, err = s.DB.CancelMaintenanceByIDs(ctx, body.MaintenanceIDs, by)
	} else {
		n, err = s.DB.CancelActiveMaintenanceForSKU(ctx, product, sku, by)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	if n == 0 {
		writeError(w, http.StatusNotFound, "not_found", "Không có cửa sổ bảo trì đang hoạt động")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"product_code": product,
		"sku_code":     sku,
		"status":       "cancelled",
		"cancelled":    n,
	})
}

func (s *Server) handleScopeMaintenanceExtend(w http.ResponseWriter, r *http.Request, product, sku string) {
	if !requireAdmin(w, r, s.Config.DevAuthBypass) {
		return
	}
	ctx := r.Context()
	var body struct {
		StartsAt    string `json:"starts_at"`
		EndsAt      string `json:"ends_at"`
		DurationMin int    `json:"duration_min"`
	}
	_ = decodeJSON(r, &body)
	startsAt, endsAt, err := parseMaintenanceWindow(body.StartsAt, body.EndsAt, body.DurationMin)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_window", err.Error())
		return
	}
	n, err := s.DB.UpdateActiveMaintenanceTimesForSKU(ctx, product, sku, startsAt, endsAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	if n == 0 {
		writeError(w, http.StatusNotFound, "not_found", "Không có cửa sổ bảo trì đang hoạt động")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"product_code": product,
		"sku_code":     sku,
		"status":       "updated",
		"updated":      n,
		"starts_at":    startsAt,
		"ends_at":      endsAt,
	})
}
