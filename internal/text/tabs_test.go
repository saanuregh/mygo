package text

import "testing"

// TestEllipsisKeepsTheSpaceBeforeIt cuts "with words" after "with ", the
// space kept before the ellipsis, as nanoui's truncation.
func TestEllipsisKeepsTheSpaceBeforeIt(t *testing.T) {
	s := newSystem()
	style := Style{Size: 12, Family: "monospace"}
	full := s.Layout(Params{Text: "withx", Style: style}) // as wide as "with "
	mark := s.Layout(Params{Text: "…", Style: style})
	l := s.Layout(Params{Text: "with words", Style: style, Width: full.Width + mark.Width + 0.5, MaxLines: 1})
	if !l.Truncated || len(l.Lines) != 1 || l.Lines[0].End != 5 {
		t.Fatalf("truncated %v, lines %+v", l.Truncated, l.Lines)
	}
}
