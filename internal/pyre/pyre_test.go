package pyre

import (
	"reflect"
	"strings"
	"testing"
)

func nonBlank(xs []string) []string {
	var out []string
	for _, x := range xs {
		if strings.TrimSpace(x) != "" {
			out = append(out, x)
		}
	}
	return out
}

func TestSplitLookbehind(t *testing.T) {
	got := MustCompile(`(?<=[。！？])`).Split("あ。い！う")
	want := []string{"あ。", "い！", "う"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Split = %q, want %q", got, want)
	}
}

func TestSplitLookbehindOrNewlines(t *testing.T) {
	got := nonBlank(MustCompile(`(?<=[。！？!?])|\n+`).Split("a。\n\nb！c\nd"))
	want := []string{"a。", "b！", "c", "d"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Split = %q, want %q", got, want)
	}
}

func TestPositionsAreRunes(t *testing.T) {
	m, ok := MustCompile(`(?<!\*)\*\*(?!\*)`).Search("𠮷あ**太字**")
	if !ok || m.Start != 2 || m.End != 4 {
		t.Errorf("Search = %+v, %v", m, ok)
	}
}

func TestMatchIsAnchored(t *testing.T) {
	re := MustCompile(`\s*(?:[*\-・]|\d+[.)])\s+`)
	if _, ok := re.Match("x - y"); ok {
		t.Error("Match が先頭以外でマッチした")
	}
	if _, ok := re.Match("  - y"); !ok {
		t.Error("Match が先頭でマッチしない")
	}
}

func TestUpperUEscape(t *testing.T) {
	re := MustCompile(`[\U0001F600-\U0001F64F]|[☀-➿]`)
	if got := re.FindAllStrings("a😀b☀"); !reflect.DeepEqual(got, []string{"😀", "☀"}) {
		t.Errorf("FindAllStrings = %q", got)
	}
}

func TestBackreference(t *testing.T) {
	re := MustCompile(`^\s{0,3}(?:(\*)\s*(?:\1\s*){2,}|(-)\s*(?:\2\s*){2,}|(_)\s*(?:\3\s*){2,})\s*$`)
	if _, ok := re.Match("* * *"); !ok {
		t.Error("区切り線にマッチしない")
	}
}

func TestSubGroup1(t *testing.T) {
	if got := MustCompile(`\*\*(.+?)\*\*`).SubGroup1("a**b**c**d**"); got != "abcd" {
		t.Errorf("SubGroup1 = %q", got)
	}
}

func TestDollarBeforeTrailingNewline(t *testing.T) {
	if _, ok := MustCompile(`[：:]$`).Search("ラベル：\n"); !ok {
		t.Error("$ が末尾の改行の前でマッチしない")
	}
}
