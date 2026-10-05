package diff

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sudame/yomiyasu-go/internal/bold"
	"github.com/sudame/yomiyasu-go/internal/pycompat"
	"github.com/sudame/yomiyasu-go/internal/pyre"
)

// 状態や性質を表す「〜ます」の語幹（動作ではないもの）
var stative = []string{"なり", "あり", "おり", "でき", "分かり", "わかり", "つながり", "変わり", "起き", "起こり", "生じ",
	"増え", "減り", "見え", "聞こえ", "残り", "続き", "終わり", "始まり", "決まり", "進み", "遅れ",
	"下回り", "上回り", "異なり", "違い", "限り", "足り", "合い", "当たり", "向き", "似", "伝わり",
	"高まり", "下がり", "上がり", "広がり", "強まり", "弱まり", "落ち", "壊れ", "崩れ", "外れ", "漏れ",
	"そろい", "揃い", "思え", "感じられ", "止まり", "消え", "困り", "迷い", "助かり"}

// 可能の形（「防げます」など）。一段動詞と見分けられないので、よく出るものだけ並べる
var potential = []string{"防げ", "書け", "読め", "使え", "言え", "選べ", "話せ", "待て", "探せ", "直せ", "残せ", "減らせ", "増やせ",
	"守れ", "作れ", "取れ", "気づけ", "見つけられ", "続けられ", "避けられ", "変えられ", "決められ", "伝えられ"}

var (
	reBoldOrTick      = pyre.MustCompile("\\*\\*|`")
	reTrailingParen   = pyre.MustCompile(`[（(][^（）()]*[）)]$`)
	reTrailingClose   = pyre.MustCompile(`[」』）)]+$`)
	reKeitai          = pyre.MustCompile(`(?:です|ます|ません|ました|でした|ましょう|ください|でしょう)$`)
	reJotaiCopula     = pyre.MustCompile(`(?:だ|である|ではない|でない)$`)
	reJotaiVerb       = pyre.MustCompile(`[るうくすつぬぶむぐたい]$`)
	reNounLike        = pyre.MustCompile(`[ァ-ヴー一-龥A-Za-z0-9]$`)
	reEndRequest      = pyre.MustCompile(`ください(?:ね)?$|(?:て|で)はいけません$|(?:て|で)はなりません$|ていただきます$|ていただけます$`)
	reEndRecommend    = pyre.MustCompile(`ましょう$|とよいです$|といいです$|をおすすめします$|をお勧めします$`)
	reEndGuess        = pyre.MustCompile(`(?:でしょう|だろう|かもしれません|かもしれない|と思います|と考えます|と感じます|はずです|ようです|気がします)$`)
	reEndDuty         = pyre.MustCompile(`(?:なければなりません|なくてはなりません|必要があります|べきです)$`)
	reEndEval         = pyre.MustCompile(`(?:重要|最重要|大切|大事|不可欠|肝心|肝要|欠かせません|鍵|カギ|最優先|必要)(?:です|でした|だ|である)?$`)
	reEndPast         = pyre.MustCompile(`(?:ました|でした|ませんでした)$`)
	reEndMasu         = pyre.MustCompile(`(?:ます|ません)$`)
	reStemTeI         = pyre.MustCompile(`(?:て|で)い$`)
	reStemPassive     = pyre.MustCompile(`(?:られ|[^しさ]れ)$`)
	reEndDesu         = pyre.MustCompile(`です$`)
	reUnitListMarker  = pyre.MustCompile(`^(?:[*\-・]|\d+[.)])\s+`)
	reUnitSentenceEnd = pyre.MustCompile(`(?<=[。！？!?])`)
	reUnitFull        = pyre.MustCompile(`[。！？!?]$`)
	reUnitDash        = pyre.MustCompile(`[—―]{1,2}`)
)

func search(re *pyre.Regexp, s string) bool {
	_, ok := re.Search(s)
	return ok
}

func hasSuffixAny(s string, suffixes []string) bool {
	for _, x := range suffixes {
		if strings.HasSuffix(s, x) {
			return true
		}
	}
	return false
}

// bareEnd は文末の判定に使う形を返す。太字や記号、文末のかっこ書きを外す。
func bareEnd(s string) string {
	t := pycompat.Strip(strings.TrimRight(pycompat.Strip(remove(reBoldOrTick, s)), "。．.！!？?"))
	prev := ""
	for first := true; first || prev != t; first = false {
		prev = t
		t = pycompat.Strip(remove(reTrailingParen, t))
	}
	return remove(reTrailingClose, t)
}

