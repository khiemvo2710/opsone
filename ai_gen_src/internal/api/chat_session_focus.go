package api

import "sync"

type chatPendingFocus struct {
	Kind    string // routing_plan | maintenance_suggestion
	PlanID  uint64
	Product string
	SKU     string
	RecID   uint64
	Summary string
}

var chatSessionFocus = struct {
	mu   sync.Mutex
	data map[string]chatPendingFocus
}{data: make(map[string]chatPendingFocus)}

func chatSessionFocusSet(sessionID string, f chatPendingFocus) {
	if sessionID == "" {
		return
	}
	chatSessionFocus.mu.Lock()
	defer chatSessionFocus.mu.Unlock()
	chatSessionFocus.data[sessionID] = f
}

func chatSessionFocusGet(sessionID string) (chatPendingFocus, bool) {
	if sessionID == "" {
		return chatPendingFocus{}, false
	}
	chatSessionFocus.mu.Lock()
	defer chatSessionFocus.mu.Unlock()
	f, ok := chatSessionFocus.data[sessionID]
	return f, ok
}

func chatSessionFocusClear(sessionID string) {
	if sessionID == "" {
		return
	}
	chatSessionFocus.mu.Lock()
	defer chatSessionFocus.mu.Unlock()
	delete(chatSessionFocus.data, sessionID)
}

func chatSessionFocusFromPending(raw map[string]any) chatPendingFocus {
	if plans, ok := raw["routing_plans"].([]map[string]any); ok && len(plans) > 0 {
		p := plans[0]
		f := chatPendingFocus{
			Kind:    "routing_plan",
			Product: strAny(p, "product_code"),
			SKU:     strAny(p, "sku_code"),
			Summary: "routing " + strAny(p, "product_code") + "/" + strAny(p, "sku_code"),
		}
		if id, ok := p["plan_id"].(float64); ok {
			f.PlanID = uint64(id)
		}
		if rv, ok := p["reason_vi"].(string); ok && rv != "" {
			f.Summary = rv
		}
		return f
	}
	if items, ok := raw["maintenance_suggestions"].([]map[string]any); ok && len(items) > 0 {
		m := items[0]
		f := chatPendingFocus{
			Kind:    "maintenance_suggestion",
			Product: strAny(m, "product_code"),
			SKU:     strAny(m, "sku_code"),
			Summary: "bảo trì " + strAny(m, "product_code") + "/" + strAny(m, "sku_code"),
		}
		if id, ok := m["id"].(float64); ok {
			f.RecID = uint64(id)
		}
		if d, ok := m["detail"].(string); ok && d != "" {
			f.Summary = d
		}
		return f
	}
	return chatPendingFocus{}
}

func strAny(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
