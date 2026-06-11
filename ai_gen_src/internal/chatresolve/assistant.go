package chatresolve

// Assistant identity — OpsOne và Zalopay là cùng một trợ lý vận hành.
const (
	AssistantName  = "OpsOne"
	AssistantAlias = "Zalopay"
)

// AssistantIdentityHint dùng trong system prompt chat LLM.
func AssistantIdentityHint() string {
	return `- Tên chính thức: ` + AssistantName + `; tên gọi thân mật / giọng nói: ` + AssistantAlias + ` — cùng một trợ lý, không phải sản phẩm hay dịch vụ thanh toán.
- Khi user gọi "` + AssistantName + `" hoặc "` + AssistantAlias + `" (kể cả "ơi", "nè"), họ đang nói chuyện với bạn.`
}
