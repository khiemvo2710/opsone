package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"opsone/internal/config"
	"opsone/internal/store"
)

// EmailParams for rendering and sending (§8.9).
type EmailParams struct {
	Product        string
	Provider       string
	Providers      []string // For batch
	SKU            string
	SKUs           []string // For batch
	HealthStatus   string
	TriggerEvent   string
	SuccessRate    float64
	PendingRate    float64
	FailRate       float64
	BreachReasons  []string
	ActionSummary  string
	IncidentID     string
	DedupeKey      string
	CycleID        *uint64
	AgentChangeID  *uint64
	MaintenanceID  *string
	Reason         string // Reason for maintenance
	IsService      bool   // True if service-level notification
	Actor          string // Who performed the action (e.g. "OpsOne" or user name)
	DeepLinkURL    string // Direct link to Dashboard for this product
	SuggestedNextStep string // Explicit next step for the ops team
}

// Service sends ops notification emails (§8.9).
type Service struct {
	DB     *store.DB
	Config config.Config
}

// NewService creates a notify service.
func NewService(db *store.DB, cfg config.Config) *Service {
	return &Service{DB: db, Config: cfg}
}

// SendIfNeeded checks cooldown and sends or logs notification.
func (s *Service) SendIfNeeded(ctx context.Context, p EmailParams) error {
	if p.DedupeKey == "" {
		p.DedupeKey = fmt.Sprintf("%s:%s:%s:%s", p.Product, p.Provider, p.SKU, p.TriggerEvent)
	}
	exists, err := s.DB.NotificationExists(ctx, p.DedupeKey)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	settings, err := s.DB.GetAgentSettings(ctx)
	if err != nil {
		return err
	}

	chat, chatJSON, _ := s.loadChat(ctx, p.Provider)
	subject, body := RenderEmailVI(p, chat)

	recipients := settings.NotificationRecipients
	if len(recipients) == 0 {
		recipients = []string{"khiemvt@vng.com.vn"}
	}
	recipientsJSON, _ := json.Marshal(recipients)

	metricsJSON, _ := json.Marshal(map[string]any{
		"success_rate": p.SuccessRate,
		"pending_rate": p.PendingRate,
		"fail_rate":    p.FailRate,
	})

	status := "sent"
	if !s.Config.NotificationMock {
		smtpFrom := settings.SmtpSender
		if smtpFrom == "" {
			smtpFrom = s.Config.SMTPFrom
		}
		if err := SendSMTPWithSender(s.Config, smtpFrom, recipients, subject, body); err != nil {
			status = "failed"
		}
	}

	return s.DB.InsertNotificationLog(ctx, p.DedupeKey, p.TriggerEvent, p.HealthStatus,
		p.Product, p.Provider, p.SKU, subject, p.ActionSummary, status,
		metricsJSON, chatJSON, recipientsJSON)
}

