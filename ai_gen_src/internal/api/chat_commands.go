package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"opsone/internal/chatresolve"
	"opsone/internal/domain"
	"opsone/internal/store"
	"opsone/internal/tools"
)

func (s *Server) tryChatCommandReply(
	ctx context.Context,
	sessionID, userMsg, actor string,
	isAdmin bool,
) (string, bool) {
	action, slots := chatresolve.DetectUIAction(userMsg)
	if action == "" {
		return "", false
	}
	reply, err := s.executeUIAction(ctx, sessionID, action, slots, userMsg, actor, isAdmin)
	if err != nil {
		return formatChatActionReply("Không thực hiện được.", err.Error(), ""), true
	}
	return reply, true
}

func (s *Server) tryChatListPendingReply(ctx context.Context, sessionID string, isAdmin bool) (string, bool) {
	raw, err := s.chatListPendingActions(ctx)
	if err != nil {
		return formatChatActionReply("Không tra được pending.", err.Error(), ""), true
	}
	data, _ := raw.(map[string]any)
	if data == nil {
		return formatChatActionReply("Không có việc chờ duyệt.", "", "Hỏi metric hoặc bảo trì nếu cần."), true
	}
	focus := chatSessionFocusFromPending(data)
	if focus.Kind != "" {
		chatSessionFocusSet(sessionID, focus)
	}
	body := formatPendingList(data)
	if body == "" {
		return formatChatActionReply("Không có việc chờ duyệt.", "", ""), true
	}
	hint := ""
	if isAdmin {
		hint = "Nói \"duyệt\" hoặc \"từ chối\" để xử lý mục đầu tiên."
	}
	return formatChatActionReply("Việc chờ duyệt:", body, hint), true
}

func formatPendingList(data map[string]any) string {
	var lines []string
	if plans, ok := data["routing_plans"].([]map[string]any); ok {
		for i, p := range plans {
			if i >= 5 {
				break
			}
			lines = append(lines, fmt.Sprintf("• Routing #%v — %s/%s",
				p["plan_id"], strAny(p, "product_code"), strAny(p, "sku_code")))
		}
	}
	if items, ok := data["maintenance_suggestions"].([]map[string]any); ok {
		for i, m := range items {
			if i >= 5 {
				break
			}
			lines = append(lines, fmt.Sprintf("• Bảo trì — %s/%s",
				strAny(m, "product_code"), strAny(m, "sku_code")))
		}
	}
	return strings.Join(lines, "\n")
}

func (s *Server) tryChatApproveRejectReply(
	ctx context.Context,
	sessionID, userMsg, actor string,
	isAdmin bool,
) (string, bool) {
	if !isAdmin {
		return formatChatActionReply(
			"Không thực hiện được.",
			"Bạn cần quyền Admin.",
			"Mở Dashboard để duyệt.",
		), true
	}

	reject := chatresolve.IsRejectShortCommand(userMsg) ||
		strings.Contains(chatresolve.NormalizeKey(userMsg), "tu choi") ||
		strings.Contains(chatresolve.NormalizeKey(userMsg), "reject")

	product := chatresolve.ExtractProductFromText(userMsg)
	sku := chatresolve.NormalizeSKU(chatresolve.ExtractSKUFromText(userMsg))

	var focus chatPendingFocus
	var hasFocus bool
	if product == "" {
		focus, hasFocus = chatSessionFocusGet(sessionID)
	} else {
		focus = chatPendingFocus{Product: product, SKU: sku}
		if strings.Contains(chatresolve.NormalizeKey(userMsg), "bao tri") ||
			strings.Contains(chatresolve.NormalizeKey(userMsg), "bảo trì") ||
			strings.Contains(chatresolve.NormalizeKey(userMsg), "maintenance") {
			focus.Kind = "maintenance_suggestion"
		} else {
			focus.Kind = "routing_plan"
		}
		hasFocus = true
	}

	if !hasFocus {
		raw, err := s.chatListPendingActions(ctx)
		if err != nil {
			return formatChatActionReply("Không tra được pending.", err.Error(), ""), true
		}
		data, _ := raw.(map[string]any)
		focus = chatSessionFocusFromPending(data)
		if focus.Kind == "" {
			return formatChatActionReply(
				"Không thực hiện được.",
				"Không có đề xuất pending.",
				"Nói \"xem pending\" để liệt kê.",
			), true
		}
		chatSessionFocusSet(sessionID, focus)
	}

	var msg string
	var err error
	if reject {
		msg, err = s.chatRejectFocus(ctx, focus, actor)
	} else {
		msg, err = s.chatApproveFocus(ctx, focus, actor)
	}
	if err != nil {
		return formatChatActionReply("Không thực hiện được.", err.Error(), ""), true
	}
	chatSessionFocusClear(sessionID)
	return formatChatActionReply(msg, "", ""), true
}

