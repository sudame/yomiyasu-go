#!/usr/bin/env bash
# 本家の Python スクリプトと yomiyasu-go の実行時間を hyperfine で比べ、結果を Markdown の表で出す。
# 入力は本家のコーパス（testdata/compat/inputs/upstream/）から作る。
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

go build -o "$work/yomiyasu-go" ./cmd/yomiyasu-go
bin="$work/yomiyasu-go"
# 利用者の設定を読まないよう、空の設定ディレクトリを使う
export XDG_CONFIG_HOME="$work/config"
export PYTHONIOENCODING=utf-8
# mise の shim を通すと起動の分だけ Python が遅く出るので、実体のパスで呼ぶ
py="$(python3 -c 'import sys; print(sys.executable)')"
lint=upstream/scripts/yomiyasu_lint.py
dif=upstream/scripts/yomiyasu_diff.py
up=testdata/compat/inputs/upstream

small_o="$up/raw_ai/01_tech_arch_sonnet_default.md"
small_r="$up/yomiyasu_rewritten/01_tech_arch_sonnet_default.md"
# 大きい入力: 書き直し前のコーパス 24 本と、それぞれの書き直し後をつなげたもの
large_o="$work/large.orig.md"
large_r="$work/large.rewrite.md"
for f in $(find "$up/raw_ai" -name '*.md' | LC_ALL=C sort); do
  cat "$f" >> "$large_o"
  printf '\n' >> "$large_o"
  cat "$up/yomiyasu_rewritten/$(basename "$f")" >> "$large_r"
  printf '\n' >> "$large_r"
done

echo "環境: $(uname -sm)、Python $("$py" -c 'import platform; print(platform.python_version())')、$(go version | cut -d' ' -f3)"
echo "小さい入力: $(wc -m < "$small_o" | tr -d ' ') 文字、大きい入力: $(wc -m < "$large_o" | tr -d ' ') 文字"
echo

run() {
  local name="$1" pycmd="$2" gocmd="$3"
  # -N: シェルを介さずに起動し、数ミリ秒の差も測れるようにする
  hyperfine -N --warmup 3 --min-runs 20 --export-markdown "$work/$name.md" \
    --command-name "yomiyasu (Python)" "$pycmd" \
    --command-name "yomiyasu-go" "$gocmd" > /dev/null
  echo "### $name"
  echo
  cat "$work/$name.md"
  echo
}

run "lint（小さい入力）" "$py $lint $small_o --json" "$bin lint $small_o --json"
run "lint（大きい入力）" "$py $lint $large_o --json" "$bin lint $large_o --json"
run "diff（小さい入力）" "$py $dif $small_o $small_r --json" "$bin diff $small_o $small_r --json"
run "diff（大きい入力）" "$py $dif $large_o $large_r --json" "$bin diff $large_o $large_r --json"
