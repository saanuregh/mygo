package text

import (
	"math"
	"testing"
)

// TestLeading makes a line a multiple of the font's own line, rounded up,
// as nanoui's leading, where LineHeight multiplies the size.
func TestLeading(t *testing.T) {
	s := newSystem()
	style := Style{Size: 12, Family: "monospace"}
	_, _, natural := s.Metrics(style)
	style.Leading = 1.35
	_, _, lh := s.Metrics(style)
	want := float32(math.Ceil(math.Ceil(float64(natural)) * 1.35))
	if lh != want {
		t.Errorf("a line of leading 1.35 is %v tall, want %v (font line %v)", lh, want, natural)
	}
	l := s.Layout(Params{Text: "one\ntwo", Style: style})
	if l.Height != 2*want {
		t.Errorf("two lines are %v tall, want %v", l.Height, 2*want)
	}
	style.LineHeight = 1.35
	if _, _, lh := s.Metrics(style); lh != 1.35*12 {
		t.Errorf("LineHeight beside Leading gives %v", lh)
	}
}

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