func (s *Server) chatApproveFocus(ctx context.Context, f chatPendingFocus, actor string) (string, error) {
	switch f.Kind {
	case "maintenance_suggestion":
		return s.chatApproveScopeMaintenance(ctx, f.Product, f.SKU, actor, "", "", 0)
	default:
		if f.PlanID > 0 {
			return s.chatApproveRoutingPlan(ctx, f.PlanID, actor, nil)
		}
		return s.chatApproveScopeRouting(ctx, f.Product, f.SKU, actor, nil)
	}
}

func (s *Server) chatRejectFocus(ctx context.Context, f chatPendingFocus, actor string) (string, error) {
	switch f.Kind {
	case "maintenance_suggestion":
		return s.chatRejectScopeMaintenance(ctx, f.Product, f.SKU, actor)
	default:
		if f.PlanID > 0 {
			return s.chatRejectRoutingPlan(ctx, f.PlanID, actor)
		}
		return s.chatRejectScopeRouting(ctx, f.Product, f.SKU, actor)
	}
}

func (s *Server) tryChatReopenReply(ctx context.Context, userMsg, actor string, isAdmin bool) (string, bool) {
	if !isAdmin {
		return formatChatActionReply("Không thực hiện được.", "Cần quyền Admin.", ""), true
	}
	product := chatresolve.ExtractProductFromText(userMsg)
	if product == "" {
		return formatChatActionReply("Không thực hiện được.", "Thiếu tên dịch vụ.", "Ví dụ: mở lại dịch vụ Garena."), true
	}
	if chatresolve.IsAmbiguousCarrierProduct(userMsg, product) {
		return formatChatActionReply(
			"Chưa rõ loại dịch vụ nào.",
			"Vui lòng nói rõ: topup, thẻ, hay data?",
			"Ví dụ: \"mở lại topup Mobifone\" / \"mở lại thẻ Mobifone\"",
		), true
	}
	sku := chatresolve.NormalizeSKU(chatresolve.ExtractSKUFromText(userMsg))
	provider := chatresolve.ExtractProviderFromText(userMsg)
	msg, err := s.chatReopenService(ctx, product, sku, provider, actor)
	if err != nil {
		return formatChatActionReply("Không thực hiện được.", err.Error(), ""), true
	}
	return formatChatActionReply(msg, "", ""), true
}

func (s *Server) tryChatSetMaintenanceReply(ctx context.Context, userMsg, actor string, isAdmin bool) (string, bool) {
	if !isAdmin {
		return formatChatActionReply("Không thực hiện được.", "Cần quyền Admin.", ""), true
	}
	product := chatresolve.ExtractProductFromText(userMsg)
	if product == "" {
		return formatChatActionReply("Không thực hiện được.", "Thiếu tên dịch vụ.", ""), true
	}
	if chatresolve.IsAmbiguousCarrierProduct(userMsg, product) {
		return formatChatActionReply(
			"Chưa rõ loại dịch vụ nào.",
			"Vui lòng nói rõ: topup, thẻ, hay data?",
			"Ví dụ: \"bảo trì topup Mobifone\" / \"bảo trì thẻ Mobifone\" / \"bảo trì data Mobifone\"",
		), true
	}
	sku := chatresolve.NormalizeSKU(chatresolve.ExtractSKUFromText(userMsg))
	provider := chatresolve.ExtractProviderFromText(userMsg)
	dur := chatresolve.ExtractDurationMinutes(userMsg)
	msg, err := s.chatSetMaintenance(ctx, product, sku, provider, dur, actor, userMsg)
	if err != nil {
		return formatChatActionReply("Không thực hiện được.", err.Error(), ""), true
	}
	return formatChatActionReply(msg, "", ""), true
}

