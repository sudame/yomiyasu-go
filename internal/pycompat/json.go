package pycompat

import (
	"fmt"
	"strconv"
	"strings"
)

// KV は Obj の 1 項目。
type KV struct {
	K string
	V any
}

// Obj はキーの順序を保った JSON オブジェクト。Python の dict の挿入順を再現する。
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
