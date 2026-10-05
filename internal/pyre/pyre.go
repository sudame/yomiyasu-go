// Package pyre は、Python の re と同じ使い方で regexp2 を呼ぶ。位置はすべてルーン単位で返す。
package pyre

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dlclark/regexp2"
)

// Match は 1 回のマッチ。Groups[0] はマッチ全体で、参加しなかったグループは空文字になる。
type Match struct {
	Start, End int
	Text       string
	Groups     []string
}

// Regexp は、探索用と先頭に固定した照合用の 2 つの正規表現を持つ。
type Regexp struct {
	re, anchored *regexp2.Regexp
}

// translate は、Python と .NET で意味の違うエスケープを .NET の書き方に直す。
//   - \UXXXXXXXX は .NET にないので、その文字そのものに置き換える。
//   - Python の \s は str.isspace() と同じく \x1c〜\x1f も含むので、.NET の \s に足す。
func translate(pattern string) string {
	rs := []rune(pattern)
	var b strings.Builder
	inClass := false
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		switch {
		case r == '\\' && i+1 < len(rs):
			n := rs[i+1]
			i++
			switch {
			case n == 'U' && i+8 < len(rs):
				v, err := strconv.ParseUint(string(rs[i+1:i+9]), 16, 32)
				if err != nil {
					panic(fmt.Sprintf("pyre: %q の \\U を読めない: %v", pattern, err))
				}
				b.WriteRune(rune(v))
				i += 8
			case n == 's' && inClass:
				b.WriteString(`\s\x1c-\x1f`)
			case n == 's':
				b.WriteString(`[\s\x1c-\x1f]`)
			case n == 'S' && inClass:
				panic(fmt.Sprintf("pyre: %q の文字クラス内の \\S は扱えない", pattern))
			case n == 'S':
				b.WriteString(`[^\s\x1c-\x1f]`)
			default:
				b.WriteRune(r)
				b.WriteRune(n)
			}
		case r == '[' && !inClass:
			inClass = true
			b.WriteRune(r)
			if i+1 < len(rs) && rs[i+1] == '^' {
				i++
				b.WriteRune('^')
			}
			if i+1 < len(rs) && rs[i+1] == ']' {
				i++
				b.WriteRune(']')
			}
		case r == ']' && inClass:
			inClass = false
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// MustCompile は Python の re の構文で書いたパターンをコンパイルする。
func MustCompile(pattern string) *Regexp {
	p := translate(pattern)
	return &Regexp{
		re:       regexp2.MustCompile(p, regexp2.None),
		anchored: regexp2.MustCompile(`\A(?:`+p+`)`, regexp2.None),
	}
}

func toMatch(m *regexp2.Match) Match {
	gs := m.Groups()
	groups := make([]string, len(gs))
	for i, g := range gs {
		groups[i] = g.String()
	}
	return Match{Start: m.Index, End: m.Index + m.Length, Text: m.String(), Groups: groups}
}

func first(re *regexp2.Regexp, s string) (Match, bool) {
	m, err := re.FindStringMatch(s)
	if err != nil {
		panic(fmt.Sprintf("pyre: %v", err))
	}
	if m == nil {
		return Match{}, false
	}
	return toMatch(m), true
}

// Search は re.search と同じ。
func (r *Regexp) Search(s string) (Match, bool) { return first(r.re, s) }

// Match は re.match と同じく、文字列の先頭でだけ照合する。
func (r *Regexp) Match(s string) (Match, bool) { return first(r.anchored, s) }

// FindAll は re.finditer と同じ。
func (r *Regexp) FindAll(s string) []Match {
	var out []Match
	m, err := r.re.FindStringMatch(s)
	for ; m != nil && err == nil; m, err = r.re.FindNextMatch(m) {
		out = append(out, toMatch(m))
	}
	if err != nil {
		panic(fmt.Sprintf("pyre: %v", err))
	}
	return out
}

// FindAllStrings はグループのないパターンでの re.findall と同じ。
func (r *Regexp) FindAllStrings(s string) []string {
	ms := r.FindAll(s)
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = m.Text
	}
	return out
}

// Count は len(re.findall(...)) と同じ。
func (r *Regexp) Count(s string) int { return len(r.FindAll(s)) }

// Split はグループのないパターンでの re.split と同じ。空白だけの断片の出方は違うことがある。
func (r *Regexp) Split(s string) []string {
	rs := []rune(s)
	var out []string
	last := 0
	for _, m := range r.FindAll(s) {
		out = append(out, string(rs[last:m.Start]))
		last = m.End
	}
	return append(out, string(rs[last:]))
}

// Sub は置換に関数を渡した re.sub と同じ。
func (r *Regexp) Sub(s string, repl func(Match) string) string {
	rs := []rune(s)
	var out []rune
	last := 0
	for _, m := range r.FindAll(s) {
		out = append(out, rs[last:m.Start]...)
		out = append(out, []rune(repl(m))...)
		last = m.End
	}
	return string(append(out, rs[last:]...))
}

// SubGroup1 は re.sub(pattern, r"\1", s) と同じ。
func (r *Regexp) SubGroup1(s string) string {
	return r.Sub(s, func(m Match) string { return m.Groups[1] })
}
