package diff

import (
	"fmt"
	"strings"

	"github.com/sudame/yomiyasu-go/internal/pycompat"
)

func strs(xs []string) []any {
	out := make([]any, len(xs))
	for i, x := range xs {
		out[i] = x
	}
	return out
}

func flagsObj(fs []Flag) []any {
	out := make([]any, len(fs))
	for i, f := range fs {
		out[i] = pycompat.Obj{{K: "note", V: f.Note}, {K: "sentences", V: strs(f.Sentences)}}
	}
	return out
}

// JSON は本家の --json と同じ出力（json.dumps(d, ensure_ascii=False, indent=1)）を返す。
func JSON(d Result) string {
	markersObj := make([]any, len(d.Markers))
	for i, m := range d.Markers {
		markersObj[i] = pycompat.Obj{
			{K: "kind", V: m.Kind}, {K: "orig", V: m.Orig}, {K: "rewrite", V: m.Rewrite},
			{K: "orig_hits", V: strs(m.OrigHits)}, {K: "rewrite_hits", V: strs(m.RwHits)},
		}
	}
	spans := make([]any, len(d.Spans))
	for i, s := range d.Spans {
		spans[i] = pycompat.Obj{
			{K: "added", V: s.Added}, {K: "was", V: s.Was}, {K: "kinds", V: strs(s.Kinds)},
			{K: "new_words", V: strs(s.NewWords)}, {K: "where", V: s.Where},
		}
	}
	logic := make([]any, len(d.Logic))
	for i, p := range d.Logic {
		logic[i] = pycompat.Obj{{K: "kind", V: p.Kind}, {K: "sentence", V: p.Sentence}}
	}
	boldObj := make([]any, len(d.Bold))
	for i, p := range d.Bold {
		boldObj[i] = pycompat.Obj{{K: "line", V: p.Line}, {K: "found", V: p.Found}, {K: "suggest", V: p.Suggest}, {K: "how", V: p.How}}
	}
	changes := make([]any, len(d.Changes))
	for i, c := range d.Changes {
		changes[i] = pycompat.Obj{
			{K: "orig", V: c.Orig}, {K: "orig_kind", V: c.OrigKind},
			{K: "rewrite", V: c.Rewrite}, {K: "rewrite_kind", V: c.RewriteKind},
		}
	}
	var stance any
	if d.Stance != "" {
		stance = d.Stance
	}
	return pycompat.Dumps(pycompat.Obj{
		{K: "markers", V: markersObj},
		{K: "new_words", V: strs(d.NewWords)},
		{K: "lost_words", V: strs(d.LostWords)},
		{K: "structure", V: strs(d.Structure)},
		{K: "spans", V: spans},
		{K: "logic", V: logic},
		{K: "bold", V: boldObj},
		{K: "endings", V: pycompat.Obj{
			{K: "stance", V: stance},
			{K: "changes", V: changes},
			{K: "flags", V: flagsObj(d.Flags)},
			{K: "orig_flags", V: flagsObj(d.OrigFlags)},
		}},
	}, 1) + "\n"
}

func orNone(xs []string) string {
	if len(xs) == 0 {
		return "なし"
	}
	return strings.Join(xs, "、")
}

// Report は本家の --json なしの出力を返す。
func Report(d Result) string {
	var lines []string
	if len(d.Markers) > 0 {
		lines = append(lines, "■ 言い回しの種類の増減（意味が変わりやすいところ）")
		for _, m := range d.Markers {
			lines = append(lines, fmt.Sprintf("- %s: %d → %d（元: %s／後: %s）", m.Kind, m.Orig, m.Rewrite, orNone(m.OrigHits), orNone(m.RwHits)))
		}
	}
	if len(d.NewWords) > 0 {
		lines = append(lines, "■ 元の文にない語: "+strings.Join(d.NewWords, "、"))
	}
	if len(d.LostWords) > 0 {
		lines = append(lines, "■ 消えた語: "+strings.Join(d.LostWords, "、"))
	}
	for _, s := range d.Structure {
		lines = append(lines, "■ "+s)
	}
	if len(d.Logic) > 0 {
		lines = append(lines, "■ つながりを確かめる場所（書き直した文。何と何をつないでいるか言えるか）")
		for _, p := range d.Logic {
			lines = append(lines, fmt.Sprintf("- %s: %s", p.Kind, truncate(p.Sentence, 44)))
		}
	}
	if len(d.Changes) > 0 {
		lines = append(lines, "■ 文末の種類が変わった文（立場に合う向きか見る）")
		for _, c := range d.Changes {
			lines = append(lines, fmt.Sprintf("- %s → %s: %s", c.OrigKind, c.RewriteKind, truncate(c.Rewrite, 44)))
		}
	}
	if len(d.Flags) > 0 {
		if d.Stance != "" {
			lines = append(lines, fmt.Sprintf("■ 文末の立場（%sの文書として見た）", d.Stance))
		} else {
			lines = append(lines, "■ 文末の立場が混ざっている候補（立場を決めてから見る）")
		}
		for _, f := range d.Flags {
			lines = append(lines, "- "+f.Note)
			for _, s := range f.Sentences {
				lines = append(lines, "  ・"+truncate(s, 44))
			}
		}
	}
	if len(d.Bold) > 0 {
		lines = append(lines, boldHead("書き直した文。"))
		lines = append(lines, boldLines(d.Bold)...)
	}
	if len(lines) == 0 {
		return "（候補なし）"
	}
	return strings.Join(lines, "\n")
}
