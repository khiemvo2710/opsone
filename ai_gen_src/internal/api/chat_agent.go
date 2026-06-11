package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"opsone/internal/catalog"
	"opsone/internal/chatresolve"
	"opsone/internal/llm"
	"opsone/internal/store"
	"opsone/internal/tools"
)

const chatMaxToolRounds = 6

var chatSessions = struct {
	mu   sync.Mutex
	data map[string][]llm.Message
}{data: make(map[string][]llm.Message)}

func chatSessionGet(sessionID string) []llm.Message {
	if sessionID == "" {
		return nil
	}
	chatSessions.mu.Lock()
	defer chatSessions.mu.Unlock()
	return append([]llm.Message(nil), chatSessions.data[sessionID]...)
}


func chatSessionAppend(sessionID string, msgs ...llm.Message) {
	if sessionID == "" {
		return
	}
	chatSessions.mu.Lock()
	defer chatSessions.mu.Unlock()
	h := chatSessions.data[sessionID]
	h = append(h, msgs...)
	if len(h) > 40 {
		h = h[len(h)-40:]
	}
	chatSessions.data[sessionID] = h
}

func (s *Server) llmClient() *llm.Client {
	return llm.NewClient(llm.Config{
		BaseURL: s.Config.LLMBaseURL(),
		APIKey:  s.Config.LLMAPIKey,
		Model:   s.Config.LLMModel,
		Timeout: time.Duration(s.Config.LLMTimeoutSec) * time.Second,
	})
}

func (s *Server) chatSystemPrompt(ctx context.Context, isAdmin bool, userDisplayName string) string {
	catalogHint := ""
	if products, err := s.DB.ListProducts(ctx, true); err == nil {
		catalogHint = chatresolve.CatalogHint(products)
	}
	admin := ""
	if isAdmin {
		admin = `
Quyền Admin: duyệt/từ chối routing hoặc bảo trì; bật/mở bảo trì; mở lại dịch vụ; đổi chế độ Tự động / Khung giờ / Chỉ đề xuất.
Luôn gọi list_pending_actions trước khi duyệt nếu chưa biết plan_id hoặc scope.
Phản hồi sau hành động: 1) Kết quả (Đã/Không) 2) Chi tiết scope/thời gian 3) Gợi ý tiếp (tuỳ chọn) — tối đa 3 dòng.`
	} else {
		admin = `
User không phải Admin: KHÔNG gọi tool duyệt/từ chối — hướng dẫn mở Dashboard hoặc liên hệ admin.`
	}
	userHint := ""
	if userDisplayName != "" {
		userHint = fmt.Sprintf(`
User chat với tên hiển thị: %s. Luôn xưng hô đúng tên này; không dùng tên khác từ lịch sử hội thoại.`, userDisplayName)
	}
	return `Bạn là trợ lý vận hành ` + chatresolve.AssistantName + ` — trả lời bằng tiếng Việt, ngắn gọn, chuyên nghiệp.
` + chatresolve.AssistantIdentityHint() + `
Bạn hỗ trợ tra cứu metric, routing, bảo trì, sự cố và (nếu được phép) duyệt đề xuất khi user yêu cầu.
` + admin + userHint + catalogHint + `
Quy tắc:
- Map tên viết tắt user (topup mobi, thẻ zing, thẻ garena, data vina…) sang đúng product_code trước khi gọi tool.
- User yêu cầu thao tác Dashboard (duyệt, bảo trì, mở lại, gia hạn, đổi chế độ…) → gọi execute_ui_action hoặc tool chuyên biệt — cùng API nút UI.
` + catalog.UIActionsPromptHint() + `
- mobi/vina/viettel là nhà mạng/dịch vụ — KHÔNG phải provider; provider chỉ ESALE, IMEDIA, SHOPPAY.
- Hỏi "dịch vụ X có đang bảo trì không" → get_maintenance(product=X), KHÔNG bắt buộc provider/sku; tóm tắt theo sku_code và provider_code từ kết quả.
- Tối đa 3 tool tra cứu mỗi lượt hội thoại trừ khi user yêu cầu duyệt (cần list_pending_actions).
- Không bịa số — chỉ dùng kết quả tool.
- User muốn đổi tên, tuổi, xưng hô hoặc avatar chat → đồng ý cập nhật, không từ chối; xác nhận ngắn gọn.
- User nói ngắn "duyệt"/"ok" hoặc "từ chối"/"không" → xử lý pending gần nhất (list_pending_actions nếu cần).
- Câu hỏi ngoài phạm vi vận hành → từ chối lịch sự.`
}

