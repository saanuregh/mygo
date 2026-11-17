//go:build linux && (amd64 || arm64)

package text

import (
	"math"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// TestFractional measures text as nanoui does with SetFractional: a line
// is the font's ascent, descent and line gap rounded up to whole DIPs,
// the baseline the ascent and half the gap from its top,
// advances keep their fractions, and SetFractional(false) is Pango's
// hinted text again.
func TestFractional(t *testing.T) {
	s := newSystem()
	if err := s.RegisterFont(gomono.TTF, "MyGo Test Fractional"); err != nil {
		t.Fatal(err)
	}
	ttf, err := sfnt.Parse(gomono.TTF)
	if err != nil {
		t.Fatal(err)
	}
	style := Style{Family: "MyGo Test Fractional", Size: 13}
	hinted := s.Layout(Params{Text: "0123456789", Style: style})
	s.SetFractional(true)
	for _, size := range []float32{11.5, 13, 14} {
		style.Size = size
		m, err := ttf.Metrics(nil, fixed.Int26_6(size*64), font.HintingNone)
		if err != nil {
			t.Fatal(err)
		}
		ascent, descent := float64(m.Ascent)/64, float64(m.Descent)/64
		wantHeight := float32(math.Ceil(float64(m.Height) / 64))
		var b sfnt.Buffer
		g, _ := ttf.GlyphIndex(&b, '0')
		adv, _ := ttf.GlyphAdvance(&b, g, fixed.Int26_6(size*64), font.HintingNone)
		l := s.Layout(Params{Text: "0123456789", Style: style})
		line := l.Lines[0]
		if line.Height != wantHeight {
			t.Errorf("size %v: line height %v, want %v (ceil of %v)", size, line.Height, wantHeight, float64(m.Height)/64)
		}
		if math.Abs(float64(line.Ascent)-ascent) > 0.02 || math.Abs(float64(line.Descent)-descent) > 0.02 {
			t.Errorf("size %v: ascent %v descent %v, want the font's %v %v", size, line.Ascent, line.Descent, ascent, descent)
		}
		if want := float32(adv) / 64 * 10; math.Abs(float64(l.Width-want)) > 0.05 {
			t.Errorf("size %v: ten digits %v wide, want %v", size, l.Width, want)
		}
		// The ascent from the top, after half the gap; the rounding's room
		// goes below.
		if gap := float64(m.Height)/64 - ascent - descent; math.Abs(float64(line.Baseline)-gap/2-ascent) > 0.02 {
			t.Errorf("size %v: baseline %v, want %v", size, line.Baseline, gap/2+ascent)
		}
	}
	if q, n, _ := s.place(s.font(style), 2, false, 10.6); n != 4 || q != 2 {
		t.Errorf("a pen at 10.6 pixels draws at quarter %d of %d, want 2 of 4", q, n)
	}
	s.SetFractional(false)
	style.Size = 13
	if again := s.Layout(Params{Text: "0123456789", Style: style}); again.Width != hinted.Width || again.Lines[0].Height != hinted.Lines[0].Height {
		t.Errorf("off again: %v by %v, want %v by %v", again.Width, again.Lines[0].Height, hinted.Width, hinted.Lines[0].Height)
	}
	if hinted.Width != float32(math.Round(float64(hinted.Width))) {
		t.Errorf("hinted ten digits %v wide, want whole DIPs", hinted.Width)
	}
}
