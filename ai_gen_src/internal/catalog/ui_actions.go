package catalog

import "strings"

// UIActionKey identifies a Dashboard button / scope action (§9.0).
type UIActionKey string

const (
	UIActionHelp            UIActionKey = "help_ui"
	UIActionListPending     UIActionKey = "list_pending"
	UIActionApproveReject   UIActionKey = "approve_reject"
	UIActionSetMaintenance  UIActionKey = "set_maintenance"
	UIActionExtendMaint     UIActionKey = "extend_maintenance"
	UIActionReopenService   UIActionKey = "reopen_service"
	UIActionRestoreBaseline UIActionKey = "restore_baseline"
	UIActionSetScopeAuto    UIActionKey = "set_scope_auto"
)

// UIActionMeta describes one operable UI action for chat/voice.
type UIActionMeta struct {
	Key         UIActionKey
	LabelVI     string
	ExampleVI   string
	AdminOnly   bool
	DashboardUI string
}

// AllUIActions is the catalog of actions the agent can run (same REST as Dashboard buttons).
func AllUIActions() []UIActionMeta {
	return []UIActionMeta{
		{UIActionListPending, "Xem việc chờ duyệt", "xem pending", false, "Hàng đề xuất routing/BT"},
		{UIActionApproveReject, "Duyệt / từ chối đề xuất", "duyệt routing topup mobi · ok · từ chối", true, "Duyệt / Từ chối"},
		{UIActionSetMaintenance, "Bật bảo trì", "bảo trì giúp tôi thẻ Garena 50.000", true, "Bảo trì SKU"},
		{UIActionExtendMaint, "Gia hạn bảo trì", "gia hạn bảo trì thẻ Garena 10.000 thêm 60 phút", true, "Gia hạn bảo trì"},
		{UIActionReopenService, "Mở lại dịch vụ", "mở lại thẻ Garena 10.000", true, "Mở lại dịch vụ"},
		{UIActionRestoreBaseline, "Trả lại routing baseline", "trả lại routing baseline thẻ Garena ESALE", true, "Trả lại / Mở lại provider"},
		{UIActionSetScopeAuto, "Đổi chế độ BT/Routing", "đặt thẻ Garena chế độ tự động", true, "Chế độ BT / Routing · Lưu"},
	}
}

// UIActionRequiresAdmin reports whether action needs Admin role.
func UIActionRequiresAdmin(key UIActionKey) bool {
	for _, a := range AllUIActions() {
		if a.Key == key {
			return a.AdminOnly
		}
	}
	return true
}

// UIActionsPromptHint returns Vietnamese catalog text for the chat system prompt.
func UIActionsPromptHint() string {
	var b strings.Builder
	b.WriteString("\nThao tác Dashboard (Admin — gọi tool execute_ui_action hoặc lệnh trực tiếp, cùng API nút UI):\n")
	for _, a := range AllUIActions() {
		if a.Key == UIActionHelp {
			continue
		}
		adm := ""
		if a.AdminOnly {
			adm = " [Admin]"
		}
		b.WriteString("- ")
		b.WriteString(a.LabelVI)
		b.WriteString(adm)
		b.WriteString(": ")
		b.WriteString(a.ExampleVI)
		b.WriteString(" (UI: ")
		b.WriteString(a.DashboardUI)
		b.WriteString(")\n")
	}
	return b.String()
}
