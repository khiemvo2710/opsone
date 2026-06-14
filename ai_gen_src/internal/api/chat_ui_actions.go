package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	"opsone/internal/catalog"
	"opsone/internal/chatresolve"
)

func (s *Server) executeUIAction(
	ctx context.Context,
	sessionID string,
	action catalog.UIActionKey,
	slots chatresolve.UIActionSlots,
	userMsg, actor string,
	isAdmin bool,
) (string, error) {
	if catalog.UIActionRequiresAdmin(action) && !isAdmin {
		return "", fmt.Errorf("cần quyền Admin — thao tác này tương đương nút trên Dashboard")
	}

	switch action {
	case catalog.UIActionHelp:
		return formatChatActionReply(s.chatUIActionsHelp(), "", ""), nil
	case catalog.UIActionListPending:
		reply, ok := s.tryChatListPendingReply(ctx, sessionID, isAdmin)
		if !ok {
			return "", fmt.Errorf("không liệt kê được pending")
		}
		return reply, nil
	case catalog.UIActionApproveReject:
		reply, ok := s.tryChatApproveRejectReply(ctx, sessionID, userMsg, actor, isAdmin)
		if !ok {
			return "", fmt.Errorf("không xử lý được duyệt/từ chối")
		}
		return reply, nil
	case catalog.UIActionSetMaintenance:
		reply, ok := s.tryChatSetMaintenanceReply(ctx, userMsg, actor, isAdmin)
		if !ok {
			return "", fmt.Errorf("không bật được bảo trì")
		}
		return reply, nil
	case catalog.UIActionExtendMaint:
		return s.chatExtendMaintenance(ctx, slots.Product, slots.SKU, slots.DurationMin, actor)
	case catalog.UIActionReopenService:
		reply, ok := s.tryChatReopenReply(ctx, userMsg, actor, isAdmin)
		if !ok {
			return "", fmt.Errorf("không mở lại được dịch vụ")
		}
		return reply, nil
	case catalog.UIActionRestoreBaseline:
		return s.chatRestoreBaseline(ctx, slots.Product, slots.SKU, slots.Provider, actor)
	case catalog.UIActionSetScopeAuto:
		reply, ok := s.tryChatSetScopeAutoReply(ctx, userMsg, actor, isAdmin)
		if !ok {
			return "", fmt.Errorf("không đổi được chế độ")
		}
		return reply, nil
	case catalog.UIActionSetAllMaintenance:
		reply, ok := s.tryChatSetAllMaintenanceReply(ctx, userMsg, actor, isAdmin)
		if !ok {
			return "", fmt.Errorf("không bật được bảo trì toàn hệ thống")
		}
		return reply, nil
	case catalog.UIActionReopenAllServices:
		reply, ok := s.tryChatReopenAllServicesReply(ctx, userMsg, actor, isAdmin)
		if !ok {
			return "", fmt.Errorf("không mở lại được toàn hệ thống")
		}
		return reply, nil
	default:
		return "", fmt.Errorf("thao tác UI chưa hỗ trợ: %s", action)
	}
}

func (s *Server) chatUIActionsHelp() string {
	var lines []string
	lines = append(lines, "OpsOne có thể thực hiện thao tác Dashboard khi bạn yêu cầu (Admin nếu ghi [Admin]):")
	for _, a := range catalog.AllUIActions() {
		adm := ""
		if a.AdminOnly {
			adm = " [Admin]"
		}
		lines = append(lines, fmt.Sprintf("• %s%s — ví dụ: \"%s\"", a.LabelVI, adm, a.ExampleVI))
	}
	lines = append(lines, "Nói rõ dịch vụ · mệnh giá · provider (ESALE/IMEDIA/SHOPPAY) khi cần.")
	return strings.Join(lines, "\n")
}

