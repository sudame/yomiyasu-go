package diff

import (
	"strings"
	"testing"
)

func TestEndingKind(t *testing.T) {
	cases := map[string]string{
		"保存してください。":    "依頼",
		"保存しましょう。":     "勧め",
		"設定を読み込みます。":   "動作（〜します）",
		"設定が変わります。":    "説明（〜ます）",
		"手順が重要です。":     "評価",
		"設定を読み込んでいます。": "説明（〜ています）",
	}
	for in, want := range cases {
		if got := endingKind(in); got != want {
			t.Errorf("endingKind(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestEndingsMostCommonKeepsFirstSeenOrderOnTies(t *testing.T) {
	out := Endings("保存してください。設定を読み込みます。保存してください。設定を読み込みます。\n", "")
	if !strings.HasPrefix(out, "■ 文末の種類（表を除く）: 依頼 2、動作（〜します） 2\n") {
		t.Errorf("Endings =\n%s", out)
	}
}
