# yomiyasu-go 実装計画

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** nanaism/yomiyasu の lint と diff を Go に移植し、規則を設定で無効にでき、その設定に合わせて SKILL.md を生成する単一バイナリ `yomiyasu-go` を作る。

**Architecture:** 本家のファイルを `upstream/` に無改変で置き、`go:embed` でバイナリに埋め込む。lint と diff は Python の挙動を一字一句まで再現する移植で、本家の Python で作った期待出力と突き合わせる互換テストで正しさを保証する。SKILL.md は、本家の原文に `patches.toml` の変換規則を当てて生成する。

**Tech Stack:** Go 1.27.1、`github.com/dlclark/regexp2`、`github.com/spf13/cobra`、`github.com/BurntSushi/toml`、golangci-lint 2.14.0、GoReleaser 2.18.2、互換テスト用の Python 3.14.8（すべて mise で固定）

**Spec:** `docs/superpowers/specs/2026-10-05-yomiyasu-go-design.md`

## Global Constraints

- Go のモジュールパスは `github.com/sudame/yomiyasu-go`。
- ツールのバージョンは `mise.toml` で固定する: go 1.27.1、golangci-lint 2.14.0、goreleaser 2.18.2、python 3.14.8。`mise exec` は付けず、`go` などをそのまま実行する。
- 取り込む本家のコミットは `986da6ffc89316a90e509d007c1efe1fc59057e6`。
- 本家から持ち込まないもの: `tests/corpus/human/`、`evals/`、`assets/`、README。
- 設定ファイルは `$XDG_CONFIG_HOME/yomiyasu-go/config.toml`（`XDG_CONFIG_HOME` が未設定か相対パスなら `~/.config/yomiyasu-go/config.toml`）。形式は `[rules.<規則ID>]` の表に `enabled = false`。
- スキル名は `yomiyasu-go`。既定の書き出し先は `~/.claude/skills/yomiyasu-go/`。
- lint と diff の標準出力・終了コードは、すべての規則が有効なとき本家の Python と完全に一致させる。指摘文（日本語）も変えない。
- 正規表現は本家のパターン文字列をそのまま使う。書き換えが要るのは `\UXXXXXXXX` だけで、それは `pyre` が読み込み時に変換する。
- シェルのコマンドは 1 回に 1 つだけ実行する。`&&`、`;`、`$(...)`、`git -C` はフックで拒否されるので、ディレクトリの移動は `cd` を別に実行する。シェルスクリプトのファイルの中ではこの制約はない。
- コミットメッセージは日本語で書き、末尾に `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>` を付ける。
- コードコメントは周りに合わせて少なめにし、日付や作業番号など古くなる情報は書かない。

## Review Focus

1. **CRLF の入力**: Windows で作ったファイルや貼り付けた文章は `\r\n` を含む。Python はテキストモードで読むので改行を `\n` にそろえてから検査する。Go 版も同じ結果を出すべきである。→ Task 5 の互換テストに `local/crlf.md` を入れる。
2. **空や空白だけの入力**: 本家は文字数が 0 のとき `bold_per_1000` と `list_ratio` を整数の `0` にする。JSON では `0` と `0.0` が違うので、ここを取り違えると互換テストが落ちる。→ Task 2 の `pycompat.Num` と Task 5 の `local/empty.md`、`local/blank.md`。
3. **BMP 外の文字**: 「𠮷」や絵文字は UTF-16 では 2 単位、UTF-8 では 4 バイトになる。Python は 1 文字として数えるので、太字の位置や `len()` の比較はすべてルーン単位で行う。→ Task 4 のテストと Task 5 の `local/astral.md`。
4. **設定ファイルの打ち間違い**: `[rules.trailng_colon]` のような誤りを黙って無視すると、利用者は規則が無効になったと思い込む。→ Task 7 で、知らない規則 ID や知らないキーを終了コード 2 のエラーにする。
5. **書き出し先の既存ファイル**: `~/.claude/skills/yomiyasu-go` が利用者の作ったディレクトリやシンボリックリンクだった場合、上書きしてはならない。→ Task 10 で、印のないディレクトリとシンボリックリンクには何も書かずに止まることをテストする。

---

## ファイル構成

```
yomiyasu-go/
├── embed.go                      # package yomiyasugo。upstream/ と LICENSE を埋め込む
├── upstream/                     # 本家のファイル（scripts/sync-upstream.sh が置く。手で編集しない）
│   ├── SKILL.md
│   ├── references/
│   ├── scripts/                  # yomiyasu_lint.py、yomiyasu_diff.py
│   └── UPSTREAM_COMMIT
├── cmd/yomiyasu-go/main.go       # os.Exit(cli.Run(...)) だけ
├── internal/
│   ├── cli/                      # cobra のコマンド定義。Run(args, stdin, stdout, stderr, getenv) int
│   ├── pycompat/                 # Python の float 表記・round・空白判定・改行・JSON 出力の再現
│   ├── pyre/                     # regexp2 を Python の re と同じ使い方で呼ぶ薄い層
│   ├── difflib/                  # difflib.SequenceMatcher（autojunk=False）の移植
│   ├── bold/                     # 太字が表示されるかの判定（lint と diff で共通）
│   ├── rules/                    # 規則 ID の一覧
│   ├── config/                   # 設定ファイルの読み込みと雛形の書き出し
│   ├── lint/                     # yomiyasu_lint.py の移植
│   ├── diff/                     # yomiyasu_diff.py の移植
│   └── skill/                    # 変換規則の適用、SKILL.md の生成、書き出し
│       └── patches.toml
├── scripts/
│   ├── sync-upstream.sh          # 本家の指定コミットを upstream/ と testdata/ に取り込む
│   └── gen-compat.sh             # 本家の Python で互換テストの期待出力を作る
├── testdata/
│   ├── upstream/bold_regressions.json
│   ├── compat/
│   │   ├── inputs/upstream/      # 本家のコーパス（sync-upstream.sh が置く）
│   │   ├── inputs/local/         # 自作の境界ケース
│   │   └── expected/             # gen-compat.sh の出力と cases.tsv
│   └── golden/                   # 生成した SKILL.md などの golden
├── .github/workflows/            # ci.yml、upstream-sync.yml、release.yml
├── .gitattributes                # testdata/** -text（CRLF の入力を壊さない）
├── .golangci.yml
├── .goreleaser.yaml
├── mise.toml
├── LICENSE（既存）
└── README.md
```

---

### Task 1: リポジトリの土台と本家の取り込み

**Files:**
- Create: `go.mod`, `mise.toml`, `.golangci.yml`, `.gitattributes`, `.gitignore`, `embed.go`, `cmd/yomiyasu-go/main.go`, `internal/cli/cli.go`, `internal/cli/cli_test.go`, `scripts/sync-upstream.sh`, `.github/workflows/ci.yml`
- Create（スクリプトが生成）: `upstream/**`, `testdata/upstream/bold_regressions.json`, `testdata/compat/inputs/upstream/**`

**Interfaces:**
- Produces: `cli.Run(args []string, stdin io.Reader, stdout, stderr io.Writer, getenv func(string) string) int`、`cli.Version`（ldflags で上書きする `var`）、`yomiyasugo.Upstream embed.FS`（`upstream/SKILL.md`、`upstream/references`、`upstream/UPSTREAM_COMMIT`、`LICENSE` を含む）

- [ ] **Step 1: mise と Go モジュールを用意する**

`mise.toml`:

```toml
[tools]
go = "1.27.1"
golangci-lint = "2.14.0"
goreleaser = "2.18.2"
python = "3.14.8"
```

実行（1 つずつ）:

```
mise trust
mise install
go mod init github.com/sudame/yomiyasu-go
go get github.com/dlclark/regexp2 github.com/spf13/cobra github.com/BurntSushi/toml
```

`.gitattributes`:

```
testdata/** -text
upstream/** -text
```

`.gitignore`:

```
/dist/
/yomiyasu-go
```

`.golangci.yml`:

```yaml
version: "2"
linters:
  default: standard
  enable:
    - errorlint
    - gocritic
    - misspell
    - revive
formatters:
  enable:
    - gofmt
    - goimports
```

- [ ] **Step 2: 本家を取り込むスクリプトを書く**

`scripts/sync-upstream.sh`（`chmod +x` する）:

```bash
#!/usr/bin/env bash
# 本家 nanaism/yomiyasu の指定コミット（既定は main）を upstream/ と testdata/ に取り込む。
set -euo pipefail
ref="${1:-main}"
root="$(cd "$(dirname "$0")/.." && pwd)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

git clone --quiet https://github.com/nanaism/yomiyasu.git "$tmp/up"
git -C "$tmp/up" checkout --quiet "$ref"
sha="$(git -C "$tmp/up" rev-parse HEAD)"
src="$tmp/up/skills/yomiyasu"

rm -rf "$root/upstream"
mkdir -p "$root/upstream"
cp "$src/SKILL.md" "$root/upstream/SKILL.md"
cp -R "$src/references" "$root/upstream/references"
cp -R "$src/scripts" "$root/upstream/scripts"
echo "$sha" > "$root/upstream/UPSTREAM_COMMIT"

in="$root/testdata/compat/inputs/upstream"
rm -rf "$in"
mkdir -p "$in"
for d in raw_ai yomiyasu_rewritten blacklist_ai edge_cases; do
  cp -R "$tmp/up/tests/corpus/$d" "$in/$d"
done

mkdir -p "$root/testdata/upstream"
cp "$tmp/up/tests/fixtures/bold_regressions.json" "$root/testdata/upstream/bold_regressions.json"
echo "$sha"
```

実行: `scripts/sync-upstream.sh 986da6ffc89316a90e509d007c1efe1fc59057e6`
期待: `986da6ffc89316a90e509d007c1efe1fc59057e6` が表示され、`upstream/SKILL.md` と `upstream/scripts/yomiyasu_lint.py` ができる。`testdata/compat/inputs/upstream/` に `human` がないことも確かめる。

- [ ] **Step 3: 埋め込み用のファイルを書く**

`embed.go`:

```go
// Package yomiyasugo は、バイナリに埋め込む本家のファイルと LICENSE を持つ。
package yomiyasugo

import "embed"

//go:embed upstream/SKILL.md upstream/references upstream/UPSTREAM_COMMIT LICENSE
var Upstream embed.FS
```

- [ ] **Step 4: CLI の入口の失敗するテストを書く**

`internal/cli/cli_test.go`:

```go
package cli

import (
	"bytes"
	"strings"
	"testing"
)

func run(t *testing.T, stdin string, env map[string]string, args ...string) (string, string, int) {
	t.Helper()
	var out, errOut bytes.Buffer
	getenv := func(k string) string { return env[k] }
	code := Run(args, strings.NewReader(stdin), &out, &errOut, getenv)
	return out.String(), errOut.String(), code
}

func TestVersionShowsUpstreamCommit(t *testing.T) {
	out, _, code := run(t, "", nil, "version")
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	if !strings.Contains(out, "986da6ffc89316a90e509d007c1efe1fc59057e6") {
		t.Errorf("version に本家のコミットがない: %q", out)
	}
}
```

実行: `go test ./internal/cli/`
期待: `Run` が未定義でコンパイルエラー。

- [ ] **Step 5: CLI の入口を書く**

`internal/cli/cli.go`:

```go
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
```

`cmd/yomiyasu-go/main.go`:

```go
package main

import (
	"os"

	"github.com/sudame/yomiyasu-go/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, os.Getenv))
}
```

- [ ] **Step 6: テストと lint が通ることを確かめる**

実行: `go test ./...`　期待: PASS
実行: `golangci-lint run`　期待: 指摘なし

- [ ] **Step 7: CI を書く**

`.github/workflows/ci.yml`（`actions/checkout` と `jdx/mise-action` は、`gh api repos/actions/checkout/releases/latest --jq .tag_name` などで最新のメジャー版を確かめてから書く）:

```yaml
name: ci
on:
  push:
    branches: [main]
  pull_request:
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
      - uses: jdx/mise-action@v3
      - run: golangci-lint run
      - run: go test ./...
```

互換テストの期待出力の再生成は Task 5 で足す。

- [ ] **Step 8: コミットする**

```
git add .
git commit -m "feat: リポジトリの土台を作り、本家の 986da6f を取り込む"
```

---

### Task 2: Python の値の表し方を再現する pycompat

