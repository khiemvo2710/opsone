package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"opsone/internal/chatresolve"
	"opsone/internal/llm"
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

func (s *Server) chatSystemPrompt(ctx context.Context, isAdmin bool) string {
	catalog := ""
	if products, err := s.DB.ListProducts(ctx, true); err == nil {
		catalog = chatresolve.CatalogHint(products)
	}
	admin := ""
	if isAdmin {
		admin = `
Quyền Admin: khi user YÊU CẦU RÕ RÀNG (duyệt/từ chối/approve/reject) routing hoặc bảo trì, bạn có thể gọi tool duyệt/từ chối.
Luôn gọi list_pending_actions trước khi duyệt nếu chưa biết plan_id hoặc scope.
Sau khi duyệt/từ chối, tóm tắt kết quả bằng tiếng Việt.`
	} else {
		admin = `
User không phải Admin: KHÔNG gọi tool duyệt/từ chối — hướng dẫn mở Dashboard hoặc liên hệ admin.`
	}
	return `Bạn là trợ lý vận hành OpsOne — trả lời bằng tiếng Việt, ngắn gọn, chuyên nghiệp.
Bạn hỗ trợ tra cứu metric, routing, bảo trì, sự cố và (nếu được phép) duyệt đề xuất khi user yêu cầu.
` + admin + catalog + `
Quy tắc:
- Map tên viết tắt user (topup mobi, thẻ zing, data vina…) sang đúng product_code trước khi gọi tool.
- mobi/vina/viettel là nhà mạng/dịch vụ — KHÔNG phải provider; provider chỉ ESALE, IMEDIA, SHOPPAY.
- Tối đa 3 tool tra cứu mỗi lượt hội thoại trừ khi user yêu cầu duyệt (cần list_pending_actions).
- Không bịa số — chỉ dùng kết quả tool.
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
		toolDef("get_maintenance", "Lấy cửa sổ bảo trì active/scheduled", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"product":  map[string]any{"type": "string"},
				"provider": map[string]any{"type": "string"},
				"sku":      map[string]any{"type": "string"},
			},
			"required": []string{"product", "provider"},
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

func (s *Server) chatAgentReply(ctx context.Context, sessionID, userMsg, actor string, isAdmin bool) (string, error) {
	client := s.llmClient()
	if !client.Enabled() {
		return s.chatReplyStub(ctx, userMsg), nil
	}

	toolsList := chatToolDefs(isAdmin)
	messages := []llm.Message{{Role: "system", Content: s.chatSystemPrompt(ctx, isAdmin)}}
	if hist := chatSessionGet(sessionID); len(hist) > 0 {
		messages = append(messages, hist...)
	}
	messages = append(messages, llm.Message{Role: "user", Content: userMsg})

	for round := 0; round < chatMaxToolRounds; round++ {
		assistant, err := client.ChatCompletion(ctx, messages, toolsList, 0.4)
		if err != nil {
			return "", err
		}
		messages = append(messages, assistant)

		if len(assistant.ToolCalls) == 0 {
			reply := strings.TrimSpace(assistant.Content)
			if reply == "" {
				reply = "Xin lỗi, tôi chưa trả lời được. Bạn thử hỏi cụ thể hơn (product, SKU, provider)."
			}
			chatSessionAppend(sessionID, llm.Message{Role: "user", Content: userMsg}, assistant)
			return reply, nil
		}

		for _, tc := range assistant.ToolCalls {
			result, execErr := s.chatExecTool(ctx, tc.Function.Name, tc.Function.Arguments, actor, isAdmin)
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
	return "Cần thêm bước xử lý — mở Dashboard hoặc thử lại câu hỏi ngắn hơn.", nil
}

func (s *Server) chatExecTool(ctx context.Context, name, argsJSON, actor string, isAdmin bool) (any, error) {
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
		out, err := s.Tools.GetMaintenance(ctx, tools.GetMaintenanceInput{
			Product:  strArg(args, "product"),
			Provider: strArg(args, "provider"),
			SKU:      strArg(args, "sku"),
		})
		return out, toolErr(err)
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
		return s.chatListPendingActions(ctx)
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
