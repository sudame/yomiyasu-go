# yomiyasu-go 設計

## 目的

[nanaism/yomiyasu](https://github.com/nanaism/yomiyasu)（MIT）を Go に移植する。本家に対する不満は次の 2 点である。

- Linter の規則を設定で変えられない。
- 検査スクリプトが Python なので、手元の Python のバージョンによって動作が変わることがある。

そこで、移植版では次の 2 点を目玉にする。

1. Linter の規則を設定ファイルで有効・無効にでき、その設定に合わせて SKILL.md が生成し直される。
2. Go の単一バイナリで配布し、実行環境による差をなくす。

本家の今後の変更には継続して追従する。

## 範囲

対象に含めるものは次のとおり。

- `yomiyasu_lint.py` と `yomiyasu_diff.py` の Go への移植（引数と JSON 出力は本家と同じにする）
- 規則ごとの有効・無効の設定
- 設定を反映した SKILL.md と references の生成と、`~/.claude/skills/yomiyasu-go/` への書き出し
- 本家の更新を検出して取り込む仕組みと、本家との互換テスト
- GitHub Releases によるバイナリ配布

対象に含めないものは次のとおり。

- 閾値・重大度・語彙リストの設定。設定ファイルの形は、これらを後から足せるようにしておく。
- 外部の規則を読み込むプラグイン機構。textlint と役割が重なり、互換テストの範囲もはっきりしなくなるため。
- プロジェクト単位の設定。Claude Code は同名のスキルが個人とプロジェクトの両方にあると個人のほうを使うため、プロジェクト単位で SKILL.md を出し分けられない。
- Claude Code プラグインとしての配布。

## 本家から引き継ぐもの・引き継がないもの

本家は MIT License（Copyright (c) 2026 nanaism）である。条件は著作権表示と許諾文を残すことだけなので、本リポジトリの `LICENSE` に本家の表示を併記する。`skill install` で書き出すスキルにも同じ `LICENSE` を置く。

| 本家のファイル | 扱い |
| --- | --- |
| `skills/yomiyasu/SKILL.md`、`references/` | `upstream/` に無改変で置き、生成の元にする |
| `skills/yomiyasu/scripts/*.py` | `upstream/` に置き、移植の元と互換テストの基準にする |
| `tests/fixtures/bold_regressions.json` | 互換テストの入力に使う |
| `tests/corpus/raw_ai/`、`yomiyasu_rewritten/` | 互換テストの入力に使う（作者が LLM で生成した文書） |
| `tests/corpus/human/` | 使わない（作者の Zenn 記事の抜粋を含み、MIT の範囲かがはっきりしない） |
| `evals/comparison_benchmark.md` | 使わない（デジタル庁の資料を含み、別に出典表示が要る） |
| `assets/` のロゴと画像 | 使わない（所属企業のロゴを含む） |

README には、本家を Go に移植した非公式版であることを明記する。

## 構成

```
yomiyasu-go/
├── upstream/                 # 本家のファイルを無改変で置く
│   └── UPSTREAM_COMMIT       # 取り込んだ本家のコミット SHA
├── cmd/yomiyasu-go/          # CLI の入口
├── internal/
│   ├── config/               # 設定の読み込みと既定値
│   ├── rules/                # 規則 ID の一覧
│   ├── lint/                 # yomiyasu_lint.py の移植
│   ├── diff/                 # yomiyasu_diff.py の移植
│   └── skill/                # SKILL.md の生成と変換規則
├── testdata/compat/          # 互換テストの入力
├── LICENSE
├── mise.toml
├── .golangci.yml
├── .goreleaser.yaml
└── .github/workflows/        # ci.yml、upstream-sync.yml、release.yml
```

`upstream/` の SKILL.md と references は `go:embed` でバイナリに埋め込む。

## CLI

| コマンド | 動作 |
| --- | --- |
| `yomiyasu-go lint [file] [--json] [--strict]` | 本家の `yomiyasu_lint.py` と同じ。無効にした規則は検出しない |
| `yomiyasu-go diff <元> <書き直し> [--stance=勧め\|決まり\|説明]` | 本家の `yomiyasu_diff.py` と同じ |
| `yomiyasu-go skill install [--dir <path>]` | SKILL.md、references、LICENSE を書き出す。既定の書き出し先は `~/.claude/skills/yomiyasu-go/` |
| `yomiyasu-go skill render` | `skill install` で書き出す SKILL.md を標準出力に表示する |
| `yomiyasu-go config init` | すべての規則を有効にした設定の雛形を書き出す |
| `yomiyasu-go version` | 自身のバージョンと、取り込んだ本家のコミットを表示する |

設定を変えたら、`skill install` を実行し直して SKILL.md に反映する。

## 設定

置き場所は `~/.config/yomiyasu-go/config.toml` だけとする。ファイルがなければ、すべての規則を有効とみなす。

```toml
[rules.trailing_colon]
enabled = false
```

無効にした規則の指摘は出力にも含めず、スコアの減点にも数えない。`metrics`（文字数や太字の数など）は規則の設定にかかわらず出力する。

規則ごとに表を分けておき、閾値や重大度を足すときは同じ表にキーを加える。設定に知らない規則 ID があれば、打ち間違いに気づけるようにエラーにする。

## SKILL.md の生成

### 規則を無効にしたときの意味

規則を無効にすると、Linter で検出しなくなるだけでなく、書き直しの指示からも外れる。たとえば `trailing_colon` を無効にすると、SKILL.md の「文末コロンを排除する」という指示も消える。Linter が許す書き方をスキルが消しにいく、という食い違いを起こさないためである。

### 変換規則

SKILL.md では、Linter の規則への言及が手順の文に入り組んでいる（例: 「絵文字、文末コロン、ダッシュ記号、…を排除します」）。そのため変換は文字列の完全一致による置換を基本にし、`internal/skill/patches.toml` に次の形で書く。

```toml
[[patch]]
when_disabled = "trailing_colon"     # 省略すると常に当てる
file = "SKILL.md"                    # upstream/ からの相対パス
find = "絵文字、文末コロン、ダッシュ記号"
replace = "絵文字、ダッシュ記号"
count = 1                            # 一致数がこれと違えば生成エラーにする
```

常に当てる変換は次の 4 つである。

- frontmatter の `name` と冒頭の見出し `# yomiyasu` を `yomiyasu-go` に置き換える（「スキル名と書き出し先」を参照）。

- `python3 <スキル配置ディレクトリ>/scripts/yomiyasu_lint.py` を `yomiyasu-go lint` に置き換える。diff も同じように置き換える。
- 「Pythonが実行できない環境では」で始まる代替手順を削る。
- 冒頭に、生成物であることと取り込んだ本家のコミットを書いた HTML コメントを入れる。

規則ごとの変換は、その規則を無効にしたときだけ当てる。対象は SKILL.md と references の両方である。変換規則を持たない規則があってもよく、その場合は Linter にだけ効く。

規則ごとにどの書き直しを止めるかは、フォーク `sudame/yomiyasu` の `feature-lint-config` ブランチで SKILL.md に足した対応表（「設定ファイルで無効にしたルール」の節）を元に決める。

`find` の一致数が `count` と違うときは生成を失敗させる。本家が文面を変えて目印が消えたことは、これで必ず検出できる。

### スキル名と書き出し先

スキル名は `yomiyasu-go` とし、本家の `yomiyasu` と区別する。本家と同じ名前を名乗らないためである。常に当てる変換で、frontmatter の `name` と冒頭の見出しを `yomiyasu-go` に置き換える。書き出し先も `~/.claude/skills/yomiyasu-go/` になるので、本家のスキルやフォーク `sudame/yomiyasu` のスキル（`~/.claude/skills/yomiyasu`）とは別の場所に置かれる。乗り換えるときは、CLAUDE.md の「yomiyasu スキルを使う」という指示を `yomiyasu-go` に書き換え、本家やフォークのスキルを外す。

`skill install` は、書き出し先に yomiyasu-go が書いた印（`.yomiyasu-go` ファイル）がなければ、何も書かずにエラーで止める。書き出し先がシンボリックリンクの場合も同じように止める。利用者が置いた別のファイルを上書きしないためである。

## 移植

- 正規表現は `github.com/dlclark/regexp2` を使う。本家は後読み、否定先読み、後方参照を多用しており、Go 標準の `regexp` では書けない。パターン文字列は本家から書き換えずに使い、本家が正規表現を変えても差し替えるだけで追従できるようにする。
- diff が類似度の計算に使っている `difflib.SequenceMatcher` は、本家が使う `autojunk=False` の動作だけを Go で実装する。
- 指摘文（日本語のメッセージ）も本家と一字一句同じにする。

## テスト

- **互換テスト**: CI で、mise で固定した Python を使って `upstream/scripts/*.py` を実行し、Go 版と同じ入力に対する `--json` 出力が完全に一致することを確かめる。対象は、すべての規則を有効にした状態だけとする。
- **規則の無効化**: 規則ごとに、無効にすると検出されなくなることを Go の単体テストで確かめる。
- **生成の golden テスト**: すべての規則を有効にした場合と、すべての規則を無効にした場合の生成結果を `testdata/golden/` にコミットし、一致を確かめる。本家を取り込む PR では、SKILL.md がどう変わるかを差分で確認できる。
- **変換規則の網羅**: 規則を 1 つずつ無効にして生成し、すべての変換規則が `count` のとおりに当たることを確かめる。

## 本家への追従

`upstream-sync.yml` を毎日 1 回実行する。

1. 本家の main の HEAD を `upstream/UPSTREAM_COMMIT` と比べる。同じなら終了する。
2. 本家の `skills/yomiyasu/`（SKILL.md、references、scripts）と互換テストの入力を `upstream/` と `testdata/compat/` にコピーし、`UPSTREAM_COMMIT` を更新する。
3. テストを実行し、結果に関係なく PR を作る。本文には、変わったファイル、落ちたテストの種類（互換のずれか、変換規則の目印が消えたか）、golden の差分を載せる。

テストが通る PR はそのままマージできる。落ちた PR は、手元で Claude に移植や変換規則の手直しを頼む。claude-code-action による自動修正は、API キーとコストが必要になるため入れない。

## 開発環境と配布

- Go、golangci-lint、互換テスト用の Python は `mise.toml` で固定する。
- `ci.yml` で golangci-lint、`go test`、互換テストを実行する。
- `release.yml` では、`v*` タグの push をきっかけに GoReleaser で macOS、Linux、Windows 向けのバイナリを GitHub Releases に置く。
- 利用者は `mise use -g github:sudame/yomiyasu-go` で入れる。リポジトリは private なので、mise から取得するには GitHub トークンが要る。

## 完了条件

- 互換テストがすべての入力で通る。
- すべての規則について、無効にしたときの Linter の動作と SKILL.md の変化がテストで確かめられている。
- `skill install` で書き出したスキルを Claude Code から使い、Step 3 と Step 4 で `yomiyasu-go` が呼ばれる。
- `upstream-sync.yml` を手動で実行し、PR が作られる。
