package fakeserver

import (
	"math"
	"strconv"
	"strings"
)

// normalizeJSONNumbers rewrites the fractional and exponent numbers of a JSON
// document the way the Cumulocity platform returns them. The platform stores
// such numbers as doubles and prints them with Java's Double.toString, so
// {"max": 19.1010101E19} comes back as {"max": 1.91010101E20}. Integer
// literals, strings and the layout of the document are left untouched.
func normalizeJSONNumbers(doc []byte) []byte {
	var out []byte
	last := 0
	i := 0
	for i < len(doc) {
		c := doc[i]
		switch {
		case c == '"':
			i++
			for i < len(doc) && doc[i] != '"' {
				if doc[i] == '\\' {
					i++
				}
				i++
			}
			i++
		case c == '-' || (c >= '0' && c <= '9'):
			start := i
			i++
			isFloat := false
			for i < len(doc) {
				d := doc[i]
				if d == '.' || d == 'e' || d == 'E' {
					isFloat = true
				} else if !(d >= '0' && d <= '9') && d != '+' && d != '-' {
					break
				}
				i++
			}
			if !isFloat {
				continue
			}
			literal := string(doc[start:i])
			f, err := strconv.ParseFloat(literal, 64)
			if err != nil || math.IsInf(f, 0) || math.IsNaN(f) {
				continue
			}
			norm := javaDoubleString(f)
			if norm == literal {
				continue
			}
			out = append(out, doc[last:start]...)
			out = append(out, norm...)
			last = i
		default:
			i++
		}
	}
	if out == nil {
		return doc
	}
	return append(out, doc[last:]...)
}

// javaDoubleString formats a float64 like Java's Double.toString: plain
// decimal notation (always with a fractional digit) for 1e-3 <= |f| < 1e7 and
// computerized scientific notation (d.dddEn) otherwise.
func javaDoubleString(f float64) string {
	if f == 0 {
		if math.Signbit(f) {
			return "-0.0"
		}
		return "0.0"
	}
	abs := math.Abs(f)
	if abs >= 1e-3 && abs < 1e7 {
		s := strconv.FormatFloat(f, 'f', -1, 64)
		if !strings.Contains(s, ".") {
			s += ".0"
		}
		return s
	}
	s := strconv.FormatFloat(f, 'e', -1, 64) // e.g. 1.91010101e+20
	mant, exp, _ := strings.Cut(s, "e")
	if !strings.Contains(mant, ".") {
		mant += ".0"
	}
	n, _ := strconv.Atoi(exp)
	return mant + "E" + strconv.Itoa(n)
}