func (s *Server) tryChatSetScopeAutoReply(ctx context.Context, userMsg, actor string, isAdmin bool) (string, bool) {
	if !isAdmin {
		return formatChatActionReply("Không thực hiện được.", "Cần quyền Admin.", ""), true
	}
	_ = actor
	product := chatresolve.ExtractProductFromText(userMsg)
	if product == "" {
		return formatChatActionReply("Không thực hiện được.", "Thiếu tên dịch vụ.", ""), true
	}
	if chatresolve.IsAmbiguousCarrierProduct(userMsg, product) {
		return formatChatActionReply(
			"Chưa rõ loại dịch vụ nào.",
			"Vui lòng nói rõ: topup, thẻ, hay data?",
			"Ví dụ: \"tự động topup Mobifone\" / \"chỉ đề xuất thẻ Mobifone\"",
		), true
	}
	sku := chatresolve.NormalizeSKU(chatresolve.ExtractSKUFromText(userMsg))
	mode := chatresolve.ParseScopeAutoMode(userMsg)
	if mode == "" {
		return formatChatActionReply(
			"Không thực hiện được.",
			"Chưa rõ chế độ.",
			"Nói: Tự động / Tự động theo khung giờ / Chỉ đề xuất.",
		), true
	}
	msg, err := s.chatSetScopeAuto(ctx, product, sku, mode, userMsg)
	if err != nil {
		return formatChatActionReply("Không thực hiện được.", err.Error(), ""), true
	}
	return formatChatActionReply(msg, "", ""), true
}

// allScopesForFilter collects (product, sku) pairs to process for a global chat command,
// applying the service-type filter(s) and handling provider-mode products (sku="").
// filters is nil/empty → all products; otherwise include products whose ServiceType is in filters.
func (s *Server) allScopesForFilter(ctx context.Context, products []domain.Product, filters []string) [][2]string {
	filterSet := make(map[string]bool, len(filters))
	for _, f := range filters {
		filterSet[f] = true
	}
	var scopes [][2]string
	for _, p := range products {
		if len(filterSet) > 0 && !filterSet[string(p.ServiceType)] {
			continue
		}
		if p.RoutingMode == domain.RoutingProvider {
			scopes = append(scopes, [2]string{p.ProductCode, ""})
			continue
		}
		skus, err := s.DB.ListSKUsForProduct(ctx, p.ProductCode, true)
		if err != nil {
			continue
		}
		for _, sk := range skus {
			scopes = append(scopes, [2]string{p.ProductCode, sk.SKUCode})
		}
	}
	return scopes
}

// allServicesScope builds a human-readable scope label from a filter list.
func allServicesScope(filters []string) string {
	labels := map[string]string{
		"card":       "thẻ",
		"topup":      "topup",
		"topup_data": "data",
	}
	if len(filters) == 0 {
		return "toàn hệ thống"
	}
	if len(filters) == 1 {
		if l, ok := labels[filters[0]]; ok {
			return "tất cả " + l
		}
	}
	parts := make([]string, 0, len(filters))
	for _, f := range filters {
		if l, ok := labels[f]; ok {
			parts = append(parts, l)
		}
	}
	return "tất cả " + strings.Join(parts, " và ")
}

func (s *Server) tryChatSetAllMaintenanceReply(ctx context.Context, userMsg, actor string, isAdmin bool) (string, bool) {
	if !isAdmin {
		return formatChatActionReply("Không thực hiện được.", "Cần quyền Admin.", ""), true
	}
	products, err := s.DB.ListProducts(ctx, true)
	if err != nil {
		return formatChatActionReply("Không thực hiện được.", err.Error(), ""), true
	}
	dur := chatresolve.ExtractDurationMinutes(userMsg)
	defaultMin := maintenanceDefaultDurationMin(ctx, s.DB)
	if dur <= 0 {
		dur = defaultMin
	}
	if dur <= 0 {
		dur = 60
	}
	startsAt, endsAt, err := parseMaintenanceWindow("", "", dur, defaultMin)
	if err != nil {
		return formatChatActionReply("Không thực hiện được.", err.Error(), ""), true
	}

	serviceFilters := chatresolve.ExtractAllServicesFilters(userMsg)
	scopes := s.allScopesForFilter(ctx, products, serviceFilters)
	if len(scopes) == 0 {
		return formatChatActionReply("Không bật được bảo trì.", "Không có dịch vụ nào phù hợp.", ""), true
	}

	// Run each scope concurrently; limit to 8 parallel DB connections.
	const maxConcurrent = 8
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	var totalProviders int64

	for _, sc := range scopes {
		wg.Add(1)
		sem <- struct{}{}
		go func(product, sku string) {
			defer wg.Done()
			defer func() { <-sem }()
			n, err := s.chatSetMaintenanceScopeAll(ctx, product, sku, startsAt, endsAt, actor)
			if err == nil {
				atomic.AddInt64(&totalProviders, int64(n))
			}
		}(sc[0], sc[1])
	}
	wg.Wait()

	if totalProviders == 0 {
		return formatChatActionReply("Không bật được bảo trì.", "Không có dịch vụ nào hoạt động.", ""), true
	}
	return formatChatActionReply(
		fmt.Sprintf("Đã bật bảo trì %s.", allServicesScope(serviceFilters)),
		fmt.Sprintf("%d provider · %s → %s.", totalProviders, startsAt.Format("15:04"), endsAt.Format("15:04")),
		"",
	), true
}