// register は敬体か常体かを返す。体言止めなどは空文字を返す。
func register(s string) string {
	t := bareEnd(s)
	if search(reKeitai, t) {
		return "敬体"
	}
	if search(reJotaiCopula, t) || (search(reJotaiVerb, t) && !search(reNounLike, t)) {
		return "常体"
	}
	return ""
}

func endingKind(s string) string {
	t := bareEnd(s)
	switch {
	case t == "":
		return ""
	case search(reEndRequest, t):
		return "依頼"
	case search(reEndRecommend, t):
		return "勧め"
	case search(reEndGuess, t):
		return "推量・考え"
	case search(reEndDuty, t):
		return "義務"
	case search(reEndEval, t):
		return "評価"
	case search(reEndPast, t):
		return "過去"
	case search(reEndMasu, t):
		stem := remove(reEndMasu, t)
		if search(reStemTeI, stem) {
			return "説明（〜ています）"
		}
		if search(reStemPassive, stem) || hasSuffixAny(stem, stative) || hasSuffixAny(stem, potential) {
			return "説明（〜ます）"
		}
		return "動作（〜します）"
	case search(reEndDesu, t):
		return "断定（〜です）"
	case search(reJotaiCopula, t):
		return "常体"
	case search(reJotaiVerb, t) && !search(reNounLike, t):
		return "常体"
	}
	return "体言止めなど"
}

type unit struct {
	where    string
	sentence string
	kind     string
	full     bool
}

func onlyTableRule(l string) bool {
	for _, r := range l {
		if !strings.ContainsRune("|-: ", r) {
			return false
		}
	}
	return true
}

// endingUnits は文末を見る単位を返す。箇条書きの 1 項目も 1 文とし、ダッシュの前も 1 つの区切りとして見る。
// markdown のときは、コードブロック・先頭の設定部分・表の区切り行を飛ばし、表のセルは「表」として扱う。
func endingUnits(t string, markdown bool) []unit {
	var out []unit
	lines := strings.Split(t, "\n")
	if markdown && len(lines) > 0 && pycompat.Strip(lines[0]) == "---" {
		for n := 1; n < len(lines); n++ {
			if lines[n] == "---" {
				lines = lines[n+1:]
				break
			}
		}
	}
	inCode := false
	for _, line := range lines {
		l := pycompat.Strip(line)
		if markdown && strings.HasPrefix(l, "```") {
			inCode = !inCode
			continue
		}
		if inCode || l == "" || strings.HasPrefix(l, "#") || l == "---" {
			continue
		}
		var where string
		var srcs []string
		switch {
		case strings.HasPrefix(l, "|"):
			if !markdown || onlyTableRule(l) {
				continue
			}
			where = "表"
			for _, c := range strings.Split(strings.Trim(l, "|"), "|") {
				srcs = append(srcs, pycompat.Strip(c))
			}
		case search(reUnitListMarker, l):
			where = "箇条書き"
			srcs = []string{remove(reUnitListMarker, l)}
		default:
			where = "地の文"
			srcs = []string{l}
		}
		for _, src := range srcs {
			for _, s := range reUnitSentenceEnd.Split(src) {
				s = pycompat.Strip(s)
				if s == "" {
					continue
				}
				full := search(reUnitFull, s)
				parts := reUnitDash.Split(s)
				for n, p := range parts {
					p = pycompat.Strip(p)
					if p == "" {
						continue
					}
					k := endingKind(p)
					last := n == len(parts)-1
					if k == "" || (!last && k == "体言止めなど") {
						continue
					}
					out = append(out, unit{where: where, sentence: p, kind: k, full: full || !last})
				}
			}
		}
	}
	return out
}

type flag struct {
	note string
	rows []unit
}

