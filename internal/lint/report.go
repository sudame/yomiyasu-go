package lint

import (
	"fmt"
	"strings"

	"github.com/sudame/yomiyasu-go/internal/pycompat"
	"github.com/sudame/yomiyasu-go/internal/rules"
)

// JSON は本家の --json と同じ出力を返す。
func JSON(r Result) string {
	findings := make([]any, len(r.Findings))
	for i, f := range r.Findings {
		findings[i] = pycompat.Obj{
			{K: "rule", V: f.Rule}, {K: "line", V: f.Line}, {K: "severity", V: f.Severity},
			{K: "message", V: f.Message}, {K: "snippet", V: f.Snippet},
		}
	}
	m := r.Metrics
	return pycompat.Dumps(pycompat.Obj{
		{K: "score", V: r.Score},
		{K: "is_clean", V: r.IsClean},
		{K: "metrics", V: pycompat.Obj{
			{K: "char_count", V: m.CharCount}, {K: "total_lines", V: m.TotalLines},
			{K: "list_lines", V: m.ListLines}, {K: "list_ratio", V: m.ListRatio},
			{K: "bold_count", V: m.BoldCount}, {K: "bold_per_1000", V: m.BoldPer1000},
		}},
		{K: "findings", V: findings},
	}, 2) + "\n"
}

// TextReport は本家の --json なしの出力を返す。disabled が空なら本家と同じ文字列になる。
func TextReport(r Result, disabled map[string]bool) string {
	var b strings.Builder
	rule := func(c string) { b.WriteString(strings.Repeat(c, 60) + "\n") }
	m := r.Metrics
	rule("=")
	fmt.Fprintf(&b, "AIっぽさ 検査レポート (スコア: %d/100)\n", r.Score)
	rule("=")
	var off []string
	for _, id := range rules.All {
		if disabled[id] {
			off = append(off, id)
		}
	}
	if len(off) > 0 {
		fmt.Fprintf(&b, "・無効にした規則: %s\n", strings.Join(off, ", "))
	}
	boldNote, listNote := "(推奨: 2.0以下 / 警告: 3.0超)", "(推奨: 15%以下 / 警告: 25%超)"
	if disabled["excess_bold"] {
		boldNote = "(検査は無効)"
	}
	if disabled["excess_list"] {
		listNote = "(検査は無効)"
	}
	fmt.Fprintf(&b, "・文字数: %d | 行数: %d\n", m.CharCount, m.TotalLines)
	fmt.Fprintf(&b, "・太字頻度: 1,000字あたり %s 個 %s\n", m.BoldPer1000, boldNote)
	fmt.Fprintf(&b, "・箇条書き比率: %s%% %s\n", ListPercent(m), listNote)
	rule("-")
	if r.IsClean {
		b.WriteString("[PASS] 設定された検査ルールによる指摘はありません。\n")
	} else {
		fmt.Fprintf(&b, "[NOTICE] %d 件の見直し候補が見つかりました。\n\n", len(r.Findings))
		for _, f := range r.Findings {
			fmt.Fprintf(&b, "L%d [%s] %s\n", f.Line, strings.ToUpper(f.Severity), f.Message)
			fmt.Fprintf(&b, "  > %s\n\n", f.Snippet)
		}
	}
	return b.String()
}
