// Package cli は yomiyasu-go のコマンドを定義する。
package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	yomiyasugo "github.com/sudame/yomiyasu-go"
)

// Version は GoReleaser が ldflags で埋める。
var Version = "dev"

// exitError は、メッセージを出し終えたあとの終了コードだけを運ぶ。
type exitError struct{ code int }

func (e exitError) Error() string { return fmt.Sprintf("exit %d", e.code) }

type env struct {
	stdin          io.Reader
	stdout, stderr io.Writer
	getenv         func(string) string
}

// Run は引数を解釈してコマンドを実行し、終了コードを返す。
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer, getenv func(string) string) int {
	e := &env{stdin: stdin, stdout: stdout, stderr: stderr, getenv: getenv}
	root := &cobra.Command{
		Use:           "yomiyasu-go",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetIn(stdin)
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs(args)
	root.AddCommand(newVersionCmd(e))
	root.AddCommand(newLintCmd(e))
	root.AddCommand(newDiffCmd(e))

	err := root.Execute()
	var ee exitError
	switch {
	case err == nil:
		return 0
	case errors.As(err, &ee):
		return ee.code
	default:
		fmt.Fprintln(stderr, err)
		return 2
	}
}

func upstreamCommit() string {
	b, err := yomiyasugo.Upstream.ReadFile("upstream/UPSTREAM_COMMIT")
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(b))
}

func newVersionCmd(e *env) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "バージョンと、取り込んだ本家のコミットを表示する",
		Args:  cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			fmt.Fprintf(e.stdout, "yomiyasu-go %s (nanaism/yomiyasu@%s)\n", Version, upstreamCommit())
			return nil
		},
	}
}
