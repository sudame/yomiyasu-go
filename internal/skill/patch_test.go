package skill

import (
	"strings"
	"testing"
)

const doc = "# T\n\n絵文字、文末コロン、ダッシュを排除します。\n\n## A\n本文A\n```\n# コードの中\n```\n### A-1\n細目\n## B\n本文B\n"

func files() map[string]string { return map[string]string{"SKILL.md": doc} }

func TestApplyDisjointTargetsInOneSentence(t *testing.T) {
	ps := []Patch{
		{ID: "emoji", Rules: []string{"emoji_prohibited"}, File: "SKILL.md", Anchor: "絵文字、文末コロン、ダッシュ", Target: "絵文字、", Count: 1},
		{ID: "colon", Rules: []string{"trailing_colon"}, File: "SKILL.md", Anchor: "絵文字、文末コロン、ダッシュ", Target: "文末コロン、", Count: 1},
	}
	got, err := Apply(ps, files(), map[string]bool{"emoji_prohibited": true, "trailing_colon": true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got["SKILL.md"], "\nダッシュを排除します。") {
		t.Errorf("got:\n%s", got["SKILL.md"])
	}
	got, _ = Apply(ps, files(), map[string]bool{"trailing_colon": true})
	if !strings.Contains(got["SKILL.md"], "絵文字、ダッシュを排除します。") {
		t.Errorf("got:\n%s", got["SKILL.md"])
	}
}

func TestSectionStopsAtSameOrHigherLevelAndIgnoresCode(t *testing.T) {
	ps := []Patch{{ID: "a", Rules: []string{"metaphor_verb"}, File: "SKILL.md", Section: "## A"}}
	got, err := Apply(ps, files(), map[string]bool{"metaphor_verb": true})
	if err != nil {
		t.Fatal(err)
	}
	want := "# T\n\n絵文字、文末コロン、ダッシュを排除します。\n\n## B\n本文B\n"
	if got["SKILL.md"] != want {
		t.Errorf("got:\n%q\nwant:\n%q", got["SKILL.md"], want)
	}
}

func TestAlwaysAndUnless(t *testing.T) {
	ps := []Patch{
		{ID: "always", File: "SKILL.md", Anchor: "# T", Replace: "# U", Count: 1},
		{ID: "only-bold", Rules: []string{"bold_not_rendered"}, Unless: []string{"excess_bold"}, File: "SKILL.md", Anchor: "本文B", Replace: "B1", Count: 1},
		{ID: "both", Rules: []string{"bold_not_rendered", "excess_bold"}, File: "SKILL.md", Anchor: "本文B", Replace: "B2", Count: 1},
	}
	cases := []struct {
		disabled map[string]bool
		want     string
	}{
		{map[string]bool{}, "本文B"},
		{map[string]bool{"bold_not_rendered": true}, "B1"},
		{map[string]bool{"bold_not_rendered": true, "excess_bold": true}, "B2"},
	}
	for _, c := range cases {
		got, err := Apply(ps, files(), c.disabled)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(got["SKILL.md"], "# U\n") || !strings.Contains(got["SKILL.md"], "\n"+c.want+"\n") {
			t.Errorf("disabled=%v got:\n%s", c.disabled, got["SKILL.md"])
		}
	}
}

func TestValidateErrors(t *testing.T) {
	cases := map[string][]Patch{
		"回":      {{ID: "x", File: "SKILL.md", Anchor: "存在しない", Count: 1}},
		"target": {{ID: "x", File: "SKILL.md", Anchor: "本文A", Target: "B", Count: 1}},
		"見出し":    {{ID: "x", File: "SKILL.md", Section: "## Z"}},
		"重な": {
			{ID: "x", Rules: []string{"emoji_prohibited"}, File: "SKILL.md", Anchor: "絵文字、文末コロン", Count: 1},
			{ID: "y", Rules: []string{"trailing_colon"}, File: "SKILL.md", Anchor: "文末コロン、ダッシュ", Count: 1},
		},
		"規則":   {{ID: "x", Rules: []string{"nope"}, File: "SKILL.md", Anchor: "本文A", Count: 1}},
		"ファイル": {{ID: "x", File: "NONE.md", Anchor: "本文A", Count: 1}},
	}
	for mention, ps := range cases {
		err := Validate(ps, files())
		if err == nil || !strings.Contains(err.Error(), mention) {
			t.Errorf("%s: err = %v", mention, err)
		}
	}
}

func TestValidateAllowsOverlapThatNeverCoApplies(t *testing.T) {
	ps := []Patch{
		{ID: "x", Rules: []string{"bold_not_rendered"}, Unless: []string{"excess_bold"}, File: "SKILL.md", Anchor: "本文B", Count: 1},
		{ID: "y", Rules: []string{"excess_bold"}, File: "SKILL.md", Anchor: "本文B", Count: 1},
	}
	if err := Validate(ps, files()); err != nil {
		t.Error(err)
	}
}

func TestLoadPatches(t *testing.T) {
	ps, err := LoadPatches([]byte("[[patch]]\nid = \"a\"\nfile = \"SKILL.md\"\nanchor = \"x\"\ncount = 1\n"))
	if err != nil || len(ps) != 1 || ps[0].ID != "a" {
		t.Errorf("LoadPatches = %+v, %v", ps, err)
	}
	if _, err := LoadPatches([]byte("[[patch]]\nid = \"a\"\nfiel = \"SKILL.md\"\n")); err == nil {
		t.Error("知らないキーを受け付けた")
	}
}
