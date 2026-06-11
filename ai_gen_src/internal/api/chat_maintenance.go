package api

import (
	"context"
	"fmt"
	"time"

	"opsone/internal/chatresolve"
	"opsone/internal/tools"
)

func chatHistoryTurns(sessionID string) []chatresolve.HistoryTurn {
	raw := chatSessionGet(sessionID)
	out := make([]chatresolve.HistoryTurn, 0, len(raw))
	for _, m := range raw {
		out = append(out, chatresolve.HistoryTurn{Role: m.Role, Content: m.Content})
	}
	return out
}

// maintenanceForChat uses the same DB rows + in-window filter as GET /dashboard/overview.
func (s *Server) maintenanceForChat(ctx context.Context, product, sku string) (tools.GetMaintenanceOutput, error) {
	now := time.Now()
	rows, err := s.DB.ListMaintenanceWindows(ctx, product, "active", 500)
	if err != nil {
		return tools.GetMaintenanceOutput{}, err
	}
	out := tools.BuildMaintenanceOutputFromRows(rows, now, "", sku)

	scheduled, err := s.DB.ListMaintenanceWindows(ctx, product, "scheduled", 200)
	if err == nil {
		for _, r := range scheduled {
			if sku != "" && r.SKUCode != sku {
				continue
			}
			if !r.StartsAt.After(now) {
				continue
			}
			item := tools.MaintenanceItem{
				MaintenanceID: r.MaintenanceID,
				ProductCode:   r.ProductCode,
				ProviderCode:  r.ProviderCode,
				SKUCode:       r.SKUCode,
				StartsAt:      r.StartsAt,
				EndsAt:        r.EndsAt,
				Status:        r.Status,
			}
			if r.Reason.Valid {
				item.Reason = r.Reason.String
			}
			out.Scheduled = append(out.Scheduled, item)
		}
	}
	out.Meta = tools.Meta{QueriedAt: now}
	return out, nil
}

func (s *Server) tryChatMaintenanceReply(ctx context.Context, sessionID, userMsg string) (string, bool) {
	hist := chatHistoryTurns(sessionID)
	if !chatresolve.ShouldLookupMaintenance(userMsg, hist) {
		return "", false
	}

	product := chatresolve.ExtractProductFromText(userMsg)
	sku := chatresolve.ExtractSKUFromText(userMsg)
	if product == "" || !chatresolve.IsKnownProduct(product) {
		product = chatresolve.ExtractProductFromHistory(hist)
	}
	if sku == "" {
		sku = chatresolve.ExtractSKUFromHistory(hist)
	}
	if product != "" && !chatresolve.IsKnownProduct(product) {
		product = ""
	}
	if product == "" {
		if chatresolve.IsGlobalMaintenanceQuery(userMsg) || chatresolve.IsMaintenanceQuery(userMsg) {
			out, err := s.maintenanceForChat(ctx, "", "")
			if err != nil {
				return fmt.Sprintf("Không tra được danh sách bảo trì: %s", err.Error()), true
			}
			return tools.FormatAllMaintenanceReply(out), true
		}
		return "Không nhận diện được dịch vụ — thử nêu rõ: *thẻ Garena*, *thẻ Mobifone 10.000*, *topup mobi*…", true
	}

	out, err := s.maintenanceForChat(ctx, product, sku)
	if err != nil {
		return fmt.Sprintf("Không tra được bảo trì **%s**: %s", product, err.Error()), true
	}
	return tools.FormatMaintenanceReply(product, sku, out), true
}
