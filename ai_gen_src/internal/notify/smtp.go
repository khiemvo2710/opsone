package notify

import (
	"fmt"
	"net/smtp"
	"strings"

	"opsone/internal/config"
)

// SendSMTP sends plain-text email via SMTP (§8.9). MailHog: localhost:1025.
func SendSMTP(cfg config.Config, to []string, subject, body string) error {
	if len(to) == 0 {
		return fmt.Errorf("no recipients")
	}
	addr := fmt.Sprintf("%s:%s", cfg.SMTPHost, cfg.SMTPPort)
	msg := buildMessage(cfg.SMTPFrom, to, subject, body)
	var auth smtp.Auth
	if cfg.SMTPUser != "" {
		auth = smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPHost)
	}
	return smtp.SendMail(addr, auth, cfg.SMTPFrom, to, []byte(msg))
}

func buildMessage(from string, to []string, subject, body string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("From: %s\r\n", from))
	b.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(to, ", ")))
	b.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(body)
	return b.String()
}