**Files:**
- Create: `internal/pycompat/num.go`, `internal/pycompat/text.go`, `internal/pycompat/json.go`, `internal/pycompat/pycompat_test.go`

**Interfaces:**
- Produces:
  - `type Num struct { F float64; IsInt bool }`、`func Int(i int) Num`、`func Float(f float64) Num`、`func (n Num) String() string`（Python の `str()` と同じ）
  - `func FloatRepr(f float64) string`、`func Round(n Num, digits int) Num`
  - `func IsSpace(r rune) bool`、`func Strip(s string) string`、`func NormalizeNewlines(s string) string`
  - `type KV struct { K string; V any }`、`type Obj []KV`、`func Dumps(v any, indent int) string`（`V` に入れてよいのは `nil`、`bool`、`int`、`Num`、`string`、`[]any`、`[]string`、`Obj`）

- [ ] **Step 1: 失敗するテストを書く**

期待値は Python 3.14 で実際に出した値である。

`internal/pycompat/pycompat_test.go`:

```go
package pycompat

import "testing"

func TestFloatRepr(t *testing.T) {
	cases := map[float64]string{
		0.0: "0.0", 1.0: "1.0", 1.03: "1.03", 0.472: "0.472", 47.2: "47.2",
		1e-05: "1e-05", 1e16: "1e+16", 1e15: "1000000000000000.0",
		123456789012345678.0: "1.2345678901234568e+17", 0.1 + 0.2: "0.30000000000000004",
	}
	for in, want := range cases {
		if got := FloatRepr(in); got != want {
			t.Errorf("FloatRepr(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestRound(t *testing.T) {
	cases := []struct {
		in     Num
		digits int
		want   string
	}{
		{Float(0.472), 2, "0.47"}, {Float(0.472), 1, "0.5"}, {Float(2.675), 2, "2.67"},
		{Float(0.125), 2, "0.12"}, {Float(1e-05), 2, "0.0"}, {Int(0), 1, "0"},
	}
	for _, c := range cases {
		if got := Round(c.in, c.digits).String(); got != c.want {
			t.Errorf("Round(%v, %d) = %q, want %q", c.in, c.digits, got, c.want)
		}
	}
}

func TestStrip(t *testing.T) {
	if got := Strip(" \x1c\x85\xa0　x​ "); got != "x​" {
		t.Errorf("Strip = %q", got)
	}
}

func TestNormalizeNewlines(t *testing.T) {
	if got := NormalizeNewlines("a\r\nb\rc\n"); got != "a\nb\nc\n" {
		t.Errorf("NormalizeNewlines = %q", got)
	}
}

func TestDumpsIndent2(t *testing.T) {
	v := Obj{
		{"a", Int(0)}, {"b", Float(0)}, {"c", []any{}}, {"d", Obj{}},
		{"e", "\"\\\n\r\t\b\f\x01\x7f <&>あ"}, {"f", nil}, {"g", true},
		{"h", []any{1, "x"}},
	}
	want := "{\n  \"a\": 0,\n  \"b\": 0.0,\n  \"c\": [],\n  \"d\": {},\n" +
		"  \"e\": \"\\\"\\\\\\n\\r\\t\\b\\f\\u0001\x7f <&>あ\",\n" +
		"  \"f\": null,\n  \"g\": true,\n  \"h\": [\n    1,\n    \"x\"\n  ]\n}"
	if got := Dumps(v, 2); got != want {
		t.Errorf("Dumps =\n%s\nwant\n%s", got, want)
	}
}

func TestDumpsIndent1(t *testing.T) {
	v := Obj{{"a", []any{1, Obj{{"b", []any{}}}}}}
	want := "{\n \"a\": [\n  1,\n  {\n   \"b\": []\n  }\n ]\n}"
	if got := Dumps(v, 1); got != want {
		t.Errorf("Dumps =\n%s\nwant\n%s", got, want)
	}
}
```

実行: `go test ./internal/pycompat/`　期待: 未定義でコンパイルエラー。

- [ ] **Step 2: 実装する**

`internal/pycompat/num.go`:

```go
// Package pycompat は、Python の値の書き表し方を Go で再現する。
package pycompat

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Num は Python の int と float を区別して持つ。JSON や f-string で 0 と 0.0 を書き分けるため。
type Num struct {
	F     float64
	IsInt bool
}

func Int(i int) Num       { return Num{F: float64(i), IsInt: true} }
func Float(f float64) Num { return Num{F: f} }

func (n Num) String() string {
	if n.IsInt {
		return strconv.FormatInt(int64(n.F), 10)
	}
	return FloatRepr(n.F)
}

// FloatRepr は Python の repr(float) と同じ文字列を返す。
func FloatRepr(f float64) string {
	switch {
	case math.IsInf(f, 1):
		return "inf"
	case math.IsInf(f, -1):
		return "-inf"
	case math.IsNaN(f):
		return "nan"
	case f == 0:
		if math.Signbit(f) {
			return "-0.0"
		}
		return "0.0"
	}
	mant, expStr, _ := strings.Cut(strconv.FormatFloat(f, 'e', -1, 64), "e")
	exp, _ := strconv.Atoi(expStr)
	if exp < -4 || exp >= 16 {
		sign := "+"
		if exp < 0 {
			sign, exp = "-", -exp
		}
		return fmt.Sprintf("%se%s%02d", mant, sign, exp)
	}
	s := strconv.FormatFloat(f, 'f', -1, 64)
	if !strings.Contains(s, ".") {
		s += ".0"
	}
	return s
}

// Round は Python の round(x, digits) と同じ値を返す。int はそのまま返す。
func Round(n Num, digits int) Num {
	if n.IsInt {
		return n
	}
	f, _ := strconv.ParseFloat(strconv.FormatFloat(n.F, 'f', digits, 64), 64)
	return Float(f)
}
```

`internal/pycompat/text.go`:

```go
package pycompat

import "strings"

// IsSpace は Python の str.isspace() と同じ判定をする。unicode.IsSpace とは \x1c〜\x1f などが違う。
func IsSpace(r rune) bool {
	switch r {
	case '\t', '\n', '\v', '\f', '\r', 0x1c, 0x1d, 0x1e, 0x1f, ' ',
		0x85, 0xa0, 0x1680, 0x2028, 0x2029, 0x202f, 0x205f, 0x3000:
		return true
	}
	return r >= 0x2000 && r <= 0x200a
}

// Strip は引数なしの str.strip() と同じ。
func Strip(s string) string { return strings.TrimFunc(s, IsSpace) }

// NormalizeNewlines は、Python がテキストモードでファイルを読むときと同じく改行を \n にそろえる。
func NormalizeNewlines(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
}
```

`internal/pycompat/json.go`:

```go
package pycompat

import (
	"fmt"
	"strconv"
	"strings"
)

// KV と Obj は、キーの順序を保った JSON オブジェクト。Python の dict の挿入順を再現する。
type KV struct {
	K string
	V any
}

type Obj []KV

// Dumps は json.dumps(v, ensure_ascii=False, indent=indent) と同じ文字列を返す。
func Dumps(v any, indent int) string {
	var b strings.Builder
	write(&b, v, indent, 0)
	return b.String()
}

func write(b *strings.Builder, v any, indent, level int) {
	pad := func(l int) string { return "\n" + strings.Repeat(" ", indent*l) }
	switch x := v.(type) {
	case nil:
		b.WriteString("null")
	case bool:
		b.WriteString(strconv.FormatBool(x))
	case int:
		b.WriteString(strconv.Itoa(x))
	case Num:
		b.WriteString(x.String())
	case string:
		quote(b, x)
	case []string:
		items := make([]any, len(x))
		for i, s := range x {
			items[i] = s
		}
		write(b, items, indent, level)
	case []any:
		if len(x) == 0 {
			b.WriteString("[]")
			return
		}
		b.WriteString("[")
		for i, item := range x {
			if i > 0 {
				b.WriteString(",")
			}
			b.WriteString(pad(level + 1))
			write(b, item, indent, level+1)
		}
		b.WriteString(pad(level) + "]")
	case Obj:
		if len(x) == 0 {
			b.WriteString("{}")
			return
		}
		b.WriteString("{")
		for i, kv := range x {
			if i > 0 {
				b.WriteString(",")
			}
			b.WriteString(pad(level + 1))
			quote(b, kv.K)
			b.WriteString(": ")
			write(b, kv.V, indent, level+1)
		}
		b.WriteString(pad(level) + "}")
	default:
		panic(fmt.Sprintf("pycompat.Dumps: 扱えない型 %T", v))
	}
}

func quote(b *strings.Builder, s string) {
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		default:
			if r < 0x20 {
				fmt.Fprintf(b, `\u%04x`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
}
```

- [ ] **Step 3: テストが通ることを確かめる**

実行: `go test ./internal/pycompat/`　期待: PASS

- [ ] **Step 4: コミットする**

```
git add internal/pycompat
git commit -m "feat: Python の数値・空白・JSON の書き表し方を再現する pycompat を足す"
```

---

### Task 3: Python の re と同じ使い方をする pyre

**Files:**
- Create: `internal/pyre/pyre.go`, `internal/pyre/pyre_test.go`

**Interfaces:**
- Produces:
  - `type Match struct { Start, End int; Text string; Groups []string }`（位置はルーン単位。`Groups[0]` は全体）
  - `func MustCompile(pattern string) *Regexp`
  - `(*Regexp).Search(s string) (Match, bool)`、`Match(s string) (Match, bool)`（先頭に固定。`re.match` 相当）、`FindAll(s string) []Match`、`FindAllStrings(s string) []string`（`re.findall` 相当。グループのないパターン専用）、`Count(s string) int`、`Split(s string) []string`、`Sub(s string, repl func(Match) string) string`、`SubGroup1(s string) string`（置換文字列 `\1` の専用版）

- [ ] **Step 1: 失敗するテストを書く**

期待値は Python 3.14 で出した値である。`Split` の 1 件目について補足する。Python は、空のマッチの直後に同じ位置から始まる空でないマッチ（`\n+`）を拾い、`'a。', '', 'b！', 'c', 'd'` を返す。regexp2 は空のマッチの次の探索を 1 文字先から始めるので、空白だけの断片の出方が変わることがある。本家はどの呼び出しでも空白だけの断片を捨てているので、テストは空白だけの断片を除いた結果で比べる。

`internal/pyre/pyre_test.go`:

```go
package pyre

import (
	"reflect"
	"strings"
	"testing"
)

func nonBlank(xs []string) []string {
	var out []string
	for _, x := range xs {
		if strings.TrimSpace(x) != "" {
			out = append(out, x)
		}
	}
	return out
}

func TestSplitLookbehind(t *testing.T) {
	got := MustCompile(`(?<=[。！？])`).Split("あ。い！う")
	want := []string{"あ。", "い！", "う"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Split = %q, want %q", got, want)
	}
}

func TestSplitLookbehindOrNewlines(t *testing.T) {
	got := nonBlank(MustCompile(`(?<=[。！？!?])|\n+`).Split("a。\n\nb！c\nd"))
	want := []string{"a。", "b！", "c", "d"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Split = %q, want %q", got, want)
	}
}

func TestPositionsAreRunes(t *testing.T) {
	m, ok := MustCompile(`(?<!\*)\*\*(?!\*)`).Search("𠮷あ**太字**")
	if !ok || m.Start != 2 || m.End != 4 {
		t.Errorf("Search = %+v, %v", m, ok)
	}
}

func TestMatchIsAnchored(t *testing.T) {
	re := MustCompile(`\s*(?:[*\-・]|\d+[.)])\s+`)
	if _, ok := re.Match("x - y"); ok {
		t.Error("Match が先頭以外でマッチした")
	}
	if _, ok := re.Match("  - y"); !ok {
		t.Error("Match が先頭でマッチしない")
	}
}

func TestUpperUEscape(t *testing.T) {
	re := MustCompile(`[\U0001F600-\U0001F64F]|[☀-➿]`)
	if got := re.FindAllStrings("a😀b☀"); !reflect.DeepEqual(got, []string{"😀", "☀"}) {
		t.Errorf("FindAllStrings = %q", got)
	}
}

func TestBackreference(t *testing.T) {
	re := MustCompile(`^\s{0,3}(?:(\*)\s*(?:\1\s*){2,}|(-)\s*(?:\2\s*){2,}|(_)\s*(?:\3\s*){2,})\s*$`)
	if _, ok := re.Match("* * *"); !ok {
		t.Error("区切り線にマッチしない")
	}
}

func TestSubGroup1(t *testing.T) {
	if got := MustCompile(`\*\*(.+?)\*\*`).SubGroup1("a**b**c**d**"); got != "abcd" {
		t.Errorf("SubGroup1 = %q", got)
	}
}

func TestDollarBeforeTrailingNewline(t *testing.T) {
	if _, ok := MustCompile(`[：:]$`).Search("ラベル：\n"); !ok {
		t.Error("$ が末尾の改行の前でマッチしない")
	}
}
```