func chatToolDefs(isAdmin bool) []llm.ToolDef {
	readOnly := []llm.ToolDef{
		toolDef("get_metrics", "Lấy success/pending/fail rate và GD. product có thể viết tắt (vd topup mobi). provider=ESALE|IMEDIA|SHOPPAY hoặc bỏ trống để xem cả 3.", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"product":  map[string]any{"type": "string", "description": "product_code hoặc viết tắt: topup mobi, thẻ zing"},
				"provider": map[string]any{"type": "string", "description": "ESALE, IMEDIA, SHOPPAY — không dùng mobi/vina/viettel"},
				"sku":      map[string]any{"type": "string"},
				"window":   map[string]any{"type": "string", "description": "vd 15m, 1h"},
			},
			"required": []string{"product"},
		}),
		toolDef("get_top_errors", "Lấy mã lỗi phổ biến (product viết tắt như get_metrics)", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"product":  map[string]any{"type": "string"},
				"provider": map[string]any{"type": "string"},
				"sku":      map[string]any{"type": "string"},
				"window":   map[string]any{"type": "string"},
				"limit":    map[string]any{"type": "integer"},
			},
			"required": []string{"product", "provider"},
		}),
		toolDef("get_routing", "Lấy tỉ lệ routing hiện tại (product_code hoặc viết tắt)", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"product": map[string]any{"type": "string"},
			},
			"required": []string{"product"},
		}),
		toolDef("get_maintenance", "Lấy bảo trì active/scheduled. Chỉ cần product (vd GARENA, thẻ garena); provider/sku tùy chọn — bỏ trống để xem tất cả mệnh giá.", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"product":  map[string]any{"type": "string", "description": "product_code hoặc viết tắt: thẻ garena, zing"},
				"provider": map[string]any{"type": "string", "description": "ESALE|IMEDIA|SHOPPAY — tùy chọn"},
				"sku":      map[string]any{"type": "string", "description": "mệnh giá — tùy chọn"},
			},
			"required": []string{"product"},
		}),
		toolDef("get_incidents", "Liệt kê sự cố gần đây", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"limit": map[string]any{"type": "integer"},
			},
		}),
		toolDef("list_pending_actions", "Liệt kê kế hoạch routing và đề xuất bảo trì chờ duyệt", map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}),
	}
	if !isAdmin {
		return readOnly
	}
	adminTools := []llm.ToolDef{
		toolDef("approve_routing_plan", "Duyệt kế hoạch routing theo plan_id", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"plan_id":      map[string]any{"type": "integer"},
				"proposed_pct": map[string]any{"type": "object"},
			},
			"required": []string{"plan_id"},
		}),
		toolDef("approve_scope_routing", "Duyệt routing cho product/sku (dùng pending plan hoặc proposed_pct)", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"product":      map[string]any{"type": "string"},
				"sku":          map[string]any{"type": "string"},
				"proposed_pct": map[string]any{"type": "object"},
			},
			"required": []string{"product"},
		}),
		toolDef("reject_routing_plan", "Từ chối kế hoạch routing theo plan_id", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"plan_id": map[string]any{"type": "integer"},
			},
			"required": []string{"plan_id"},
		}),
		toolDef("reject_scope_routing", "Từ chối routing pending của product/sku", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"product": map[string]any{"type": "string"},
				"sku":     map[string]any{"type": "string"},
			},
			"required": []string{"product", "sku"},
		}),
		toolDef("approve_scope_maintenance", "Duyệt đề xuất bảo trì SKU", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"product":      map[string]any{"type": "string"},
				"sku":          map[string]any{"type": "string"},
				"starts_at":    map[string]any{"type": "string"},
				"ends_at":      map[string]any{"type": "string"},
				"duration_min": map[string]any{"type": "integer"},
			},
			"required": []string{"product", "sku"},
		}),
		toolDef("reject_scope_maintenance", "Từ chối đề xuất bảo trì SKU", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"product": map[string]any{"type": "string"},
				"sku":     map[string]any{"type": "string"},
			},
			"required": []string{"product", "sku"},
		}),
		toolDef("set_maintenance", "Bật bảo trì thủ công (admin). Bỏ sku để áp toàn dịch vụ; bỏ provider để tất cả provider đang routing.", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"product":      map[string]any{"type": "string"},
				"sku":          map[string]any{"type": "string"},
				"provider":     map[string]any{"type": "string", "description": "ESALE|IMEDIA|SHOPPAY — tùy chọn"},
				"duration_min": map[string]any{"type": "integer"},
				"starts_at":    map[string]any{"type": "string"},
				"ends_at":      map[string]any{"type": "string"},
			},
			"required": []string{"product"},
		}),
		toolDef("reopen_service", "Mở lại dịch vụ — hủy BT active + baseline routing. Bỏ sku để mở toàn dịch vụ.", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"product": map[string]any{"type": "string"},
				"sku":     map[string]any{"type": "string"},
			},
			"required": []string{"product"},
		}),
		toolDef("set_scope_auto", "Đặt chế độ routing/BT: auto | time_window | recommend_only (Chỉ đề xuất).", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"product":      map[string]any{"type": "string"},
				"sku":          map[string]any{"type": "string"},
				"auto_action":  map[string]any{"type": "string", "description": "auto | time_window | recommend_only"},
				"window_start": map[string]any{"type": "string"},
				"window_end":   map[string]any{"type": "string"},
			},
			"required": []string{"product", "auto_action"},
		}),
		toolDef("execute_ui_action", "Thực hiện thao tác Dashboard (cùng API nút UI). action: list_pending|approve_reject|set_maintenance|extend_maintenance|reopen_service|restore_baseline|set_scope_auto. decision=reject khi approve_reject từ chối.", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"action":       map[string]any{"type": "string"},
				"product":      map[string]any{"type": "string"},
				"sku":          map[string]any{"type": "string"},
				"provider":     map[string]any{"type": "string"},
				"duration_min": map[string]any{"type": "integer"},
				"auto_action":  map[string]any{"type": "string"},
				"decision":     map[string]any{"type": "string", "description": "approve | reject — cho approve_reject"},
				"utterance":    map[string]any{"type": "string", "description": "câu user gốc (tuỳ chọn)"},
			},
			"required": []string{"action"},
		}),
	}
	return append(readOnly, adminTools...)
}

