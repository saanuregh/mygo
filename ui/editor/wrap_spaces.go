package editor

import (
	"math"
	"unicode/utf8"
)

// wrapBreaksAtSpaces is wrapBreaks for State.WrapAtSpaces, as editors
// that count columns wrap: a character takes a column, a wide one (as
// CJK, or an emoji with its joiners and modifiers) two, a mark none; a
// row breaks after the last space or tab that fits, else before the
// character that does not, never inside a character. segs are s's
// segments (nil without tabs or inlays): an inlay goes with the
// character after it, its own spaces no place to break, and one at the
// line's end takes the last word down with it when it does not fit.
func (v *view) wrapBreaksAtSpaces(s string, segs []dispSeg) []int {
	cw := v.charWidth
	if cw <= 0 {
		return nil
	}
	cols := int(math.Floor(float64(v.wrapWidth/cw) + 0.01))
	// The units a row may break before: a character (a tab's spaces one)
	// with the inlays before it, from at to end, w columns.
	type unit struct {
		at, end, w int
		space      bool
	}
	units := make([]unit, 0, len(s))
	pending, hint := -1, 0
	si := 0
	segEnd := func(i int) int {
		if i+1 < len(segs) {
			return segs[i+1].disp
		}
		return len(s)
	}
	for i := 0; i < len(s); {
		for si+1 < len(segs) && segs[si+1].disp <= i {
			si++
		}
		kind := segText
		if si < len(segs) {
			kind = segs[si].kind
		}
		if kind == segInlay {
			end := segEnd(si)
			if pending < 0 {
				pending = i
			}
			for j := i; j < end; {
				n, w := v.cluster(s, j)
				hint += w
				j = n
			}
			i = end
			continue
		}
		at := i
		if pending >= 0 {
			at = pending
		}
		if kind == segTab {
			end := segEnd(si)
			units = append(units, unit{at: at, end: end, w: hint + end - i, space: true})
			i = end
		} else {
			n, w := v.cluster(s, i)
			units = append(units, unit{at: at, end: n, w: hint + w, space: s[i] == ' ' || s[i] == '\t'})
			i = n
		}
		pending, hint = -1, 0
	}
	var breaks []int
	start, col := 0, 0
	last := -1 // the unit after the last space or tab in the row
	for k := 0; k < len(units); {
		u := units[k]
		if col+u.w > cols && k > start {
			at := k
			if last > start {
				at = last
			}
			breaks = append(breaks, units[at].at)
			start, last = at, -1
			col = 0
			for j := at; j < k; j++ {
				col += units[j].w
			}
			continue
		}
		col += u.w
		if u.space {
			last = k + 1
		}
		k++
	}
	if hint > 0 && len(units) > start && col+hint > cols && last > start && last < len(units) {
		breaks = append(breaks, units[last].at)
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
