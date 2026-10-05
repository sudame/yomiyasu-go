package bold

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

type regression struct {
	ID                   string `json:"id"`
	Text                 string `json:"text"`
	ExpectedBoldProblems int    `json:"expected_bold_problems"`
	ExpectedLineNumbers  []int  `json:"expected_line_numbers"`
}

func TestUpstreamRegressions(t *testing.T) {
	b, err := os.ReadFile("../../testdata/upstream/bold_regressions.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []regression
	if err := json.Unmarshal(b, &cases); err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		t.Run(c.ID, func(t *testing.T) {
			ps := Problems(c.Text, true)
			if len(ps) != c.ExpectedBoldProblems {
				t.Fatalf("問題の数 = %d, want %d: %+v", len(ps), c.ExpectedBoldProblems, ps)
			}
			lines := []int{}
			for _, p := range ps {
				lines = append(lines, p.Line)
			}
			want := c.ExpectedLineNumbers
			if want == nil {
				want = []int{}
			}
			if !reflect.DeepEqual(lines, want) {
				t.Errorf("行番号 = %v, want %v", lines, want)
			}
		})
	}
}

func TestAstralBeforeBold(t *testing.T) {
	ps := Problems("𠮷は**「重要」**です。", true)
	if len(ps) != 1 || ps[0].How != "かっこの内側だけを太字にする" {
		t.Fatalf("Problems = %+v", ps)
	}
	if ps[0].Suggest != "𠮷は「**重要**」です。" {
		t.Errorf("Suggest = %q", ps[0].Suggest)
	}
}
