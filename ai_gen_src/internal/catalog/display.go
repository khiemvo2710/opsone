package catalog

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"opsone/internal/domain"
)

// fallbackLabels when DB label is empty (matches products.label in seed).
var fallbackLabels = map[string]string{
	"MOBIFONE":      "Thẻ Mobifone",
	"VINAPHONE":     "Thẻ Vinaphone",
	"VIETTEL":       "Thẻ Viettel",
	"ZING":          "Thẻ Zing",
	"GARENA":        "Thẻ Garena",
	"TOPUP_MOBI":    "Topup Mobifone",
	"TOPUP_VINA":    "Topup Vinaphone",
	"TOPUP_VIETTEL": "Topup Viettel",
	"DATA_MOBI":     "Data Mobifone",
	"DATA_VINA":     "Data Vinaphone",
	"DATA_VIETTEL":  "Data Viettel",
}

// ProductDisplayLabel returns human-readable product name for UI/chat.
func ProductDisplayLabel(p domain.Product) string {
	if p.Label != "" {
		return p.Label
	}
	return ProductDisplayLabelCode(p.ProductCode, nil)
}

// ProductDisplayLabelCode maps product_code to label; labels from DB override fallback.
func ProductDisplayLabelCode(code string, labels map[string]string) string {
	if labels != nil {
		if l, ok := labels[code]; ok && l != "" {
			return l
		}
	}
	if l, ok := fallbackLabels[code]; ok {
		return l
	}
	return humanizeProductCode(code)
}

// JoinProductDisplayLabels formats codes as "Data Mobifone, Data Viettel, ...".
func JoinProductDisplayLabels(codes []string, labels map[string]string) string {
	if len(codes) == 0 {
		return ""
	}
	seen := make(map[string]struct{}, len(codes))
	out := make([]string, 0, len(codes))
	for _, code := range codes {
		if code == "" {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		out = append(out, ProductDisplayLabelCode(code, labels))
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}

func humanizeProductCode(code string) string {
	parts := strings.Split(strings.ToLower(code), "_")
	for i, p := range parts {
		if len(p) == 0 {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}

// ProductLabelMap builds product_code → label from catalog rows.
func ProductLabelMap(products []domain.Product) map[string]string {
	m := make(map[string]string, len(products))
	for _, p := range products {
		if p.Label != "" {
			m[p.ProductCode] = p.Label
		}
	}
	return m
}

var productCodeToken = regexp.MustCompile(`^[A-Z][A-Z0-9_]+$`)

// FormatCycleHealthSummary rewrites legacy summaries like
// "Sự cố: [DATA_MOBI DATA_VIETTEL ...]" into human-readable product labels.
func FormatCycleHealthSummary(summary string, labels map[string]string) string {
	if summary == "" || !strings.HasPrefix(summary, "Sự cố:") {
		return summary
	}
	rest := strings.TrimSpace(strings.TrimPrefix(summary, "Sự cố:"))
	rest = strings.TrimPrefix(rest, "[")
	rest = strings.TrimSuffix(rest, "]")
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return summary
	}

	// Already comma-separated labels (no raw tokens)?
	if strings.Contains(rest, ",") && !containsRawProductCodes(rest) {
		return summary
	}

	parts := strings.Fields(strings.NewReplacer(",", " ", ";", " ", "[", " ", "]", " ").Replace(rest))
	codes := make([]string, 0, len(parts))
	for _, part := range parts {
		token := strings.TrimSpace(part)
		if token == "" {
			continue
		}
		if productCodeToken.MatchString(token) {
			codes = append(codes, token)
		}
	}
	if len(codes) == 0 {
		return summary
	}
	return fmt.Sprintf("Sự cố: %s", JoinProductDisplayLabels(codes, labels))
}

func containsRawProductCodes(text string) bool {
	for _, token := range strings.Fields(strings.ReplaceAll(text, ",", " ")) {
		if productCodeToken.MatchString(strings.TrimSpace(token)) {
			return true
		}
	}
	return false
}
