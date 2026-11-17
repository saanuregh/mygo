package editor

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// PreserveCase returns replacement in the case pattern of matched, as VS
// Code's preserve case gives it: when both have as many parts split by
// '-', or else by '_', each part takes its match's pattern; otherwise an
// all upper match gives the replacement upper, an all lower one lower, and
// one whose first letter is upper gives the replacement's first letter
// upper. Any other match, and one without letters, leaves it as typed.
func PreserveCase(matched, replacement string) string {
	for _, sep := range []string{"-", "_"} {
		if !strings.Contains(matched, sep) || !strings.Contains(replacement, sep) {
			continue
		}
		mp, rp := strings.Split(matched, sep), strings.Split(replacement, sep)
		if len(mp) != len(rp) {
			continue
		}
		for i := range rp {
			rp[i] = preserveCaseWhole(mp[i], rp[i])
		}
		return strings.Join(rp, sep)
	}
	return preserveCaseWhole(matched, replacement)
}

// preserveCaseWhole is PreserveCase without parts.
func preserveCaseWhole(matched, replacement string) string {
	upper, lower := strings.ToUpper(matched), strings.ToLower(matched)
	switch {
	case upper == lower || replacement == "":
		return replacement
	case matched == upper:
		return strings.ToUpper(replacement)
	case matched == lower:
		return strings.ToLower(replacement)
	}
	if first, _ := utf8.DecodeRuneInString(matched); unicode.IsUpper(first) {
		r, n := utf8.DecodeRuneInString(replacement)
		return string(unicode.ToUpper(r)) + replacement[n:]
	}
	return replacement
}
