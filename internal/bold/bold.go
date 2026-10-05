// Package bold は、本家の太字の表示判定（bold_problems とその補助関数）を移植する。lint と diff の両方から使う。
package bold

import (
	"sort"
	"strings"
	"unicode"

	"github.com/sudame/yomiyasu-go/internal/pycompat"
	"github.com/sudame/yomiyasu-go/internal/pyre"
)

// Problem は太字にならない ** の 1 か所と、直し方の案。
type Problem struct {
	Line    int
	Found   string
	Suggest string
	How     string
}

// none は Python の空文字（範囲外の文字）を表す。
const none rune = -1

var asciiPunct = func() map[rune]bool {
	m := map[rune]bool{}
	for _, r := range "!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~" {
		m[r] = true
	}
	return m
}()

var brackets = map[rune]rune{
	'「': '」', '『': '』', '（': '）', '(': ')', '【': '】', '〔': '〕', '［': '］', '[': ']',
	'〈': '〉', '《': '》', '“': '”', '‘': '’', '＜': '＞',
}

var (
	reBackticks    = pyre.MustCompile("`+")
	reDelim        = pyre.MustCompile(`(?<!\*)\*\*(?!\*)`)
	reTrailingBS   = pyre.MustCompile(`\\*$`)
	reListMarker   = pyre.MustCompile(`^\s{0,3}(?:[*+-]|\d+[.)])\s+`)
	reIndent       = pyre.MustCompile(`^\s{0,3}`)
	reFence        = pyre.MustCompile("^\\s{0,3}(`{3,}|~{3,})(.*)$")
	reThematic     = pyre.MustCompile(`^\s{0,3}(?:(\*)\s*(?:\1\s*){2,}|(-)\s*(?:\2\s*){2,}|(_)\s*(?:\3\s*){2,})\s*$`)
	reTableSep     = pyre.MustCompile(`^\s{0,3}\|?\s*:?-{1,}:?\s*(\|\s*:?-{1,}:?\s*)+\|?\s*$`)
	reSetext       = pyre.MustCompile(`^\s{0,3}(=+|-+)\s*$`)
	reATXHeading   = pyre.MustCompile(`^\s{0,3}#{1,6}(\s+|$)`)
	punctuationEnd = "。、．，！？!?"
)

func isWS(r rune) bool { return r == none || pycompat.IsSpace(r) }

func punctGFM(r rune) bool { return r != none && (asciiPunct[r] || unicode.In(r, unicode.P)) }

func punctNew(r rune) bool { return r != none && unicode.In(r, unicode.P, unicode.S) }

func canOpen(prev, nxt rune) bool {
	for _, p := range []func(rune) bool{punctGFM, punctNew} {
		ok := !isWS(nxt) && (!p(nxt) || isWS(prev) || p(prev))
		if !ok {
			return false
		}
	}
	return true
}

func canClose(prev, nxt rune) bool {
	for _, p := range []func(rune) bool{punctGFM, punctNew} {
		ok := !isWS(prev) && (!p(prev) || isWS(nxt) || p(nxt))
		if !ok {
			return false
		}
	}
	return true
}

// at は Python の text[p] if 0 <= p < len(text) else "" と同じ。
func at(t []rune, p int) rune {
	if p >= 0 && p < len(t) {
		return t[p]
	}
	return none
}

func codeSpans(t []rune) [][2]int {
	var runs [][2]int
	for _, m := range reBackticks.FindAll(string(t)) {
		runs = append(runs, [2]int{m.Start, m.End})
	}
	var spans [][2]int
	for k := 0; k < len(runs); k++ {
		s, e := runs[k][0], runs[k][1]
		for m := k + 1; m < len(runs); m++ {
			if runs[m][1]-runs[m][0] == e-s {
				spans = append(spans, [2]int{s, runs[m][1]})
				k = m
				break
			}
		}
	}
	return spans
}

