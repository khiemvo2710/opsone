package chatresolve

import "testing"

func TestIsApproveShortCommand(t *testing.T) {
	for _, msg := range []string{"ok", "duyệt", "đồng ý", "approve"} {
		if !IsApproveShortCommand(msg) {
			t.Fatalf("expected approve for %q", msg)
		}
	}
	if IsApproveShortCommand("metric topup mobi") {
		t.Fatal("long query should not be short approve")
	}
}

func TestIsRejectShortCommand(t *testing.T) {
	for _, msg := range []string{"không", "từ chối", "reject"} {
		if !IsRejectShortCommand(msg) {
			t.Fatalf("expected reject for %q", msg)
		}
	}
}

func TestParseScopeAutoMode(t *testing.T) {
	if got := ParseScopeAutoMode("đặt garena chế độ tự động"); got != "auto" {
		t.Fatalf("auto: got %q", got)
	}
	if got := ParseScopeAutoMode("tự động theo khung giờ 8h-22h"); got != "time_window" {
		t.Fatalf("time_window: got %q", got)
	}
	if got := ParseScopeAutoMode("chỉ đề xuất routing"); got != "recommend_only" {
		t.Fatalf("recommend_only: got %q", got)
	}
}

func TestIsSetMaintenanceCommand(t *testing.T) {
	cases := []struct {
		msg  string
		want bool
	}{
		{"bật bảo trì thẻ zing 30 phút", true},
		{"bảo trì giúp tôi thẻ Garena toàn bộ mệnh giá", true},
		{"bảo trì thẻ garena", true},
		{"thẻ zing có đang bảo trì không", false},
		{"thẻ Garena có đang bảo trì không", false},
		{"toàn bộ mệnh giá thẻ Garena có đang bảo trì hay không", false},
	}
	for _, c := range cases {
		if got := IsSetMaintenanceCommand(c.msg); got != c.want {
			t.Fatalf("IsSetMaintenanceCommand(%q) = %v, want %v", c.msg, got, c.want)
		}
	}
}

func TestShouldLookupMaintenance_setCommand(t *testing.T) {
	msg := "bảo trì giúp tôi thẻ Garena toàn bộ mệnh giá"
	if ShouldLookupMaintenance(msg, nil) {
		t.Fatal("set maintenance command should not trigger status lookup")
	}
}
