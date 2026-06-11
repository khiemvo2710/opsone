package chatresolve

import "testing"

func TestResolveProductPair_topupMobi(t *testing.T) {
	p, prov := ResolveProductPair("topup", "mobi")
	if p != "TOPUP_MOBI" || prov != "" {
		t.Fatalf("got product=%q provider=%q", p, prov)
	}
}

func TestResolveProduct_combined(t *testing.T) {
	if got := ResolveProduct("topup mobi"); got != "TOPUP_MOBI" {
		t.Fatalf("got %q", got)
	}
	if got := ResolveProduct("thẻ zing"); got != "ZING" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveProvider_carrierNotProvider(t *testing.T) {
	if got := ResolveProvider("mobi"); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
	if got := ResolveProvider("esale"); got != "ESALE" {
		t.Fatalf("got %q", got)
	}
}

func TestNormalizeToolArgs(t *testing.T) {
	args := NormalizeToolArgs(map[string]any{
		"product":  "topup",
		"provider": "mobi",
	})
	if args["product"] != "TOPUP_MOBI" {
		t.Fatalf("product=%v", args["product"])
	}
	if _, ok := args["provider"]; ok {
		t.Fatalf("provider should be cleared, got %v", args["provider"])
	}
}