func pairsInBlock(t []rune) [][2]int {
	code := codeSpans(t)
	var pos []int
	for _, m := range reDelim.FindAll(string(t)) {
		p := m.Start
		inCode := false
		for _, c := range code {
			if c[0] <= p && p < c[1] {
				inCode = true
				break
			}
		}
		if inCode {
			continue
		}
		bs := 0
		if bm, ok := reTrailingBS.Search(string(t[:p])); ok {
			bs = len([]rune(bm.Text))
		}
		if bs%2 == 1 {
			continue
		}
		pos = append(pos, p)
	}

	var pairs [][2]int
	used := map[int]bool{}

	// 第1段: 前後の空白による開閉の判定と、スタックでの照合
	var stack []int
	for _, p := range pos {
		prev := none
		if p > 0 {
			prev = t[p-1]
		}
		nxt := at(t, p+2)
		openOK := !isWS(nxt)
		closeOK := !isWS(prev)
		if closeOK && len(stack) > 0 {
			opener := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if opener+2 < p {
				pairs = append(pairs, [2]int{opener, p})
				used[opener] = true
				used[p] = true
			}
		} else if openOK {
			stack = append(stack, p)
		}
	}

	// 第2段: 内側の空白のために開閉の条件から外れた太字の候補を組にする
	var unpaired []int
	for _, p := range pos {
		if !used[p] {
			unpaired = append(unpaired, p)
		}
	}
	idx := 0
	for idx < len(unpaired)-1 {
		p1, p2 := unpaired[idx], unpaired[idx+1]
		crossed := false
		for _, pr := range pairs {
			if (p1 < pr[0] && pr[0] < p2) || (p1 < pr[1] && pr[1] < p2) {
				crossed = true
				break
			}
		}
		if crossed {
			idx++
			continue
		}
		inner := t[p1+2 : p2]
		if pycompat.Strip(string(inner)) != "" {
			if isWS(inner[0]) || isWS(inner[len(inner)-1]) {
				pairs = append(pairs, [2]int{p1, p2})
				used[p1] = true
				used[p2] = true
				idx += 2
				continue
			}
		}
		idx++
	}

	sort.SliceStable(pairs, func(a, b int) bool { return pairs[a][0] < pairs[b][0] })
	return pairs
}

func pairOK(t []rune, i, j int) bool {
	return canOpen(at(t, i-1), at(t, i+2)) && canClose(at(t, j-1), at(t, j+2))
}

// closeOf は s の先頭のかっこに対応する閉じかっこの位置を返す。なければ -1。
func closeOf(s []rune) int {
	o, c, depth := s[0], brackets[s[0]], 0
	for k, x := range s {
		switch x {
		case o:
			depth++
		case c:
			depth--
			if depth == 0 {
				return k
			}
		}
	}
	return -1
}

type try struct {
	middle string
	how    string
}

// fix は k 番目の太字（i と j の **）の直し方の案を返す。案がなければ ok は false。
func fix(t []rune, i, j, k int) (middle, how string, ok bool) {
	inner := t[i+2 : j]
	var tries []try
	if len(inner) >= 3 {
		if _, isBracket := brackets[inner[0]]; isBracket && closeOf(inner) == len(inner)-1 {
			tries = append(tries, try{
				string(inner[0]) + "**" + string(inner[1:len(inner)-1]) + "**" + string(inner[len(inner)-1]),
				"かっこの内側だけを太字にする",
			})
		}
	}
	if len(inner) >= 2 && strings.ContainsRune(punctuationEnd, inner[len(inner)-1]) {
		tries = append(tries, try{
			"**" + string(inner[:len(inner)-1]) + "**" + string(inner[len(inner)-1]),
			"句読点を太字の外に出す",
		})
	}
	body := []rune(string(inner))
	if isWS(at(t, i+2)) || isWS(at(t, j-1)) {
		body = []rune(pycompat.Strip(string(inner)))
	}
	left, right := "", ""
	if !canOpen(at(t, i-1), at(body, 0)) {
		left = " "
	}
	if !canClose(at(body, len(body)-1), at(t, j+2)) {
		right = " "
	}
	h := "文字に接する側に半角スペースを入れる"
	if string(body) != string(inner) && left == "" && right == "" {
		h = "太字の内側の空白を取る"
	}
	tries = append(tries, try{left + "**" + string(body) + "**" + right, h})
	for _, tr := range tries {
		cand := []rune(string(t[:i]) + tr.middle + string(t[j+2:]))
		pairs := pairsInBlock(cand)
		if k < len(pairs) && pairOK(cand, pairs[k][0], pairs[k][1]) {
			return tr.middle, tr.how, true
		}
	}
	return "", "手で直す", false
}

// lineContainers は、行のリストの印と引用の深さ、その内側の中身を簡易に解析する。
func lineContainers(line string) (isList bool, depth int, content string) {
	rem := []rune(line)
	if m, ok := reListMarker.Match(line); ok {
		isList = true
		rem = rem[m.End:]
	}
	p := 0
	for {
		if m, ok := reIndent.Match(string(rem[p:])); ok {
			p += m.End
		}
		if p < len(rem) && rem[p] == '>' {
			depth++
			p++
			if p < len(rem) && rem[p] == ' ' {
				p++
			}
		} else {
			break
		}
	}
	return isList, depth, string(rem[p:])
}

type numbered struct {
	no   int
	line string
}

