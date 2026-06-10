package output

import (
	"fmt"
	"strings"

	"opsone/internal/rules"
)

// HealthLabel returns Vietnamese label for health status (§8.2).
func HealthLabel(status string) string {
	switch status {
	case "red":
		return "Đang có vấn đề"
	case "yellow":
		return "Đang theo dõi / xử lý"
	default:
		return "Hệ thống OK"
	}
}

// IncidentSummary builds fallback incident text (§7.6.2, §8.3).
func IncidentSummary(product, provider, sku string, success, fail float64, evidence []rules.RuleResult, breachReasons []string) (summary, headline string) {
	var parts []string
	scope := product
	if sku != "" {
		scope = fmt.Sprintf("%s SKU %s", product, sku)
	}
	parts = append(parts, fmt.Sprintf("%s qua %s tỷ lệ lỗi %.0f%%, thành công %.0f%%", scope, provider, fail, success))
	if len(breachReasons) > 0 {
		parts = append(parts, breachReasons[0])
	}
	for _, e := range evidence {
		if e.Triggered && e.MessageVi != "" {
			parts = append(parts, e.MessageVi)
			break
		}
	}
	summary = strings.Join(parts, " — ")
	if len(summary) > 200 {
		summary = summary[:197] + "..."
	}
	headline = fmt.Sprintf("%s · %s lỗi %.0f%%", product, provider, fail)
	if len(headline) > 60 {
		headline = headline[:57] + "..."
	}
	return summary, headline
}

// RoutingPlanReason builds Vietnamese reason for routing plan (§7.6.3).
func RoutingPlanReason(product, badProvider string, evidence []rules.RuleResult) string {
	for _, e := range evidence {
		if e.SuggestedAction == "routing" && e.MessageVi != "" {
			return fmt.Sprintf("%s: %s", product, e.MessageVi)
		}
	}
	return fmt.Sprintf("%s: Đề xuất chuyển traffic khỏi %s đang suy giảm", product, badProvider)
}

// MaintenanceDetail builds maintenance recommendation text (§8.5).
func MaintenanceDetail(product, provider string, reasons []string) string {
	msg := fmt.Sprintf("Đề xuất bảo trì %s — provider %s (chỉ 1 luồng active hoặc không có backup healthy)", product, provider)
	if len(reasons) > 0 {
		msg += ". " + reasons[0]
	}
	return msg
}

// MonitorRecommendation when data insufficient (§7.5).
func MonitorRecommendation(product string) string {
	return fmt.Sprintf("%s: Chưa đủ dữ liệu — theo dõi thêm 15 phút", product)
}

// SeverityFromEvidence maps bumps to low/medium/high.
func SeverityFromEvidence(base string, evidence []rules.RuleResult) string {
	bump := 0
	for _, e := range evidence {
		bump += e.SeverityBump
	}
	switch {
	case bump >= 3:
		return "high"
	case bump >= 1:
		return "medium"
	default:
		if base == "high" {
			return "high"
		}
		return "low"
	}
}
