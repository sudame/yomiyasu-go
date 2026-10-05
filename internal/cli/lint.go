package cli

import (
	"fmt"
	"io"
	"os"
	"unicode/utf8"

	"github.com/spf13/cobra"

	"github.com/sudame/yomiyasu-go/internal/config"
	"github.com/sudame/yomiyasu-go/internal/lint"
	"github.com/sudame/yomiyasu-go/internal/pycompat"
)

func newLintCmd(e *env) *cobra.Command {
	var asJSON, strict bool
	cmd := &cobra.Command{
		Use:   "lint [file]",
		Short: "日本語の表現とMarkdownの書式を、設定されたルールで点検します。指摘は見直し候補です。",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			path, err := config.DefaultPath(e.getenv)
			if err != nil {
				fmt.Fprintf(e.stderr, "設定ファイルのエラー: %v\n", err)
				return exitError{2}
			}
			cfg, err := config.Load(path)
			if err != nil {
				fmt.Fprintf(e.stderr, "設定ファイルのエラー: %v\n", err)
				return exitError{2}
			}
			// Python はファイルから読むときだけ改行を \n にそろえ、標準入力の改行はそのまま読む。
			var text string
			if len(args) == 1 {
				raw, err := os.ReadFile(args[0])
				if err == nil && !utf8.Valid(raw) {
					err = fmt.Errorf("UTF-8 として読めない")
				}
				if err != nil {
					fmt.Fprintf(e.stderr, "Error opening file %s: %v\n", args[0], err)
					return exitError{2}
				}
				text = pycompat.NormalizeNewlines(string(raw))
			} else {
				raw, err := io.ReadAll(e.stdin)
				if err != nil {
					return err
				}
				text = string(raw)
			}
			disabled := cfg.Disabled
			r := lint.Lint(text, disabled)
			if asJSON {
				fmt.Fprint(e.stdout, lint.JSON(r))
			} else {
				fmt.Fprint(e.stdout, lint.TextReport(r, disabled))
			}
			if strict {
				for _, f := range r.Findings {
					if f.Severity == "warn" || f.Severity == "error" {
						return exitError{1}
					}
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "JSON形式で出力")
	cmd.Flags().BoolVar(&strict, "strict", false, "警告が1件でもあれば非ゼロ（終了コード1）で終了")
	return cmd
}
