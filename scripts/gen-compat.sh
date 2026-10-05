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

# 標準入力から読むケース（Python は標準入力の改行を変換しない）
for f in testdata/compat/inputs/local/crlf.md testdata/compat/inputs/local/cr.md testdata/compat/inputs/local/astral.md; do
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
