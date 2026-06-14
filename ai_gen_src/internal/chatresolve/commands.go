package chatresolve

import (
	"regexp"
	"strconv"
	"strings"
)

var durationMinPattern = regexp.MustCompile(`(\d+)\s*(?:phut|p|min|minutes?)`)
var percentPattern = regexp.MustCompile(`(\d{1,3})\s*(?:%|phan tram|phần trăm|percent)`)

var approveTokens = []string{
	"duyet", "dong y", "ok", "approve", "yes",
}

var rejectTokens = []string{
	"tu choi", "reject", "bo qua", "khong",
}

var listPendingTokens = []string{
	"xem pending", "cho duyet", "cho duyệt", "viec cho duyet", "việc chờ duyệt",
	"liet ke pending", "liệt kê pending", "pending",
}

var reopenTokens = []string{
	"mo lai dich vu", "mở lại dịch vụ", "mo lai", "mở lại", "reopen", "ket thuc bao tri", "kết thúc bảo trì",
	"mo bao tri", "mở bảo trì", "bo bao tri", "bỏ bảo trì",
}

var scopeAutoTokens = []string{
	"che do", "chế độ", "dat che do", "đặt chế độ", "cau hinh", "cấu hình", "scope auto", "tu dong", "tự động", "chi de xuat", "chỉ đề xuất",
}

// CommandKey normalizes user command text (lowercase, fold Vietnamese).
func CommandKey(msg string) string {
	return trimMsgKey(msg)
}

func trimMsgKey(msg string) string {
	key := strings.Trim(NormalizeKey(msg), " .!?,;:")
	return foldCommandKey(key)
}

func foldCommandKey(s string) string {
	repl := strings.NewReplacer(
		"à", "a", "á", "a", "ả", "a", "ã", "a", "ạ", "a",
		"ă", "a", "ằ", "a", "ắ", "a", "ẳ", "a", "ẵ", "a", "ặ", "a",
		"â", "a", "ầ", "a", "ấ", "a", "ẩ", "a", "ẫ", "a", "ậ", "a",
		"è", "e", "é", "e", "ẻ", "e", "ẽ", "e", "ẹ", "e",
		"ê", "e", "ề", "e", "ế", "e", "ể", "e", "ễ", "e", "ệ", "e",
		"ì", "i", "í", "i", "ỉ", "i", "ĩ", "i", "ị", "i",
		"ò", "o", "ó", "o", "ỏ", "o", "õ", "o", "ọ", "o",
		"ô", "o", "ồ", "o", "ố", "o", "ổ", "o", "ỗ", "o", "ộ", "o",
		"ơ", "o", "ờ", "o", "ớ", "o", "ở", "o", "ỡ", "o", "ợ", "o",
		"ù", "u", "ú", "u", "ủ", "u", "ũ", "u", "ụ", "u",
		"ư", "u", "ừ", "u", "ứ", "u", "ử", "u", "ữ", "u", "ự", "u",
		"ỳ", "y", "ý", "y", "ỷ", "y", "ỹ", "y", "ỵ", "y",
		"đ", "d",
	)
	return repl.Replace(s)
}

func tokenMatch(key string, tokens []string) bool {
	for _, t := range tokens {
		if key == t {
			return true
		}
	}
	return false
}

func containsAny(key string, tokens []string) bool {
	for _, t := range tokens {
		if strings.Contains(key, t) {
			return true
		}
	}
	return false
}

// IsApproveShortCommand is true for bare approval ("ok", "duyệt", …).
func IsApproveShortCommand(msg string) bool {
	key := trimMsgKey(msg)
	if tokenMatch(key, approveTokens) {
		return true
	}
	for _, t := range approveTokens {
		if strings.HasPrefix(key, t+" ") && len(strings.Fields(key)) <= 4 {
			return true
		}
	}
	return false
}

// IsRejectShortCommand is true for bare rejection ("không", "từ chối", …).
func IsRejectShortCommand(msg string) bool {
	key := trimMsgKey(msg)
	if tokenMatch(key, rejectTokens) {
		return true
	}
	for _, t := range rejectTokens {
		if strings.HasPrefix(key, t+" ") && len(strings.Fields(key)) <= 4 {
			return true
		}
	}
	return false
}

// IsListPendingCommand asks to list pending routing/maintenance items.
func IsListPendingCommand(msg string) bool {
	key := trimMsgKey(msg)
	return containsAny(key, listPendingTokens)
}

