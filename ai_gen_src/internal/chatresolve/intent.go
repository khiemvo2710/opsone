package chatresolve

import "strings"

// HistoryTurn is one prior chat message used for maintenance follow-up detection.
type HistoryTurn struct {
	Role    string
	Content string
}

// ChatIntent is a coarse routing label for direct reply + FAQ stats.
type ChatIntent string

const (
	IntentMaintenance ChatIntent = "maintenance"
	IntentMetrics     ChatIntent = "metrics"
	IntentUnknown     ChatIntent = "unknown"
)

var maintenanceQueryTokens = []string{
	"bao tri", "bảo trì", "maintenance",
	"dang bao tri", "đang bảo trì",
	"co bao tri", "có bảo trì",
	"dang bt", "đang bt", "co bt", "có bt",
	"co dang bao tri", "có đang bảo trì",
}

var metricsQueryTokens = []string{
	"pending", "gd pending", "gd pend",
	"quay don", "quay đơn", "quay don icel",
	"banding", "treo", "stuck", "ket don", "kẹt đơn",
	"ty le pending", "tỷ lệ pending", "ti le pending",
	"success", "fail", "metric", "chi so", "chỉ số",
	"gd fail", "gd loi", "gd lỗi",
	"co bi pending", "có bị pending", "dang bi pending", "đang bị pending",
	"co dang pending", "có đang pending",
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
	if IsKnownProduct(upper) {
		return upper
	}
	return ""
}

var globalMaintenanceTokens = []string{
	"ngoai ra", "ngoài ra",
	"con dich vu", "còn dịch vụ", "con loai dich vu", "còn loại dịch vụ",
	"dich vu nao", "dịch vụ nào", "loai dich vu nao", "loại dịch vụ nào",
	"tat ca", "tất cả", "toan bo", "toàn bộ",
	"dich vu khac", "dịch vụ khác", "nhung dich vu", "những dịch vụ",
}

// IsGlobalMaintenanceQuery is true when user asks which services are under maintenance (no single product).
func IsGlobalMaintenanceQuery(msg string) bool {
	key := NormalizeKey(msg)
	for _, token := range globalMaintenanceTokens {
		if strings.Contains(key, token) {
			return true
		}
	}
	return false
}

var maintenanceScopeTokens = []string{
	"the ", "thẻ ", "menh gia", "mệnh giá", "menh gia the", "mệnh giá thẻ",
}

// IsMetricsQuery reports whether the user asks about success/pending/fail or GD counts.
func IsMetricsQuery(msg string) bool {
	key := NormalizeKey(msg)
	for _, token := range metricsQueryTokens {
		if strings.Contains(key, token) {
			return true
		}
	}
	return false
}

// IsMaintenanceScopeQuery is true when the message names a card product and denomination.
func IsMaintenanceScopeQuery(msg string) bool {
	if IsMetricsQuery(msg) {
		return false
	}
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

// ShouldLookupMetrics decides whether to query metrics for chat direct reply.
func ShouldLookupMetrics(userMsg string, hist []HistoryTurn) bool {
	if IsMetricsQuery(userMsg) {
		// Only trigger direct metrics if we have a product in current msg OR history
		if ExtractProductFromText(userMsg) != "" {
			return true
		}
		if ExtractProductFromHistory(hist) != "" {
			return true
		}
	}
	return false
}

// DetectChatIntent picks the best direct-reply route (metrics before maintenance).
func DetectChatIntent(userMsg string, hist []HistoryTurn) ChatIntent {
	if ShouldLookupMetrics(userMsg, hist) {
		return IntentMetrics
	}
	if ShouldLookupMaintenance(userMsg, hist) {
		return IntentMaintenance
	}
	return IntentUnknown
}

// ShouldLookupMaintenance decides whether to query maintenance_windows for chat.
func ShouldLookupMaintenance(userMsg string, hist []HistoryTurn) bool {
	if IsMetricsQuery(userMsg) {
		return false
	}
	if IsSetMaintenanceCommand(userMsg) {
		return false
	}
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

// ExtractProductFromHistory scans recent turns (user or assistant) for a product_code.
func ExtractProductFromHistory(hist []HistoryTurn) string {
	for i := len(hist) - 1; i >= 0; i-- {
		if code := ExtractProductFromText(hist[i].Content); code != "" {
			return code
		}
	}
	return ""
}

// ExtractSKUFromHistory scans recent turns (user or assistant) for a denomination.
func ExtractSKUFromHistory(hist []HistoryTurn) string {
	for i := len(hist) - 1; i >= 0; i-- {
		if sku := ExtractSKUFromText(hist[i].Content); sku != "" {
			return sku
		}
	}
	return ""
}
