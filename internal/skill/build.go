package skill

import (
	_ "embed"
	"fmt"
	"io/fs"
	"strings"

	"github.com/sudame/yomiyasu-go/internal/rules"
)

//go:embed patches.toml
var patchesTOML []byte

// Build は、本家の SKILL.md と references に変換規則を当て、LICENSE を添えたスキルの中身を返す。
// キーは書き出し先からの相対パス（SKILL.md、references/...、LICENSE）。
func Build(up fs.FS, disabled map[string]bool, version string) (map[string]string, error) {
	files := map[string]string{}
	err := fs.WalkDir(up, "upstream", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel := strings.TrimPrefix(p, "upstream/")
		if rel != "SKILL.md" && !strings.HasPrefix(rel, "references/") {
			return nil
		}
		b, err := fs.ReadFile(up, p)
		files[rel] = string(b)
		return err
	})
	if err != nil {
		return nil, err
	}
	patches, err := LoadPatches(patchesTOML)
	if err != nil {
		return nil, err
	}
	out, err := Apply(patches, files, disabled)
	if err != nil {
		return nil, err
	}
	commit, err := fs.ReadFile(up, "upstream/UPSTREAM_COMMIT")
	if err != nil {
		return nil, err
	}
	out["SKILL.md"] = insertHeader(out["SKILL.md"], header(strings.TrimSpace(string(commit)), version, disabled))
	license, err := fs.ReadFile(up, "LICENSE")
	if err != nil {
		return nil, err
	}
	out["LICENSE"] = string(license)
	return out, nil
}

func header(commit, version string, disabled map[string]bool) string {
	var off []string
	for _, id := range rules.All {
		if disabled[id] {
			off = append(off, id)
		}
	}
	list := "なし"
	if len(off) > 0 {
		list = strings.Join(off, ", ")
	}
	return fmt.Sprintf("<!-- yomiyasu-go が生成した。元: nanaism/yomiyasu@%s（MIT License）。yomiyasu-go %s。無効にした規則: %s。"+
		"このファイルは直接編集せず、設定を変えて yomiyasu-go skill install を実行し直す。 -->\n", commit, version, list)
}

// insertHeader は frontmatter の直後に header を入れる。
func insertHeader(skill, header string) string {
	if rest, ok := strings.CutPrefix(skill, "---\n"); ok {
		if i := strings.Index(rest, "\n---\n"); i >= 0 {
			cut := len("---\n") + i + len("\n---\n")
			return skill[:cut] + header + skill[cut:]
		}
	}
	return header + skill
}
