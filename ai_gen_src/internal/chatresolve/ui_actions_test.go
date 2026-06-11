package chatresolve

import (
	"testing"

	"opsone/internal/catalog"
)

func TestDetectUIAction_extendAndRestore(t *testing.T) {
	cases := []struct {
		msg    string
		want   catalog.UIActionKey
		hasSKU bool
	}{
		{"gia hạn bảo trì thẻ Garena 10.000 thêm 60 phút", catalog.UIActionExtendMaint, true},
		{"trả lại routing baseline thẻ zing ESALE", catalog.UIActionRestoreBaseline, false},
		{"mở lại thẻ Garena 10.000", catalog.UIActionReopenService, true},
		{"Mở bảo trì thẻ Garena 10.000", catalog.UIActionReopenService, true},
		{"bỏ bảo trì thẻ Garena 10.000", catalog.UIActionReopenService, true},
		{"agent có thể làm gì trên ui", catalog.UIActionHelp, false},
	}
	for _, tc := range cases {
		got, slots := DetectUIAction(tc.msg)
		if got != tc.want {
			t.Fatalf("DetectUIAction(%q)=%q want %q", tc.msg, got, tc.want)
		}
		if tc.hasSKU && slots.SKU == "" {
			t.Fatalf("DetectUIAction(%q) missing sku", tc.msg)
		}
	}
}

func TestIsRestoreBaseline_notReopen(t *testing.T) {
	msg := "mở lại dịch vụ Garena"
	if IsRestoreBaselineCommand(msg) {
		t.Fatal("reopen should not be restore baseline")
	}
}