func (s *Server) tryChatReopenAllServicesReply(ctx context.Context, userMsg, actor string, isAdmin bool) (string, bool) {
	if !isAdmin {
		return formatChatActionReply("Không thực hiện được.", "Cần quyền Admin.", ""), true
	}
	products, err := s.DB.ListProducts(ctx, true)
	if err != nil {
		return formatChatActionReply("Không thực hiện được.", err.Error(), ""), true
	}

	serviceFilters := chatresolve.ExtractAllServicesFilters(userMsg)
	scopes := s.allScopesForFilter(ctx, products, serviceFilters)

	// Run each scope concurrently; limit to 8 parallel DB connections.
	const maxConcurrent = 8
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	var totalCancelled int64

	for _, sc := range scopes {
		wg.Add(1)
		sem <- struct{}{}
		go func(product, sku string) {
			defer wg.Done()
			defer func() { <-sem }()
			n, _ := s.DB.CancelActiveMaintenanceForSKU(ctx, product, sku, actor)
			if n > 0 {
				atomic.AddInt64(&totalCancelled, n)
				_ = s.DB.CancelPendingRoutingPlansForScope(ctx, product, sku)
				_, _, _ = s.applyScopeReopenRouting(ctx, product, sku, actor,
					fmt.Sprintf("Chat mở lại toàn hệ thống %s/%s", product, sku))
			}
		}(sc[0], sc[1])
	}
	wg.Wait()

	if totalCancelled == 0 {
		return formatChatActionReply("Không có bảo trì active.", "Tất cả dịch vụ đang hoạt động bình thường.", ""), true
	}
	return formatChatActionReply(
		fmt.Sprintf("Đã mở lại %s.", allServicesScope(serviceFilters)),
		fmt.Sprintf("Hủy %d cửa sổ bảo trì; routing baseline được khôi phục.", totalCancelled),
		"",
	), true
}

