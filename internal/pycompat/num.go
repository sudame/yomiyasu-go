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

// Int は Python の int を表す Num を返す。
func Int(i int) Num { return Num{F: float64(i), IsInt: true} }

// Float は Python の float を表す Num を返す。
func Float(f float64) Num { return Num{F: f} }

// String は Python の str() と同じ文字列を返す。
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
