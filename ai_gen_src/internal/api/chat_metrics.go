package api

import (
	"context"
	"fmt"
	"math"
	"time"

	"opsone/internal/chatresolve"
	"opsone/internal/tools"
)

func (s *Server) tryChatMetricsReply(ctx context.Context, sessionID, userMsg string) (string, bool) {
	hist := chatHistoryTurns(sessionID)
	if !chatresolve.ShouldLookupMetrics(userMsg, hist) {
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
	if product == "" || !chatresolve.IsKnownProduct(product) {
		return "Không nhận diện được dịch vụ — thử: *thẻ Mobifone 50.000 có GD pending không*, *topup mobi pending*…", true
	}

	p, err := s.DB.GetProductByCode(ctx, product)
	if err != nil {
		return fmt.Sprintf("Không tra được **%s**: %s", product, err.Error()), true
	}
	if p.RoutingMode == "sku" && sku == "" {
		return fmt.Sprintf("**%s** cần mệnh giá/SKU — ví dụ: *thẻ Mobifone 50.000 GD pending*.", product), true
	}

	res, err := s.metricsForChat(ctx, product, sku)
	if err != nil {
		return fmt.Sprintf("Không tra được metric **%s**: %s", product, err.Error()), true
	}
	pendingFocused := chatresolve.IsMetricsQuery(userMsg)
	return tools.FormatMetricsReply(res, pendingFocused), true
}

func (s *Server) metricsForChat(ctx context.Context, product, sku string) (tools.MetricsChatResult, error) {
	settings, err := s.DB.GetAgentSettings(ctx)
	if err != nil {
		return tools.MetricsChatResult{}, err
	}
	th, err := s.DB.GetProductThreshold(ctx, product)
	if err != nil {
		return tools.MetricsChatResult{}, err
	}
	dur, winLabel, _ := tools.ParseWindow("")
	since := time.Now().Add(-dur)

	providers := []string{"ESALE", "IMEDIA", "SHOPPAY"}
	rows := make([]tools.ProviderMetricsLine, 0, len(providers))
	for _, prov := range providers {
		line := tools.ProviderMetricsLine{Provider: prov}
		m, ok, err := s.DB.GetMetricsInWindow(ctx, settings.DataSource, product, sku, prov, since)
		if err != nil {
			return tools.MetricsChatResult{}, err
		}
		if !ok {
			rows = append(rows, line)
			continue
		}
		pTxn := uint(math.Round(float64(m.TotalTransactions) * m.PendingRate / 100))
		fTxn := uint(math.Round(float64(m.TotalTransactions) * m.FailRate / 100))
		line.HasData = true
		line.SuccessRate = m.SuccessRate
		line.PendingRate = m.PendingRate
		line.FailRate = m.FailRate
		line.PendingTxn = pTxn
		line.FailTxn = fTxn
		rows = append(rows, line)
	}
	return tools.BuildMetricsChatResult(product, sku, winLabel, th, rows), nil
}
