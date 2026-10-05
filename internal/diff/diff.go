// Package diff は本家の yomiyasu_diff.py の比較を移植する。
package diff

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sudame/yomiyasu-go/internal/bold"
	"github.com/sudame/yomiyasu-go/internal/difflib"
	"github.com/sudame/yomiyasu-go/internal/pycompat"
	"github.com/sudame/yomiyasu-go/internal/pyre"
)

type marker struct {
	name string
	re   *pyre.Regexp
}

// 文末や言い回しの種類。数が増えた・減ったものを候補にする
var markers = []marker{
	{"依頼", pyre.MustCompile(`(?:て|で)ください`)},
	{"勧誘", pyre.MustCompile(`ましょう`)},
	{"義務", pyre.MustCompile(`なければ(?:なりません|ならない)|なくては(?:なりません|ならない)|ねばならない|必要があ(?:ります|る)|べき`)},
	{"評価", pyre.MustCompile(`大切|重要|大事|不可欠|欠かせ|肝心|肝要`)},
	{"可能", pyre.MustCompile(`でき(?:ます|る|ません|ない)|(?<![しさ])(?:ら|れ)(?:ます|ません)(?=[。、がけし]|$)|(?<=[作使書読言防守残伝送続進])(?:れ|え|け|め|せ)(?:ます|る)(?=[。、がけし]|$)`)},
	{"推量", pyre.MustCompile(`でしょう|だろう|かもしれ|はず|と思(?:います|う)|ようです|らしい|おそれ|たいところ`)},
	{"念押し", pyre.MustCompile(`のです|んです|こそ|まさに|必ず|絶対|常に`)},
	{"意志", pyre.MustCompile(`(?:に|ように|ことに)し(?:ます|ている|ています)`)},
	{"条件", pyre.MustCompile(`(?<!例)(?<!たと)(?:れ|え|け|せ|て|ね|め|べ)ば(?![かり])|なら(?=[、。]|$|\s)|たら(?=[、。]|$|\s)|場合`)},
	{"説明化", pyre.MustCompile(`ことが挙げられ|ということ|ことです|ことになります`)},
	{"つなぎ", pyre.MustCompile(`まず|また(?!は)|そして|さらに|次に|最後に|ただし|しかし|つまり|そのため|ので(?!す)|によって|ことで|ことにより`)},
}

var content = pyre.MustCompile(`[一-龥々〆ヵヶ]{2,}|[ァ-ヴー]{2,}|[A-Za-z][A-Za-z0-9_.+#/-]+`)

var logicPatterns = []marker{
	{"文頭のつなぎ", pyre.MustCompile(`^(?:ただし|しかし|一方|また|さらに|つまり|そのため|したがって|だから|それでも|なお|そこで|ところが)`)},
	{"主題の「も」", pyre.MustCompile(`^(?!それで)[^、。]{0,17}[^、。てでり]も、`)},
	{"予告だけの文", pyre.MustCompile(`^.{0,28}(?:が|も)あります。$|次の(?:点|こと|とおり|通り)です|以下の(?:点|こと|とおり|通り)`)},
	{"文頭の指示語", pyre.MustCompile(`^(?:これ|それ(?!でも|から)|こう(?:した|して|する|いう)|そう(?:した|して|する|いう)|この|その)(?!して)`)},
}

var (
	reBold          = pyre.MustCompile(`\*\*(.+?)\*\*`)
	reListMarkers   = pyre.MustCompile(`(?m)^\s*(?:[*\-・]|\d+[.)])\s+`)
	reHeadingMarks  = pyre.MustCompile(`(?m)^#+\s*`)
	reBlanks        = pyre.MustCompile(`[ \t]+`)
	reHasList       = pyre.MustCompile(`(?m)^\s*(?:[*\-・]|\d+[.)])\s+\S`)
	reSentenceBreak = pyre.MustCompile(`(?<=[。！？!?])|\n+`)
	reListLine      = pyre.MustCompile(`\s*(?:[*\-・]|\d+[.)])\s+`)
)

func remove(re *pyre.Regexp, s string) string {
	return re.Sub(s, func(pyre.Match) string { return "" })
}

func normalize(t string) string {
	t = reBold.SubGroup1(t)
	t = remove(reListMarkers, t)
	t = remove(reHeadingMarks, t)
	t = reBlanks.Sub(t, func(pyre.Match) string { return " " })
	return pycompat.Strip(t)
}

func hasList(t string) bool { return search(reHasList, t) }

func sentences(t string) []string {
	var out []string
	for _, s := range reSentenceBreak.Split(t) {
		if pycompat.Strip(s) != "" {
			out = append(out, s)
		}
	}
	return out
}

// paragraphs は段落を返す。続いた箇条書きは 1 つの段落とみなす。
func paragraphs(t string) []string {
	var blocks []string
	prevList := false
	for _, line := range strings.Split(t, "\n") {
		if pycompat.Strip(line) == "" {
			prevList = false
			continue
		}
		_, isList := reListLine.Match(line)
		if isList && prevList {
			continue
		}
		blocks = append(blocks, line)
		prevList = isList
	}
	return blocks
}

// LogicPoint は、つながりを確かめる場所。
type LogicPoint struct {
	Kind     string
	Sentence string
}

func logicPoints(t string) []LogicPoint {
	var pts []LogicPoint
	for _, s := range sentences(normalize(t)) {
		s2 := pycompat.Strip(s)
		for _, p := range logicPatterns {
			if search(p.re, s2) {
				pts = append(pts, LogicPoint{p.name, s2})
			}
		}
	}
	return pts
}