func stanceFlags(t, stance string, markdown bool) ([]unit, []flag) {
	rows := endingUnits(t, markdown)
	var act, ask, rec, ev []unit
	var askIdx []int
	for i, r := range rows {
		if r.where == "表" {
			continue
		}
		switch r.kind {
		case "動作（〜します）":
			act = append(act, r)
		case "勧め", "依頼":
			ask = append(ask, r)
			askIdx = append(askIdx, i)
			if r.kind == "勧め" {
				rec = append(rec, r)
			}
		case "評価":
			ev = append(ev, r)
		}
	}
	var flags []flag
	switch stance {
	case "勧め":
		if len(act) > 0 {
			flags = append(flags, flag{"勧めの文書に、動作を表す「〜します」がある。書き手の予定や手順・道具の説明か、読み手への依頼かを原文と文脈から確認する", act})
		}
	case "決まり":
		if len(rec) > 0 || len(ev) > 0 {
			flags = append(flags, flag{"決まり・手順の文書に、勧めや評価の文末がある。決まりそのものなら決まりの形（〜します）にする。決まりの理由や前提を述べる文なら残す。「〜してください」はそのままでよい", concat(rec, ev)})
		}
	case "説明":
		var body []unit
		for n, r := range ask {
			if askIdx[n] != len(rows)-1 {
				body = append(body, r)
			}
		}
		if len(body) > 0 {
			flags = append(flags, flag{"事実・結果・考えの文書に、読み手への勧めや頼みがある（最後の1文を除く）。立場が合っているか見る", body})
		}
	default:
		if len(act) > 0 && len(ask) > 0 {
			flags = append(flags, flag{"動作の「〜します」と、勧め・依頼が同じ文章にある。「〜します」が書き手の側の予定・決まった手順なのか、読み手にしてほしい行動なのかを見る", concat(act, ask)})
		} else if len(act) > 0 && len(ev) > 0 {
			flags = append(flags, flag{"動作の「〜します」と、評価（〜が重要です など）が同じ文章にある。各文の働きと原文の強さが保たれているか確認する", concat(act, ev)})
		}
	}
	// 敬体と常体。文として書かれた単位（。で終わるもの、ダッシュの前）だけを数える。表と、。のない箇条書きは数えない
	var jotai, keitai []unit
	for _, r := range rows {
		if r.where == "表" || !r.full {
			continue
		}
		switch register(r.sentence) {
		case "常体":
			jotai = append(jotai, r)
		case "敬体":
			keitai = append(keitai, r)
		}
	}
	if len(jotai) > 0 && len(keitai) > len(jotai) {
		flags = append(flags, flag{"敬体の文の中に、常体の文がある。そろえるときは「する → します」と形だけで変えず、立場に合う形にする", jotai})
	} else if len(keitai) > 0 && len(jotai) > len(keitai) {
		flags = append(flags, flag{"常体の文の中に、敬体の文がある", keitai})
	}
	return rows, flags
}

func concat(a, b []unit) []unit {
	out := append([]unit{}, a...)
	return append(out, b...)
}

// truncate は、n 文字を超える文を n 文字で切って「…」を付ける。
func truncate(s string, n int) string {
	rs := []rune(s)
	if len(rs) <= n {
		return s
	}
	return string(rs[:n]) + "…"
}

func boldHead(where string) string {
	return fmt.Sprintf("■ 太字にならない書き方（%sGitHub などで ** がそのまま表示される。表示可否と案の範囲を確認して直す）", where)
}

func boldLines(ps []bold.Problem) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		suggest := p.Suggest
		if suggest == "" {
			suggest = "（手で直す）"
		}
		out[i] = fmt.Sprintf("- %d行目: %s → %s（%s）", p.Line, p.Found, suggest, p.How)
	}
	return out
}

// Endings は --endings の出力全体を返す。stance が空文字なら立場を指定しない。
func Endings(text, stance string) string {
	var b strings.Builder
	rows, flags := stanceFlags(text, stance, true)

	// Counter.most_common と同じく、数の多い順に並べ、同じ数なら最初に出た順を保つ
	var kinds []string
	counts := map[string]int{}
	for _, r := range rows {
		if r.where == "表" {
			continue
		}
		if counts[r.kind] == 0 {
			kinds = append(kinds, r.kind)
		}
		counts[r.kind]++
	}
	sort.SliceStable(kinds, func(a, c int) bool { return counts[kinds[a]] > counts[kinds[c]] })
	parts := make([]string, len(kinds))
	for i, k := range kinds {
		parts[i] = fmt.Sprintf("%s %d", k, counts[k])
	}
	b.WriteString("■ 文末の種類（表を除く）: " + strings.Join(parts, "、") + "\n")

	for _, f := range flags {
		b.WriteString("■ " + f.note + "\n")
		for _, r := range f.rows {
			b.WriteString("  ・" + truncate(r.sentence, 60) + "\n")
		}
	}
	if bp := bold.Problems(text, true); len(bp) > 0 {
		b.WriteString(boldHead("") + "\n")
		b.WriteString(strings.Join(boldLines(bp), "\n") + "\n")
	}
	return b.String()
}
