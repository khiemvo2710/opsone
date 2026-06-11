package catalog

import "testing"

func TestFormatCycleHealthSummary_legacyBracket(t *testing.T) {
	in := "Sự cố: [DATA_MOBI DATA_VIETTEL GARENA MOBIFONE TOPUP_VIETTEL VIETTEL VINAPHONE ZING]"
	out := FormatCycleHealthSummary(in, nil)
	want := "Sự cố: Data Mobifone, Data Viettel, Thẻ Garena, Thẻ Mobifone, Thẻ Viettel, Thẻ Vinaphone, Thẻ Zing, Topup Viettel"
	if out != want {
		t.Fatalf("got %q want %q", out, want)
	}
}

func TestFormatCycleHealthSummary_alreadyFormatted(t *testing.T) {
	in := "Sự cố: Data Mobifone, Thẻ Garena"
	if got := FormatCycleHealthSummary(in, nil); got != in {
		t.Fatalf("expected unchanged, got %q", got)
	}
}
