package markdown

import "unicode/utf8"

// Mark is a range of a view's source, the bytes [Start, End), that
// Options.Marks highlights; Current marks the one a find bar is on.
type Mark struct {
	Start, End int
	Current    bool
}

// marksWithin appends to out the marks that overlap [start, end).
func marksWithin(out, marks []Mark, start, end int) []Mark {
	for _, m := range marks {
		if m.Start < end && m.End > start && m.End > m.Start {
			out = append(out, m)
		}
	}
	return out
}

// sameMarks reports whether marked is the marks that overlap [start, end).
func sameMarks(marked, marks []Mark, start, end int) bool {
	n := 0
	for _, m := range marks {
		if m.Start < end && m.End > start && m.End > m.Start {
			if n >= len(marked) || marked[n] != m {
				return false
			}
			n++
		}
	}
	return n == len(marked)
}

// markAt returns the mark covering source byte i, or nil.
func (sb *spanBuilder) markAt(i int) *Mark {
	for k := range sb.marks {
		if m := &sb.marks[k]; m.Start <= i && i < m.End {
			return m
		}
	}
	return nil
}

// addMarked adds text cut where the marks over it start and end. Text
// that is not its source's bytes is marked whole when a mark touches it.
func (sb *spanBuilder) addMarked(text string, src, end int, exact bool, s spanStyle, link int, alpha float32) {
	if !exact || end-src != len(text) {
		var mark *Mark
		for k := range sb.marks {
			if m := &sb.marks[k]; m.Start < max(end, src+1) && m.End > src {
				mark = m
				break
			}
		}
		sb.addRun(text, src, end, exact, s, link, alpha, mark)
		return
	}
	for len(text) > 0 {
		mark := sb.markAt(src)
		n := 1
		for n < len(text) && (sb.markAt(src+n) == mark || !utf8.RuneStart(text[n])) {
			n++
		}
		sb.addRun(text[:n], src, src+n, exact, s, link, alpha, mark)
		text, src = text[n:], src+n
	}
}
