package skill

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	yomiyasugo "github.com/sudame/yomiyasu-go"
	"github.com/sudame/yomiyasu-go/internal/rules"
)

var update = flag.Bool("update", false, "golden を書き直す")

// 規則を無効にしたとき、生成物のどこにも残ってはならない文言。
var vanish = map[string][]string{
	"sentence_end_repetition":   {"3連続した場合にのみ", "3文以上連続した場合にのみ"},
	"excess_bold":               {"1,000文字あたり1〜2箇所"},
	"excess_list":               {"15%以下"},
	"bold_not_rendered":         {"太字として正しく表示される書き方に整えます", "かっこの内側だけを太字にします", "### 原則 14:", "bold_not_rendered（太字にならない書き方）が出た場合は"},
	"emoji_prohibited":          {"絵文字は使いません", "絵文字を使わない", "絵文字や装飾としての文末コロン", "絵文字、文末コロン"},
	"redundant_bracket":         {"情報量の増えない言い換えカッコ", "重複する補足カッコを削り", "### 原則 10:"},
	"unnatural_halfwidth_space": {"和欧文間の不自然な半角空白", "余計な半角空白を除去します", "### 原則 11:"},
	"trailing_colon":            {"文末コロン、", "装飾としての文末コロン", "文末の装飾としてのコロン"},
	"slop_vocabulary":           {"### 内容に合わない大げさな名詞の整理", "## 3. 2026年急増語", "## 4. 体験を大げさに見せる熟語", "## 5. 具体的に見えて意味が曖昧な言葉"},
	"metaphor_verb":             {"### 比喩動詞の具体化", "### 原則 4: 比喩動詞の具体化", "## 1. 比喩的に使われる動詞", "比喩動詞や言い回しは、ふだん使う言葉"},
	"meta_filler":               {"「重要なのは」「大事なのは」が評価そのものを担っているときは", "## 6. 不要な前置きと定型の結び", "「重要なのは」が評価を担っているときは述語に残し"},
	"negative_parallelism":      {"否定対比（AではなくB）は、否定を外しても主張が変わらないときだけ"},
}

func joinAll(files map[string]string) string {
	var b strings.Builder
	for _, v := range files {
		b.WriteString(v)
	}
	return b.String()
}

func TestEveryRuleHasVanishingPhrases(t *testing.T) {
	for _, id := range rules.All {
		if len(vanish[id]) == 0 {
			t.Errorf("%s の文言がない", id)
		}
	}
}

func TestPhrasesExistWhenEnabledAndVanishWhenDisabled(t *testing.T) {
	enabled, err := Build(yomiyasugo.Upstream, map[string]bool{}, "test")
	if err != nil {
		t.Fatal(err)
	}
	all := joinAll(enabled)
	for id, phrases := range vanish {
		for _, p := range phrases {
			if !strings.Contains(all, p) {
				t.Errorf("%s: 有効なのに %q がない（本家の文面と食い違っている）", id, p)
			}
		}
		got, err := Build(yomiyasugo.Upstream, map[string]bool{id: true}, "test")
		if err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		text := joinAll(got)
		for _, p := range phrases {
			if strings.Contains(text, p) {
				t.Errorf("%s を無効にしても %q が残っている", id, p)
			}
		}
	}
}

func TestAlwaysPatches(t *testing.T) {
	got, err := Build(yomiyasugo.Upstream, map[string]bool{}, "test")
	if err != nil {
		t.Fatal(err)
	}
	s := got["SKILL.md"]
	for _, want := range []string{"name: yomiyasu-go\n", "\n# yomiyasu-go\n", "yomiyasu-go lint <対象ファイル>", "yomiyasu-go diff 元の文.txt 書き直した文.txt", "<!-- yomiyasu-go が生成した。元: nanaism/yomiyasu@986da6ffc89316a90e509d007c1efe1fc59057e6"} {
		if !strings.Contains(s, want) {
			t.Errorf("SKILL.md に %q がない", want)
		}
	}
	for _, gone := range []string{"python3 ", "Pythonが実行できない環境では", "name: yomiyasu\n"} {
		if strings.Contains(s, gone) {
			t.Errorf("SKILL.md に %q が残っている", gone)
		}
	}
	if !strings.Contains(got["LICENSE"], "Copyright (c) 2026 nanaism") {
		t.Error("LICENSE に本家の表示がない")
	}
}

func TestGolden(t *testing.T) {
	allDisabled := map[string]bool{}
	for _, id := range rules.All {
		allDisabled[id] = true
	}
	for name, disabled := range map[string]map[string]bool{"all-enabled": {}, "all-disabled": allDisabled} {
		got, err := Build(yomiyasugo.Upstream, disabled, "test")
		if err != nil {
			t.Fatal(err)
		}
		for rel, content := range got {
			p := filepath.Join("../../testdata/golden", name, rel)
			if *update {
				if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
				continue
			}
			want, err := os.ReadFile(p)
			if err != nil {
				t.Fatalf("%s: %v（go test ./internal/skill/ -update で作る）", p, err)
			}
			if string(want) != content {
				t.Errorf("%s が golden と違う（意図した変化なら -update で書き直す）", p)
			}
		}
	}
}
