package game

import (
	"strings"
	"testing"
)

func TestLocalizeExactAndFormattedMessages(t *testing.T) {
	cases := []struct{ lang, in, contains string }{
		{"en", "bạn không có món này", "do not have"},
		{"ja", "bạn không có món này", "持っていません"},
		{"en", "cần 490 vàng, bạn còn 100", "need 490 gold; you have 100"},
		{"ja", "cần 490 vàng, bạn còn 100", "490金貨必要"},
		{"en", "Đã đăng bán 2× Kiếm sắt giá 90 vàng (phí 5).", "Listed 2× Iron sword for 90 gold"},
		{"ja", "Đã đăng bán 2× Kiếm sắt giá 90 vàng (phí 5).", "鉄の剣×2を90金貨で出品"},
		{"en", "💎 Ghép 6 mảnh thành Hồng Ngọc cấp 1.", "Forged 6 shards"},
		{"ja", "💎 Ghép 6 mảnh thành Hồng Ngọc cấp 1.", "ルビー"},
	}
	for _, tc := range cases {
		got := localizeText(tc.lang, tc.in)
		if !strings.Contains(got, tc.contains) {
			t.Errorf("%s: %q -> %q, want %q", tc.lang, tc.in, got, tc.contains)
		}
	}
}

func TestLocalizeTextFallsBackAndLeavesPetNamesAlone(t *testing.T) {
	in := "Momo hạ gục Heo Rừng hoang, nhặt 19 🪙!"
	if got := localizeText("vi", in); got != in {
		t.Fatalf("Vietnamese changed: %q", got)
	}
	// The pet's own name remains Momo; known generated names may be localised later.
	if got := localizeText("en", in); !strings.HasPrefix(got, "Momo ") {
		t.Fatalf("pet name was altered: %q", got)
	}
	if got := localizeText("fr", in); got != in {
		t.Fatalf("unsupported locale should use source text: %q", got)
	}
}

func TestDuelReplayTextIsLocalisedWithoutMutatingStoredHistory(t *testing.T) {
	in := duelResult{Turns: []duelTurn{{Text: "Momo dùng 🔥 Phun lửa Lv2: −18"}, {Text: "Momo cắn một phát: −8"}}}
	out := localizeDuel("ja", in)
	if !strings.Contains(out.Turns[0].Text, "火炎") {
		t.Fatalf("skill name did not localise: %q", out.Turns[0].Text)
	}
	if in.Turns[0].Text != "Momo dùng 🔥 Phun lửa Lv2: −18" {
		t.Fatalf("localisation mutated stored history: %q", in.Turns[0].Text)
	}
}

func TestRuntimeRulesCompileAndHaveBothTranslations(t *testing.T) {
	for i, r := range formattedText {
		if r.re == nil || r.en == "" || r.ja == "" {
			t.Errorf("rule %d incomplete: %+v", i, r)
		}
	}
	for source, pair := range exactText {
		if source == "" || pair[0] == "" || pair[1] == "" {
			t.Errorf("exact translation incomplete: %q %+v", source, pair)
		}
	}
}
