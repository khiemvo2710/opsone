package chatresolve

import "testing"

func TestNormalizeSKU(t *testing.T) {
	cases := map[string]string{
		"10.000":   "10000",
		"10.000đ":  "10000",
		"10000":    "10000",
		"100.000":  "100000",
	}
	for in, want := range cases {
		if got := NormalizeSKU(in); got != want {
			t.Fatalf("NormalizeSKU(%q)=%q want %q", in, got, want)
		}
	}
}

func TestExtractSKUFromText_mobifone(t *testing.T) {
	msg := "Ý tôi nói là thẻ Mobifone 10.000"
	if got := ExtractSKUFromText(msg); got != "10000" {
		t.Fatalf("got %q want 10000", got)
	}
	if got := ExtractProductFromText(msg); got != "MOBIFONE" {
		t.Fatalf("product=%q want MOBIFONE", got)
	}
	if !IsMaintenanceScopeQuery(msg) {
		t.Fatal("expected maintenance scope query")
	}
}
