package tools

import (
	"context"
	"fmt"
	"math"

	"opsone/internal/domain"
	"opsone/internal/store"
)

// RoutingUpdateItem one SKU update for sku_batch.
type RoutingUpdateItem struct {
	SKU     string             `json:"sku"`
	Routing map[string]float64 `json:"routing"`
}

// UpdateRoutingInput §6.7.
type UpdateRoutingInput struct {
	Product     string
	ServiceType string
	Scope       string
	SKU         string
	Routing     map[string]float64
	Updates     []RoutingUpdateItem
	TriggerType string
	ExecutedBy  string
	Reason      string
	CycleID     *uint64
	PlanID      *uint64
	IncidentID  *string
}

// UpdateRoutingOutput §6.7.
type UpdateRoutingOutput struct {
	Applied      bool     `json:"applied"`
	ChangeLogIDs []uint64 `json:"change_log_ids"`
	Product      string   `json:"product"`
	Scope        string   `json:"scope"`
	Meta         Meta     `json:"meta"`
}

// UpdateRouting applies routing change and writes agent_change_log (§6.7).
func (r *Registry) UpdateRouting(ctx context.Context, in UpdateRoutingInput) (UpdateRoutingOutput, error) {
	if in.Product == "" {
		return UpdateRoutingOutput{}, newErr("invalid_input", "Thiếu product")
	}
	p, err := r.DB.GetProductByCode(ctx, in.Product)
	if err != nil {
		return UpdateRoutingOutput{}, newErr("product_not_found", "Không tìm thấy sản phẩm")
	}
	if in.ServiceType != "" && string(p.ServiceType) != in.ServiceType {
		return UpdateRoutingOutput{}, newErr("service_type_mismatch", "service_type không khớp sản phẩm")
	}
	if err := validateScopeForProduct(p, in.Scope); err != nil {
		return UpdateRoutingOutput{}, err
	}

	trigger := in.TriggerType
	if trigger == "" {
		trigger = "auto"
	}
	by := in.ExecutedBy
	if by == "" {
		by = "opsone-agent"
	}

	var changeIDs []uint64
	switch in.Scope {
	case "provider":
		id, err := r.applyScopeRouting(ctx, in.Product, "provider", "", in.Routing, trigger, by, in.Reason, in.CycleID, in.PlanID, in.IncidentID)
		if err != nil {
			return UpdateRoutingOutput{}, err
		}
		changeIDs = append(changeIDs, id)
	case "sku":
		if in.SKU == "" {
			return UpdateRoutingOutput{}, newErr("sku_required", "Thiếu sku cho scope=sku")
		}
		id, err := r.applyScopeRouting(ctx, in.Product, "sku", in.SKU, in.Routing, trigger, by, in.Reason, in.CycleID, in.PlanID, in.IncidentID)
		if err != nil {
			return UpdateRoutingOutput{}, err
		}
		changeIDs = append(changeIDs, id)
	case "sku_batch":
		if len(in.Updates) == 0 {
			return UpdateRoutingOutput{}, newErr("invalid_input", "sku_batch cần mảng updates")
		}
		for _, u := range in.Updates {
			id, err := r.applyScopeRouting(ctx, in.Product, "sku", u.SKU, u.Routing, trigger, by, in.Reason, in.CycleID, in.PlanID, in.IncidentID)
			if err != nil {
				return UpdateRoutingOutput{}, err
			}
			changeIDs = append(changeIDs, id)
		}
	default:
		return UpdateRoutingOutput{}, newErr("invalid_scope", "Scope không hợp lệ")
	}

	ds, _ := r.dataSource(ctx)
	r.markRecoveryStart(ctx, in)
	r.resolveIncidentAfterRouting(ctx, in)
	return UpdateRoutingOutput{
		Applied:      true,
		ChangeLogIDs: changeIDs,
		Product:      in.Product,
		Scope:        in.Scope,
		Meta:         Meta{DataSource: ds},
	}, nil
}

