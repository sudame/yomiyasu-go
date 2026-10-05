// Command yomiyasu-go は、yomiyasu の lint と diff の Go 版と、スキルの書き出しを行う。
package main

import (
	"os"

	"github.com/sudame/yomiyasu-go/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, os.Getenv))
}