func (s *Server) chatReopenService(ctx context.Context, product, sku, provider, actor string) (string, error) {
	labels, _ := s.DB.ProductLabelMap(ctx)
	label := product
	if l, ok := labels[product]; ok && l != "" {
		label = l
	}

	if sku == "" {
		skus, err := s.DB.ListSKUsForProduct(ctx, product, true)
		if err != nil {
			return "", err
		}
		if len(skus) == 0 {
			// provider-mode product (e.g. TOPUP_MOBI): no SKUs, scope addressed with empty SKU.
			cancelled, _ := s.DB.CancelActiveMaintenanceForSKU(ctx, product, "", actor)
			_ = s.DB.CancelPendingRoutingPlansForScope(ctx, product, "")
			_, _, _ = s.applyScopeReopenRouting(ctx, product, "", actor, fmt.Sprintf("Chat mở lại dịch vụ %s", product))
			return formatChatActionReply(
				fmt.Sprintf("Đã mở lại dịch vụ %s (toàn bộ).", label),
				fmt.Sprintf("Hủy %d cửa sổ BT; routing baseline khôi phục.", cancelled),
				"",
			), nil
		}
		var summaries []string
		for _, sk := range skus {
			m, err := s.chatReopenService(ctx, product, sk.SKUCode, provider, actor)
			if err != nil {
				continue
			}
			summaries = append(summaries, m)
		}
		if len(summaries) == 0 {
			return "", fmt.Errorf("không mở lại được scope nào cho %s", product)
		}
		return formatChatActionReply(
			fmt.Sprintf("Đã mở lại dịch vụ %s (toàn bộ).", label),
			strings.Join(summaries, "\n"),
			"",
		), nil
	}

	cancelled, err := s.DB.CancelActiveMaintenanceForSKU(ctx, product, sku, actor)
	if err != nil {
		return "", err
	}
	if cancelled == 0 {
		if ids := s.activeMaintenanceIDsForScope(ctx, product, sku); len(ids) > 0 {
			extra, err := s.DB.CancelMaintenanceByIDs(ctx, ids, actor)
			if err != nil {
				return "", err
			}
			cancelled += extra
		}
	}
	if n, err := s.DB.CountActiveMaintenanceForSKU(ctx, product, sku); err != nil {
		return "", err
	} else if n > 0 {
		return "", fmt.Errorf("vẫn còn bảo trì active (%d cửa sổ) — thử Mở lại trên Dashboard", n)
	}
	_ = s.DB.CancelPendingRoutingPlansForScope(ctx, product, sku)
	reason := fmt.Sprintf("Chat mở lại dịch vụ %s/%s", product, sku)

	var detail string
	if provider != "" {
		// Named provider → route 100% to that provider, others 0%.
		allProviders, err := s.DB.GetRoutingForScope(ctx, product, sku)
		if err != nil {
			return "", err
		}
		newRouting := map[string]float64{}
		for _, p := range allProviders {
			if p.ProviderCode == provider {
				newRouting[p.ProviderCode] = 100
			} else {
				newRouting[p.ProviderCode] = 0
			}
		}
		if _, ok := newRouting[provider]; !ok {
			newRouting[provider] = 100
		}
		if _, rerr := s.Tools.UpdateRouting(ctx, tools.UpdateRoutingInput{
			Product:     product,
			Scope:       "sku",
			SKU:         sku,
			Routing:     newRouting,
			TriggerType: "manual_temp",
			ExecutedBy:  actor,
			Reason:      reason + " qua " + provider,
		}); rerr != nil {
			return "", rerr
		}
		detail = fmt.Sprintf("Hủy %d cửa sổ BT; routing 100%% → %s.", cancelled, provider)
	} else {
		out, _, err := s.applyScopeReopenRouting(ctx, product, sku, actor, reason)
		if err != nil {
			return "", err
		}
		detail = fmt.Sprintf("Hủy %d cửa sổ BT; routing baseline áp dụng %v.", cancelled, out.Applied)
	}

	if cancelled == 0 {
		return formatChatActionReply(
			fmt.Sprintf("Không có BT active cho %s · %s.", label, sku),
			detail,
			"",
		), nil
	}
	return formatChatActionReply(
		fmt.Sprintf("Đã mở lại dịch vụ %s · %s.", label, sku),
		detail,
		"",
	), nil
}

func (s *Server) chatSetMaintenance(
	ctx context.Context,
	product, sku, provider string,
	durationMin int,
	actor, userMsg string,
) (string, error) {
	defaultMin := maintenanceDefaultDurationMin(ctx, s.DB)
	if durationMin <= 0 {
		durationMin = defaultMin
	}
	if durationMin <= 0 {
		durationMin = 60
	}
	startsAt, endsAt, err := parseMaintenanceWindow("", "", durationMin, defaultMin)
	if err != nil {
		return "", err
	}

	labels, _ := s.DB.ProductLabelMap(ctx)
	label := product
	if l, ok := labels[product]; ok && l != "" {
		label = l
	}

	if sku == "" {
		skus, err := s.DB.ListSKUsForProduct(ctx, product, true)
		if err != nil {
			return "", err
		}
		if len(skus) == 0 {
			// provider-mode product (e.g. TOPUP_MOBI): no SKUs, use provider-scope path.
			n, err := s.chatSetMaintenanceScopeAll(ctx, product, "", startsAt, endsAt, actor)
			if err != nil {
				return "", err
			}
			return formatChatActionReply(
				fmt.Sprintf("Đã bật bảo trì %s (toàn dịch vụ).", label),
				fmt.Sprintf("%d provider · %s → %s.", n, startsAt.Format("15:04"), endsAt.Format("15:04")),
				"",
			), nil
		}
		total := 0
		for _, sk := range skus {
			n, err := s.chatSetMaintenanceScope(ctx, product, sk.SKUCode, provider, startsAt, endsAt, actor, userMsg)
			if err != nil {
				continue
			}
			total += n
		}
		if total == 0 {
			return "", fmt.Errorf("không bật được bảo trì cho %s", product)
		}
		return formatChatActionReply(
			fmt.Sprintf("Đã bật bảo trì %s (toàn dịch vụ).", label),
			fmt.Sprintf("%d provider · %s → %s.", total, startsAt.Format("15:04"), endsAt.Format("15:04")),
			"",
		), nil
	}

	n, err := s.chatSetMaintenanceScope(ctx, product, sku, provider, startsAt, endsAt, actor, userMsg)
	if err != nil {
		return "", err
	}
	scope := label + " · " + sku
	if provider != "" {
		scope += " · " + provider
	}
	return formatChatActionReply(
		fmt.Sprintf("Đã bật bảo trì %s.", scope),
		fmt.Sprintf("%d provider · %s → %s.", n, startsAt.Format("15:04"), endsAt.Format("15:04")),
		"",
	), nil
}

