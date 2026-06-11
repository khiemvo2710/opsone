package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	"opsone/internal/chatresolve"
	"opsone/internal/store"
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
	sku := chatresolve.NormalizeSKU(chatresolve.ExtractSKUFromText(userMsg))
	msg, err := s.chatReopenService(ctx, product, sku, actor)
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

func (s *Server) chatReopenService(ctx context.Context, product, sku, actor string) (string, error) {
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
			return "", fmt.Errorf("không có SKU cho %s", product)
		}
		var summaries []string
		for _, sk := range skus {
			m, err := s.chatReopenService(ctx, product, sk.SKUCode, actor)
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
	out, _, err := s.applyScopeReopenRouting(ctx, product, sku, actor, reason)
	if err != nil {
		return "", err
	}
	detail := fmt.Sprintf("Hủy %d cửa sổ BT; routing baseline áp dụng %v.", cancelled, out.Applied)
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
			return "", fmt.Errorf("không có SKU cho %s", product)
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

func (s *Server) chatSetMaintenanceScope(
	ctx context.Context,
	product, sku, provider string,
	startsAt, endsAt time.Time,
	actor, _ string,
) (int, error) {
	providers, err := s.DB.GetRoutingForScope(ctx, product, sku)
	if err != nil {
		return 0, err
	}
	reason := skuWideMaintenanceMarker
	if provider != "" {
		reason = provider + " — chat bảo trì thủ công"
		targets := []string{provider}
		applied, err := applyMaintenanceTargets(ctx, s.Tools, product, sku, actor, reason, targets, startsAt, endsAt)
		return len(applied), err
	}
	targets := maintenanceTargetsForScope(reason, "", providers)
	if len(targets) == 0 {
		return 0, fmt.Errorf("không xác định provider cho %s/%s", product, sku)
	}
	applied, err := applyMaintenanceTargets(ctx, s.Tools, product, sku, actor, reason, targets, startsAt, endsAt)
	return len(applied), err
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
