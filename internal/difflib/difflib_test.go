package difflib

import (
	"reflect"
	"testing"
)

func TestRatioAndOpcodes(t *testing.T) {
	cases := []struct {
		a, b  string
		ratio float64
		ops   []Opcode
	}{
		{"abcd", "bcde", 0.75, []Opcode{{"delete", 0, 1, 0, 0}, {"equal", 1, 4, 0, 3}, {"insert", 4, 4, 3, 4}}},
		{"今日は晴れです。", "明日は雨です。", 0.6666666666666666, []Opcode{{"replace", 0, 1, 0, 1}, {"equal", 1, 3, 1, 3}, {"replace", 3, 5, 3, 4}, {"equal", 5, 8, 4, 7}}},
		{"", "", 1.0, nil},
		{"abxcd", "abcd", 0.8888888888888888, []Opcode{{"equal", 0, 2, 0, 2}, {"delete", 2, 3, 2, 2}, {"equal", 3, 5, 2, 4}}},
		{"aaaa", "aa", 0.6666666666666666, []Opcode{{"equal", 0, 2, 0, 2}, {"delete", 2, 4, 2, 2}}},
	}
	for _, c := range cases {
		m := New([]rune(c.a), []rune(c.b))
		if got := m.Ratio(); got != c.ratio {
			t.Errorf("Ratio(%q, %q) = %v, want %v", c.a, c.b, got, c.ratio)
		}
		if got := m.Opcodes(); !reflect.DeepEqual(got, c.ops) {
			t.Errorf("Opcodes(%q, %q) = %v, want %v", c.a, c.b, got, c.ops)
		}
	}
}
