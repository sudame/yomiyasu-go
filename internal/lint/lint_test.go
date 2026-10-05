package lint

import (
	"os"
	"slices"
	"testing"

	"github.com/sudame/yomiyasu-go/internal/rules"
)

func rulesOf(r Result) []string {
	var out []string
	for _, f := range r.Findings {
		out = append(out, f.Rule)
	}
	return out
}

func TestSilentKowareruIsNotCountedTwice(t *testing.T) {
	r := Lint("データが静かに壊れる。\n", nil)
	n := 0
	for _, f := range r.Findings {
		if f.Rule == "metaphor_verb" {
			n++
		}
	}
	if n != 1 {
		t.Errorf("metaphor_verb = %d 件, want 1: %v", n, rulesOf(r))
	}
}

func TestMetricsKeepIntZeroForEmptyText(t *testing.T) {
	m := Lint("", nil).Metrics
	if !m.ListRatio.IsInt || !m.BoldPer1000.IsInt {
		t.Errorf("空の文書の比率は int の 0 になる: %+v", m)
	}
}

func TestEachRuleCanBeDisabled(t *testing.T) {
	dense, err := os.ReadFile("../../testdata/compat/inputs/local/dense.md")
	if err != nil {
		t.Fatal(err)
	}
	samples := map[string]string{
		"excess_bold":               string(dense),
		"excess_list":               string(dense),
		"sentence_end_repetition":   "これは本です。あれも本です。それも本です。\n",
		"bold_not_rendered":         "これは**「重要」**です。\n",
		"emoji_prohibited":          "楽しい一日でした😀\n",
		"redundant_bracket":         "# 説明（概要）\n",
		"unnatural_halfwidth_space": "この README を読みます\n",
		"trailing_colon":            "次の項目：\n",
		"slop_vocabulary":           "解像度を上げます\n",
		"metaphor_verb":             "地味に効きます\n",
		"meta_filler":               "重要なのは、設計です\n",
		"negative_parallelism":      "速さではなく正しさを選びます\n",
	}
	for _, id := range rules.All {
		t.Run(id, func(t *testing.T) {
			text, ok := samples[id]
			if !ok {
				t.Fatalf("%s の例文がない", id)
			}
			if !slices.Contains(rulesOf(Lint(text, nil)), id) {
				t.Fatalf("有効なときに %s を検出しない: %v", id, rulesOf(Lint(text, nil)))
			}
			r := Lint(text, map[string]bool{id: true})
			if slices.Contains(rulesOf(r), id) {
				t.Errorf("無効にしても %s を検出した", id)
			}
			if r.Score != score(r.Findings) {
				t.Errorf("無効にした規則が点数に残っている: %d", r.Score)
			}
		})
	}
}
