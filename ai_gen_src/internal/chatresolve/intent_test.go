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