func toolDef(name, desc string, params map[string]any) llm.ToolDef {
	return llm.ToolDef{
		Type: "function",
		Function: llm.FunctionSpec{
			Name:        name,
			Description: desc,
			Parameters:  params,
		},
	}
}

func (s *Server) chatAgentReply(ctx context.Context, sessionID, userMsg, userDisplayName, actor string, isAdmin bool, out *ChatTurnOutcome) (string, error) {
	hist := chatHistoryTurns(sessionID)
	intent := chatresolve.DetectChatIntent(userMsg, hist)
	if out == nil {
		out = chatTurnOutcomeInit(intent)
	} else if out.IntentKey == "" && intent != chatresolve.IntentUnknown {
		out.IntentKey = string(intent)
	}

	if reply, ok := s.tryChatMetricsReply(ctx, sessionID, userMsg); ok {
		out.setDirectRoute("direct_metrics", userMsg)
		s.recordChatIntentHit(intent, userMsg, out.Route, out.ActionResult)
		chatSessionAppend(sessionID, llm.Message{Role: "user", Content: userMsg}, llm.Message{Role: "assistant", Content: reply})
		return reply, nil
	}
	if reply, ok := s.tryChatCommandReply(ctx, sessionID, userMsg, actor, isAdmin); ok {
		out.setDirectRoute("direct_"+chatCommandRouteKey(userMsg), userMsg)
		s.recordChatIntentHit(intent, userMsg, out.Route, out.ActionResult)
		chatSessionAppend(sessionID, llm.Message{Role: "user", Content: userMsg}, llm.Message{Role: "assistant", Content: reply})
		return reply, nil
	}
	if reply, ok := s.tryChatMaintenanceReply(ctx, sessionID, userMsg); ok {
		out.setDirectRoute("direct_maintenance", userMsg)
		s.recordChatIntentHit(intent, userMsg, out.Route, out.ActionResult)
		chatSessionAppend(sessionID, llm.Message{Role: "user", Content: userMsg}, llm.Message{Role: "assistant", Content: reply})
		return reply, nil
	}

	client := s.llmClient()
	if !client.Enabled() {
		out.setDirectRoute("stub", userMsg)
		s.recordChatIntentHit(intent, userMsg, out.Route, out.ActionResult)
		return s.chatReplyStub(ctx, userMsg), nil
	}

	toolsList := chatToolDefs(isAdmin)
	messages := []llm.Message{{Role: "system", Content: s.chatSystemPrompt(ctx, isAdmin, userDisplayName)}}
	if hist := chatSessionGet(sessionID); len(hist) > 0 {
		messages = append(messages, hist...)
	}
	messages = append(messages, llm.Message{Role: "user", Content: userMsg})
	out.Route = "llm"

	for round := 0; round < chatMaxToolRounds; round++ {
		assistant, err := client.ChatCompletion(ctx, messages, toolsList, 0.4)
		if err != nil {
			out.ActionResult = store.ChatActionError
			s.recordChatIntentHit(intent, userMsg, out.Route, out.ActionResult)
			return "", err
		}
		messages = append(messages, assistant)

		if len(assistant.ToolCalls) == 0 {
			reply := strings.TrimSpace(assistant.Content)
			if reply == "" {
				reply = "Xin lỗi, tôi chưa trả lời được. Bạn thử hỏi cụ thể hơn (product, SKU, provider)."
			}
			out.ActionResult = store.ChatActionSuccess
			s.recordChatIntentHit(intent, userMsg, out.Route, out.ActionResult)
			chatSessionAppend(sessionID, llm.Message{Role: "user", Content: userMsg}, assistant)
			return reply, nil
		}

		for _, tc := range assistant.ToolCalls {
			out.appendTool(tc.Function.Name)
			result, execErr := s.chatExecTool(ctx, sessionID, tc.Function.Name, tc.Function.Arguments, actor, isAdmin)
			if execErr != nil {
				result = map[string]any{"error": execErr.Error()}
			}
			raw, _ := json.Marshal(result)
			messages = append(messages, llm.Message{
				Role:       "tool",
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
				Content:    string(raw),
			})
		}
	}
	out.ActionResult = store.ChatActionNoOp
	s.recordChatIntentHit(intent, userMsg, out.Route, out.ActionResult)
	return "Cần thêm bước xử lý — mở Dashboard hoặc thử lại câu hỏi ngắn hơn.", nil
}

