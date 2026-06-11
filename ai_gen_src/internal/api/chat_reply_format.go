package api

import "strings"

func formatChatActionReply(result, detail, hint string) string {
	var parts []string
	if r := strings.TrimSpace(result); r != "" {
		parts = append(parts, r)
	}
	if d := strings.TrimSpace(detail); d != "" {
		parts = append(parts, d)
	}
	if h := strings.TrimSpace(hint); h != "" {
		parts = append(parts, h)
	}
	return strings.Join(parts, "\n")
}
