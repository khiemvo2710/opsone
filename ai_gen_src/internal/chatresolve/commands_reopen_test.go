package chatresolve

import "testing"

func TestIsReopenServiceCommand_garenaSku(t *testing.T) {
	msg := "mở lại thẻ Garena 10.000"
	if !IsReopenServiceCommand(msg) {
		t.Fatalf("expected reopen command for %q", msg)
	}
	if IsSetMaintenanceCommand(msg) {
		t.Fatal("reopen must not match set maintenance")
	}
	if ExtractSKUFromText(msg) != "10000" {
		t.Fatalf("sku=%q want 10000", ExtractSKUFromText(msg))
	}
	if ExtractProductFromText(msg) != "GARENA" {
		t.Fatalf("product=%q want GARENA", ExtractProductFromText(msg))
	}

	for _, m := range []string{
		"Mở bảo trì thẻ Garena 10.000",
		"bỏ bảo trì thẻ Garena 10.000",
		"mở bảo trì thẻ garena 10.000",
		"bỏ bảo trì thẻ garena 10.000",
	} {
		if !IsReopenServiceCommand(m) {
			t.Errorf("expected reopen command for %q", m)
		}
		if IsSetMaintenanceCommand(m) {
			t.Errorf("reopen must not match set maintenance for %q", m)
		}
	}
}
