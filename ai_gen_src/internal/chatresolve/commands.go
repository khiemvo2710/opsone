package chatresolve

import (
	"regexp"
	"strconv"
	"strings"
)

var durationMinPattern = regexp.MustCompile(`(\d+)\s*(?:phut|p|min|minutes?)`)

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

// ExtractProviderFromText finds ESALE|IMEDIA|SHOPPAY in message.
func ExtractProviderFromText(msg string) string {
	key := strings.ToUpper(NormalizeKey(msg))
	for _, p := range []string{"ESALE", "IMEDIA", "SHOPPAY"} {
		if strings.Contains(key, p) {
			return p
		}
	}
	return ""
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