func (s *Service) loadChat(ctx context.Context, provider string) (store.ChatEscalation, []byte, error) {
	c, ok, err := s.DB.GetChatEscalation(ctx, provider)
	if err != nil {
		return store.ChatEscalation{}, nil, err
	}
	if !ok {
		c = store.ChatEscalation{
			ProviderCode:  provider,
			ChatAppName:   "Microsoft Teams",
			ChatGroupName: "[OpsOne] Support",
			MentionTags:   "@oncall",
		}
	}
	raw, _ := json.Marshal(c)
	return c, raw, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Email renderer — type-specific layouts
// ─────────────────────────────────────────────────────────────────────────────

// RenderEmailVI builds subject + plain-text body based on TriggerEvent.
func RenderEmailVI(p EmailParams, chat store.ChatEscalation) (subject, body string) {
	switch p.TriggerEvent {
	case "incident_open":
		return renderIncidentOpen(p, chat)
	case "breach":
		return renderBreach(p, chat)
	case "routing_applied":
		return renderRoutingApplied(p, chat)
	case "recovery_failed":
		return renderRecoveryFailed(p, chat)
	case "maintenance_active":
		return renderMaintenanceActive(p, chat)
	case "maintenance_scheduled":
		return renderMaintenanceScheduled(p, chat)
	case "maintenance_completed":
		return renderMaintenanceCompleted(p, chat)
	case "maintenance_cancelled":
		return renderMaintenanceCancelled(p, chat)
	default:
		return renderGeneric(p, chat)
	}
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func scopeLabel(p EmailParams) string {
	parts := []string{p.Product}
	if len(p.SKUs) > 0 {
		parts = append(parts, strings.Join(p.SKUs, "+"))
	} else if p.SKU != "" {
		parts = append(parts, p.SKU)
	}
	if len(p.Providers) > 0 {
		parts = append(parts, strings.Join(p.Providers, "+"))
	} else if p.Provider != "" {
		parts = append(parts, p.Provider)
	}
	return strings.Join(parts, " / ")
}

func healthIcon(status string) string {
	switch status {
	case "red":
		return "🔴"
	case "yellow":
		return "🟡"
	case "green":
		return "🟢"
	default:
		return "⚪"
	}
}

func healthLabel(status string) string {
	switch status {
	case "red":
		return "SỰ CỐ"
	case "yellow":
		return "CẢNH BÁO"
	case "green":
		return "ỔN ĐỊNH"
	default:
		return "KHÔNG XÁC ĐỊNH"
	}
}

func writeSep(buf *bytes.Buffer) {
	buf.WriteString("--------------------------------\n")
}
func writeFooter(buf *bytes.Buffer) {
	tz, _ := time.LoadLocation("Asia/Ho_Chi_Minh")
	ts := time.Now().In(tz).Format("02/01/2006 15:04:05 ICT")
	buf.WriteString("\nThời điểm: ")
	buf.WriteString(ts)
	buf.WriteString("\nOpsOne - Hệ thống giám sát thanh toán tự động\n")
}

// renderMaintenanceActive renders "maintenance_active" email.
func renderMaintenanceActive(p EmailParams, _ store.ChatEscalation) (subject, body string) {
	scope := scopeLabel(p)
	actor := p.Actor
	if actor == "" {
		actor = "OpsOne"
	}
	subject = fmt.Sprintf("[OpsOne] BẢO TRÌ BẮT ĐẦU | %s - %s", scope, actor)

	var buf bytes.Buffer
	buf.WriteString("BẢO TRÌ ĐÃ KÍCH HOẠT - ")
	buf.WriteString(scope)
	buf.WriteString("\n")
	writeSep(&buf)
	buf.WriteString("\nPHẠM VI BẢO TRÌ\n")
	writeSep(&buf)
	buf.WriteString("\nSản phẩm:  ")
	buf.WriteString(p.Product)
	buf.WriteString("\n")
	if len(p.SKUs) > 0 {
		buf.WriteString("SKU:       ")
		buf.WriteString(strings.Join(p.SKUs, ", "))
		buf.WriteString("\n")
	} else if p.SKU != "" {
		buf.WriteString("SKU:       ")
		buf.WriteString(p.SKU)
		buf.WriteString("\n")
	}

	reason := p.Reason
	if reason == "" {
		reason = "Bảo trì dịch vụ thủ công"
	}
	buf.WriteString("\nLÝ DO\n")
	writeSep(&buf)
	buf.WriteString("\n")
	buf.WriteString(reason)
	buf.WriteString("\n")

	if p.ActionSummary != "" {
		buf.WriteString("\nHÀNH ĐỘNG\n")
		writeSep(&buf)
		buf.WriteString("\n")
		buf.WriteString(p.ActionSummary)
		buf.WriteString("\n")
	}

	buf.WriteString("\nNGƯỜI THỰC HIỆN\n")
	writeSep(&buf)
	buf.WriteString("\n")
	buf.WriteString(actor)
	buf.WriteString("\n")

	buf.WriteString("\nLƯU Ý\n")
	writeSep(&buf)
	buf.WriteString("\n")
	buf.WriteString("- Routing tới provider đang bảo trì đã bị tạm dừng\n")
	buf.WriteString("- Hệ thống sẽ tự động mở lại khi bảo trì kết thúc\n")
	buf.WriteString("- Mở Dashboard để gia hạn hoặc hủy bảo trì sớm nếu cần\n")

	writeFooter(&buf)
	return subject, buf.String()
}

func renderMaintenanceScheduled(p EmailParams, _ store.ChatEscalation) (subject, body string) {
	scope := scopeLabel(p)
	actor := p.Actor
	if actor == "" {
		actor = "OpsOne"
	}
	subject = fmt.Sprintf("[OpsOne] BẢO TRÌ LÊN LỊCH | %s - %s", scope, actor)

	var buf bytes.Buffer
	buf.WriteString("BẢO TRÌ ĐÃ LÊN LỊCH - ")
	buf.WriteString(scope)
	buf.WriteString("\n")
	writeSep(&buf)
	buf.WriteString("\nSản phẩm:  ")
	buf.WriteString(p.Product)
	buf.WriteString("\n")
	if len(p.SKUs) > 0 {
		buf.WriteString("SKU:       ")
		buf.WriteString(strings.Join(p.SKUs, ", "))
		buf.WriteString("\n")
	} else if p.SKU != "" {
		buf.WriteString("SKU:       ")
		buf.WriteString(p.SKU)
		buf.WriteString("\n")
	}
	if p.Reason != "" {
		buf.WriteString("\nLý do: ")
		buf.WriteString(p.Reason)
		buf.WriteString("\n")
	}
	buf.WriteString("\nNgười lên lịch: ")
	buf.WriteString(actor)
	buf.WriteString("\n")
	buf.WriteString("\nLưu ý:\n")
	buf.WriteString("- Bảo trì sẽ tự động kích hoạt theo lịch đã đặt\n")
	buf.WriteString("- Mở Dashboard để xem chi tiết hoặc hủy lịch\n")
	writeFooter(&buf)
	return subject, buf.String()
}

func renderMaintenanceCompleted(p EmailParams, _ store.ChatEscalation) (subject, body string) {
	scope := scopeLabel(p)
	actor := p.Actor
	if actor == "" {
		actor = "OpsOne"
	}
	subject = fmt.Sprintf("[OpsOne] BẢO TRÌ KẾT THÚC | %s - %s", scope, actor)

	var buf bytes.Buffer
	buf.WriteString("BẢO TRÌ ĐÃ KẾT THÚC - ")
	buf.WriteString(scope)
	buf.WriteString("\n")
	writeSep(&buf)
	buf.WriteString("\nSản phẩm:  ")
	buf.WriteString(p.Product)
	buf.WriteString("\n")
	if len(p.SKUs) > 0 {
		buf.WriteString("SKU:       ")
		buf.WriteString(strings.Join(p.SKUs, ", "))
		buf.WriteString("\n")
	} else if p.SKU != "" {
		buf.WriteString("SKU:       ")
		buf.WriteString(p.SKU)
		buf.WriteString("\n")
	}
	if p.ActionSummary != "" {
		buf.WriteString("\nHành động: ")
		buf.WriteString(p.ActionSummary)
		buf.WriteString("\n")
	}
	buf.WriteString("Người thực hiện: ")
	buf.WriteString(actor)
	buf.WriteString("\n")
	buf.WriteString("\nLưu ý:\n")
	buf.WriteString("- Routing đã được khôi phục về trạng thái bình thường\n")
	buf.WriteString("- Kiểm tra Dashboard để xác nhận metrics ổn định\n")
	writeFooter(&buf)
	return subject, buf.String()
}

func renderMaintenanceCancelled(p EmailParams, _ store.ChatEscalation) (subject, body string) {
	scope := scopeLabel(p)
	actor := p.Actor
	if actor == "" {
		actor = "OpsOne"
	}
	subject = fmt.Sprintf("[OpsOne] BẢO TRÌ ĐÃ HỦY | %s - %s", scope, actor)

	var buf bytes.Buffer
	buf.WriteString("BẢO TRÌ ĐÃ HỦY - ")
	buf.WriteString(scope)
	buf.WriteString("\n")
	writeSep(&buf)
	buf.WriteString("\nSản phẩm:  ")
	buf.WriteString(p.Product)
	buf.WriteString("\n")
	if len(p.SKUs) > 0 {
		buf.WriteString("SKU:       ")
		buf.WriteString(strings.Join(p.SKUs, ", "))
		buf.WriteString("\n")
	} else if p.SKU != "" {
		buf.WriteString("SKU:       ")
		buf.WriteString(p.SKU)
		buf.WriteString("\n")
	}
	buf.WriteString("Người hủy: ")
	buf.WriteString(actor)
	buf.WriteString("\n")
	buf.WriteString("\nLưu ý:\n")
	buf.WriteString("- Bảo trì đã bị hủy trước khi kết thúc\n")
	buf.WriteString("- Routing trở về trạng thái trước bảo trì\n")
	writeFooter(&buf)
	return subject, buf.String()
}

func renderIncidentOpen(p EmailParams, _ store.ChatEscalation) (subject, body string) {
	scope := scopeLabel(p)
	subject = fmt.Sprintf("[OpsOne] 🔴 SỰ CỐ MỚI | %s", scope)
	if p.IncidentID != "" {
		subject = fmt.Sprintf("[OpsOne] 🔴 SỰ CỐ MỚI %s | %s", p.IncidentID, scope)
	}

	var buf bytes.Buffer
	buf.WriteString("🔴 SỰ CỐ MỚI - ")
	buf.WriteString(scope)
	if p.IncidentID != "" {
		buf.WriteString(fmt.Sprintf(" [%s]", p.IncidentID))
	}
	buf.WriteString("\n")
	writeSep(&buf)
	buf.WriteString(fmt.Sprintf("\nSuccess: %.0f%%  |  Pending: %.0f%%  |  Lỗi: %.0f%%\n",
		p.SuccessRate, p.PendingRate, p.FailRate))
	if len(p.BreachReasons) > 0 {
		buf.WriteString("Ngưỡng vượt: ")
		buf.WriteString(strings.Join(p.BreachReasons, "; "))
		buf.WriteString("\n")
	}
	if p.ActionSummary != "" {
		buf.WriteString("\nTóm tắt: ")
		buf.WriteString(p.ActionSummary)
		buf.WriteString("\n")
	}
	if p.DeepLinkURL != "" {
		buf.WriteString("\nXem Dashboard: ")
		buf.WriteString(p.DeepLinkURL)
		buf.WriteString("\n")
	}
	writeFooter(&buf)
	return subject, buf.String()
}

func renderBreach(p EmailParams, chat store.ChatEscalation) (subject, body string) {
	scope := scopeLabel(p)
	icon := healthIcon(p.HealthStatus)
	label := healthLabel(p.HealthStatus)
	subject = fmt.Sprintf("[OpsOne] %s %s | %s", icon, label, scope)

	var buf bytes.Buffer
	buf.WriteString(icon)
	buf.WriteString(" ")
	buf.WriteString(label)
	buf.WriteString(" - ")
	buf.WriteString(scope)
	buf.WriteString("\n")
	writeSep(&buf)
	buf.WriteString(fmt.Sprintf("\nSuccess: %.0f%%  |  Pending: %.0f%%  |  Lỗi: %.0f%%\n",
		p.SuccessRate, p.PendingRate, p.FailRate))
	if len(p.BreachReasons) > 0 {
		buf.WriteString("Ngưỡng vượt: ")
		buf.WriteString(strings.Join(p.BreachReasons, "; "))
		buf.WriteString("\n")
	}
	if p.ActionSummary != "" {
		buf.WriteString("\nHành động: ")
		buf.WriteString(p.ActionSummary)
		buf.WriteString("\n")
	}
	if p.DeepLinkURL != "" {
		buf.WriteString("\nXem Dashboard: ")
		buf.WriteString(p.DeepLinkURL)
		buf.WriteString("\n")
	}
	if chat.ChatGroupName != "" {
		buf.WriteString(fmt.Sprintf("\nLeo thang: %s | %s | %s\n",
			chat.ChatAppName, chat.ChatGroupName, chat.MentionTags))
	}
	writeFooter(&buf)
	return subject, buf.String()
}

func renderRoutingApplied(p EmailParams, _ store.ChatEscalation) (subject, body string) {
	scope := scopeLabel(p)
	subject = fmt.Sprintf("[OpsOne] ĐIỀU CHỈNH ROUTING | %s", scope)

	var buf bytes.Buffer
	buf.WriteString("ROUTING ĐÃ ĐIỀU CHỈNH - ")
	buf.WriteString(scope)
	buf.WriteString("\n")
	writeSep(&buf)
	if p.ActionSummary != "" {
		buf.WriteString("\n")
		buf.WriteString(p.ActionSummary)
		buf.WriteString("\n")
	}
	if p.Actor != "" {
		buf.WriteString("Người thực hiện: ")
		buf.WriteString(p.Actor)
		buf.WriteString("\n")
	}
	writeFooter(&buf)
	return subject, buf.String()
}

func renderRecoveryFailed(p EmailParams, chat store.ChatEscalation) (subject, body string) {
	scope := scopeLabel(p)
	subject = fmt.Sprintf("[OpsOne] PHỤC HỒI THẤT BẠI | %s", scope)

	var buf bytes.Buffer
	buf.WriteString("PHỤC HỒI THẤT BẠI - ")
	buf.WriteString(scope)
	buf.WriteString("\n")
	writeSep(&buf)
	buf.WriteString(fmt.Sprintf("\nSuccess: %.0f%%  |  Pending: %.0f%%  |  Lỗi: %.0f%%\n",
		p.SuccessRate, p.PendingRate, p.FailRate))
	if p.ActionSummary != "" {
		buf.WriteString("\n")
		buf.WriteString(p.ActionSummary)
		buf.WriteString("\n")
	}
	if chat.ChatGroupName != "" {
		buf.WriteString(fmt.Sprintf("\nLeo thang: %s | %s | %s\n",
			chat.ChatAppName, chat.ChatGroupName, chat.MentionTags))
	}
	writeFooter(&buf)
	return subject, buf.String()
}

func renderGeneric(p EmailParams, _ store.ChatEscalation) (subject, body string) {
	scope := scopeLabel(p)
	subject = fmt.Sprintf("[OpsOne] %s | %s", p.TriggerEvent, scope)

	var buf bytes.Buffer
	buf.WriteString(p.TriggerEvent)
	buf.WriteString(" - ")
	buf.WriteString(scope)
	buf.WriteString("\n")
	writeSep(&buf)
	if p.ActionSummary != "" {
		buf.WriteString("\n")
		buf.WriteString(p.ActionSummary)
		buf.WriteString("\n")
	}
	writeFooter(&buf)
	return subject, buf.String()
}