実行: `go test ./internal/pyre/`　期待: 未定義でコンパイルエラー。

- [ ] **Step 2: 実装する**

`internal/pyre/pyre.go`:

```go
// Package pyre は、Python の re と同じ使い方で regexp2 を呼ぶ。位置はすべてルーン単位で返す。
package pyre

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/dlclark/regexp2"
)

type Match struct {
	Start, End int
	Text       string
	Groups     []string
}

type Regexp struct {
	re, anchored *regexp2.Regexp
}

// .NET の構文には \UXXXXXXXX がないので、その文字そのものに置き換える。
var upperU = regexp.MustCompile(`\\U([0-9A-Fa-f]{8})`)

func MustCompile(pattern string) *Regexp {
	p := upperU.ReplaceAllStringFunc(pattern, func(s string) string {
		n, err := strconv.ParseUint(s[2:], 16, 32)
		if err != nil {
			panic(err)
		}
		return string(rune(n))
	})
	return &Regexp{
		re:       regexp2.MustCompile(p, regexp2.None),
		anchored: regexp2.MustCompile(`\A(?:`+p+`)`, regexp2.None),
	}
}

func toMatch(m *regexp2.Match) Match {
	gs := m.Groups()
	groups := make([]string, len(gs))
	for i, g := range gs {
		groups[i] = g.String()
	}
	return Match{Start: m.Index, End: m.Index + m.Length, Text: m.String(), Groups: groups}
}

func first(re *regexp2.Regexp, s string) (Match, bool) {
	m, err := re.FindStringMatch(s)
	if err != nil {
		panic(fmt.Sprintf("pyre: %v", err))
	}
	if m == nil {
		return Match{}, false
	}
	return toMatch(m), true
}

func (r *Regexp) Search(s string) (Match, bool) { return first(r.re, s) }
func (r *Regexp) Match(s string) (Match, bool)  { return first(r.anchored, s) }

func (r *Regexp) FindAll(s string) []Match {
	var out []Match
	m, err := r.re.FindStringMatch(s)
	for ; m != nil && err == nil; m, err = r.re.FindNextMatch(m) {
		out = append(out, toMatch(m))
	}
	if err != nil {
		panic(fmt.Sprintf("pyre: %v", err))
	}
	return out
}

func (r *Regexp) FindAllStrings(s string) []string {
	ms := r.FindAll(s)
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = m.Text
	}
	return out
}

func (r *Regexp) Count(s string) int { return len(r.FindAll(s)) }

func (r *Regexp) Split(s string) []string {
	rs := []rune(s)
	var out []string
	last := 0
	for _, m := range r.FindAll(s) {
		out = append(out, string(rs[last:m.Start]))
		last = m.End
	}
	return append(out, string(rs[last:]))
}

func (r *Regexp) Sub(s string, repl func(Match) string) string {
	rs := []rune(s)
	var out []rune
	last := 0
	for _, m := range r.FindAll(s) {
		out = append(out, rs[last:m.Start]...)
		out = append(out, []rune(repl(m))...)
		last = m.End
	}
	return string(append(out, rs[last:]...))
}

func (r *Regexp) SubGroup1(s string) string {
	return r.Sub(s, func(m Match) string { return m.Groups[1] })
}
```

- [ ] **Step 3: テストが通ることを確かめる**

実行: `go test ./internal/pyre/`　期待: PASS。`TestSplitLookbehindOrNewlines` が落ちた場合は、空白だけの断片以外に差がないかを確かめる。差があれば Python の `re.split` の結果と突き合わせて原因を調べる。

- [ ] **Step 4: コミットする**

```
git add internal/pyre go.mod go.sum
git commit -m "feat: Python の re と同じ使い方で regexp2 を呼ぶ pyre を足す"
```

---

### Task 4: difflib と太字の判定を移植する

**Files:**
- Create: `internal/difflib/difflib.go`, `internal/difflib/difflib_test.go`, `internal/bold/bold.go`, `internal/bold/bold_test.go`

**Interfaces:**
- Consumes: `pyre.MustCompile`、`pycompat.IsSpace`、`pycompat.Strip`
- Produces:
  - `difflib.Opcode{Tag string; I1, I2, J1, J2 int}`、`difflib.New(a, b []rune) *Matcher`、`(*Matcher).Ratio() float64`、`(*Matcher).Opcodes() []Opcode`
  - `bold.Problem{Line int; Found, Suggest, How string}`、`bold.Problems(text string, skipFrontmatter bool) []Problem`

- [ ] **Step 1: difflib の失敗するテストを書く**

期待値は Python 3.14 の `difflib.SequenceMatcher(None, a, b, autojunk=False)` で出した値である。

`internal/difflib/difflib_test.go`:

```go
package difflib

import (
	"reflect"
	"testing"
)

func TestRatioAndOpcodes(t *testing.T) {
	cases := []struct {
		a, b  string
		ratio float64
		ops   []Opcode
	}{
		{"abcd", "bcde", 0.75, []Opcode{{"delete", 0, 1, 0, 0}, {"equal", 1, 4, 0, 3}, {"insert", 4, 4, 3, 4}}},
		{"今日は晴れです。", "明日は雨です。", 0.6666666666666666, []Opcode{{"replace", 0, 1, 0, 1}, {"equal", 1, 3, 1, 3}, {"replace", 3, 5, 3, 4}, {"equal", 5, 8, 4, 7}}},
		{"", "", 1.0, nil},
		{"abxcd", "abcd", 0.8888888888888888, []Opcode{{"equal", 0, 2, 0, 2}, {"delete", 2, 3, 2, 2}, {"equal", 3, 5, 2, 4}}},
		{"aaaa", "aa", 0.6666666666666666, []Opcode{{"equal", 0, 2, 0, 2}, {"delete", 2, 4, 2, 2}}},
	}
	for _, c := range cases {
		m := New([]rune(c.a), []rune(c.b))
		if got := m.Ratio(); got != c.ratio {
			t.Errorf("Ratio(%q, %q) = %v, want %v", c.a, c.b, got, c.ratio)
		}
		if got := m.Opcodes(); !reflect.DeepEqual(got, c.ops) {
			t.Errorf("Opcodes(%q, %q) = %v, want %v", c.a, c.b, got, c.ops)
		}
	}
}
```

実行: `go test ./internal/difflib/`　期待: 未定義でコンパイルエラー。

- [ ] **Step 2: difflib を実装する**

CPython の `difflib.SequenceMatcher` から、`isjunk=None` かつ `autojunk=False` のときに通る処理だけを移す。junk がないので、`find_longest_match` の後半にある junk 用の 2 つのループは不要になる。

`internal/difflib/difflib.go`:

```go
// Package difflib は Python の difflib.SequenceMatcher（isjunk=None, autojunk=False）を移植する。
package difflib

import "sort"

type Opcode struct {
	Tag            string
	I1, I2, J1, J2 int
}

type Matcher struct {
	a, b     []rune
	b2j      map[rune][]int
	matching [][3]int
}

func New(a, b []rune) *Matcher {
	m := &Matcher{a: a, b: b, b2j: map[rune][]int{}}
	for j, r := range b {
		m.b2j[r] = append(m.b2j[r], j)
	}
	return m
}

func (m *Matcher) findLongestMatch(alo, ahi, blo, bhi int) (besti, bestj, bestsize int) {
	besti, bestj = alo, blo
	j2len := map[int]int{}
	for i := alo; i < ahi; i++ {
		newj2len := map[int]int{}
		for _, j := range m.b2j[m.a[i]] {
			if j < blo {
				continue
			}
			if j >= bhi {
				break
			}
			k := j2len[j-1] + 1
			newj2len[j] = k
			if k > bestsize {
				besti, bestj, bestsize = i-k+1, j-k+1, k
			}
		}
		j2len = newj2len
	}
	for besti > alo && bestj > blo && m.a[besti-1] == m.b[bestj-1] {
		besti, bestj, bestsize = besti-1, bestj-1, bestsize+1
	}
	for besti+bestsize < ahi && bestj+bestsize < bhi && m.a[besti+bestsize] == m.b[bestj+bestsize] {
		bestsize++
	}
	return besti, bestj, bestsize
}

func (m *Matcher) matchingBlocks() [][3]int {
	if m.matching != nil {
		return m.matching
	}
	la, lb := len(m.a), len(m.b)
	queue := [][4]int{{0, la, 0, lb}}
	var blocks [][3]int
	for len(queue) > 0 {
		q := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		alo, ahi, blo, bhi := q[0], q[1], q[2], q[3]
		i, j, k := m.findLongestMatch(alo, ahi, blo, bhi)
		if k == 0 {
			continue
		}
		blocks = append(blocks, [3]int{i, j, k})
		if alo < i && blo < j {
			queue = append(queue, [4]int{alo, i, blo, j})
		}
		if i+k < ahi && j+k < bhi {
			queue = append(queue, [4]int{i + k, ahi, j + k, bhi})
		}
	}
	sort.Slice(blocks, func(x, y int) bool {
		for n := 0; n < 3; n++ {
			if blocks[x][n] != blocks[y][n] {
				return blocks[x][n] < blocks[y][n]
			}
		}
		return false
	})
	var out [][3]int
	i1, j1, k1 := 0, 0, 0
	for _, b := range blocks {
		if i1+k1 == b[0] && j1+k1 == b[1] {
			k1 += b[2]
			continue
		}
		if k1 > 0 {
			out = append(out, [3]int{i1, j1, k1})
		}
		i1, j1, k1 = b[0], b[1], b[2]
	}
	if k1 > 0 {
		out = append(out, [3]int{i1, j1, k1})
	}
	m.matching = append(out, [3]int{la, lb, 0})
	return m.matching
}

func (m *Matcher) Opcodes() []Opcode {
	var out []Opcode
	i, j := 0, 0
	for _, b := range m.matchingBlocks() {
		ai, bj, size := b[0], b[1], b[2]
		tag := ""
		switch {
		case i < ai && j < bj:
			tag = "replace"
		case i < ai:
			tag = "delete"
		case j < bj:
			tag = "insert"
		}
		if tag != "" {
			out = append(out, Opcode{tag, i, ai, j, bj})
		}
		i, j = ai+size, bj+size
		if size > 0 {
			out = append(out, Opcode{"equal", ai, i, bj, j})
		}
	}
	return out
}

func (m *Matcher) Ratio() float64 {
	matches := 0
	for _, b := range m.matchingBlocks() {
		matches += b[2]
	}
	total := len(m.a) + len(m.b)
	if total == 0 {
		return 1.0
	}
	return 2.0 * float64(matches) / float64(total)
}
```

実行: `go test ./internal/difflib/`　期待: PASS

- [ ] **Step 3: 太字の判定の失敗するテストを書く**

本家の回帰テストデータをそのまま使う。

`internal/bold/bold_test.go`:

```go
package bold

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

type regression struct {
	ID                   string `json:"id"`
	Text                 string `json:"text"`
	ExpectedBoldProblems int    `json:"expected_bold_problems"`
	ExpectedLineNumbers  []int  `json:"expected_line_numbers"`
}

func TestUpstreamRegressions(t *testing.T) {
	b, err := os.ReadFile("../../testdata/upstream/bold_regressions.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []regression
	if err := json.Unmarshal(b, &cases); err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		t.Run(c.ID, func(t *testing.T) {
			ps := Problems(c.Text, true)
			if len(ps) != c.ExpectedBoldProblems {
				t.Fatalf("問題の数 = %d, want %d: %+v", len(ps), c.ExpectedBoldProblems, ps)
			}
			lines := []int{}
			for _, p := range ps {
				lines = append(lines, p.Line)
			}
			want := c.ExpectedLineNumbers
			if want == nil {
				want = []int{}
			}
			if !reflect.DeepEqual(lines, want) {
				t.Errorf("行番号 = %v, want %v", lines, want)
			}
		})
	}
}

func TestAstralBeforeBold(t *testing.T) {
	ps := Problems("𠮷は**「重要」**です。", true)
	if len(ps) != 1 || ps[0].How != "かっこの内側だけを太字にする" {
		t.Fatalf("Problems = %+v", ps)
	}
	if ps[0].Suggest != "𠮷は「**重要**」です。" {
		t.Errorf("Suggest = %q", ps[0].Suggest)
	}
}
```