// chatSetMaintenanceScopeAll is used exclusively by global "bảo trì tất cả" commands.
// Unlike chatSetMaintenanceScope it:
//  1. Bypasses chatMaintenanceTargets — forces ALL providers into maintenance.
//  2. Uses a no-rollback loop — each provider succeeds or fails independently,
//     so a single provider failure (e.g. SHOPPAY disabled) does NOT cancel the
//     already-applied ESALE/IMEDIA windows (applyMaintenanceTargets rolls back on error).
func (s *Server) chatSetMaintenanceScopeAll(
	ctx context.Context,
	product, sku string,
	startsAt, endsAt time.Time,
	actor string,
) (int, error) {
	_ = s.DB.CancelPendingRoutingPlansForScope(ctx, product, sku)
	// Cancel existing windows first — prevents duplicates and stale orphans.
	_, _ = s.DB.CancelActiveMaintenanceForSKU(ctx, product, sku, actor)

	providers, err := s.DB.GetRoutingForScope(ctx, product, sku)
	if err != nil {
		return 0, err
	}
	if len(providers) == 0 {
		return 0, fmt.Errorf("không có provider cho %s/%s", product, sku)
	}

	// No-rollback: apply each provider independently.
	count := 0
	for i, p := range providers {
		_, err := s.Tools.SetMaintenance(ctx, tools.SetMaintenanceInput{
			Product:     product,
			Provider:    p.ProviderCode,
			SKU:         sku,
			StartsAt:    startsAt,
			EndsAt:      endsAt,
			TriggerType: "manual_temp",
			Reason:      skuWideMaintenanceMarker,
			Status:      "active",
			Seq:         i,
			SkipNotify:  true,
			Actor:       actor,
		})
		if err == nil {
			count++
		}
	}
	if count == 0 {
		return 0, fmt.Errorf("không bật được bảo trì cho bất kỳ provider nào của %s/%s", product, sku)
	}
	return count, nil
}