func validateScopeForProduct(p domain.Product, scope string) error {
	switch p.RoutingMode {
	case domain.RoutingProvider:
		if scope != "provider" {
			return newErr("scope_mismatch", "Topup tiền chỉ dùng scope=provider")
		}
	case domain.RoutingSKU:
		if scope != "sku" && scope != "sku_batch" {
			return newErr("scope_mismatch", "Thẻ/topup data chỉ dùng scope=sku hoặc sku_batch")
		}
	}
	return nil
}

func (r *Registry) applyScopeRouting(ctx context.Context, product, scope, sku string,
	target map[string]float64,
	trigger, by, reason string, cycleID, planID *uint64, incidentID *string) (uint64, error) {

	if err := validateRoutingPct(target); err != nil {
		return 0, err
	}

	before, err := r.DB.BuildRoutingSnapshot(ctx, product, scope, sku)
	if err != nil {
		return 0, err
	}

	for provider, pct := range target {
		if err := r.DB.UpdateTrafficPct(ctx, product, sku, provider, pct, by); err != nil {
			return 0, fmt.Errorf("update traffic_pct: %w", err)
		}
	}

	after, err := r.DB.BuildRoutingSnapshot(ctx, product, scope, sku)
	if err != nil {
		return 0, err
	}

	return r.DB.InsertAgentChangeLog(ctx, store.AgentChangeInsert{
		ProductCode:   product,
		Scope:         scope,
		SKUCode:       sku,
		RoutingBefore: before,
		RoutingAfter:  after,
		TriggerType:   trigger,
		ExecutedBy:    by,
		Reason:        reason,
		CycleID:       cycleID,
		RoutingPlanID: planID,
		IncidentID:    incidentID,
	})
}

func (r *Registry) resolveIncidentAfterRouting(ctx context.Context, in UpdateRoutingInput) {
	if in.TriggerType != "admin_approve" && in.TriggerType != "auto" {
		return
	}
	action := "auto"
	if in.TriggerType == "admin_approve" {
		action = "admin_approve"
	}
	by := in.ExecutedBy
	if by == "" {
		by = "opsone-agent"
	}
	if in.IncidentID != nil && *in.IncidentID != "" {
		_ = r.DB.UpdateIncidentHandled(ctx, *in.IncidentID, store.IncidentHandleUpdate{
			Status:           "resolved",
			HandledBy:        by,
			ResolutionAction: action,
		})
		return
	}
	sku := in.SKU
	if in.Scope == "provider" {
		sku = ""
	}
	_ = r.DB.ResolveOpenIncidentForScope(ctx, in.Product, sku, in.CycleID, by, action)
}

func (r *Registry) markRecoveryStart(ctx context.Context, in UpdateRoutingInput) {
	if in.TriggerType != "admin_approve" && in.TriggerType != "auto" && in.TriggerType != "manual_baseline" {
		return
	}
	applyCycle := uint64(0)
	if in.CycleID != nil && *in.CycleID > 0 {
		applyCycle = *in.CycleID
	} else if id, err := r.DB.LatestCompletedCycleID(ctx); err == nil {
		applyCycle = id
	}
	sku := in.SKU
	if in.Scope == "provider" {
		sku = ""
	}
	_ = r.DB.MarkRecoveryStart(ctx, in.Product, sku, applyCycle)
}

func validateRoutingPct(m map[string]float64) error {
	if len(m) == 0 {
		return newErr("invalid_routing", "Routing rỗng")
	}
	var sum float64
	for _, pct := range m {
		if pct < 0 || pct > 100 {
			return newErr("guardrail_violation", "Tỷ lệ phải trong 0–100%")
		}
		sum += pct
	}
	if math.Abs(sum-100) > 0.01 {
		return newErr("invalid_routing", fmt.Sprintf("Tổng routing phải = 100%% (hiện %.1f%%)", sum))
	}
	return nil
}