実行: `go test ./internal/bold/`　期待: 未定義でコンパイルエラー。

- [ ] **Step 4: 太字の判定を移植する**

移植元は `upstream/scripts/yomiyasu_lint.py` の 222〜536 行目である（`BOLD_ASCII_PUNCT` から `_bold_short` まで）。diff 側の 250〜564 行目は同じ内容なので、`bold` パッケージを 1 つ作って両方から使う。関数は上から順に、次の対応で移す。

| Python | Go |
| --- | --- |
| `BOLD_ASCII_PUNCT`, `BOLD_BRACKETS` | `asciiPunct`（`map[rune]bool`）、`brackets`（`map[rune]rune`） |
| `_bold_ws(ch)` | `isWS(ch rune, ok bool) bool`。Python の `""`（範囲外）は `ok == false` で表す。判定は `pycompat.IsSpace` |
| `_bold_punct_gfm`, `_bold_punct_new` | `unicode.In(r, unicode.P)`、`unicode.In(r, unicode.P, unicode.S)` |
| `_bold_can_open`, `_bold_can_close`, `_bold_code_spans`, `_bold_pairs_in_block`, `_bold_pair_ok`, `_bold_close_of`, `_bold_fix`, `_line_containers` | 同じ名前の非公開関数（`canOpen` など） |
| `bold_problems` | `Problems` |
| `_bold_short` | `short` |

移植の決まりは次のとおり。

- 文字列は最初に `[]rune` にし、添字・`len()`・スライスはすべてルーン単位で扱う。
- 正規表現は `pyre.MustCompile` に、Python の raw 文字列の中身をバッククォートでそのまま渡す。パッケージ変数として 1 度だけコンパイルする。`re.match` は `.Match`、`re.search` は `.Search`、`re.finditer` は `.FindAll` を使う。
- `rstrip("\r")` は `strings.TrimRight(s, "\r")`、`strip()` は `pycompat.Strip`、`lstrip()` は `strings.TrimLeftFunc(s, pycompat.IsSpace)` を使う。
- `bisect.bisect_right` は `sort.Search(len(offsets), func(i int) bool { return offsets[i] > idx }) - 1` にする。
- `_bold_fix` の戻り値 `None` は、`(string, bool)` の `false` で表す。

実行: `go test ./internal/bold/`　期待: PASS

- [ ] **Step 5: コミットする**

```
git add internal/difflib internal/bold
git commit -m "feat: difflib.SequenceMatcher と太字の判定を移植する"
```

---

### Task 5: lint を移植し、互換テストを作る

**Files:**
- Create: `internal/lint/lint.go`, `internal/lint/report.go`, `internal/lint/lint_test.go`, `internal/cli/lint.go`, `internal/cli/compat_test.go`, `scripts/gen-compat.sh`, `testdata/compat/inputs/local/*.md`
- Modify: `internal/cli/cli.go`（`newLintCmd` を登録）、`.github/workflows/ci.yml`

**Interfaces:**
- Consumes: `bold.Problems`、`pyre`、`pycompat`
- Produces:
  - `lint.Finding{Rule string; Line int; Severity, Message, Snippet string}`
  - `lint.Metrics{CharCount, TotalLines, ListLines, BoldCount int; ListRatio, BoldPer1000 pycompat.Num}`
  - `lint.Result{Score int; IsClean bool; Metrics Metrics; Findings []Finding}`
  - `lint.Lint(text string, disabled map[string]bool) Result`
  - `lint.JSON(r Result) string`（`indent=2`）、`lint.TextReport(r Result, disabled map[string]bool) string`
  - `cli` の `lint [file] [--json] [--strict]`

- [ ] **Step 1: 互換テストの期待出力を作るスクリプトを書く**

`scripts/gen-compat.sh`（`chmod +x` する）。本家の Python を実行し、ケースごとに標準出力と終了コードを保存する。Go 側のテストは `cases.tsv` を読んで同じ引数で `cli.Run` を呼ぶので、ケースの一覧はこのスクリプトだけが決める。

```bash
#!/usr/bin/env bash
# 本家の Python で互換テストの期待出力を作る。cases.tsv の各行は「ID<TAB>stdin のファイル<TAB>引数...」。
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"
export PYTHONIOENCODING=utf-8
lint=upstream/scripts/yomiyasu_lint.py
dif=upstream/scripts/yomiyasu_diff.py
out=testdata/compat/expected
rm -rf "$out"
mkdir -p "$out"
cases="$out/cases.tsv"
: > "$cases"
n=0

# 1 ケースを実行して記録する。$1=stdin に渡すファイル（なければ -）、$2=Python スクリプト、残り=引数
record() {
  local stdin="$1" script="$2"
  shift 2
  n=$((n + 1))
  local id
  id=$(printf '%04d' "$n")
  local sub=(lint)
  [ "$script" = "$dif" ] && sub=(diff)
  set +e
  if [ "$stdin" = "-" ]; then
    python3 "$script" "$@" > "$out/$id.out" < /dev/null
  else
    python3 "$script" "$@" > "$out/$id.out" < "$stdin"
  fi
  echo $? > "$out/$id.exit"
  set -e
  (IFS=$'\t'; echo "$id	$stdin	${sub[*]}	$*") >> "$cases"
}

inputs=$(find testdata/compat/inputs -type f -name '*.md' | LC_ALL=C sort)
for f in $inputs; do
  record - "$lint" "$f" --json
  record - "$lint" "$f"
  record - "$lint" "$f" --strict --json
  record - "$dif" --endings "$f"
done

# 標準入力から読むケース
for f in testdata/compat/inputs/local/crlf.md testdata/compat/inputs/local/astral.md; do
  record "$f" "$lint" --json
done

# 存在しないファイル（stdout と終了コードだけを比べる）
record - "$lint" testdata/compat/inputs/does-not-exist.md --json

# diff の組。本家のコーパスは raw_ai/X と yomiyasu_rewritten/X、blacklist_ai/X と yomiyasu_rewritten/X_bl
up=testdata/compat/inputs/upstream
pairs=()
for f in $(find "$up/raw_ai" -name '*.md' | LC_ALL=C sort); do
  pairs+=("$f" "$up/yomiyasu_rewritten/$(basename "$f")")
done
for f in $(find "$up/blacklist_ai" -name '*.md' | LC_ALL=C sort); do
  pairs+=("$f" "$up/yomiyasu_rewritten/$(basename "$f" .md)_bl.md")
done
for f in $(find testdata/compat/inputs/local/diff -name '*.orig.md' | LC_ALL=C sort); do
  pairs+=("$f" "${f%.orig.md}.rewrite.md")
done
for ((i = 0; i < ${#pairs[@]}; i += 2)); do
  o="${pairs[i]}"
  r="${pairs[i+1]}"
  record - "$dif" "$o" "$r" --json
  record - "$dif" "$o" "$r"
  for s in 勧め 決まり 説明 報告 不明; do
    record - "$dif" "$o" "$r" "--stance=$s" --json
  done
done

# 使い方の表示
record - "$dif" only-one-arg.md
echo "$n cases"
```

`cases.tsv` の 3 列目は Go 側のサブコマンド名（`lint` か `diff`）、4 列目以降は Python に渡した引数をタブ区切りで並べたものである。

- [ ] **Step 2: 自作の境界ケースを置く**

`testdata/compat/inputs/local/` に次のファイルを作る。`crlf.md` だけは、エディタではなく `printf` で書いて CRLF を保つ。

`crlf.md`（`printf` で作る）:

```
printf '# 見出し\r\n\r\n重要なのは、設計です。これは大事です。次も大事です。\r\n\r\n- 項目：\r\n- **「強調」**です。\r\n' > testdata/compat/inputs/local/crlf.md
```

`empty.md`: 空のファイル（`: > testdata/compat/inputs/local/empty.md`）

`blank.md`: 空白と改行だけ（`printf '  \n\n\t\n' > testdata/compat/inputs/local/blank.md`）

`astral.md`:

```markdown
𠮷野家で**「牛丼」**を食べました😀。地味に効きます。
𠮷は**必須です。**詳しくは下に書きます。
```

`frontmatter.md`:

```markdown
---
title: 解像度
---

本文の解像度を上げます。重要なのは、設計ではなく運用です。
```

`dense.md`（300 字を超え、太字と箇条書きが多い文書。`excess_bold` と `excess_list` を発火させる）: 「**太字**を含む 20 字ほどの文」を地の文で 10 行、続けて `- 箇条書きの項目です。` を 10 行書く。

`diff/stance.orig.md` と `diff/stance.rewrite.md`:

```markdown
設定ファイルを読み込みます。必ず保存しましょう。
- 手順を確認する
- 結果を共有する
```

```markdown
設定ファイルを読み込みます。保存してください。手順と結果を確認し、共有することが重要です。
```

- [ ] **Step 3: 期待出力を作る**

実行: `scripts/gen-compat.sh`
期待: `NNN cases` が表示され、`testdata/compat/expected/` に `cases.tsv`、`0001.out`、`0001.exit` などができる。

- [ ] **Step 4: 互換テストを書く（失敗する）**

`internal/cli/compat_test.go`:

```go
package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func TestCompatWithUpstreamPython(t *testing.T) {
	root := repoRoot(t)
	t.Chdir(root)
	cases, err := os.ReadFile("testdata/compat/expected/cases.tsv")
	if err != nil {
		t.Fatal(err)
	}
	env := map[string]string{"XDG_CONFIG_HOME": t.TempDir(), "HOME": t.TempDir()}
	for _, line := range strings.Split(strings.TrimRight(string(cases), "\n"), "\n") {
		cols := strings.Split(line, "\t")
		id, stdinPath, args := cols[0], cols[1], cols[2:]
		t.Run(id+" "+strings.Join(args, " "), func(t *testing.T) {
			var stdin []byte
			if stdinPath != "-" {
				if stdin, err = os.ReadFile(stdinPath); err != nil {
					t.Fatal(err)
				}
			}
			var out, errOut bytes.Buffer
			code := Run(args, bytes.NewReader(stdin), &out, &errOut, func(k string) string { return env[k] })
			wantOut, _ := os.ReadFile(filepath.Join("testdata/compat/expected", id+".out"))
			wantExitRaw, _ := os.ReadFile(filepath.Join("testdata/compat/expected", id+".exit"))
			wantExit, _ := strconv.Atoi(strings.TrimSpace(string(wantExitRaw)))
			if code != wantExit {
				t.Errorf("終了コード = %d, want %d（stderr: %s）", code, wantExit, errOut.String())
			}
			if out.String() != string(wantOut) {
				t.Errorf("標準出力が違う\n--- got\n%s\n--- want\n%s", out.String(), wantOut)
			}
		})
	}
}
```

実行: `go test ./internal/cli/ -run TestCompat`　期待: `lint` が未知のコマンドで FAIL。

- [ ] **Step 5: lint を移植する**

移植元は `upstream/scripts/yomiyasu_lint.py` の 17〜80 行目（定数）と、82〜220 行目、539〜719 行目（関数）である。太字の判定は Task 4 の `bold.Problems` を呼ぶ。

| Python | Go |
| --- | --- |
| `EMOJI_PATTERN`, `SLOP_WORDS`, `METAPHOR_VERB_PATTERNS`, `FILLER_PATTERNS`, `NEGATIVE_PARALLELISM_PATTERN` | 同じ順序のパッケージ変数。パターンは `pyre.MustCompile` に文字列をそのまま渡す |
| `get_frontmatter_line_count` | `frontmatterLineCount(lines []string) int` |
| `extract_plain_sentences` | `plainSentences(text string) []sentence`（`sentence{line int; text string}`） |
| `check_sentence_end_repetitions` | `sentenceEndRepetitions` |
| `analyze_markdown_metrics` | `metrics(text string) Metrics` |
| `lint_text` | `Lint`。規則を無効にする処理は Task 7 で足すので、ここでは `disabled` を受け取るだけで使わない |