func (s *Server) chatSetMaintenanceScope(
	ctx context.Context,
	product, sku, provider string,
	startsAt, endsAt time.Time,
	actor, _ string,
) (int, error) {
	// Cancel any pending routing plans for this scope — manual maintenance supersedes them.
	_ = s.DB.CancelPendingRoutingPlansForScope(ctx, product, sku)

	providers, err := s.DB.GetRoutingForScope(ctx, product, sku)
	if err != nil {
		return 0, err
	}
	reason := skuWideMaintenanceMarker
	if provider != "" {
		reason = provider + " — chat bảo trì thủ công"

		// Build set of providers already in active maintenance.
		inMaint := map[string]bool{}
		if windows, err2 := s.DB.ListActiveMaintenance(ctx, product, "", sku); err2 == nil {
			now := time.Now()
			for _, w := range windows {
				if w.EndsAt.After(now) {
					inMaint[w.ProviderCode] = true
				}
			}
		}

		// Count active providers OTHER than the named one.
		var activeOthers []domain.RoutingPct
		for _, p := range providers {
			if p.ProviderCode == provider || inMaint[p.ProviderCode] {
				continue
			}
			if p.TrafficPct > 0 {
				activeOthers = append(activeOthers, p)
			}
		}

		if len(activeOthers) > 0 {
			// Other active providers exist → routing-only: zero out the named provider
			// and redistribute its traffic proportionally. No maintenance window created.
			drop := 0.0
			newRouting := map[string]float64{}
			for _, p := range providers {
				newRouting[p.ProviderCode] = p.TrafficPct
				if p.ProviderCode == provider {
					drop = p.TrafficPct
				}
			}
			newRouting[provider] = 0
			totalOther := 0.0
			for _, p := range activeOthers {
				totalOther += p.TrafficPct
			}
			if totalOther > 0 {
				for _, p := range activeOthers {
					newRouting[p.ProviderCode] = p.TrafficPct + (p.TrafficPct/totalOther)*drop
				}
			} else {
				equal := 100.0 / float64(len(activeOthers))
				for _, p := range activeOthers {
					newRouting[p.ProviderCode] = equal
				}
			}
			final := roundToInt100(newRouting)
			if _, rerr := s.Tools.UpdateRouting(ctx, tools.UpdateRoutingInput{
				Product:     product,
				Scope:       "sku",
				SKU:         sku,
				Routing:     final,
				TriggerType: "manual_temp",
				ExecutedBy:  actor,
				Reason:      reason,
			}); rerr != nil {
				return 0, rerr
			}
			// Cancel any lingering maintenance window for the named provider.
			// Routing-only means the provider is excluded via traffic=0,
			// not via a maintenance window — stale windows would block "Mở lại dịch vụ".
			_, _ = s.DB.CancelActiveMaintenanceForSKUProvider(ctx, product, sku, provider, actor)
			return 1, nil
		}

		// Named provider is the only active one → put ALL providers into maintenance.
		var allTargets []string
		for _, p := range providers {
			allTargets = append(allTargets, p.ProviderCode)
		}
		if len(allTargets) == 0 {
			allTargets = []string{provider}
		}
		applied, err := applyMaintenanceTargets(ctx, s.Tools, product, sku, actor, reason, allTargets, startsAt, endsAt)
		return len(applied), err
	}

	// When no provider is specified, prefer providers that are routing-excluded
	// (TrafficPct == 0, not already in an active maintenance window).
	// This handles the common case: agent routing plan moved bad provider to 0%,
	// user then says "bảo trì" to formalise the de-facto exclusion.
	// Fall back to active (TrafficPct > 0) providers if no exclusion is found.
	targets := chatMaintenanceTargets(ctx, s.DB, providers, product, sku)
	if len(targets) == 0 {
		return 0, fmt.Errorf("không xác định provider cho %s/%s", product, sku)
	}
	// Use a specific reason when only 1 provider is targeted — avoids the
	// sku-wide maintenance marker being stored for a single-provider maintenance.
	if len(targets) == 1 {
		reason = targets[0] + " — chat bảo trì thủ công"
	}
	applied, err := applyMaintenanceTargets(ctx, s.Tools, product, sku, actor, reason, targets, startsAt, endsAt)
	return len(applied), err
}

