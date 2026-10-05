# yomiyasu-go

[nanaism/yomiyasu](https://github.com/nanaism/yomiyasu) を Go に移植した非公式版です。本家は AI が生成した日本語を読みやすく整える Claude Code のスキルで、MIT License で公開されています。このリポジトリは本家の作者とは関係ありません。

## 本家との違い

- lint の規則を、設定ファイルで 1 つずつ無効にできます。
- 設定に合わせて SKILL.md と references を生成します。規則を無効にすると、lint で検出しなくなるだけでなく、スキルの書き直しの指示からも外れます。
- 検査スクリプトを Go の単一バイナリにしたので、手元の Python のバージョンによって動作が変わりません。

lint と diff の出力は、本家の Python スクリプトと同じです。設定ファイルがなければ、本家と同じ規則で検査します。

## 入れ方

```bash
mise use -g github:sudame/yomiyasu-go
```

リポジトリが private の間は、取得に GitHub のトークン（環境変数 `GITHUB_TOKEN` など）が要ります。

## 使い方

```bash
# すべての規則を有効にした設定の雛形を書き出す
yomiyasu-go config init

# 無効にしたい規則を enabled = false にする
$EDITOR ~/.config/yomiyasu-go/config.toml

# 設定を反映したスキルを ~/.claude/skills/yomiyasu-go/ に書き出す
yomiyasu-go skill install
```

設定を変えたら、`yomiyasu-go skill install` を実行し直して SKILL.md に反映します。設定ファイルの場所は `~/.config/yomiyasu-go/config.toml` です。環境変数 `XDG_CONFIG_HOME` に絶対パスを設定している場合は、`$XDG_CONFIG_HOME/yomiyasu-go/config.toml` を使います。

```toml
[rules.trailing_colon]
enabled = false
```

設定に知らない規則 ID や項目があると、打ち間違いに気づけるようにエラーで止まります。

スキルの書き出し先に yomiyasu-go が書いた印（`.yomiyasu-go`）がない場合や、書き出し先がシンボリックリンクの場合は、何も書かずに止まります。利用者が置いたファイルを上書きしないためです。`--dir` で書き出し先を変えられます。

ほかのコマンドは次のとおりです。

| コマンド | 動作 |
| --- | --- |
| `yomiyasu-go lint [file] [--json] [--strict]` | 本家の `yomiyasu_lint.py` と同じ検査をする。無効にした規則は検出しない |
| `yomiyasu-go diff <元の文> <書き直した文> [--stance=勧め\|決まり\|説明] [--json]` | 本家の `yomiyasu_diff.py` と同じ比較をする |
| `yomiyasu-go skill render` | `skill install` で書き出す SKILL.md を標準出力に表示する |
| `yomiyasu-go version` | 自身のバージョンと、取り込んだ本家のコミットを表示する |

`yomiyasu-go diff` を引数なしで実行したときの使い方の表示は、本家と出力をそろえるため `python3 yomiyasu_diff.py` のままにしています。

## 規則

| 規則 ID | 無効にしたときにしなくなる書き直し |
| --- | --- |
| `excess_bold` | 太字の数を減らすこと |
| `excess_list` | 箇条書きの比率を下げるために地の文へまとめること |
| `sentence_end_repetition` | 同一文末が続くことを理由にした文末の調整 |
| `bold_not_rendered` | 太字が表示される形への修正 |
| `emoji_prohibited` | 絵文字の削除 |
| `redundant_bracket` | 情報量の増えない補足カッコの削除 |
| `unnatural_halfwidth_space` | 和欧文間の半角空白の削除 |
| `trailing_colon` | 文末コロンの削除 |
| `slop_vocabulary` | 大げさな語や意味の曖昧な語の言い換え |
| `metaphor_verb` | 比喩動詞や、lint が検出する比喩の型の言い換え |
| `meta_filler` | 前置き、定型の結び、自己ラベリングの削除や書き換え |
| `negative_parallelism` | 否定対比（AではなくB）を肯定文にすること |

無効にした規則の指摘は lint の出力に含まれず、スコアの減点にも数えません。文字数や太字の数などの計測値は、設定にかかわらず出力します。

## 本家への追従

本家のファイルは `upstream/` に手を加えずに置き、`internal/skill/patches.toml` の変換規則を当てて SKILL.md を生成します。

`upstream-sync` ワークフローが毎日 1 回、本家の main を取り込みます。更新があれば、テストの結果を本文に載せた PR を作ります。テストは次の 3 つです。

- 互換テスト: 本家の Python で作った期待出力（`testdata/compat/expected/`）と、Go 版の出力が一致するかを確かめます。
- 変換規則の検査: 変換規則の目印が本家の文面から消えていないかを確かめます。
- golden: 生成した SKILL.md の変化を `testdata/golden/` の差分で確かめます。

## 乗り換え

本家やフォークの yomiyasu スキルから乗り換えるときは、CLAUDE.md などにある「yomiyasu スキルを使う」という指示を `yomiyasu-go` に書き換え、元のスキルを外します。スキル名が `yomiyasu-go` なので、元のスキルと同じ場所には書き出しません。

## 開発

Go、golangci-lint、互換テスト用の Python などは `mise.toml` で固定しています。

```bash
mise install
golangci-lint run
go test ./...

# 互換テストの期待出力を本家の Python で作り直す
scripts/gen-compat.sh

# 生成した SKILL.md の golden を書き直す
go test ./internal/skill/ -update -run TestGolden
```

## ライセンス

MIT License です。本家の著作権表示を `LICENSE` に併記しています。`skill install` で書き出すスキルにも同じ `LICENSE` を置きます。
