package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"opsone/internal/agent"
	"opsone/internal/tools"
)

// getPendingSuggestionsForSSE returns pending routing plans and maintenance suggestions.
// Used to trigger auto-chat notifications.
func (s *Server) getPendingSuggestionsForSSE(ctx context.Context) (map[string]any, error) {
	plans, err := s.DB.ListPendingRoutingPlansPerScope(ctx)
	if err != nil {
		return nil, err
	}
	maint, err := s.DB.ListPendingMaintenanceByScope(ctx)
	if err != nil {
		return nil, err
	}

	result := map[string]any{
		"routing_plans":           make([]map[string]any, 0),
		"maintenance_suggestions": make([]map[string]any, 0),
		"has_suggestions":         len(plans) > 0 || len(maint) > 0,
		"plan_count":              len(plans),
		"maintenance_count":       len(maint),
	}

	routing := make([]map[string]any, 0, len(plans))
	for _, p := range plans {
		item := map[string]any{
			"product_code": p.ProductCode,
			"sku_code":     p.SKUCode,
			"status":       p.Status,
			"plan_id":      p.ID,
		}
		var parsed struct {
			ProposedPct map[string]float64 `json:"proposed_pct"`
			ReasonVI    string             `json:"reason_vi"`
		}
		if json.Unmarshal(p.PlanJSON, &parsed) == nil {
			if len(parsed.ProposedPct) > 0 {
				item["proposed_pct"] = parsed.ProposedPct
			}
			if parsed.ReasonVI != "" {
				item["reason_vi"] = parsed.ReasonVI
			}
		}
		routing = append(routing, item)
	}
	result["routing_plans"] = routing

	maintItems := make([]map[string]any, 0, len(maint))
	for _, rec := range maint {
		maintItems = append(maintItems, map[string]any{
			"product_code":  rec.ProductCode,
			"sku_code":      rec.SKUCode,
			"provider_code": rec.ProviderCode,
			"detail":        rec.Detail,
			"id":            rec.ID,
		})
	}
	result["maintenance_suggestions"] = maintItems

	return result, nil
}

// formatSuggestionSystemMessage builds a system message for auto-opened chat with pending suggestions.
func formatSuggestionSystemMessage(suggestions map[string]any) string {
	if !getBool(suggestions, "has_suggestions") {
		return ""
	}

	var lines []string
	lines = append(lines, "📢 **Có việc mới cần xử lý!**\n")

	// Add routing plans
	if plans, ok := suggestions["routing_plans"].([]map[string]any); ok && len(plans) > 0 {
		lines = append(lines, "🔄 **Đề xuất Routing:** (Thay đổi phân phối traffic)")
		for i, p := range plans {
			if i >= 3 {
				lines = append(lines, fmt.Sprintf("   • ...và %d kế hoạch khác", len(plans)-3))
				break
			}
			productCode := strAny(p, "product_code")
			skuCode := strAny(p, "sku_code")
			reasonVI := strAny(p, "reason_vi")
			pctStr := ""
			if proposed, ok := p["proposed_pct"].(map[string]any); ok {
				parts := make([]string, 0)
				for provider := range proposed {
					parts = append(parts, fmt.Sprintf("%s: %.0f%%", provider, getFloat(proposed, provider)))
				}
				if len(parts) > 0 {
					pctStr = " → " + strings.Join(parts, " / ")
				}
			}
			detail := fmt.Sprintf("%s/%s%s", productCode, skuCode, pctStr)
			if reasonVI != "" {
				detail += fmt.Sprintf(" (%s)", reasonVI)
			}
			lines = append(lines, fmt.Sprintf("   • %s", detail))
		}
		lines = append(lines, "")
	}

	// Add maintenance suggestions
	if items, ok := suggestions["maintenance_suggestions"].([]map[string]any); ok && len(items) > 0 {
		lines = append(lines, "🔧 **Đề xuất Bảo trì:**")
		for i, m := range items {
			if i >= 3 {
				lines = append(lines, fmt.Sprintf("   • ...và %d bảo trì khác", len(items)-3))
				break
			}
			productCode := strAny(m, "product_code")
			skuCode := strAny(m, "sku_code")
			detail := strAny(m, "detail")
			detailStr := fmt.Sprintf("%s/%s", productCode, skuCode)
			if detail != "" {
				detailStr += fmt.Sprintf(" — %s", detail)
			}
			lines = append(lines, fmt.Sprintf("   • %s", detailStr))
		}
		lines = append(lines, "")
	}

	lines = append(lines, "💡 **Hành động:**")
	lines = append(lines, "   • Gõ \"xem pending\" để xem chi tiết")
	lines = append(lines, "   • Admin: duyệt/từ chối ngay hoặc vào Dashboard")

	return strings.Join(lines, "\n")
}