// IsReopenServiceCommand asks to end maintenance and restore service.
func IsReopenServiceCommand(msg string) bool {
	key := trimMsgKey(msg)
	if IsMaintenanceQuery(msg) && !containsAny(key, reopenTokens) {
		return false
	}
	return containsAny(key, reopenTokens)
}

// IsMaintenanceStatusQuery is true when user asks whether maintenance is active (not a set command).
func IsMaintenanceStatusQuery(msg string) bool {
	key := trimMsgKey(msg)
	statusHints := []string{
		"co dang", "co bao tri", "co bt", "co dang bao tri", "co dang bt",
		"dang bao tri", "dang bt",
		"hay khong", "hong khong", "phai khong", "co khong",
		"liet ke", "xem ", "check ", "tra cuu", "hien co", "hien tai",
		"tinh trang", "trang thai",
	}
	for _, h := range statusHints {
		if strings.Contains(key, h) {
			return true
		}
	}
	if strings.Contains(msg, "?") || strings.Contains(msg, "？") {
		return true
	}
	if strings.Contains(key, " co ") && strings.Contains(key, " khong") {
		return true
	}
	return false
}

// IsSetMaintenanceCommand asks to proactively start maintenance.
func IsSetMaintenanceCommand(msg string) bool {
	key := trimMsgKey(msg)
	if IsReopenServiceCommand(msg) {
		return false
	}
	if !strings.Contains(key, "bao tri") && !strings.Contains(key, " bt") {
		return false
	}
	if IsMaintenanceStatusQuery(msg) {
		return false
	}
	actionHints := []string{
		"bat bao tri", "bat bt", "kich hoat",
		"giup toi", "giup em", "giup ",
		"dat bao tri", "dat bt", "thuc hien",
		"cho toi", "cho em",
		"toan bo menh gia", "toan bo", "tat ca menh gia", "tat ca",
		"ca dich vu", "toan dich vu",
	}
	for _, h := range actionHints {
		if strings.Contains(key, h) {
			return true
		}
	}
	if strings.HasPrefix(key, "bao tri ") && ExtractProductFromText(msg) != "" {
		return true
	}
	return strings.Contains(key, "bat bao tri")
}

// allScopeTokens — broad terms meaning "all" or "whole system".
var allScopeTokens = []string{
	"tat ca", "toan bo", "toan he thong", "moi ", "all ",
}

// allObjectTokens — terms meaning "services / products".
var allObjectTokens = []string{
	"dich vu", "san pham", "dv", "sp", "service",
}

// serviceTypeTokens — service-type words that act as the "object" in phrases like
// "tất cả thẻ", "tất cả topup", "tất cả data".
// Includes "top up" (space) and common STT mishears ("tot ap", "top ap").
var serviceTypeTokens = []string{
	"topup", "top up", "tot ap", "top ap", "cap ap", " nap", " data",
}

// isAllServicesPhrase returns true when the key contains an "all" word AND a "service"
// (or service-type) word, regardless of words in between.
func isAllServicesPhrase(key string) bool {
	hasScope := containsAny(key, allScopeTokens)
	if !hasScope {
		return strings.Contains(key, "toan he thong")
	}
	if containsAny(key, allObjectTokens) {
		return true
	}
	// "tất cả thẻ" / "tất cả topup" / "tất cả data"
	if containsAny(key, serviceTypeTokens) {
		return true
	}
	// "the" at end of string or surrounded by spaces (thẻ / card)
	if strings.HasSuffix(key, " the") || strings.Contains(key, " the ") {
		return true
	}
	return false
}

// ExtractAllServicesFilters returns the set of service_type filters when the user
// qualifies "tất cả" with one or more service types ("thẻ", "topup", "data").
// Multiple types can appear in one message: "data và top up" → ["topup_data","topup"].
// Returns nil (empty) when no type qualifier is found → caller treats as "all services".
func ExtractAllServicesFilters(msg string) []string {
	key := trimMsgKey(msg)
	seen := map[string]bool{}
	var filters []string
	add := func(f string) {
		if !seen[f] {
			seen[f] = true
			filters = append(filters, f)
		}
	}
	if containsAny(key, []string{"topup", "top up", "tot ap", "top ap", "cap ap", " nap"}) {
		add("topup")
	}
	if strings.Contains(key, " data") || strings.HasSuffix(key, "data") {
		add("topup_data")
	}
	if strings.HasSuffix(key, " the") || strings.Contains(key, " the ") {
		add("card")
	}
	return filters
}