func (s *Server) chatExecTool(ctx context.Context, sessionID, name, argsJSON, actor string, isAdmin bool) (any, error) {
	var args map[string]any
	if argsJSON != "" {
		_ = json.Unmarshal([]byte(argsJSON), &args)
	}
	if args == nil {
		args = map[string]any{}
	}
	args = chatresolve.NormalizeToolArgs(args)

	switch name {
	case "get_metrics":
		product := strArg(args, "product")
		provider := strArg(args, "provider")
		sku := strArg(args, "sku")
		window := strArg(args, "window")
		if product != "" && provider == "" {
			out, err := s.chatMetricsAllProviders(ctx, product, sku, window)
			return out, err
		}
		out, err := s.Tools.GetMetrics(ctx, tools.GetMetricsInput{
			Product: product, Provider: provider, SKU: sku, Window: window,
		})
		return out, toolErr(err)
	case "get_top_errors":
		out, err := s.Tools.GetTopErrors(ctx, tools.GetTopErrorsInput{
			Product:  strArg(args, "product"),
			Provider: strArg(args, "provider"),
			SKU:      strArg(args, "sku"),
			Window:   strArg(args, "window"),
			Limit:    intArg(args, "limit"),
		})
		return out, toolErr(err)
	case "get_routing":
		out, err := s.Tools.GetRouting(ctx, tools.GetRoutingInput{Product: strArg(args, "product")})
		return out, toolErr(err)
	case "get_maintenance":
		product := strArg(args, "product")
		if product == "" {
			product = chatresolve.ExtractProductFromText(strArg(args, "provider"))
		}
		provider := strArg(args, "provider")
		if provider != "" {
			if _, ok := map[string]struct{}{"ESALE": {}, "IMEDIA": {}, "SHOPPAY": {}}[provider]; !ok {
				if product == "" {
					product = chatresolve.ResolveProduct(provider)
				}
				provider = ""
			}
		}
		sku := strArg(args, "sku")
		if sku != "" {
			sku = chatresolve.NormalizeSKU(sku)
		}
		out, err := s.maintenanceForChat(ctx, product, sku)
		if err != nil {
			return nil, toolErr(err)
		}
		if provider != "" {
			out = tools.FilterMaintenanceByProvider(out, provider)
		}
		return tools.EnrichMaintenanceOutput(product, out), nil
	case "get_incidents":
		limit := intArg(args, "limit")
		if limit <= 0 {
			limit = 5
		}
		rows, err := s.DB.ListIncidents(ctx, nil, limit, 0)
		if err != nil {
			return nil, err
		}
		items := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			summary := ""
			if row.Summary.Valid {
				summary = row.Summary.String
			}
			items = append(items, map[string]any{
				"incident_id":  row.IncidentID,
				"product_code": row.ProductCode,
				"severity":     row.Severity,
				"status":       row.Status,
				"summary":      summary,
			})
		}
		return map[string]any{"incidents": items}, nil
	case "list_pending_actions":
		out, err := s.chatListPendingActions(ctx)
		if err == nil {
			if data, ok := out.(map[string]any); ok {
				if f := chatSessionFocusFromPending(data); f.Kind != "" {
					chatSessionFocusSet(sessionID, f)
				}
			}
		}
		return out, err
	case "approve_routing_plan":
		if !isAdmin {
			return nil, fmt.Errorf("cần quyền Admin")
		}
		msg, err := s.chatApproveRoutingPlan(ctx, uint64(intArg(args, "plan_id")), actor, pctArg(args, "proposed_pct"))
		return map[string]any{"message": msg}, err
	case "approve_scope_routing":
		if !isAdmin {
			return nil, fmt.Errorf("cần quyền Admin")
		}
		msg, err := s.chatApproveScopeRouting(ctx, strArg(args, "product"), strArg(args, "sku"), actor, pctArg(args, "proposed_pct"))
		return map[string]any{"message": msg}, err
	case "reject_routing_plan":
		if !isAdmin {
			return nil, fmt.Errorf("cần quyền Admin")
		}
		msg, err := s.chatRejectRoutingPlan(ctx, uint64(intArg(args, "plan_id")), actor)
		return map[string]any{"message": msg}, err
	case "reject_scope_routing":
		if !isAdmin {
			return nil, fmt.Errorf("cần quyền Admin")
		}
		msg, err := s.chatRejectScopeRouting(ctx, strArg(args, "product"), strArg(args, "sku"), actor)
		return map[string]any{"message": msg}, err
	case "approve_scope_maintenance":
		if !isAdmin {
			return nil, fmt.Errorf("cần quyền Admin")
		}
		msg, err := s.chatApproveScopeMaintenance(ctx,
			strArg(args, "product"), strArg(args, "sku"), actor,
			strArg(args, "starts_at"), strArg(args, "ends_at"), intArg(args, "duration_min"))
		return map[string]any{"message": msg}, err
	case "reject_scope_maintenance":
		if !isAdmin {
			return nil, fmt.Errorf("cần quyền Admin")
		}
		msg, err := s.chatRejectScopeMaintenance(ctx, strArg(args, "product"), strArg(args, "sku"), actor)
		return map[string]any{"message": msg}, err
	case "set_maintenance":
		if !isAdmin {
			return nil, fmt.Errorf("cần quyền Admin")
		}
		product := strArg(args, "product")
		sku := strArg(args, "sku")
		if sku != "" {
			sku = chatresolve.NormalizeSKU(sku)
		}
		msg, err := s.chatSetMaintenance(ctx, product, sku, strArg(args, "provider"),
			intArg(args, "duration_min"), actor, "")
		if err != nil {
			return nil, err
		}
		return map[string]any{"message": msg}, nil
	case "reopen_service":
		if !isAdmin {
			return nil, fmt.Errorf("cần quyền Admin")
		}
		sku := strArg(args, "sku")
		if sku != "" {
			sku = chatresolve.NormalizeSKU(sku)
		}
		msg, err := s.chatReopenService(ctx, strArg(args, "product"), sku, actor)
		return map[string]any{"message": msg}, err
	case "set_scope_auto":
		if !isAdmin {
			return nil, fmt.Errorf("cần quyền Admin")
		}
		sku := strArg(args, "sku")
		if sku != "" {
			sku = chatresolve.NormalizeSKU(sku)
		}
		msg, err := s.chatSetScopeAuto(ctx, strArg(args, "product"), sku, strArg(args, "auto_action"),
			fmt.Sprintf("%s %s", strArg(args, "window_start"), strArg(args, "window_end")))
		return map[string]any{"message": msg}, err
	case "execute_ui_action":
		if !isAdmin {
			return nil, fmt.Errorf("cần quyền Admin")
		}
		msg, err := s.executeUIActionFromTool(ctx, sessionID, args, actor, isAdmin)
		if err != nil {
			return nil, err
		}
		return map[string]any{"message": msg}, nil
	default:
		return nil, fmt.Errorf("tool không hỗ trợ: %s", name)
	}
}

