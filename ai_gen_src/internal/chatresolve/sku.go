package chatresolve

import (
	"regexp"
	"strings"
)

var skuDenomPattern = regexp.MustCompile(`\d{1,3}(?:\.\d{3})+|\d{4,7}`)

// NormalizeSKU maps user/LLM denomination text to catalog sku_code (vd 10.000 → 10000).
func NormalizeSKU(raw string) string {
	var digits strings.Builder
	for _, r := range strings.ToUpper(strings.TrimSpace(raw)) {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}
	return digits.String()
}

// ExtractSKUFromText finds a card/topup denomination in free text.
func ExtractSKUFromText(msg string) string {
	key := NormalizeKey(msg)
	matches := skuDenomPattern.FindAllString(key, -1)
	if len(matches) == 0 {
		return ""
	}
	best := ""
	for _, m := range matches {
		norm := NormalizeSKU(m)
		if len(norm) >= 4 && len(norm) > len(best) {
			best = norm
		}
	}
	return best
}