移植の決まりは Task 4 と同じである。lint 特有の注意点は次のとおり。

- `metrics` の `bold_per_1000` と `list_ratio` は、分母が 0 のとき `pycompat.Int(0)`、それ以外は `pycompat.Round(pycompat.Float(x), 2)` と `pycompat.Round(pycompat.Float(x), 3)` にする。
- 指摘文の f-string に埋め込む数値は `Num.String()` で書く（`{metrics['bold_per_1000']}` は `metrics.BoldPer1000.String()`、`{round(metrics['list_ratio']*100, 1)}` は `pycompat.Round(mul100(metrics.ListRatio), 1).String()`）。`mul100` は `IsInt` を保ったまま 100 倍する。
- `len(s_clean) > 3` はルーン数で比べる。
- `' '.join(emoji_matches[:3])` は、最大 3 件をスペースでつなぐ。
- スコアは `max(0, 100 - penalty)`。`warn` と `error` は 5 点、それ以外は 2 点を引く。

`lint.JSON`:

```go
func JSON(r Result) string {
	findings := make([]any, len(r.Findings))
	for i, f := range r.Findings {
		findings[i] = pycompat.Obj{
			{K: "rule", V: f.Rule}, {K: "line", V: f.Line}, {K: "severity", V: f.Severity},
			{K: "message", V: f.Message}, {K: "snippet", V: f.Snippet},
		}
	}
	m := r.Metrics
	return pycompat.Dumps(pycompat.Obj{
		{K: "score", V: r.Score},
		{K: "is_clean", V: r.IsClean},
		{K: "metrics", V: pycompat.Obj{
			{K: "char_count", V: m.CharCount}, {K: "total_lines", V: m.TotalLines},
			{K: "list_lines", V: m.ListLines}, {K: "list_ratio", V: m.ListRatio},
			{K: "bold_count", V: m.BoldCount}, {K: "bold_per_1000", V: m.BoldPer1000},
		}},
		{K: "findings", V: findings},
	}, 2) + "\n"
}
```

`lint.TextReport` は、`main()` の 745〜761 行目にある `print` を、末尾の改行も含めて同じ文字列にする。`print("=" * 60)` は `strings.Repeat("=", 60) + "\n"`、`print(f"...\n")` は末尾に改行が 2 つ付く。

- [ ] **Step 6: lint コマンドを書く**

`internal/cli/lint.go`:

```go
package cli

import (
	"fmt"
	"io"
	"os"
	"unicode/utf8"

	"github.com/spf13/cobra"

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
			var raw []byte
			var err error
			if len(args) == 1 {
				raw, err = os.ReadFile(args[0])
				if err == nil && !utf8.Valid(raw) {
					err = fmt.Errorf("UTF-8 として読めない")
				}
				if err != nil {
					fmt.Fprintf(e.stderr, "Error opening file %s: %v\n", args[0], err)
					return exitError{2}
				}
			} else if raw, err = io.ReadAll(e.stdin); err != nil {
				return err
			}
			disabled := map[string]bool{}
			r := lint.Lint(pycompat.NormalizeNewlines(string(raw)), disabled)
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
```

`cli.go` の `Run` に `root.AddCommand(newLintCmd(e))` を足す。

- [ ] **Step 7: lint のケースが通ることを確かめる**

実行: `go test ./internal/cli/ -run 'TestCompat/.* lint'`
期待: lint のケースはすべて PASS。diff のケースは Task 6 で通すので、ここでは FAIL のままでよい。

lint のケースが落ちたら、出力の差から原因の関数を特定し、その関数の Python と Go を 1 行ずつ見比べる。

- [ ] **Step 8: CI で期待出力の再生成と差分の確認を足す**

`.github/workflows/ci.yml` の `go test ./...` の前に、次の 2 つを足す。

```yaml
      - run: scripts/gen-compat.sh
      - run: git diff --exit-code testdata/compat/expected
```

- [ ] **Step 9: コミットする**

```
git add .
git commit -m "feat: lint を移植し、本家の Python との互換テストを足す"
```

---

### Task 6: diff を移植する

**Files:**
- Create: `internal/diff/diff.go`, `internal/diff/endings.go`, `internal/diff/report.go`, `internal/diff/diff_test.go`, `internal/cli/diff.go`
- Modify: `internal/cli/cli.go`

**Interfaces:**
- Consumes: `bold.Problems`、`difflib.New`、`pyre`、`pycompat`
- Produces: `diff.Diff(orig, rewrite string, stance string) pycompat.Obj`（`stance` が空文字なら Python の `None`）、`diff.Report(d pycompat.Obj) string`、`diff.Endings(text, stance string) string`（`--endings` の出力全体）、`cli` の `diff`

- [ ] **Step 1: diff の単体テストを書く**

互換テストが主な検証になるので、単体テストは Python の挙動が分かりにくい 2 点だけを押さえる。

`internal/diff/diff_test.go`:

```go
package diff

import (
	"strings"
	"testing"
)

func TestEndingKind(t *testing.T) {
	cases := map[string]string{
		"保存してください。":    "依頼",
		"保存しましょう。":     "勧め",
		"設定を読み込みます。":   "動作（〜します）",
		"設定が変わります。":    "説明（〜ます）",
		"手順が重要です。":     "評価",
		"設定を読み込んでいます。": "説明（〜ています）",
	}
	for in, want := range cases {
		if got := endingKind(in); got != want {
			t.Errorf("endingKind(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestEndingsMostCommonKeepsFirstSeenOrderOnTies(t *testing.T) {
	out := Endings("保存してください。設定を読み込みます。保存してください。設定を読み込みます。\n", "")
	if !strings.HasPrefix(out, "■ 文末の種類（表を除く）: 依頼 2、動作（〜します） 2\n") {
		t.Errorf("Endings =\n%s", out)
	}
}
```

実行: `go test ./internal/diff/`　期待: 未定義でコンパイルエラー。

- [ ] **Step 2: diff を移植する**

移植元は `upstream/scripts/yomiyasu_diff.py` の 21〜248 行目と 567〜696 行目である。太字の判定は `bold.Problems` を呼ぶ。

| Python | Go |
| --- | --- |
| `MARKERS`（dict） | `markers []struct{ name string; re *pyre.Regexp }`。dict の順序を保つため、スライスにする |
| `CONTENT`, `LOGIC`, `STANCES`, `STATIVE`, `POTENTIAL`, `EVAL_END` | 同じ順序のパッケージ変数 |
| `normalize`, `has_list`, `sentences`, `paragraphs`, `logic_points`, `bare_end`, `register`, `ending_kind`, `ending_units`, `stance_flags`, `ending_changes`, `count`, `context`, `diff`, `bold_head`, `bold_lines`, `report` | 同じ名前（camel case）の関数 |
| `__main__` の引数処理 | `internal/cli/diff.go` |

diff 特有の注意点は次のとおり。

- `diff()` の返り値は `pycompat.Obj` で組む。キーの順序は `markers`、`new_words`、`lost_words`、`structure`、`spans`、`logic`、`bold`、`endings` である。`endings` の中は `stance`、`changes`、`flags`、`orig_flags` の順。`stance` は、空文字なら `nil`、それ以外は文字列を入れる。
- `bold` の各要素は `pycompat.Obj{{"line", p.Line}, {"found", p.Found}, {"suggest", p.Suggest}, {"how", p.How}}`。
- `sorted(...)` は `sort.Strings` でよい。UTF-8 のバイト順はコードポイント順と同じなので、Python と同じ並びになる。
- `set(...)` は `map[string]bool` にし、`sorted` の前に重複を除く。
- `w not in o` は `!strings.Contains(o, w)` にする。
- `ending_units` の `lines.index("---", 1)` は、2 行目以降で最初に `"---"` と完全一致する行を探す。見つからなければ何もしない。
- `set(l) <= set("|-: ")` は、`l` のすべての文字が `|`、`-`、`:`、空白のどれかであること。
- `stance_flags` の `r is not rows[-1]` は、要素の同一性で比べる。`ending_units` が返すスライスの添字で比べると確実である。
- `Counter.most_common()` は、数の多い順に並べ、同じ数のものは最初に出た順を保つ。`sort.SliceStable` で数の降順に並べる。
- `json.dumps(d, ensure_ascii=False, indent=1)` は `pycompat.Dumps(d, 1)`。`print` の改行も付ける。

- [ ] **Step 3: diff コマンドを書く**

本家の diff は argparse を使わず、`--` で始まらない引数を位置引数として扱う。この挙動を再現するため、cobra のフラグ解析を切る。

`internal/cli/diff.go`:

```go
package cli

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sudame/yomiyasu-go/internal/diff"
	"github.com/sudame/yomiyasu-go/internal/pycompat"
)

var stances = map[string]string{
	"勧め": "勧め", "読み手に勧める": "勧め", "決まり": "決まり", "手順": "決まり",
	"説明": "説明", "報告": "説明", "体験": "説明",
}

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
			read := func(p string) (string, error) {
				b, err := os.ReadFile(p)
				if err != nil {
					return "", err
				}
				return pycompat.NormalizeNewlines(string(b)), nil
			}
			if slices.Contains(argv, "--endings") && len(args) > 0 {
				t, err := read(args[0])
				if err != nil {
					return err
				}
				fmt.Fprint(e.stdout, diff.Endings(t, stance))
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
			if slices.Contains(argv, "--json") {
				fmt.Fprintln(e.stdout, pycompat.Dumps(d, 1))
			} else {
				fmt.Fprintln(e.stdout, diff.Report(d))
			}
			return nil
		},
	}
}
```

使い方の文言は互換テストのため本家と同じにしておく。`yomiyasu-go` 向けの文言にするかどうかは、Task 12 の README を書くときに決める。

`cli.go` の `Run` に `root.AddCommand(newDiffCmd(e))` を足す。

- [ ] **Step 4: すべての互換テストが通ることを確かめる**

実行: `go test ./...`　期待: PASS（互換テストのすべてのケースを含む）

- [ ] **Step 5: コミットする**

```
git add .
git commit -m "feat: diff を移植する"
```

---

### Task 7: 設定ファイルで規則を無効にする

**Files:**
- Create: `internal/rules/rules.go`, `internal/config/config.go`, `internal/config/config_test.go`, `internal/cli/config.go`, `internal/cli/lint_config_test.go`
- Modify: `internal/lint/lint.go`, `internal/lint/report.go`, `internal/cli/lint.go`, `internal/cli/cli.go`

**Interfaces:**
- Produces:
  - `rules.All []string`、`rules.Known(id string) bool`
  - `config.Config{Path string; Disabled map[string]bool}`、`config.DefaultPath(getenv func(string) string) (string, error)`、`config.Load(path string) (config.Config, error)`、`config.Init(path string) error`
  - `cli` の `config init`。lint は設定を読んで無効な規則を除く

- [ ] **Step 1: 規則 ID の一覧を書く**

`internal/rules/rules.go`:

```go
// Package rules は lint の規則 ID を持つ。順序は本家の lint が指摘を出す順に合わせる。
package rules

import "slices"

var All = []string{
	"excess_bold",
	"excess_list",
	"sentence_end_repetition",
	"bold_not_rendered",
	"emoji_prohibited",
	"redundant_bracket",
	"unnatural_halfwidth_space",
	"trailing_colon",
	"slop_vocabulary",
	"metaphor_verb",
	"meta_filler",
	"negative_parallelism",
}

func Known(id string) bool { return slices.Contains(All, id) }
```

- [ ] **Step 2: 設定の失敗するテストを書く**

`internal/config/config_test.go`:

```go
package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestDefaultPath(t *testing.T) {
	env := map[string]string{"HOME": "/home/u"}
	get := func(k string) string { return env[k] }
	if p, _ := DefaultPath(get); p != "/home/u/.config/yomiyasu-go/config.toml" {
		t.Errorf("DefaultPath = %q", p)
	}
	env["XDG_CONFIG_HOME"] = "/xdg"
	if p, _ := DefaultPath(get); p != "/xdg/yomiyasu-go/config.toml" {
		t.Errorf("DefaultPath = %q", p)
	}
	env["XDG_CONFIG_HOME"] = "relative"
	if p, _ := DefaultPath(get); p != "/home/u/.config/yomiyasu-go/config.toml" {
		t.Errorf("相対パスの XDG_CONFIG_HOME を使った: %q", p)
	}
}

func TestLoadMissingFileEnablesAll(t *testing.T) {
	c, err := Load(filepath.Join(t.TempDir(), "none.toml"))
	if err != nil || len(c.Disabled) != 0 || c.Path != "" {
		t.Errorf("Load = %+v, %v", c, err)
	}
}

func TestLoadDisabled(t *testing.T) {
	p := write(t, "[rules.trailing_colon]\nenabled = false\n\n[rules.emoji_prohibited]\nenabled = true\n")
	c, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if !c.Disabled["trailing_colon"] || c.Disabled["emoji_prohibited"] || c.Path != p {
		t.Errorf("Load = %+v", c)
	}
}

func TestLoadRejectsMistakes(t *testing.T) {
	cases := map[string]string{
		"[rules.trailng_colon]\nenabled = false\n":        "trailng_colon",
		"[rules.trailing_colon]\nenable = false\n":        "enable",
		"[rules.trailing_colon]\nenabled = \"false\"\n":   "enabled",
		"[rule.trailing_colon]\nenabled = false\n":        "rule",
		"[rules.trailing_colon\nenabled = false\n":        "",
	}
	for body, mention := range cases {
		_, err := Load(write(t, body))
		if err == nil {
			t.Errorf("誤りを受け付けた: %q", body)
			continue
		}
		if !strings.Contains(err.Error(), mention) {
			t.Errorf("エラーに %q が含まれない: %v", mention, err)
		}
	}
}

func TestInitWritesAllRulesAndRefusesOverwrite(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sub", "config.toml")
	if err := Init(p); err != nil {
		t.Fatal(err)
	}
	c, err := Load(p)
	if err != nil || len(c.Disabled) != 0 {
		t.Fatalf("雛形を読み直せない: %+v, %v", c, err)
	}
	if err := Init(p); err == nil {
		t.Error("既存の設定を上書きした")
	}
}
```

実行: `go test ./internal/config/`　期待: 未定義でコンパイルエラー。

- [ ] **Step 3: 設定を実装する**

`internal/config/config.go`:

```go
// Package config は ~/.config/yomiyasu-go/config.toml を読み書きする。
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/sudame/yomiyasu-go/internal/rules"
)

type Config struct {
	Path     string // 読んだ設定ファイル。ファイルがなければ空
	Disabled map[string]bool
}

func DefaultPath(getenv func(string) string) (string, error) {
	if x := getenv("XDG_CONFIG_HOME"); filepath.IsAbs(x) {
		return filepath.Join(x, "yomiyasu-go", "config.toml"), nil
	}
	home := getenv("HOME")
	if home == "" {
		return "", errors.New("HOME が設定されていないので、設定ファイルの場所を決められない")
	}
	return filepath.Join(home, ".config", "yomiyasu-go", "config.toml"), nil
}

type fileFormat struct {
	Rules map[string]struct {
		Enabled *bool `toml:"enabled"`
	} `toml:"rules"`
}

func Load(path string) (Config, error) {
	c := Config{Disabled: map[string]bool{}}
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	var f fileFormat
	md, err := toml.Decode(string(b), &f)
	if err != nil {
		return c, fmt.Errorf("%s: %w", path, err)
	}
	if und := md.Undecoded(); len(und) > 0 {
		keys := make([]string, len(und))
		for i, k := range und {
			keys[i] = k.String()
		}
		return c, fmt.Errorf("%s: 知らない設定項目がある: %s", path, strings.Join(keys, ", "))
	}
	for id, r := range f.Rules {
		if !rules.Known(id) {
			return c, fmt.Errorf("%s: 知らない規則 ID %q がある（使える ID: %s）", path, id, strings.Join(rules.All, ", "))
		}
		if r.Enabled != nil && !*r.Enabled {
			c.Disabled[id] = true
		}
	}
	c.Path = path
	return c, nil
}

func Init(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	var b strings.Builder
	b.WriteString("# yomiyasu-go の設定。enabled = false にした規則は、lint で検出せず、SKILL.md の書き直しの指示からも外れる。\n")
	b.WriteString("# 変えたら yomiyasu-go skill install を実行し直して SKILL.md に反映する。\n")
	for _, id := range rules.All {
		fmt.Fprintf(&b, "\n[rules.%s]\nenabled = true\n", id)
	}
	_, err = f.WriteString(b.String())
	return err
}
```

実行: `go test ./internal/config/`　期待: PASS。`enabled = "false"` で型の誤りが `toml.Decode` から返り、メッセージに `enabled` が含まれることも確かめる。含まれない場合は、`fmt.Errorf` で `rules.<id>.enabled は true か false` と言い直す。

- [ ] **Step 4: lint で無効な規則を除く失敗するテストを書く**

`internal/cli/lint_config_test.go`:

```go
package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLintSkipsDisabledRules(t *testing.T) {
	xdg := t.TempDir()
	cfg := filepath.Join(xdg, "yomiyasu-go", "config.toml")
	if err := os.MkdirAll(filepath.Dir(cfg), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte("[rules.trailing_colon]\nenabled = false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{"XDG_CONFIG_HOME": xdg}
	out, _, code := run(t, "項目：\n", env, "lint", "--json")
	if code != 0 || strings.Contains(out, "trailing_colon") || !strings.Contains(out, `"score": 100`) {
		t.Errorf("code=%d out=%s", code, out)
	}
}

func TestLintConfigErrorExits2(t *testing.T) {
	xdg := t.TempDir()
	cfg := filepath.Join(xdg, "yomiyasu-go", "config.toml")
	if err := os.MkdirAll(filepath.Dir(cfg), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte("[rules.nope]\nenabled = false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, errOut, code := run(t, "本文です。\n", map[string]string{"XDG_CONFIG_HOME": xdg}, "lint")
	if code != 2 || !strings.Contains(errOut, "設定ファイルのエラー") || !strings.Contains(errOut, "nope") {
		t.Errorf("code=%d stderr=%s", code, errOut)
	}
}

func TestLintTextReportMarksDisabledMetrics(t *testing.T) {
	xdg := t.TempDir()
	cfg := filepath.Join(xdg, "yomiyasu-go", "config.toml")
	if err := os.MkdirAll(filepath.Dir(cfg), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte("[rules.excess_bold]\nenabled = false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, _, _ := run(t, "本文です。\n", map[string]string{"XDG_CONFIG_HOME": xdg}, "lint")
	if !strings.Contains(out, "(検査は無効)") || !strings.Contains(out, "無効にした規則: excess_bold") {
		t.Errorf("out=%s", out)
	}
}

func TestConfigInit(t *testing.T) {
	xdg := t.TempDir()
	env := map[string]string{"XDG_CONFIG_HOME": xdg}
	if _, errOut, code := run(t, "", env, "config", "init"); code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errOut)
	}
	if _, err := os.Stat(filepath.Join(xdg, "yomiyasu-go", "config.toml")); err != nil {
		t.Fatal(err)
	}
	if _, _, code := run(t, "", env, "config", "init"); code == 0 {
		t.Error("既存の設定を上書きした")
	}
}
```

実行: `go test ./internal/cli/ -run 'TestLint|TestConfig'`　期待: FAIL

- [ ] **Step 5: lint と CLI に設定を通す**

- `lint.Lint` の最後で、スコアを計算する前に `disabled` に含まれる規則の指摘を除く。`metrics` はそのまま出す。
- `lint.TextReport` は、`disabled` が空なら本家と同じ文字列を出す。空でないときは次のように変える。
  - 罫線の直後に `・無効にした規則: <rules.All の順に , 区切り>` の行を足す。
  - `excess_bold` が無効なら、太字頻度の行の括弧書きを `(検査は無効)` にする。`excess_list` も箇条書き比率の行で同じようにする。
- `internal/cli/lint.go` では、ファイルを読む前に設定を読む。

```go
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
```

さらに、`disabled := map[string]bool{}` を `disabled := cfg.Disabled` に置き換える。

`internal/cli/config.go`:

```go
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
```

`cli.go` の `Run` に `root.AddCommand(newConfigCmd(e))` を足す。

- [ ] **Step 6: すべてのテストが通ることを確かめる**

実行: `go test ./...`　期待: PASS。互換テストは設定ファイルがない状態で走るので、結果は変わらない。

- [ ] **Step 7: コミットする**

```
git add .
git commit -m "feat: 設定ファイルで lint の規則を無効にできるようにする"
```

---

### Task 8: 変換規則を当てる仕組み

**Files:**
- Create: `internal/skill/patch.go`, `internal/skill/patch_test.go`

**Interfaces:**
- Consumes: `rules.Known`
- Produces:
  - `skill.Patch{ID string; Rules, Unless []string; File, Anchor, Target, Section, Replace string; Count int}`
  - `skill.LoadPatches(b []byte) ([]Patch, error)`
  - `skill.Validate(patches []Patch, files map[string]string) error`
  - `skill.Apply(patches []Patch, files map[string]string, disabled map[string]bool) (map[string]string, error)`

変換規則の意味は次のとおり。

- `rules` にある規則がすべて無効で、`unless` にある規則がすべて有効なときに当てる。`rules` が空なら、設定にかかわらず常に当てる。
- `anchor` は元の文書にちょうど `count` 回現れなければならない。置き換えるのは各 `anchor` の中の `target` の部分である（`target` を省くと `anchor` 全体）。`target` は `anchor` の中にちょうど 1 回現れなければならない。
- `section` を書いたときは、その文字列と完全に一致する見出しの行から、同じ深さかより浅い次の見出しの手前までを `replace` に置き換える。コードブロックの中の `#` で始まる行は見出しとみなさない。
- 置き換える範囲は、すべて元の文書の上で決める。そのため、同じ文の別の部分を書き換える変換規則どうしは互いの結果に影響しない。同時に当たりうる 2 つの変換規則（片方の `rules` ともう片方の `unless` に同じ規則がない組）の範囲が重なっていたら、`Validate` がエラーにする。
- `Validate` は設定にかかわらずすべての変換規則を検査する。本家の文面が変わって目印が消えたことは、どの設定でも生成の時点で分かる。

- [ ] **Step 1: 失敗するテストを書く**

`internal/skill/patch_test.go`:

```go
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
		"回": {{ID: "x", File: "SKILL.md", Anchor: "存在しない", Count: 1}},
		"target": {{ID: "x", File: "SKILL.md", Anchor: "本文A", Target: "B", Count: 1}},
		"見出し": {{ID: "x", File: "SKILL.md", Section: "## Z"}},
		"重な": {
			{ID: "x", Rules: []string{"emoji_prohibited"}, File: "SKILL.md", Anchor: "絵文字、文末コロン", Count: 1},
			{ID: "y", Rules: []string{"trailing_colon"}, File: "SKILL.md", Anchor: "文末コロン、ダッシュ", Count: 1},
		},
		"規則": {{ID: "x", Rules: []string{"nope"}, File: "SKILL.md", Anchor: "本文A", Count: 1}},
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
```

実行: `go test ./internal/skill/`　期待: 未定義でコンパイルエラー。

- [ ] **Step 2: 実装する**

`internal/skill/patch.go`:

```go
// Package skill は、本家の SKILL.md と references に変換規則を当てて、yomiyasu-go のスキルを作る。
package skill

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/sudame/yomiyasu-go/internal/rules"
)

type Patch struct {
	ID      string   `toml:"id"`
	Rules   []string `toml:"rules"`
	Unless  []string `toml:"unless"`
	File    string   `toml:"file"`
	Anchor  string   `toml:"anchor"`
	Target  string   `toml:"target"`
	Section string   `toml:"section"`
	Replace string   `toml:"replace"`
	Count   int      `toml:"count"`
}

func LoadPatches(b []byte) ([]Patch, error) {
	var f struct {
		Patch []Patch `toml:"patch"`
	}
	md, err := toml.Decode(string(b), &f)
	if err != nil {
		return nil, err
	}
	if und := md.Undecoded(); len(und) > 0 {
		return nil, fmt.Errorf("patches.toml に知らないキーがある: %v", und)
	}
	return f.Patch, nil
}

type span struct {
	start, end int
	patch      *Patch
}

func (p *Patch) applies(disabled map[string]bool) bool {
	for _, r := range p.Rules {
		if !disabled[r] {
			return false
		}
	}
	for _, r := range p.Unless {
		if disabled[r] {
			return false
		}
	}
	return true
}

func coApplicable(p, q *Patch) bool {
	need := append(slices.Clone(p.Rules), q.Rules...)
	for _, u := range append(slices.Clone(p.Unless), q.Unless...) {
		if slices.Contains(need, u) {
			return false
		}
	}
	return true
}

func locate(p *Patch, text string) ([]span, error) {
	if p.Section != "" {
		return locateSection(p, text)
	}
	if n := strings.Count(text, p.Anchor); p.Anchor == "" || n != p.Count {
		return nil, fmt.Errorf("変換規則 %s: %s で anchor が %d 回見つかった（期待 %d 回）: %q", p.ID, p.File, n, p.Count, p.Anchor)
	}
	target := p.Target
	if target == "" {
		target = p.Anchor
	}
	if strings.Count(p.Anchor, target) != 1 {
		return nil, fmt.Errorf("変換規則 %s: target が anchor の中にちょうど 1 回ない: %q", p.ID, target)
	}
	off := strings.Index(p.Anchor, target)
	var out []span
	for idx := 0; ; {
		j := strings.Index(text[idx:], p.Anchor)
		if j < 0 {
			break
		}
		s := idx + j + off
		out = append(out, span{s, s + len(target), p})
		idx += j + len(p.Anchor)
	}
	return out, nil
}

func headingLevel(line string) int {
	n := len(line) - len(strings.TrimLeft(line, "#"))
	if n == 0 || n > 6 || (len(line) > n && line[n] != ' ') {
		return 0
	}
	return n
}

func locateSection(p *Patch, text string) ([]span, error) {
	lines := strings.SplitAfter(text, "\n")
	start, level, found := -1, 0, 0
	pos, inCode := 0, false
	end := len(text)
	for _, l := range lines {
		trimmed := strings.TrimRight(l, "\n")
		if strings.HasPrefix(trimmed, "```") {
			inCode = !inCode
		}
		if !inCode {
			lv := headingLevel(trimmed)
			if trimmed == p.Section {
				found++
				if start < 0 {
					start, level = pos, lv
				}
			} else if start >= 0 && end == len(text) && lv > 0 && lv <= level {
				end = pos
			}
		}
		pos += len(l)
	}
	if found != 1 || level == 0 {
		return nil, fmt.Errorf("変換規則 %s: %s で見出し %q が %d 回見つかった（期待 1 回）", p.ID, p.File, p.Section, found)
	}
	return []span{{start, end, p}}, nil
}

func Validate(patches []Patch, files map[string]string) error {
	byFile := map[string][]span{}
	for i := range patches {
		p := &patches[i]
		for _, r := range append(slices.Clone(p.Rules), p.Unless...) {
			if !rules.Known(r) {
				return fmt.Errorf("変換規則 %s: 知らない規則 %q", p.ID, r)
			}
		}
		text, ok := files[p.File]
		if !ok {
			return fmt.Errorf("変換規則 %s: ファイル %s がない", p.ID, p.File)
		}
		spans, err := locate(p, text)
		if err != nil {
			return err
		}
		byFile[p.File] = append(byFile[p.File], spans...)
	}
	for file, spans := range byFile {
		for i := range spans {
			for j := i + 1; j < len(spans); j++ {
				a, b := spans[i], spans[j]
				if a.patch != b.patch && a.start < b.end && b.start < a.end && coApplicable(a.patch, b.patch) {
					return fmt.Errorf("%s で変換規則 %s と %s の範囲が重なっている", file, a.patch.ID, b.patch.ID)
				}
			}
		}
	}
	return nil
}