// Change は、書き直しで文末の種類が変わった文。
type Change struct {
	Orig, OrigKind, Rewrite, RewriteKind string
}

// endingChanges は、書き直しで文末の種類が変わった文を、似ている元の文と組にして返す。
func endingChanges(o, r string) []Change {
	ou, ru := endingUnits(o, false), endingUnits(r, false)
	var changes []Change
	for _, x := range ru {
		var best *unit
		score := 0.0
		for i := range ou {
			sc := difflib.New([]rune(ou[i].sentence), []rune(x.sentence)).Ratio()
			if sc > score {
				best, score = &ou[i], sc
			}
		}
		if best != nil && score >= 0.45 && best.kind != x.kind {
			kind := best.kind
			if best.where == "箇条書き" {
				kind += "・箇条書き"
			}
			changes = append(changes, Change{best.sentence, kind, x.sentence, x.kind})
		}
	}
	return changes
}

// Flag は、文末の立場について見直す候補。
type Flag struct {
	Note      string
	Sentences []string
}

func toFlags(fs []flag) []Flag {
	out := make([]Flag, len(fs))
	for i, f := range fs {
		ss := make([]string, len(f.rows))
		for n, r := range f.rows {
			ss[n] = r.sentence
		}
		out[i] = Flag{f.note, ss}
	}
	return out
}

// Marker は、言い回しの種類の数の増減。
type Marker struct {
	Kind             string
	Orig, Rewrite    int
	OrigHits, RwHits []string
}

// Span は、書き直しで足した部分。
type Span struct {
	Added, Was string
	Kinds      []string
	NewWords   []string
	Where      string
}

// Result は diff() の戻り値。
type Result struct {
	Markers   []Marker
	NewWords  []string
	LostWords []string
	Structure []string
	Spans     []Span
	Logic     []LogicPoint
	Bold      []bold.Problem
	Stance    string
	Changes   []Change
	Flags     []Flag
	OrigFlags []Flag
}

func context(t []rune, i, j int) string {
	const width = 18
	a, b := max(0, i-width), min(len(t), j+width)
	return string(t[a:i]) + "［" + string(t[i:j]) + "］" + string(t[j:b])
}

func uniqueSorted(xs []string, keep func(string) bool) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, x := range xs {
		if !seen[x] && keep(x) {
			seen[x] = true
			out = append(out, x)
		}
	}
	sort.Strings(out)
	return out
}

// Diff は元の文と書き直した文を比べる。stance が空文字なら立場を指定しない。
func Diff(origRaw, rwRaw, stance string) Result {
	o, r := normalize(origRaw), normalize(rwRaw)
	_, flags := stanceFlags(rwRaw, stance, false)
	_, origFlags := stanceFlags(origRaw, stance, false)
	d := Result{
		Markers: []Marker{}, Structure: []string{}, Spans: []Span{},
		Logic:   logicPoints(rwRaw),
		Bold:    bold.Problems(rwRaw, true),
		Stance:  stance,
		Changes: endingChanges(origRaw, rwRaw),
		Flags:   toFlags(flags), OrigFlags: toFlags(origFlags),
	}

	// 1. 種類ごとの数の増減
	for _, m := range markers {
		a, b := m.re.Count(o), m.re.Count(r)
		if a != b {
			d.Markers = append(d.Markers, Marker{m.name, a, b, m.re.FindAllStrings(o), m.re.FindAllStrings(r)})
		}
	}

	// 2. 元にない語・消えた語（語の単位で、文のどこかに出てくるかを見る）
	d.NewWords = uniqueSorted(content.FindAllStrings(r), func(w string) bool { return !strings.Contains(o, w) })
	d.LostWords = uniqueSorted(content.FindAllStrings(o), func(w string) bool { return !strings.Contains(r, w) })

	// 3. 構造
	if hasList(origRaw) && !hasList(rwRaw) {
		d.Structure = append(d.Structure, "箇条書きを地の文にした。各項目の文末（指示・説明・評価）が元と同じか見る")
	}
	po, pr := len(paragraphs(origRaw)), len(paragraphs(rwRaw))
	if pr < po {
		d.Structure = append(d.Structure, fmt.Sprintf("段落をまとめた（%d → %d）。まとめた段落の話題が1つか見る", po, pr))
	}
	if so, sr := len(sentences(o)), len(sentences(r)); sr != so {
		d.Structure = append(d.Structure, fmt.Sprintf("文の数が変わった（%d → %d）", so, sr))
	}

	// 4. 足した部分（文字単位の差分）。種類か新しい語に当たるものだけ残す
	or, rr := []rune(o), []rune(r)
	for _, op := range difflib.New(or, rr).Opcodes() {
		if op.Tag != "insert" && op.Tag != "replace" {
			continue
		}
		seg := string(rr[op.J1:op.J2])
		around := string(rr[max(0, op.J1-3):min(len(rr), op.J2+3)])
		kinds := []string{}
		for _, m := range markers {
			if search(m.re, around) {
				kinds = append(kinds, m.name)
			}
		}
		words := []string{}
		for _, w := range content.FindAllStrings(seg) {
			if !strings.Contains(o, w) {
				words = append(words, w)
			}
		}
		if len(kinds) > 0 || len(words) > 0 {
			d.Spans = append(d.Spans, Span{seg, string(or[op.I1:op.I2]), kinds, words, context(rr, op.J1, op.J2)})
		}
	}
	return d
}
