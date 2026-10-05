package pycompat

import "testing"

func TestFloatRepr(t *testing.T) {
	cases := map[float64]string{
		0.0: "0.0", 1.0: "1.0", 1.03: "1.03", 0.472: "0.472", 47.2: "47.2",
		1e-05: "1e-05", 1e16: "1e+16", 1e15: "1000000000000000.0",
		123456789012345678.0: "1.2345678901234568e+17",
	}
	// 定数のまま足すと Go では厳密に 0.3 になるため、変数を通して float64 で足す。
	a, b := 0.1, 0.2
	cases[a+b] = "0.30000000000000004"
	for in, want := range cases {
		if got := FloatRepr(in); got != want {
			t.Errorf("FloatRepr(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestRound(t *testing.T) {
	cases := []struct {
		in     Num
		digits int
		want   string
	}{
		{Float(0.472), 2, "0.47"}, {Float(0.472), 1, "0.5"}, {Float(2.675), 2, "2.67"},
		{Float(0.125), 2, "0.12"}, {Float(1e-05), 2, "0.0"}, {Int(0), 1, "0"},
	}
	for _, c := range cases {
		if got := Round(c.in, c.digits).String(); got != c.want {
			t.Errorf("Round(%v, %d) = %q, want %q", c.in, c.digits, got, c.want)
		}
	}
}

func TestStrip(t *testing.T) {
	if got := Strip(" \x1c\u0085\u00a0\u3000x\u200b "); got != "x\u200b" {
		t.Errorf("Strip = %q", got)
	}
}

func TestNormalizeNewlines(t *testing.T) {
	if got := NormalizeNewlines("a\r\nb\rc\n"); got != "a\nb\nc\n" {
		t.Errorf("NormalizeNewlines = %q", got)
	}
}

func TestDumpsIndent2(t *testing.T) {
	v := Obj{
		{"a", Int(0)}, {"b", Float(0)}, {"c", []any{}}, {"d", Obj{}},
		{"e", "\"\\\n\r\t\b\f\x01\x7f <&>あ"}, {"f", nil}, {"g", true},
		{"h", []any{1, "x"}},
	}
	want := "{\n  \"a\": 0,\n  \"b\": 0.0,\n  \"c\": [],\n  \"d\": {},\n" +
		"  \"e\": \"\\\"\\\\\\n\\r\\t\\b\\f\\u0001\x7f <&>あ\",\n" +
		"  \"f\": null,\n  \"g\": true,\n  \"h\": [\n    1,\n    \"x\"\n  ]\n}"
	if got := Dumps(v, 2); got != want {
		t.Errorf("Dumps =\n%s\nwant\n%s", got, want)
	}
}

func TestDumpsIndent1(t *testing.T) {
	v := Obj{{"a", []any{1, Obj{{"b", []any{}}}}}}
	want := "{\n \"a\": [\n  1,\n  {\n   \"b\": []\n  }\n ]\n}"
	if got := Dumps(v, 1); got != want {
		t.Errorf("Dumps =\n%s\nwant\n%s", got, want)
	}
}
