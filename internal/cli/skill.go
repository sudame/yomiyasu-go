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

	var dir string
	install := &cobra.Command{
		Use:   "install",
		Short: "設定を反映した SKILL.md、references、LICENSE を書き出す",
		Args:  cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			files, err := e.buildSkill()
			if err != nil {
				return err
			}
			if dir == "" {
				if dir, err = skill.DefaultDir(e.getenv); err != nil {
					return err
				}
			}
			if err := skill.Install(dir, files, "yomiyasu-go "+Version+" (nanaism/yomiyasu@"+upstreamCommit()+")"); err != nil {
				return err
			}
			fmt.Fprintln(e.stdout, dir)
			return nil
		},
	}
	install.Flags().StringVar(&dir, "dir", "", "書き出し先（既定は ~/.claude/skills/yomiyasu-go）")
	cmd.AddCommand(install)
	return cmd
}