func Apply(patches []Patch, files map[string]string, disabled map[string]bool) (map[string]string, error) {
	if err := Validate(patches, files); err != nil {
		return nil, err
	}
	byFile := map[string][]span{}
	for i := range patches {
		p := &patches[i]
		if !p.applies(disabled) {
			continue
		}
		spans, _ := locate(p, files[p.File])
		byFile[p.File] = append(byFile[p.File], spans...)
	}
	out := make(map[string]string, len(files))
	for name, text := range files {
		spans := byFile[name]
		sort.Slice(spans, func(i, j int) bool { return spans[i].start < spans[j].start })
		var b strings.Builder
		last := 0
		for _, s := range spans {
			b.WriteString(text[last:s.start])
			b.WriteString(s.patch.Replace)
			last = s.end
		}
		b.WriteString(text[last:])
		out[name] = b.String()
	}
	return out, nil
}
```

- [ ] **Step 3: テストが通ることを確かめる**

実行: `go test ./internal/skill/`　期待: PASS

- [ ] **Step 4: コミットする**

```
git add internal/skill
git commit -m "feat: 本家の文面に変換規則を当てる仕組みを足す"
```

---

### Task 9: 変換規則を書き、SKILL.md を生成する

**Files:**
- Create: `internal/skill/patches.toml`, `internal/skill/build.go`, `internal/skill/build_test.go`, `internal/cli/skill.go`, `testdata/golden/all-enabled/**`, `testdata/golden/all-disabled/**`
- Modify: `internal/cli/cli.go`

**Interfaces:**
- Consumes: `skill.LoadPatches`、`skill.Apply`、`yomiyasugo.Upstream`、`config`
- Produces:
  - `skill.Build(up fs.FS, disabled map[string]bool, version string) (map[string]string, error)`。キーは書き出し先からの相対パス（`SKILL.md`、`references/...`、`LICENSE`）
  - `cli` の `skill render`（`SKILL.md` を標準出力へ）

- [ ] **Step 1: 規則ごとの文言のテストを書く**

規則を無効にしたら消える文言と、有効なら残る文言を規則ごとに並べる。下の表の文言は、本家 986da6f の文面から取ったものである。テストを書く前に、それぞれの文言が `upstream/` の中に実際にあるかを grep で確かめる。全角と半角の違いなどで見つからない文言は、本家の実際の文面に合わせて直す（文言の意図は変えない）。

規則ごとにどの書き直しを止めるかの判断は、フォーク `sudame/yomiyasu` の `feature-lint-config` ブランチで SKILL.md に足した「設定ファイルで無効にしたルール」の表に従う。

`internal/skill/build_test.go`:

```go
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
```

実行: `go test ./internal/skill/`　期待: `Build` が未定義でコンパイルエラー。

- [ ] **Step 2: 生成を実装する**

`internal/skill/build.go`:

```go
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
```

- [ ] **Step 3: 常に当てる変換規則を書く**

`internal/skill/patches.toml` の先頭に書く。`anchor` は本家 986da6f の `upstream/SKILL.md` からそのままコピーする。

```toml
# 本家 nanaism/yomiyasu の SKILL.md と references に当てる変換規則。
# rules の規則がすべて無効で、unless の規則がすべて有効なときに当てる。rules がなければ常に当てる。
# anchor は元の文書にちょうど count 回現れること。置き換えるのは anchor の中の target（省略時は anchor 全体）。

[[patch]]
id = "name"
file = "SKILL.md"
anchor = "name: yomiyasu\n"
replace = "name: yomiyasu-go\n"
count = 1

[[patch]]
id = "title"
file = "SKILL.md"
anchor = "\n# yomiyasu\n"
replace = "\n# yomiyasu-go\n"
count = 1

[[patch]]
id = "lint-command"
file = "SKILL.md"
anchor = "# スキル配置先（${CLAUDE_SKILL_DIR}等）を基準にスクリプトの絶対パスを解決して実行\npython3 <スキル配置ディレクトリ>/scripts/yomiyasu_lint.py <対象ファイル>"
replace = "yomiyasu-go lint <対象ファイル>"
count = 1

[[patch]]
id = "diff-command"
file = "SKILL.md"
anchor = "python3 <スキル配置ディレクトリ>/scripts/yomiyasu_diff.py 元の文.txt"
replace = "yomiyasu-go diff 元の文.txt"
count = 1

[[patch]]
id = "no-python-fallback"
file = "SKILL.md"
anchor = "Pythonが実行できない環境では、"
replace = "yomiyasu-go が実行できない環境では、"
count = 1
```

実行: `grep -rn 'python\|Python\|scripts/' upstream/SKILL.md upstream/references`
期待: 上の変換規則で置き換える箇所のほかに、Python やスクリプトの場所に触れた記述がないこと。ほかにもあれば、同じ形の変換規則を足す。

- [ ] **Step 4: 規則ごとの変換規則を書く**

`vanish` の文言が消え、文章として崩れないように、規則ごとに変換規則を足す。作業の順序は次のとおり。

1. `grep -rn '<vanish の文言>' upstream/` で、文言がある場所をすべて洗い出す。
2. 次の方針で変換規則を書く。
   - references の「### 原則 N」や slop-catalog の「## N.」のように、1 つの規則だけを扱う節は `section` で節ごと消す。
   - 複数の規則を並べた文（例: 「絵文字、文末コロン、ダッシュ記号（em dash）、情報量の増えない言い換えカッコ、和欧文間の不自然な半角空白を排除します。」）は、規則ごとに `anchor` を同じ文にし、`target` でその規則の語句（「絵文字、」「文末コロン、」など）だけを消す。並びの最後の語句を消すときは、直前の読点を `target` に含める。
   - 2 つの規則が同じ文を別の形で書き換える必要があり、`target` を分けられないときは、`unless` を使って組み合わせごとの変換規則にする。例えば SKILL.md の「元の文で太字にならない書き方に…」から「…そのまま維持します。」までの文は、次の 3 つに分ける。
     - `bold_not_rendered` だけが無効なとき: 太字の表示の段落を消し、「太字を減らす決まり（domains/tech.md の1,000文字あたり1〜2箇所など）はそのまま維持します。」だけを残す。
     - `excess_bold` だけが無効なとき: 「これは太字を増やす決まりではなく、…維持します。」を「これは太字を増やす決まりではありません。」にする。
     - 両方が無効なとき: 段落ごと消す。
   - 消すと手順が成り立たない文は、消さずに「〜は行いません」と書き換える。例えば `sentence_end_repetition` の「文末のリズム調整は、AIっぽさを直すついでに同一文末が3連続した場合にのみ行います。」は、「同一文末が続くことを理由にした文末のリズム調整は行いません。」にする。gemini-syntax.md の原則 8 の同じ趣旨の文も同じように書き換える。
3. `go test ./internal/skill/ -run TestPhrases` を通す。
4. 規則を 1 つずつ無効にして `go run ./cmd/yomiyasu-go skill render` の出力を読み、文が途中で切れていないか、番号付きの手順の中身が空になっていないかを目で確かめる（`skill render` は Step 6 で作るので、この確認は Step 6 のあとに行う）。

`unless` を使った組の書き方の例:

```toml
[[patch]]
id = "bold-paragraph-keep-excess"
rules = ["bold_not_rendered"]
unless = ["excess_bold"]
file = "SKILL.md"
anchor = "また、Markdownとして表示される文章（GitHubや技術記事など）で太字を残すときは、"  # 実際には段落の最後の「維持します。」までをコピーする
replace = "太字を減らす決まり（domains/tech.md の1,000文字あたり1〜2箇所など）はそのまま維持します。"
count = 1
```

- [ ] **Step 5: golden を作って中身を読む**

実行: `go test ./internal/skill/ -update -run TestGolden`
実行: `go test ./internal/skill/`　期待: PASS

`testdata/golden/all-disabled/SKILL.md` を通して読み、文の崩れがないか確かめる。崩れていたら Step 4 に戻る。

- [ ] **Step 6: skill render コマンドを書く**

`internal/cli/skill.go`:

```go
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
```

`cli.go` の `Run` に `root.AddCommand(newSkillCmd(e))` を足す。Step 4 の 4 番目の確認をここで行う。

- [ ] **Step 7: コミットする**

```
git add .
git commit -m "feat: 設定に合わせて SKILL.md を生成する"
```

---

### Task 10: スキルを書き出す

**Files:**
- Create: `internal/skill/install.go`, `internal/skill/install_test.go`
- Modify: `internal/cli/skill.go`

**Interfaces:**
- Consumes: `skill.Build`
- Produces: `skill.MarkerName = ".yomiyasu-go"`、`skill.Install(dir string, files map[string]string, marker string) error`、`skill.DefaultDir(getenv func(string) string) (string, error)`、`cli` の `skill install [--dir <path>]`

- [ ] **Step 1: 失敗するテストを書く**

`internal/skill/install_test.go`:

```go
package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var sample = map[string]string{"SKILL.md": "s", "references/a.md": "a", "LICENSE": "l"}

func TestInstallIntoNewDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "yomiyasu-go")
	if err := Install(dir, sample, "v"); err != nil {
		t.Fatal(err)
	}
	for rel, want := range sample {
		b, err := os.ReadFile(filepath.Join(dir, rel))
		if err != nil || string(b) != want {
			t.Errorf("%s = %q, %v", rel, b, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, MarkerName)); err != nil {
		t.Error("印がない")
	}
}

func TestReinstallRemovesStaleFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "yomiyasu-go")
	if err := Install(dir, sample, "v"); err != nil {
		t.Fatal(err)
	}
	if err := Install(dir, map[string]string{"SKILL.md": "s2"}, "v"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "references/a.md")); !os.IsNotExist(err) {
		t.Error("古いファイルが残った")
	}
}

func TestInstallRefusesForeignDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "yomiyasu-go")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := Install(dir, sample, "v")
	if err == nil || !strings.Contains(err.Error(), MarkerName) {
		t.Fatalf("err = %v", err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "SKILL.md")); string(b) != "mine" {
		t.Error("利用者のファイルを上書きした")
	}
}

func TestInstallRefusesSymlink(t *testing.T) {
	base := t.TempDir()
	real := filepath.Join(base, "real")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(real, MarkerName), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "yomiyasu-go")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	err := Install(link, sample, "v")
	if err == nil || !strings.Contains(err.Error(), "シンボリックリンク") {
		t.Fatalf("err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(real, "SKILL.md")); !os.IsNotExist(err) {
		t.Error("リンク先に書き込んだ")
	}
}

func TestInstallRefusesFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "yomiyasu-go")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Install(p, sample, "v"); err == nil {
		t.Error("ファイルを上書きした")
	}
}

func TestDefaultDir(t *testing.T) {
	get := func(k string) string { return map[string]string{"HOME": "/home/u"}[k] }
	if d, _ := DefaultDir(get); d != "/home/u/.claude/skills/yomiyasu-go" {
		t.Errorf("DefaultDir = %q", d)
	}
}
```

実行: `go test ./internal/skill/ -run 'TestInstall|TestReinstall|TestDefaultDir'`　期待: 未定義でコンパイルエラー。

- [ ] **Step 2: 実装する**

`internal/skill/install.go`:

```go
package skill

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// MarkerName は yomiyasu-go が書き出したディレクトリに置く印。印のないディレクトリには書き込まない。
const MarkerName = ".yomiyasu-go"

func DefaultDir(getenv func(string) string) (string, error) {
	home := getenv("HOME")
	if home == "" {
		return "", errors.New("HOME が設定されていないので、書き出し先を決められない")
	}
	return filepath.Join(home, ".claude", "skills", "yomiyasu-go"), nil
}

func Install(dir string, files map[string]string, marker string) error {
	fi, err := os.Lstat(dir)
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return err
	case fi.Mode()&fs.ModeSymlink != 0:
		return fmt.Errorf("%s はシンボリックリンクなので書き出さない。リンクを外してから実行し直す", dir)
	case !fi.IsDir():
		return fmt.Errorf("%s はディレクトリではないので書き出さない", dir)
	default:
		if _, err := os.Stat(filepath.Join(dir, MarkerName)); err != nil {
			return fmt.Errorf("%s には %s がなく、yomiyasu-go が書き出したディレクトリか分からないので書き出さない。中身を確かめて外してから実行し直す", dir, MarkerName)
		}
		if err := os.RemoveAll(dir); err != nil {
			return err
		}
	}
	for rel, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return os.WriteFile(filepath.Join(dir, MarkerName), []byte(marker+"\n"), 0o644)
}
```

`internal/cli/skill.go` の `newSkillCmd` に `install` を足す。

```go
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
```

- [ ] **Step 3: テストが通ることを確かめる**

実行: `go test ./...`　期待: PASS

- [ ] **Step 4: 手で書き出して確かめる**

実行: `go run ./cmd/yomiyasu-go skill install --dir /private/tmp/claude-501/yomiyasu-go-check`（スクラッチパッドなど一時的な場所に書き出す）
期待: 書き出し先のパスが表示され、`SKILL.md`、`references/`、`LICENSE`、`.yomiyasu-go` がある。

- [ ] **Step 5: コミットする**

```
git add .
git commit -m "feat: 設定を反映したスキルを書き出す skill install を足す"
```

---

### Task 11: 本家の更新を毎日取り込む

**Files:**
- Create: `.github/workflows/upstream-sync.yml`

**Interfaces:**
- Consumes: `scripts/sync-upstream.sh`、`scripts/gen-compat.sh`

- [ ] **Step 1: ワークフローを書く**

`GITHUB_TOKEN` で作った PR では、ほかのワークフロー（ci.yml）が動かない。そのため、テストはこのジョブの中で実行し、結果を PR の本文に載せる。

`.github/workflows/upstream-sync.yml`:

```yaml
name: upstream-sync
on:
  schedule:
    - cron: "0 0 * * *"
  workflow_dispatch:
permissions:
  contents: write
  pull-requests: write
jobs:
  sync:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
      - uses: jdx/mise-action@v3
      - name: 本家を取り込む
        id: sync
        run: |
          old="$(cat upstream/UPSTREAM_COMMIT)"
          new="$(scripts/sync-upstream.sh main | tail -n1)"
          echo "old=$old" >> "$GITHUB_OUTPUT"
          echo "new=$new" >> "$GITHUB_OUTPUT"
      - name: 既に PR があるか確かめる
        id: exists
        if: steps.sync.outputs.old != steps.sync.outputs.new
        env:
          GH_TOKEN: ${{ github.token }}
        run: |
          n="$(gh pr list --head "upstream-sync/${{ steps.sync.outputs.new }}" --state open --json number --jq length)"
          echo "count=$n" >> "$GITHUB_OUTPUT"
      - name: テストして PR を作る
        if: steps.sync.outputs.old != steps.sync.outputs.new && steps.exists.outputs.count == '0'
        env:
          GH_TOKEN: ${{ github.token }}
          OLD: ${{ steps.sync.outputs.old }}
          NEW: ${{ steps.sync.outputs.new }}
        run: |
          set +e
          scripts/gen-compat.sh > /tmp/gen.log 2>&1
          go test ./... > /tmp/test.log 2>&1
          test_status=$?
          go test ./internal/skill/ -update -run TestGolden > /dev/null 2>&1
          set -e
          branch="upstream-sync/$NEW"
          git config user.name "github-actions[bot]"
          git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
          git switch -c "$branch"
          git add -A
          git commit -m "chore: 本家 nanaism/yomiyasu を ${NEW:0:7} に更新する"
          git push origin "$branch"
          {
            echo "本家 nanaism/yomiyasu を \`${OLD:0:7}\` から \`${NEW:0:7}\` に更新する。"
            echo
            echo "- 本家の差分: https://github.com/nanaism/yomiyasu/compare/$OLD...$NEW"
            if [ "$test_status" = 0 ]; then
              echo "- テスト: 通った"
            else
              echo "- テスト: 落ちた（互換テストの失敗は移植のずれ、TestPhrases や Validate の失敗は変換規則の目印の消失）"
            fi
            echo
            echo "<details><summary>go test の出力（末尾 200 行）</summary>"
            echo
            echo '```'
            tail -n 200 /tmp/test.log
            echo '```'
            echo "</details>"
            echo
            echo "golden（生成した SKILL.md）の差分は、この PR の testdata/golden/ で確かめる。"
          } > /tmp/body.md
          gh pr create --title "本家を ${NEW:0:7} に更新する" --body-file /tmp/body.md --base main --head "$branch"
```

- [ ] **Step 2: lint で確かめる**

実行: `mise use -g actionlint`（入っていなければ）
実行: `actionlint .github/workflows/upstream-sync.yml`　期待: 指摘なし

- [ ] **Step 3: コミットする**

```
git add .github/workflows/upstream-sync.yml
git commit -m "ci: 本家の更新を毎日取り込んで PR を作る"
```

GitHub の設定で「Allow GitHub Actions to create and approve pull requests」を有効にする必要がある。これはリポジトリの持ち主が手で行うので、完了報告で伝える。

---

### Task 12: リリースと README

**Files:**
- Create: `.goreleaser.yaml`, `.github/workflows/release.yml`, `README.md`

- [ ] **Step 1: GoReleaser の設定を書く**

`.goreleaser.yaml`:

```yaml
version: 2
builds:
  - main: ./cmd/yomiyasu-go
    binary: yomiyasu-go
    env:
      - CGO_ENABLED=0
    goos: [darwin, linux, windows]
    goarch: [amd64, arm64]
    ldflags:
      - -s -w -X github.com/sudame/yomiyasu-go/internal/cli.Version={{.Version}}
archives:
  - name_template: "{{ .ProjectName }}_{{ .Os }}_{{ .Arch }}"
    formats: [tar.gz]
    format_overrides:
      - goos: windows
        formats: [zip]
    files:
      - LICENSE
      - README.md
checksum:
  name_template: checksums.txt
```

実行: `goreleaser check`　期待: 指摘なし
実行: `goreleaser build --snapshot --clean --single-target`　期待: `dist/` にバイナリができる

- [ ] **Step 2: リリースのワークフローを書く**

`.github/workflows/release.yml`:

```yaml
name: release
on:
  push:
    tags: ["v*"]
permissions:
  contents: write
jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
        with:
          fetch-depth: 0
      - uses: jdx/mise-action@v3
      - run: go test ./...
      - run: goreleaser release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

- [ ] **Step 3: README を書く**

`README.md` には次のことを書く。日本語の部分は yomiyasu スキルで推敲する。

- nanaism/yomiyasu を Go に移植した非公式版であること。本家へのリンクと、本家が MIT License であること。
- 本家との違い: 規則を設定で無効にできること、設定に合わせて SKILL.md を生成すること、単一バイナリであること。
- 入れ方: `mise use -g github:sudame/yomiyasu-go`。リポジトリが private の間は GitHub トークンが要ること。
- 使い方: `config init` → 設定を編集 → `skill install` の流れ。設定を変えたら `skill install` を実行し直すこと。
- 規則 ID の一覧（`rules.All` の順）と、無効にしたときに止まる書き直し。
- 本家への追従の仕組み（upstream-sync と互換テスト）。
- 乗り換えるときは、CLAUDE.md の「yomiyasu スキルを使う」を `yomiyasu-go` に書き換え、本家やフォークのスキルを外すこと。

- [ ] **Step 4: diff の使い方の文言を決める**

Task 6 で、diff の使い方の文言は互換テストのため本家と同じ `python3 yomiyasu_diff.py …` にした。このままだと `yomiyasu-go diff` の利用者には分かりにくい。次のどちらかを選び、README に書く。

- 本家と同じ文言のままにする（互換テストをそのまま通す）。
- `yomiyasu-go diff …` に変え、互換テストの使い方のケースだけ標準出力の比較から外す（`cases.tsv` の該当ケースを終了コードだけの比較にする）。

迷ったら前者にする。

- [ ] **Step 5: コミットする**

```
git add .goreleaser.yaml .github/workflows/release.yml README.md
git commit -m "feat: GoReleaser でのリリースと README を足す"
```

---

## 完了の確認

すべての Task が終わったら、次を順に確かめる。

1. `golangci-lint run` と `go test ./...` が通る。
2. `scripts/gen-compat.sh` を実行し直しても `git diff --exit-code testdata/compat/expected` が差分なしで終わる。
3. `go run ./cmd/yomiyasu-go skill install` で `~/.claude/skills/yomiyasu-go/` に書き出し、Claude Code で yomiyasu-go スキルを使う。Step 3 と Step 4 で `yomiyasu-go lint` と `yomiyasu-go diff` が呼ばれることを確かめる。ここは利用者に依頼する。
4. GitHub の設定で Actions による PR の作成を許可してもらったあと、`upstream-sync` を手動で実行し、本家が更新されていれば PR が作られることを確かめる。
