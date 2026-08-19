package reference

// whitespaceV1 is the frozen whitespace table of the V1 language. Engines must
// never delegate this definition to their own Unicode tables, whose versions
// may differ.
var whitespaceV1 = map[rune]struct{}{
	0x0009: {}, 0x000A: {}, 0x000B: {}, 0x000C: {}, 0x000D: {},
	0x0020: {}, 0x0085: {}, 0x00A0: {}, 0x1680: {},
	0x2000: {}, 0x2001: {}, 0x2002: {}, 0x2003: {}, 0x2004: {}, 0x2005: {},
	0x2006: {}, 0x2007: {}, 0x2008: {}, 0x2009: {}, 0x200A: {},
	0x2028: {}, 0x2029: {}, 0x202F: {}, 0x205F: {}, 0x3000: {}, 0xFEFF: {},
}

// IsWhitespaceV1 reports whether the code point belongs to the frozen table.
func IsWhitespaceV1(r rune) bool {
	_, ok := whitespaceV1[r]
	return ok
}

// WhitespaceV1 returns the frozen table as a sorted slice of code points.
func WhitespaceV1() []rune {
	out := make([]rune, 0, len(whitespaceV1))
	for r := range whitespaceV1 {
		out = append(out, r)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j-1] > out[j]; j-- {
			out[j-1], out[j] = out[j], out[j-1]
		}
	}
	return out
}

// upperASCII maps only a..z to A..Z, never consulting a locale.
func upperASCII(s string) string {
	b := []byte(s)
	changed := false
	for i := range b {
		if b[i] >= 'a' && b[i] <= 'z' {
			b[i] -= 'a' - 'A'
			changed = true
		}
	}
	if !changed {
		return s
	}
	return string(b)
}

// lowerASCII maps only A..Z to a..z, never consulting a locale.
func lowerASCII(s string) string {
	b := []byte(s)
	changed := false
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 'a' - 'A'
			changed = true
		}
	}
	if !changed {
		return s
	}
	return string(b)
}

// trimASCII removes only U+0009..U+000D and U+0020 at both ends.
func trimASCII(s string) string {
	start := 0
	for start < len(s) && isASCIISpace(s[start]) {
		start++
	}
	end := len(s)
	for end > start && isASCIISpace(s[end-1]) {
		end--
	}
	return s[start:end]
}

func isASCIISpace(c byte) bool {
	return c == 0x20 || (c >= 0x09 && c <= 0x0D)
}

func isASCIIDigit(r rune) bool { return r >= '0' && r <= '9' }

func isASCIIUpper(r rune) bool { return r >= 'A' && r <= 'Z' }

func isASCIIAlnum(r rune) bool { return isASCIIDigit(r) || isASCIIUpper(r) }
