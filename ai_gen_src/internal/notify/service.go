package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"opsone/internal/config"
	"opsone/internal/store"
)

// EmailParams for rendering and sending (§8.9).
type EmailParams struct {
	Product        string
	Provider       string
	SKU            string
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
}

// Service sends ops notification emails (§8.9).
type Service struct {
	DB     *store.DB
	Config config.Config
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

	chat, chatJSON, _ := s.loadChat(ctx, p.Provider)
	subject, body := RenderEmailVI(p, chat)
	recipients := []string{"ops-team@company.local"}
	recipientsJSON, _ := json.Marshal(recipients)
	metricsJSON, _ := json.Marshal(map[string]any{
		"success_rate": p.SuccessRate,
		"pending_rate": p.PendingRate,
		"fail_rate":    p.FailRate,
	})

	status := "sent"
	if s.Config.NotificationMock {
		status = "sent"
	} else {
		if err := SendSMTP(s.Config, recipients, subject, body); err != nil {
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
		c = store.ChatEscalation{ProviderCode: provider, ChatAppName: "Microsoft Teams", ChatGroupName: "[OpsOne] Support", MentionTags: "@oncall"}
	}
	raw, _ := json.Marshal(c)
	return c, raw, nil
}

// RenderEmailVI builds subject + plain text body (§8.9).
func RenderEmailVI(p EmailParams, chat store.ChatEscalation) (subject, body string) {
	icon := "🔴"
	if p.HealthStatus == "yellow" {
		icon = "🟡"
	}
	subject = fmt.Sprintf("[OpsOne %s] %s — %s", icon, p.Product, p.ActionSummary)

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "── Tình trạng hiện tại ──\n")
	fmt.Fprintf(&buf, "Sản phẩm:     %s\n", p.Product)
	fmt.Fprintf(&buf, "Nhà cung cấp: %s\n", p.Provider)
	fmt.Fprintf(&buf, "Thành công:   %.0f%%  |  Pending: %.0f%%  |  Lỗi: %.0f%%\n", p.SuccessRate, p.PendingRate, p.FailRate)
	if len(p.BreachReasons) > 0 {
		fmt.Fprintf(&buf, "Ngưỡng vượt:  %s\n", joinReasons(p.BreachReasons))
	}
	if p.ActionSummary != "" {
		fmt.Fprintf(&buf, "\n── Hành động OpsOne ──\n%s\n", p.ActionSummary)
	}
	fmt.Fprintf(&buf, "\n── Leo thang ──\n")
	fmt.Fprintf(&buf, "  Ứng dụng: %s | Nhóm: %s | Tag: %s\n", chat.ChatAppName, chat.ChatGroupName, chat.MentionTags)
	fmt.Fprintf(&buf, "\nThời điểm: %s\n", time.Now().Format("2006-01-02 15:04 -0700"))
	return subject, buf.String()
}

func joinReasons(r []string) string {
	if len(r) == 0 {
		return ""
	}
	out := r[0]
	for i := 1; i < len(r); i++ {
		out += "; " + r[i]
	}
	return out
}
