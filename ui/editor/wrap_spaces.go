package editor

import (
	"math"
	"unicode/utf8"
)

// wrapBreaksAtSpaces is wrapBreaks for State.WrapAtSpaces, as editors
// that count columns wrap: a character takes a column, a wide one (as
// CJK, or an emoji with its joiners and modifiers) two, a mark none; a
// row breaks after the last space or tab that fits, else before the
// character that does not, never inside a character.
func (v *view) wrapBreaksAtSpaces(s string) []int {
	cw := v.charWidth
	if cw <= 0 {
		return nil
	}
	cols := int(math.Floor(float64(v.wrapWidth/cw) + 0.01))
	var breaks []int
	start, col := 0, 0
	last := -1 // after the last space or tab in the row
	for i := 0; i < len(s); {
		n, w := v.cluster(s, i)
		if col+w > cols && i > start {
			at := i
			if last > start {
				at = last
			}
			breaks = append(breaks, at)
			start, last = at, -1
			col = 0
			for j := at; j < i; {
				m, cw := v.cluster(s, j)
				col += cw
				j = m
			}
			continue
		}
		col += w
		if s[i] == ' ' || s[i] == '\t' {
			last = n
		}
		i = n
	}
	return breaks
}

// cluster returns the end of the character at i of s, with its marks and
// what joiners join to it, and the columns it takes.
func (v *view) cluster(s string, i int) (end, cols int) {
	r, size := utf8.DecodeRuneInString(s[i:])
	end = i + size
	if r < 0x80 {
		if r < ' ' {
			return end, 0
		}
		return end, 1
	}
	if isRegional(r) && end < len(s) {
		if d, dsize := utf8.DecodeRuneInString(s[end:]); isRegional(d) {
			end += dsize
		}
	}
	wide := false
	for end < len(s) {
		c, size := utf8.DecodeRuneInString(s[end:])
		if c == 0x200d {
			end += size
			if end < len(s) {
				_, size = utf8.DecodeRuneInString(s[end:])
				end += size
			}
			continue
		}
		if !extends(c) {
			break
		}
		if c == 0xfe0f {
			wide = true
		}
		end += size
	}
	if extends(r) {
		return end, 0
	}
	// Measured against the editor's column: wider than one is two.
	if wide || v.widths.of(r) > v.charWidth*1.25 {
		return end, 2
	}
	return end, 1
}