// chatMaintenanceTargets picks which providers to put in maintenance when the user
// issues a scope-level "bảo trì" command without naming a specific provider.
//
// Priority:
//  1. Agent pending recommendation for this scope → use the recommended provider(s).
//     This correctly handles both "IMEDIA de-routed to 0% by agent" and
//     "IMEDIA still active but agent already flagged it as bad".
//  2. Routing-excluded (TrafficPct==0, not in maintenance) — legacy fallback when
//     there is no recommendation but the agent already zeroed the bad provider.
//  3. Active (TrafficPct>0) providers — standard "put whole service in maintenance".
func chatMaintenanceTargets(
	ctx context.Context,
	db *store.DB,
	providers []domain.RoutingPct,
	product, sku string,
) []string {
	// Build set of providers already in active maintenance
	maintSet := map[string]bool{}
	if existing, err := db.ListActiveMaintenance(ctx, product, "", sku); err == nil {
		now := time.Now()
		for _, w := range existing {
			if w.EndsAt.After(now) {
				maintSet[w.ProviderCode] = true
			}
		}
	}

	// 1. Check agent pending recommendation
	if rec, found, err := db.LatestPendingMaintenanceForScope(ctx, product, sku); err == nil && found {
		targets := maintenanceTargetsFromRecommendation(rec, providers)
		// Filter out providers already in maintenance
		var filtered []string
		for _, t := range targets {
			if !maintSet[t] {
				filtered = append(filtered, t)
			}
		}
		if len(filtered) > 0 {
			return filtered
		}
	}

	// 1b. Pending routing plan: if plan proposes moving an ACTIVE provider to 0%,
	// that provider is the bad one to maintain — handles the case where the dashboard
	// shows a routing suggestion but no DB maintenance recommendation exists yet.
	if plan, ok, _ := db.GetPendingRoutingPlanForScope(ctx, product, sku); ok {
		var parsed struct {
			ProposedPct map[string]float64 `json:"proposed_pct"`
		}
		if json.Unmarshal(plan.PlanJSON, &parsed) == nil && len(parsed.ProposedPct) > 0 {
			var planTargets []string
			for _, p := range providers {
				if maintSet[p.ProviderCode] {
					continue
				}
				newPct, inPlan := parsed.ProposedPct[p.ProviderCode]
				if inPlan && newPct == 0 && p.TrafficPct > 0 {
					planTargets = append(planTargets, p.ProviderCode)
				}
			}
			if len(planTargets) > 0 {
				return planTargets
			}
		}
	}

	var excluded []string // TrafficPct==0, not in maintenance
	var active []string   // TrafficPct>0, not in maintenance
	for _, p := range providers {
		if maintSet[p.ProviderCode] {
			continue
		}
		if p.TrafficPct == 0 {
			excluded = append(excluded, p.ProviderCode)
		} else {
			active = append(active, p.ProviderCode)
		}
	}

	// 2. Routing-excluded (agent already de-routed them)
	if len(excluded) > 0 {
		return excluded
	}
	// 3. Active providers
	return active
}

func (s *Server) chatSetScopeAuto(ctx context.Context, product, sku, mode, userMsg string) (string, error) {
	cfg := store.ScopeAutoConfig{
		ProductCode: product,
		SKUCode:     sku,
		AutoAction:  store.NormalizeScopeAutoAction(mode),
	}
	if mode == "time_window" {
		cfg.WindowStart, cfg.WindowEnd = parseScopeAutoWindowFromText(userMsg)
	}
	if err := s.DB.UpsertScopeAutoConfig(ctx, cfg); err != nil {
		return "", err
	}
	saved, _ := s.DB.GetScopeAutoConfig(ctx, product, sku)
	if store.ShouldAutoApplyScope(saved, time.Now()) {
		if sku == "" {
			_ = s.DB.CancelPendingRoutingPlansForProduct(ctx, product)
		} else {
			_ = s.DB.CancelPendingRoutingPlansForScope(ctx, product, sku)
		}
	}
	labels, _ := s.DB.ProductLabelMap(ctx)
	label := product
	if l, ok := labels[product]; ok && l != "" {
		label = l
	}
	scope := label
	if sku != "" {
		scope += " · " + sku
	}
	detail := "Routing/BT = " + chatresolve.ScopeAutoModeLabel(saved.AutoAction)
	if saved.AutoAction == "time_window" && saved.WindowStart != "" {
		detail += fmt.Sprintf(" (%s → %s)", saved.WindowStart, saved.WindowEnd)
	}
	return formatChatActionReply(
		fmt.Sprintf("Đã cập nhật %s.", scope),
		detail,
		"Agent sẽ theo chế độ mới từ chu kỳ tiếp theo.",
	), nil
}

func parseScopeAutoWindowFromText(msg string) (start, end string) {
	key := chatresolve.NormalizeKey(msg)
	for _, sep := range []string{" den ", " đến ", "-", "–"} {
		if idx := strings.Index(key, sep); idx > 0 {
			a := strings.TrimSpace(key[:idx])
			b := strings.TrimSpace(key[idx+len(sep):])
			if len(a) >= 2 && len(b) >= 2 {
				return normalizeTimeWindowToken(a), normalizeTimeWindowToken(b)
			}
		}
	}
	return "08:00", "22:00"
}

func normalizeTimeWindowToken(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.Contains(raw, ":") {
		return raw
	}
	if strings.HasSuffix(raw, "h") {
		h := strings.TrimSuffix(raw, "h")
		if len(h) == 1 {
			return "0" + h + ":00"
		}
		return h + ":00"
	}
	return raw
}