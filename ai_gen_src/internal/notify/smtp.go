package notify

import (
	"fmt"
	"net/smtp"
	"strings"

	"opsone/internal/config"
)

// SendSMTP sends plain-text email via SMTP (§8.9).
func SendSMTP(cfg config.Config, to []string, subject, body string) error {
	return SendSMTPWithSender(cfg, cfg.SMTPFrom, to, subject, body)
}

// SendSMTPWithSender sends plain-text email with custom from address.
func SendSMTPWithSender(cfg config.Config, from string, to []string, subject, body string) error {
	if len(to) == 0 {
		return fmt.Errorf("no recipients")
	}
	if from == "" {
		from = cfg.SMTPFrom
	}
	addr := fmt.Sprintf("%s:%s", cfg.SMTPHost, cfg.SMTPPort)
	msg := buildMessage(from, to, subject, body)
	var auth smtp.Auth
	if cfg.SMTPUser != "" {
		auth = smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPHost)
	}
	return smtp.SendMail(addr, auth, from, to, []byte(msg))
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
