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
