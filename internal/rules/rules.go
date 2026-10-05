// Package rules は lint の規則 ID を持つ。順序は本家の lint が指摘を出す順に合わせる。
package rules

import "slices"

// All は規則 ID の一覧。
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

// Known は id が規則 ID の一覧にあるかを返す。
func Known(id string) bool { return slices.Contains(All, id) }
