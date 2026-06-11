package chatresolve

import "strings"

// HistoryTurn is one prior chat message used for maintenance follow-up detection.
type HistoryTurn struct {
	Role    string
	Content string
}

var maintenanceQueryTokens = []string{
	"bao tri", "bảo trì", "maintenance",
	"dang bao tri", "đang bảo trì",
	"co bao tri", "có bảo trì",
	"dang bt", "đang bt", "co bt", "có bt",
	"co dang bao tri", "có đang bảo trì",
}

// IsMaintenanceQuery reports whether the user asks about service maintenance windows.
func IsMaintenanceQuery(msg string) bool {
	key := NormalizeKey(msg)
	for _, token := range maintenanceQueryTokens {
		if strings.Contains(key, token) {
			return true
		}
	}
	return false
}

// ExtractProductFromText maps free-text (user message or tool arg) to product_code.
// Uses longest matching alias to avoid "the" beating "the garena".
func ExtractProductFromText(msg string) string {
	key := NormalizeKey(msg)
	if key == "" {
		return ""
	}
	if code := lookupProduct(key); code != "" {
		return code
	}

	bestCode := ""
	bestLen := 0
	for alias, code := range staticProductAliases {
		if !strings.Contains(key, alias) {
			continue
		}
		if len(alias) > bestLen {
			bestLen = len(alias)
			bestCode = code
		}
	}
	if bestCode != "" {
		return bestCode
	}

	upper := strings.ToUpper(strings.ReplaceAll(key, " ", "_"))
	if _, ok := knownProductCodes[upper]; ok {
		return upper
	}
	return ResolveProduct(key)
}

var maintenanceScopeTokens = []string{
	"the ", "thẻ ", "menh gia", "mệnh giá", "menh gia the", "mệnh giá thẻ",
}

// IsMaintenanceScopeQuery is true when the message names a card product and denomination.
func IsMaintenanceScopeQuery(msg string) bool {
	if ExtractProductFromText(msg) == "" || ExtractSKUFromText(msg) == "" {
		return false
	}
	key := NormalizeKey(msg)
	for _, token := range maintenanceScopeTokens {
		if strings.Contains(key, token) {
			return true
		}
	}
	return false
}

// ShouldLookupMaintenance decides whether to query maintenance_windows for chat.
func ShouldLookupMaintenance(userMsg string, hist []HistoryTurn) bool {
	if IsMaintenanceQuery(userMsg) || IsMaintenanceScopeQuery(userMsg) {
		return true
	}
	if ExtractProductFromText(userMsg) == "" && ExtractSKUFromText(userMsg) == "" {
		return false
	}
	for i := len(hist) - 1; i >= 0; i-- {
		if hist[i].Role != "user" {
			continue
		}
		if IsMaintenanceQuery(hist[i].Content) || IsMaintenanceScopeQuery(hist[i].Content) {
			return true
		}
	}
	return false
}

// ExtractProductFromHistory scans recent user turns for a product_code.
func ExtractProductFromHistory(hist []HistoryTurn) string {
	for i := len(hist) - 1; i >= 0; i-- {
		if hist[i].Role != "user" {
			continue
		}
		if code := ExtractProductFromText(hist[i].Content); code != "" {
			return code
		}
	}
	return ""
}

// ExtractSKUFromHistory scans recent user turns for a denomination.
func ExtractSKUFromHistory(hist []HistoryTurn) string {
	for i := len(hist) - 1; i >= 0; i-- {
		if hist[i].Role != "user" {
			continue
		}
		if sku := ExtractSKUFromText(hist[i].Content); sku != "" {
			return sku
		}
	}
	return ""
}
