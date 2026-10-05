package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/sudame/yomiyasu-go/internal/config"
)

func newConfigCmd(e *env) *cobra.Command {
	cmd := &cobra.Command{Use: "config", Short: "設定ファイルを扱う"}
	cmd.AddCommand(&cobra.Command{
		Use:   "init",
		Short: "すべての規則を有効にした設定の雛形を書き出す",
		Args:  cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			path, err := config.DefaultPath(e.getenv)
			if err != nil {
				return err
			}
			if err := config.Init(path); err != nil {
				return err
			}
			fmt.Fprintln(e.stdout, path)
			return nil
		},
	})
	return cmd
}
