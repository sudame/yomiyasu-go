package cli

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sudame/yomiyasu-go/internal/config"
	"github.com/sudame/yomiyasu-go/internal/diff"
	"github.com/sudame/yomiyasu-go/internal/pycompat"
)

var stances = map[string]string{
	"勧め": "勧め", "読み手に勧める": "勧め", "決まり": "決まり", "手順": "決まり",
	"説明": "説明", "報告": "説明", "体験": "説明",
}

// newDiffCmd は本家の diff と同じく、argparse を使わずに引数を読む。
// -- で始まらない引数を位置引数とし、--stance=、--json、--endings だけを見る。
func newDiffCmd(e *env) *cobra.Command {
	return &cobra.Command{
		Use:                "diff <元の文> <書き直した文> [--stance=勧め|決まり|説明] [--json]",
		Short:              "元の文と書き直した文を比べ、見直す候補を出す",
		DisableFlagParsing: true,
		RunE: func(_ *cobra.Command, argv []string) error {
			var args []string
			stance := ""
			for _, a := range argv {
				if !strings.HasPrefix(a, "--") {
					args = append(args, a)
				}
				if v, ok := strings.CutPrefix(a, "--stance="); ok {
					stance = stances[v]
				}
			}
			// bold_not_rendered を無効にしたら、太字を直す候補も出さない
			cfg := config.Config{Disabled: map[string]bool{}}
			if path, err := config.DefaultPath(e.getenv); err == nil {
				if cfg, err = config.Load(path); err != nil {
					fmt.Fprintf(e.stderr, "設定ファイルのエラー: %v\n", err)
					return exitError{2}
				}
			}
			checkBold := !cfg.Disabled["bold_not_rendered"]
			// 本家は読めないファイルで例外を出して終了コード 1 で終わる
			read := func(p string) (string, error) {
				b, err := os.ReadFile(p)
				if err != nil {
					fmt.Fprintln(e.stderr, err)
					return "", exitError{1}
				}
				return pycompat.NormalizeNewlines(string(b)), nil
			}
			if slices.Contains(argv, "--endings") && len(args) > 0 {
				t, err := read(args[0])
				if err != nil {
					return err
				}
				fmt.Fprint(e.stdout, diff.Endings(t, stance, checkBold))
				return nil
			}
			if len(args) < 2 {
				fmt.Fprintln(e.stdout, "使い方: python3 yomiyasu_diff.py 元の文.txt 書き直した文.txt [--stance=勧め|決まり|説明] [--json]")
				fmt.Fprintln(e.stdout, "　　　  python3 yomiyasu_diff.py --endings ファイル [--stance=勧め|決まり|説明]")
				return exitError{1}
			}
			o, err := read(args[0])
			if err != nil {
				return err
			}
			r, err := read(args[1])
			if err != nil {
				return err
			}
			d := diff.Diff(o, r, stance)
			if !checkBold {
				d.Bold = nil
			}
			if slices.Contains(argv, "--json") {
				fmt.Fprint(e.stdout, diff.JSON(d))
			} else {
				fmt.Fprintln(e.stdout, diff.Report(d))
			}
			return nil
		},
	}
}
