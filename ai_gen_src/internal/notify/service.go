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
	Actor          string // Who performed the action (Manual name or 'OpsOne')
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

	// Load settings from DB
	settings, err := s.DB.GetAgentSettings(ctx)
	if err != nil {
		return err
	}

	chat, chatJSON, _ := s.loadChat(ctx, p.Provider)
	subject, body := RenderEmailVI(p, chat)

	recipients := settings.NotificationRecipients
	if len(recipients) == 0 {
		recipients = []string{"ops-team@company.local"} // fallback
	}
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
		// Use SmtpSender from DB if not empty
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
	} else if p.HealthStatus == "green" {
		icon = "🟢"
	}
	
	title := p.Product
	if len(p.SKUs) > 0 {
		title = fmt.Sprintf("%s — %d SKU", p.Product, len(p.SKUs))
	} else if p.SKU != "" {
		title = fmt.Sprintf("%s — SKU %s", p.Product, p.SKU)
	}
	
	subject = fmt.Sprintf("[OpsOne %s] %s — %s", icon, title, p.ActionSummary)

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "── Tình trạng hiện tại ──\n")
	fmt.Fprintf(&buf, "Sản phẩm:     %s\n", p.Product)
	if len(p.SKUs) > 0 {
		fmt.Fprintf(&buf, "Danh sách SKU: %s\n", strings.Join(p.SKUs, ", "))
	} else if p.SKU != "" {
		fmt.Fprintf(&buf, "SKU:          %s\n", p.SKU)
	}
	if len(p.Providers) > 0 {
		fmt.Fprintf(&buf, "Providers:    %s\n", strings.Join(p.Providers, ", "))
	} else if p.Provider != "" {
		fmt.Fprintf(&buf, "Nhà cung cấp: %s\n", p.Provider)
	}
	
	if p.TriggerEvent != "maintenance_active" && p.TriggerEvent != "maintenance_completed" && p.TriggerEvent != "maintenance_cancelled" {
		fmt.Fprintf(&buf, "Thành công:   %.0f%%  |  Pending: %.0f%%  |  Lỗi: %.0f%%\n", p.SuccessRate, p.PendingRate, p.FailRate)
		if len(p.BreachReasons) > 0 {
			fmt.Fprintf(&buf, "Ngưỡng vượt:  %s\n", joinReasons(p.BreachReasons))
		}
	}

	if p.ActionSummary != "" {
		fmt.Fprintf(&buf, "\n── Hành động OpsOne ──\n%s\n", p.ActionSummary)
	}
	
	if p.Reason != "" {
		fmt.Fprintf(&buf, "\n── Nguyên nhân bảo trì ──\n%s\n", p.Reason)
	}

	if p.Actor != "" {
		fmt.Fprintf(&buf, "Người thực hiện: %s\n", p.Actor)
	}

	if chat.ProviderCode != "" {
		fmt.Fprintf(&buf, "\n── Leo thang ──\n")
		fmt.Fprintf(&buf, "  Ứng dụng: %s | Nhóm: %s | Tag: %s\n", chat.ChatAppName, chat.ChatGroupName, chat.MentionTags)
	}
	
	fmt.Fprintf(&buf, "\nThời điểm: %s\n", time.Now().Format("2006-01-02 15:04:05 -0700"))
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
