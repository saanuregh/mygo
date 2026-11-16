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

// TestTrimLineStarts leaves out the spaces starting a line after the
// first, as nanoui's wrapped text, and keeps the first line's.
func TestTrimLineStarts(t *testing.T) {
	s := newSystem()
	style := Style{Size: 12, Family: "monospace"}
	l := s.Layout(Params{Text: "  a\n    b", Style: style, Width: 400, TrimLineStarts: true})
	if len(l.Lines) != 2 || l.Lines[0].Start != 0 || l.Lines[1].Start != 8 {
		t.Fatalf("lines %+v", l.Lines)
	}
	if g := l.Lines[1].Glyphs; len(g) != 1 || g[0].X != 0 {
		t.Fatalf("the second line's glyphs %+v", g)
	}
}