func getBool(m map[string]any, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

func getFloat(m map[string]any, key string) float64 {
	if v, ok := m[key].(float64); ok {
		return v
	}
	return 0
}

func (s *Server) chatListPendingActions(ctx context.Context) (any, error) {
	plans, err := s.DB.ListPendingRoutingPlansPerScope(ctx)
	if err != nil {
		return nil, err
	}
	maint, err := s.DB.ListPendingMaintenanceByScope(ctx)
	if err != nil {
		return nil, err
	}
	routing := make([]map[string]any, 0, len(plans))
	for _, p := range plans {
		item := map[string]any{
			"type":         "routing_plan",
			"plan_id":      p.ID,
			"product_code": p.ProductCode,
			"sku_code":     p.SKUCode,
			"status":       p.Status,
		}
		var parsed struct {
			ProposedPct map[string]float64 `json:"proposed_pct"`
			ReasonVI    string             `json:"reason_vi"`
		}
		if json.Unmarshal(p.PlanJSON, &parsed) == nil {
			if len(parsed.ProposedPct) > 0 {
				item["proposed_pct"] = parsed.ProposedPct
			}
			if parsed.ReasonVI != "" {
				item["reason_vi"] = parsed.ReasonVI
			}
		}
		routing = append(routing, item)
	}
	maintItems := make([]map[string]any, 0, len(maint))
	for key, rec := range maint {
		maintItems = append(maintItems, map[string]any{
			"type":          "maintenance_suggestion",
			"scope_key":     key,
			"product_code":  rec.ProductCode,
			"sku_code":      rec.SKUCode,
			"provider_code": rec.ProviderCode,
			"detail":        rec.Detail,
			"id":            rec.ID,
		})
	}
	return map[string]any{
		"routing_plans":           routing,
		"maintenance_suggestions": maintItems,
	}, nil
}

func (s *Server) chatApproveRoutingPlan(ctx context.Context, planID uint64, actor string, proposedPct map[string]float64) (string, error) {
	plan, err := s.DB.GetRoutingPlan(ctx, planID)
	if err != nil {
		return "", fmt.Errorf("không tìm thấy kế hoạch routing #%d", planID)
	}
	if plan.Status != "pending_approve" && plan.Status != "draft" {
		return "", fmt.Errorf("kế hoạch #%d không ở trạng thái chờ duyệt", planID)
	}
	var parsed struct {
		Scope       string             `json:"scope"`
		SKU         string             `json:"sku"`
		ProposedPct map[string]float64 `json:"proposed_pct"`
	}
	if err := json.Unmarshal(plan.PlanJSON, &parsed); err != nil {
		return "", fmt.Errorf("plan_json không hợp lệ")
	}
	routing := parsed.ProposedPct
	if len(proposedPct) > 0 {
		routing = proposedPct
	}
	if len(routing) == 0 {
		return "", fmt.Errorf("thiếu proposed_pct")
	}
	scope := plan.Scope
	if scope == "" {
		scope = parsed.Scope
	}
	sku := plan.SKUCode
	if sku == "" {
		sku = parsed.SKU
	}
	var cycleID *uint64
	if plan.CycleID.Valid {
		c := uint64(plan.CycleID.Int64)
		cycleID = &c
	}
	incidentID, _ := s.DB.FindOpenIncidentForScope(ctx, plan.ProductCode, sku, cycleID)
	var incPtr *string
	if incidentID != "" {
		incPtr = &incidentID
	}
	pid := planID
	out, err := s.Tools.UpdateRouting(ctx, tools.UpdateRoutingInput{
		Product:     plan.ProductCode,
		Scope:       scope,
		SKU:         sku,
		Routing:     routing,
		TriggerType: "admin_approve",
		ExecutedBy:  actor,
		Reason:      fmt.Sprintf("chat approve routing plan #%d", planID),
		PlanID:      &pid,
		CycleID:     cycleID,
		IncidentID:  incPtr,
	})
	if err != nil {
		return "", err
	}
	if err := s.DB.UpdateRoutingPlanStatus(ctx, planID, "executed", actor); err != nil {
		return "", err
	}
	return fmt.Sprintf("Đã duyệt kế hoạch routing #%d — áp dụng %v", planID, out.Applied), nil
}

func (s *Server) chatApproveScopeRouting(ctx context.Context, product, sku, actor string, proposedPct map[string]float64) (string, error) {
	if len(proposedPct) == 0 {
		plan, ok, err := s.DB.GetPendingRoutingPlanForScope(ctx, product, sku)
		if err != nil {
			return "", err
		}
		if !ok {
			return "", fmt.Errorf("không có kế hoạch routing chờ duyệt cho %s/%s", product, sku)
		}
		return s.chatApproveRoutingPlan(ctx, plan.ID, actor, nil)
	}
	prod, err := s.DB.GetProductByCode(ctx, product)
	if err != nil {
		return "", fmt.Errorf("không tìm thấy sản phẩm %s", product)
	}
	scope := "sku"
	if sku == "" {
		scope = "provider"
	}
	var plan agent.RoutingPlanJSON
	planID, err := s.DB.InsertRoutingPlan(ctx, nil, product, scope, sku, plan, "pending_approve")
	if err != nil {
		return "", err
	}
	out, err := s.Tools.UpdateRouting(ctx, tools.UpdateRoutingInput{
		Product:     product,
		ServiceType: string(prod.ServiceType),
		Scope:       scope,
		SKU:         sku,
		Routing:     proposedPct,
		TriggerType: "admin_approve",
		ExecutedBy:  actor,
		Reason:      fmt.Sprintf("chat approve routing %s/%s", product, sku),
		PlanID:      &planID,
	})
	if err != nil {
		_ = s.DB.UpdateRoutingPlanStatus(ctx, planID, "cancelled", actor)
		return "", err
	}
	if err := s.DB.UpdateRoutingPlanStatus(ctx, planID, "executed", actor); err != nil {
		return "", err
	}
	return fmt.Sprintf("Đã duyệt routing %s/%s — áp dụng %v", product, sku, out.Applied), nil
}

func (s *Server) chatRejectRoutingPlan(ctx context.Context, planID uint64, actor string) (string, error) {
	plan, err := s.DB.GetRoutingPlan(ctx, planID)
	if err != nil {
		return "", fmt.Errorf("không tìm thấy kế hoạch #%d", planID)
	}
	if plan.Status != "pending_approve" && plan.Status != "draft" {
		return "", fmt.Errorf("kế hoạch #%d không chờ duyệt", planID)
	}
	if err := s.DB.UpdateRoutingPlanStatus(ctx, planID, "rejected", actor); err != nil {
		return "", err
	}
	return fmt.Sprintf("Đã từ chối kế hoạch routing #%d (%s/%s)", planID, plan.ProductCode, plan.SKUCode), nil
}

func (s *Server) chatRejectScopeRouting(ctx context.Context, product, sku, actor string) (string, error) {
	plan, ok, err := s.DB.GetPendingRoutingPlanForScope(ctx, product, sku)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("không có kế hoạch chờ duyệt cho %s/%s", product, sku)
	}
	return s.chatRejectRoutingPlan(ctx, plan.ID, actor)
}

