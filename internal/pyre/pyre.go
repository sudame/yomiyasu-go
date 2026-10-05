// Package pyre は、Python の re と同じ使い方で regexp2 を呼ぶ。位置はすべてルーン単位で返す。
package pyre

import (
	"fmt"
	"regexp"
	"strconv"

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

// .NET の構文には \UXXXXXXXX がないので、その文字そのものに置き換える。
var upperU = regexp.MustCompile(`\\U([0-9A-Fa-f]{8})`)

// MustCompile は Python の re の構文で書いたパターンをコンパイルする。
func MustCompile(pattern string) *Regexp {
	p := upperU.ReplaceAllStringFunc(pattern, func(s string) string {
		n, err := strconv.ParseUint(s[2:], 16, 32)
		if err != nil {
			panic(err)
		}
		return string(rune(n))
	})
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
