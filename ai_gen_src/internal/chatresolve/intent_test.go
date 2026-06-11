package chatresolve

import "testing"

func TestIsMaintenanceQuery(t *testing.T) {
	if !IsMaintenanceQuery("thẻ Garena có đang bảo trì không") {
		t.Fatal("expected maintenance query")
	}
	if !IsMaintenanceQuery("thẻ garena có bảo trì không") {
		t.Fatal("expected có bảo trì query")
	}
	if !IsMaintenanceQuery("toan bo menh gia the garena co dang bao tri") {
		t.Fatal("expected maintenance query without diacritics")
	}
	if IsMaintenanceQuery("Garena có sự cố không") {
		t.Fatal("incident query should not match maintenance")
	}
}

func TestExtractProductFromText_noFakeProduct(t *testing.T) {
	msg := "Ngoài ra còn loại dịch vụ nào đang bảo trì không"
	if got := ExtractProductFromText(msg); got != "" {
		t.Fatalf("got %q want empty", got)
	}
	if !IsGlobalMaintenanceQuery(msg) {
		t.Fatal("expected global maintenance query")
	}
}

func TestShouldLookupMaintenance_followUp(t *testing.T) {
	hist := []HistoryTurn{
		{Role: "user", Content: "thẻ Mobifone có đang bảo trì không"},
		{Role: "assistant", Content: "không có"},
	}
	msg := "Ý tôi nói là thẻ Mobifone 10.000"
	if !ShouldLookupMaintenance(msg, hist) {
		t.Fatal("expected follow-up maintenance lookup")
	}
}

func TestExtractProductFromText_garena(t *testing.T) {
	msg := "tôi hỏi là toàn bộ mệnh giá thẻ Garena có đang bảo trì hay không"
	if got := ExtractProductFromText(msg); got != "GARENA" {
		t.Fatalf("got %q want GARENA", got)
	}
}

func TestIsMetricsQuery_pendingBanding(t *testing.T) {
	msg := "hiện tại thẻ Mobifone 50.000 đi qua rồi quay đơn icel có đang bị banding không"
	if !IsMetricsQuery(msg) {
		t.Fatal("expected metrics/pending query")
	}
	if ShouldLookupMaintenance(msg, nil) {
		t.Fatal("pending query must not route to maintenance")
	}
	if !ShouldLookupMetrics(msg, nil) {
		t.Fatal("expected metrics lookup")
	}
	if DetectChatIntent(msg, nil) != IntentMetrics {
		t.Fatalf("got %q want metrics", DetectChatIntent(msg, nil))
	}
}

func TestIsMaintenanceScopeQuery_notMetrics(t *testing.T) {
	msg := "thẻ Mobifone 50.000 có đang bảo trì không"
	if !IsMaintenanceScopeQuery(msg) {
		t.Fatal("expected maintenance scope query")
	}
	pending := "thẻ Mobifone 50.000 có GD pending không"
	if IsMaintenanceScopeQuery(pending) {
		t.Fatal("pending scope must not match maintenance scope")
	}
}