func (s *Server) chatApproveScopeMaintenance(ctx context.Context, product, sku, actor string, startsAt, endsAt string, durationMin int) (string, error) {
	defaultMin := maintenanceDefaultDurationMin(ctx, s.DB)
	st, en, err := parseMaintenanceWindow(startsAt, endsAt, durationMin, defaultMin)
	if err != nil {
		return "", err
	}
	rec, ok, err := s.DB.LatestPendingMaintenanceForScope(ctx, product, sku)
	detail := "Đề xuất bảo trì — chat duyệt"
	providerCode := ""
	if err == nil && ok {
		detail = rec.Detail
		providerCode = rec.ProviderCode
	}
	providers, err := s.DB.GetRoutingForScope(ctx, product, sku)
	if err != nil {
		return "", err
	}
	targets := maintenanceTargetsForScope(detail, providerCode, providers)
	if len(targets) == 0 {
		return "", fmt.Errorf("không xác định được provider bảo trì")
	}
	applied, err := applyMaintenanceTargets(
		ctx, s.Tools, product, sku, actor,
		fmt.Sprintf("chat approve maintenance %s/%s", product, sku),
		targets, st, en,
	)
	if err != nil {
		return "", err
	}
	if ok && rec.ID > 0 {
		_ = s.DB.DeleteRecommendation(ctx, rec.ID)
	}
	return fmt.Sprintf("Đã duyệt bảo trì %s/%s (%d provider) — %s → %s", product, sku, len(applied), st.Format("15:04"), en.Format("15:04")), nil
}

func (s *Server) chatRejectScopeMaintenance(ctx context.Context, product, sku, actor string) (string, error) {
	rec, ok, err := s.DB.LatestPendingMaintenanceForScope(ctx, product, sku)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("không có đề xuất bảo trì chờ duyệt cho %s/%s", product, sku)
	}
	if err := s.DB.DeleteRecommendation(ctx, rec.ID); err != nil {
		return "", err
	}
	_ = actor
	return fmt.Sprintf("Đã từ chối đề xuất bảo trì %s/%s", product, sku), nil
}
