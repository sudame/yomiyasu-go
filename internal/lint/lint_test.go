package lint

import "testing"

func rulesOf(r Result) []string {
	var out []string
	for _, f := range r.Findings {
		out = append(out, f.Rule)
	}
	return out
}

func TestSilentKowareruIsNotCountedTwice(t *testing.T) {
	r := Lint("データが静かに壊れる。\n", nil)
	n := 0
	for _, f := range r.Findings {
		if f.Rule == "metaphor_verb" {
			n++
		}
	}
	if n != 1 {
		t.Errorf("metaphor_verb = %d 件, want 1: %v", n, rulesOf(r))
	}
}

func TestMetricsKeepIntZeroForEmptyText(t *testing.T) {
	m := Lint("", nil).Metrics
	if !m.ListRatio.IsInt || !m.BoldPer1000.IsInt {
		t.Errorf("空の文書の比率は int の 0 になる: %+v", m)
	}
}
