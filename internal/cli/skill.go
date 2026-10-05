package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	yomiyasugo "github.com/sudame/yomiyasu-go"
	"github.com/sudame/yomiyasu-go/internal/config"
	"github.com/sudame/yomiyasu-go/internal/skill"
)

func (e *env) buildSkill() (map[string]string, error) {
	path, err := config.DefaultPath(e.getenv)
	if err != nil {
		return nil, fmt.Errorf("設定ファイルのエラー: %w", err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		return nil, fmt.Errorf("設定ファイルのエラー: %w", err)
	}
	return skill.Build(yomiyasugo.Upstream, cfg.Disabled, Version)
}

func newSkillCmd(e *env) *cobra.Command {
	cmd := &cobra.Command{Use: "skill", Short: "Claude Code のスキルを生成する"}
	cmd.AddCommand(&cobra.Command{
		Use:   "render",
		Short: "skill install で書き出す SKILL.md を標準出力に表示する",
		Args:  cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			files, err := e.buildSkill()
			if err != nil {
				return err
			}
			fmt.Fprint(e.stdout, files["SKILL.md"])
			return nil
		},
	})
	return cmd
}