// IsSetAllMaintenanceCommand is true when the user wants to put ALL services into maintenance.
// Returns false when a specific product is named — e.g. "bảo trì tất cả thẻ Garena" is
// a per-product command (all SKUs of Garena), not a global all-services command.
func IsSetAllMaintenanceCommand(msg string) bool {
	key := trimMsgKey(msg)
	if !strings.Contains(key, "bao tri") && !strings.Contains(key, " bt") {
		return false
	}
	if IsReopenServiceCommand(msg) {
		return false
	}
	if !isAllServicesPhrase(key) {
		return false
	}
	// A named product → product-scoped command, not global.
	if ExtractProductFromText(msg) != "" {
		return false
	}
	return true
}

// IsReopenAllServicesCommand is true when the user wants to cancel ALL maintenance / reopen everything.
// Returns false when a specific product is named.
func IsReopenAllServicesCommand(msg string) bool {
	key := trimMsgKey(msg)
	if !containsAny(key, reopenTokens) || !isAllServicesPhrase(key) {
		return false
	}
	if ExtractProductFromText(msg) != "" {
		return false
	}
	return true
}

// IsSetScopeAutoCommand asks to change routing/maintenance auto mode.
func IsSetScopeAutoCommand(msg string) bool {
	key := trimMsgKey(msg)
	if !containsAny(key, scopeAutoTokens) {
		return false
	}
	return ParseScopeAutoMode(msg) != "" || strings.Contains(key, "chi de xuat") ||
		strings.Contains(key, "chỉ đề xuất") || strings.Contains(key, "khung gio") ||
		strings.Contains(key, "khung giờ")
}

// ParseScopeAutoMode maps Vietnamese to auto_action code.
func ParseScopeAutoMode(msg string) string {
	key := NormalizeKey(msg)
	if strings.Contains(key, "chi de xuat") || strings.Contains(key, "chỉ đề xuất") ||
		strings.Contains(key, "recommend only") {
		return "recommend_only"
	}
	if strings.Contains(key, "khung gio") || strings.Contains(key, "khung giờ") ||
		strings.Contains(key, "time window") || strings.Contains(key, "theo khung") {
		return "time_window"
	}
	if strings.Contains(key, "tu dong") || strings.Contains(key, "tự động") ||
		strings.Contains(key, " auto") || strings.HasSuffix(key, " auto") {
		if strings.Contains(key, "khung") {
			return "time_window"
		}
		return "auto"
	}
	return ""
}

// ExtractDurationMinutes parses duration from text (default 0 = caller picks default).
func ExtractDurationMinutes(msg string) int {
	key := NormalizeKey(msg)
	m := durationMinPattern.FindStringSubmatch(key)
	if len(m) < 2 {
		return 0
	}
	n, _ := strconv.Atoi(m[1])
	return n
}

// ExtractProviderFromText finds ESALE|IMEDIA|SHOPPAY in message, including STT aliases.
func ExtractProviderFromText(msg string) string {
	key := NormalizeKey(msg)
	// Check STT aliases first (e.g. "excel"→ESALE, "ai media"→IMEDIA)
	if code, ok := staticProviderAliases[key]; ok {
		return code
	}
	for alias, code := range staticProviderAliases {
		if strings.Contains(key, alias) {
			return code
		}
	}
	// Exact uppercase match
	upper := strings.ToUpper(key)
	for _, p := range []string{"ESALE", "IMEDIA", "SHOPPAY"} {
		if strings.Contains(upper, p) {
			return p
		}
	}
	return ""
}

// ExtractProviderPct returns the percentage (0–100) following a provider mention.
// E.g. "qua ai Media 100%" → ("IMEDIA", 100). Returns ("", -1) if not found.
func ExtractProviderPct(msg string) (provider string, pct int) {
	provider = ExtractProviderFromText(msg)
	if provider == "" {
		return "", -1
	}
	key := NormalizeKey(msg)
	m := percentPattern.FindStringSubmatch(key)
	if len(m) < 2 {
		return provider, -1
	}
	n, err := strconv.Atoi(m[1])
	if err != nil || n < 0 || n > 100 {
		return provider, -1
	}
	return provider, n
}

// ScopeAutoModeLabel returns Vietnamese label for auto_action.
func ScopeAutoModeLabel(mode string) string {
	switch mode {
	case "auto":
		return "Tự động"
	case "time_window":
		return "Tự động theo khung giờ"
	default:
		return "Chỉ đề xuất"
	}
}
