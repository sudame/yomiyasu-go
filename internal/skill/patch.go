// Package skill は、本家の SKILL.md と references に変換規則を当てて、yomiyasu-go のスキルを作る。
package skill

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/sudame/yomiyasu-go/internal/rules"
)

// Patch は変換規則の 1 つ。Rules の規則がすべて無効で、Unless の規則がすべて有効なときに当てる。
// Rules が空なら常に当てる。
type Patch struct {
	ID      string   `toml:"id"`
	Rules   []string `toml:"rules"`
	Unless  []string `toml:"unless"`
	File    string   `toml:"file"`
	Anchor  string   `toml:"anchor"`
	Target  string   `toml:"target"`
	Section string   `toml:"section"`
	Replace string   `toml:"replace"`
	Count   int      `toml:"count"`
}

// LoadPatches は patches.toml を読む。知らないキーがあればエラーにする。
func LoadPatches(b []byte) ([]Patch, error) {
	var f struct {
		Patch []Patch `toml:"patch"`
	}
	md, err := toml.Decode(string(b), &f)
	if err != nil {
		return nil, err
	}
	if und := md.Undecoded(); len(und) > 0 {
		return nil, fmt.Errorf("patches.toml に知らないキーがある: %v", und)
	}
	return f.Patch, nil
}

type span struct {
	start, end int
	patch      *Patch
}

func (p *Patch) applies(disabled map[string]bool) bool {
	for _, r := range p.Rules {
		if !disabled[r] {
			return false
		}
	}
	for _, r := range p.Unless {
		if disabled[r] {
			return false
		}
	}
	return true
}

// coApplicable は、2 つの変換規則が同時に当たりうるかを返す。
func coApplicable(p, q *Patch) bool {
	need := append(slices.Clone(p.Rules), q.Rules...)
	for _, u := range append(slices.Clone(p.Unless), q.Unless...) {
		if slices.Contains(need, u) {
			return false
		}
	}
	return true
}

func locate(p *Patch, text string) ([]span, error) {
	if p.Section != "" {
		return locateSection(p, text)
	}
	if n := strings.Count(text, p.Anchor); p.Anchor == "" || n != p.Count {
		return nil, fmt.Errorf("変換規則 %s: %s で anchor が %d 回見つかった（期待 %d 回）: %q", p.ID, p.File, n, p.Count, p.Anchor)
	}
	target := p.Target
	if target == "" {
		target = p.Anchor
	}
	if strings.Count(p.Anchor, target) != 1 {
		return nil, fmt.Errorf("変換規則 %s: target が anchor の中にちょうど 1 回ない: %q", p.ID, target)
	}
	off := strings.Index(p.Anchor, target)
	var out []span
	for idx := 0; ; {
		j := strings.Index(text[idx:], p.Anchor)
		if j < 0 {
			break
		}
		s := idx + j + off
		out = append(out, span{s, s + len(target), p})
		idx += j + len(p.Anchor)
	}
	return out, nil
}

func headingLevel(line string) int {
	n := len(line) - len(strings.TrimLeft(line, "#"))
	if n == 0 || n > 6 || (len(line) > n && line[n] != ' ') {
		return 0
	}
	return n
}

// locateSection は、Section と一致する見出しの行から、同じ深さかより浅い次の見出しの手前までの範囲を返す。
func locateSection(p *Patch, text string) ([]span, error) {
	lines := strings.SplitAfter(text, "\n")
	start, level, found := -1, 0, 0
	pos, inCode := 0, false
	end := len(text)
	for _, l := range lines {
		trimmed := strings.TrimRight(l, "\n")
		if strings.HasPrefix(trimmed, "```") {
			inCode = !inCode
		}
		if !inCode {
			lv := headingLevel(trimmed)
			if trimmed == p.Section {
				found++
				if start < 0 {
					start, level = pos, lv
				}
			} else if start >= 0 && end == len(text) && lv > 0 && lv <= level {
				end = pos
			}
		}
		pos += len(l)
	}
	if found != 1 || level == 0 {
		return nil, fmt.Errorf("変換規則 %s: %s で見出し %q が %d 回見つかった（期待 1 回）", p.ID, p.File, p.Section, found)
	}
	return []span{{start, end, p}}, nil
}

// Validate は、設定にかかわらずすべての変換規則が元の文書に当たることと、
// 同時に当たりうる変換規則どうしの範囲が重ならないことを確かめる。
func Validate(patches []Patch, files map[string]string) error {
	byFile := map[string][]span{}
	for i := range patches {
		p := &patches[i]
		for _, r := range append(slices.Clone(p.Rules), p.Unless...) {
			if !rules.Known(r) {
				return fmt.Errorf("変換規則 %s: 知らない規則 %q", p.ID, r)
			}
		}
		text, ok := files[p.File]
		if !ok {
			return fmt.Errorf("変換規則 %s: ファイル %s がない", p.ID, p.File)
		}
		spans, err := locate(p, text)
		if err != nil {
			return err
		}
		byFile[p.File] = append(byFile[p.File], spans...)
	}
	for file, spans := range byFile {
		for i := range spans {
			for j := i + 1; j < len(spans); j++ {
				a, b := spans[i], spans[j]
				if a.patch != b.patch && a.start < b.end && b.start < a.end && coApplicable(a.patch, b.patch) {
					return fmt.Errorf("%s で変換規則 %s と %s の範囲が重なっている", file, a.patch.ID, b.patch.ID)
				}
			}
		}
	}
	return nil
}

// Apply は、disabled に合う変換規則を当てた文書を返す。置き換える範囲はすべて元の文書の上で決める。
func Apply(patches []Patch, files map[string]string, disabled map[string]bool) (map[string]string, error) {
	if err := Validate(patches, files); err != nil {
		return nil, err
	}
	byFile := map[string][]span{}
	for i := range patches {
		p := &patches[i]
		if !p.applies(disabled) {
			continue
		}
		spans, _ := locate(p, files[p.File])
		byFile[p.File] = append(byFile[p.File], spans...)
	}
	out := make(map[string]string, len(files))
	for name, text := range files {
		spans := byFile[name]
		sort.Slice(spans, func(i, j int) bool { return spans[i].start < spans[j].start })
		var b strings.Builder
		last := 0
		for _, s := range spans {
			b.WriteString(text[last:s.start])
			b.WriteString(s.patch.Replace)
			last = s.end
		}
		b.WriteString(text[last:])
		out[name] = b.String()
	}
	return out, nil
}
