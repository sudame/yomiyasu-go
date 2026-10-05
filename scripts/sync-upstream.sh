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