func (s *Server) chatExtendMaintenance(ctx context.Context, product, sku string, durationMin int, actor string) (string, error) {
	_ = actor
	if product == "" {
		return "", fmt.Errorf("thiếu tên dịch vụ — ví dụ: gia hạn bảo trì thẻ Garena 10.000")
	}
	defaultMin := maintenanceDefaultDurationMin(ctx, s.DB)
	if durationMin <= 0 {
		durationMin = defaultMin
	}
	if durationMin <= 0 {
		durationMin = 60
	}

	if sku == "" {
		skus, err := s.DB.ListSKUsForProduct(ctx, product, true)
		if err != nil {
			return "", err
		}
		var parts []string
		for _, sk := range skus {
			m, err := s.chatExtendMaintenance(ctx, product, sk.SKUCode, durationMin, actor)
			if err != nil {
				continue
			}
			parts = append(parts, stripActionReplyWrapper(m))
		}
		if len(parts) == 0 {
			return "", fmt.Errorf("không gia hạn được scope nào cho %s", product)
		}
		labels, _ := s.DB.ProductLabelMap(ctx)
		label := product
		if l, ok := labels[product]; ok && l != "" {
			label = l
		}
		return formatChatActionReply(
			fmt.Sprintf("Đã gia hạn bảo trì %s (toàn dịch vụ).", label),
			strings.Join(parts, "\n"),
			"",
		), nil
	}

	active, err := s.DB.CountActiveMaintenanceForSKU(ctx, product, sku)
	if err != nil {
		return "", err
	}
	if active == 0 {
		return "", fmt.Errorf("không có bảo trì active cho %s/%s", product, sku)
	}

	rows, err := s.DB.ListMaintenanceWindows(ctx, product, "active", 500)
	if err != nil {
		return "", err
	}
	if sched, err := s.DB.ListMaintenanceWindows(ctx, product, "scheduled", 200); err == nil {
		rows = append(rows, sched...)
	}

	var startsAt time.Time
	var maxEnd time.Time
	for _, r := range rows {
		if r.SKUCode != sku {
			continue
		}
		if startsAt.IsZero() || r.StartsAt.Before(startsAt) {
			startsAt = r.StartsAt
		}
		if maxEnd.IsZero() || r.EndsAt.After(maxEnd) {
			maxEnd = r.EndsAt
		}
	}
	if maxEnd.IsZero() {
		return "", fmt.Errorf("không tìm thấy cửa sổ BT cho %s/%s", product, sku)
	}
	newEnd := maxEnd.Add(time.Duration(durationMin) * time.Minute)

	n, err := s.DB.UpdateActiveMaintenanceTimesForSKU(ctx, product, sku, startsAt, newEnd)
	if err != nil {
		return "", err
	}
	if n == 0 {
		return "", fmt.Errorf("thời gian bảo trì không đổi — thử tăng duration")
	}

	labels, _ := s.DB.ProductLabelMap(ctx)
	label := product
	if l, ok := labels[product]; ok && l != "" {
		label = l
	}
	return formatChatActionReply(
		fmt.Sprintf("Đã gia hạn bảo trì %s · %s.", label, sku),
		fmt.Sprintf("%d cửa sổ · kết thúc ~%s (+%d phút).", n, newEnd.Format("15:04 02/01"), durationMin),
		"",
	), nil
}

func (s *Server) chatRestoreBaseline(ctx context.Context, product, sku, provider, actor string) (string, error) {
	if product == "" {
		return "", fmt.Errorf("thiếu tên dịch vụ — ví dụ: trả lại routing baseline thẻ Garena")
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
		var parts []string
		for _, sk := range skus {
			m, err := s.chatRestoreBaseline(ctx, product, sk.SKUCode, provider, actor)
			if err != nil {
				continue
			}
			parts = append(parts, stripActionReplyWrapper(m))
		}
		if len(parts) == 0 {
			return "", fmt.Errorf("không trả baseline được scope nào cho %s", product)
		}
		return formatChatActionReply(
			fmt.Sprintf("Đã trả routing baseline %s (toàn dịch vụ).", label),
			strings.Join(parts, "\n"),
			"",
		), nil
	}

	reason := fmt.Sprintf("Chat trả baseline routing %s/%s", product, sku)
	if provider != "" {
		reason += " · " + provider
	}
	out, proposed, err := s.applyScopeReopenRouting(ctx, product, sku, actor, reason)
	if err != nil {
		return "", err
	}
	scope := label + " · " + sku
	if provider != "" {
		scope += " · " + provider
	}
	detail := fmt.Sprintf("Baseline áp dụng %v.", out.Applied)
	if len(proposed) > 0 {
		parts := make([]string, 0, len(proposed))
		for prov, pct := range proposed {
			parts = append(parts, fmt.Sprintf("%s %.0f%%", prov, pct))
		}
		detail += " " + strings.Join(parts, " | ") + "."
	}
	return formatChatActionReply(
		fmt.Sprintf("Đã trả routing baseline %s.", scope),
		detail,
		"BT active (nếu có) vẫn giữ — dùng \"mở lại dịch vụ\" để hủy BT.",
	), nil
}

func stripActionReplyWrapper(reply string) string {
	return strings.TrimSpace(reply)
}

// executeUIActionFromTool is called by LLM tool execute_ui_action.
func (s *Server) executeUIActionFromTool(ctx context.Context, sessionID string, args map[string]any, actor string, isAdmin bool) (string, error) {
	action := catalog.UIActionKey(strings.TrimSpace(strArg(args, "action")))
	if action == "" {
		return "", fmt.Errorf("thiếu action")
	}
	slots := chatresolve.UIActionSlots{
		Product:     chatresolve.ExtractProductFromText(strArg(args, "product")),
		SKU:         chatresolve.NormalizeSKU(strArg(args, "sku")),
		Provider:    strings.ToUpper(strArg(args, "provider")),
		DurationMin: intArg(args, "duration_min"),
		AutoAction:  strArg(args, "auto_action"),
		Reject:      strings.EqualFold(strArg(args, "decision"), "reject"),
	}
	userMsg := strArg(args, "utterance")
	if action == catalog.UIActionApproveReject {
		if slots.Reject {
			userMsg = "từ chối " + slots.Product + " " + slots.SKU
		} else {
			userMsg = "duyệt " + slots.Product + " " + slots.SKU
		}
	}
	if userMsg == "" {
		userMsg = fmt.Sprintf("%s %s %s", action, slots.Product, slots.SKU)
	}
	if action == catalog.UIActionSetScopeAuto && slots.AutoAction != "" {
		userMsg = fmt.Sprintf("đặt chế độ %s %s %s", slots.AutoAction, slots.Product, slots.SKU)
	}
	reply, err := s.executeUIAction(ctx, sessionID, action, slots, userMsg, actor, isAdmin)
	if err != nil {
		return "", err
	}
	return stripActionReplyWrapper(reply), nil
}
