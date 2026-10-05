// Package yomiyasugo は、バイナリに埋め込む本家のファイルと LICENSE を持つ。
package yomiyasugo

import "embed"

// Upstream は、本家の SKILL.md と references、取り込んだコミット、LICENSE を持つ。
//
//go:embed upstream/SKILL.md upstream/references upstream/UPSTREAM_COMMIT LICENSE
var Upstream embed.FS