func strArg(args map[string]any, key string) string {
	v, ok := args[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprint(t)
	}
}

func intArg(args map[string]any, key string) int {
	v, ok := args[key]
	if !ok || v == nil {
		return 0
	}
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	default:
		return 0
	}
}

func pctArg(args map[string]any, key string) map[string]float64 {
	v, ok := args[key]
	if !ok || v == nil {
		return nil
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	out := make(map[string]float64)
	for k, val := range m {
		if f, ok := val.(float64); ok {
			out[strings.ToUpper(k)] = f
		}
	}
	return out
}

func (s *Server) chatMetricsAllProviders(ctx context.Context, product, sku, window string) (map[string]any, error) {
	providers := []string{"ESALE", "IMEDIA", "SHOPPAY"}
	items := make([]map[string]any, 0, len(providers))
	for _, prov := range providers {
		out, err := s.Tools.GetMetrics(ctx, tools.GetMetricsInput{
			Product: product, Provider: prov, SKU: sku, Window: window,
		})
		if err != nil {
			continue
		}
		items = append(items, map[string]any{
			"provider": prov,
			"metrics":  out,
		})
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("không có metric cho %s (thử window 15m hoặc 1h)", product)
	}
	return map[string]any{
		"product":   product,
		"providers": items,
		"note":      "Tổng hợp ESALE + IMEDIA + SHOPPAY",
	}, nil
}

func toolErr(err error) error {
	if err == nil {
		return nil
	}
	return err
}

func (s *Server) chatReplyStub(ctx context.Context, msg string) string {
	return s.chatReply(ctx, msg)
}