// Problems は太字にならない ** の場所と直し方の案を返す。
// コードブロック・インラインコード・HTML の行・先頭の設定部分は見ない。
func Problems(text string, skipFrontmatter bool) []Problem {
	var out []Problem
	lines := strings.Split(text, "\n")
	start := 0
	if skipFrontmatter && len(lines) > 0 && strings.TrimRight(lines[0], "\r") == "---" {
		for n := 1; n < len(lines); n++ {
			if strings.TrimRight(lines[n], "\r") == "---" {
				start = n + 1
				break
			}
		}
	}

	var blocks [][]numbered
	var curr []numbered
	var fenceChar rune
	fenceLen := 0
	inFence := false
	currDepth := 0
	inTable := false

	flush := func() {
		if len(curr) > 0 {
			blocks = append(blocks, curr)
			curr = nil
		}
		currDepth = 0
		inTable = false
	}

	for no := start; no < len(lines); no++ {
		line := strings.TrimRight(lines[no], "\r")
		lineNo := no + 1

		isList, depth, content := lineContainers(line)

		// フェンスコードブロックの開始と終了
		m, fenced := reFence.Match(content)
		if inFence {
			if fenced {
				g1 := []rune(m.Groups[1])
				if g1[0] == fenceChar && len(g1) >= fenceLen && pycompat.Strip(m.Groups[2]) == "" {
					inFence = false
				}
			}
			continue
		}
		if fenced {
			g1 := []rune(m.Groups[1])
			// 情報文字列にバッククォートを含む ``` はフェンスではない
			if g1[0] != '`' || !strings.Contains(m.Groups[2], "`") {
				flush()
				inFence, fenceChar, fenceLen = true, g1[0], len(g1)
				continue
			}
		}

		if pycompat.Strip(content) == "" {
			flush()
			continue
		}

		if strings.HasPrefix(strings.TrimLeftFunc(content, pycompat.IsSpace), "<") {
			flush()
			continue
		}

		if _, ok := reThematic.Match(content); ok {
			flush()
			continue
		}

		// GFM の表の区切り行（外周のパイプがないものを含む）
		if _, ok := reTableSep.Match(content); ok {
			if len(curr) > 0 {
				hdr := curr[len(curr)-1]
				curr = curr[:len(curr)-1]
				flush()
				blocks = append(blocks, []numbered{hdr})
			} else {
				flush()
			}
			inTable = true
			continue
		}

		// Setext 見出しの下線
		if len(curr) > 0 {
			if _, ok := reSetext.Match(content); ok {
				flush()
				continue
			}
		}

		// ATX 見出し
		if _, ok := reATXHeading.Match(content); ok {
			flush()
			blocks = append(blocks, []numbered{{lineNo, line}})
			continue
		}

		// 表の行（外周のパイプがあるもの、または表の続きの行）
		if strings.HasPrefix(content, "|") || (inTable && strings.Contains(content, "|")) {
			flush()
			blocks = append(blocks, []numbered{{lineNo, line}})
			inTable = true
			continue
		}
		inTable = false

		// リスト項目の開始
		_, listInContent := reListMarker.Match(content)
		if isList || listInContent {
			flush()
			currDepth = depth
			curr = append(curr, numbered{lineNo, line})
			continue
		}

		// 引用の深さの変化（直前の深さが 1 以上で、今の深さが 0 なら続きの行とみなす）
		lazy := currDepth > 0 && depth == 0
		if len(curr) > 0 && depth != currDepth && !lazy {
			flush()
			currDepth = depth
		}
		if len(curr) == 0 {
			currDepth = depth
		}
		curr = append(curr, numbered{lineNo, line})
	}
	flush()

	for _, block := range blocks {
		ls := make([]string, len(block))
		for n, b := range block {
			ls[n] = b.line
		}
		t := []rune(strings.Join(ls, "\n"))
		offsets := []int{0}
		for _, b := range block[:len(block)-1] {
			offsets = append(offsets, offsets[len(offsets)-1]+len([]rune(b.line))+1)
		}
		lineOf := func(idx int) int {
			return block[sort.Search(len(offsets), func(n int) bool { return offsets[n] > idx })-1].no
		}

		for k, pr := range pairsInBlock(t) {
			i, j := pr[0], pr[1]
			if pairOK(t, i, j) {
				continue
			}
			middle, how, ok := fix(t, i, j, k)
			pre := string(t[max(0, i-4):i])
			post := string(t[j+2 : min(len(t), j+6)])
			found := pre + short(string(t[i:j+2])) + post
			suggest := ""
			if ok {
				suggest = pre + short(middle) + post
			}
			out = append(out, Problem{Line: lineOf(i), Found: found, Suggest: suggest, How: how})
		}
	}
	return out
}

// short は、長い太字を直すところ（両端）だけに縮める。
func short(s string) string {
	rs := []rune(s)
	if len(rs) <= 30 {
		return s
	}
	return string(rs[:12]) + "…" + string(rs[len(rs)-12:])
}
