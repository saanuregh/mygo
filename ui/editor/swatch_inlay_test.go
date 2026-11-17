package editor

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// A swatch inlay draws a square of its color before the text after it,
// takes two characters' room, and a press on the square reaches
// OnInlayClick with its index in SetInlays' list.
func TestSwatchInlayDrawsAndClicks(t *testing.T) {
	st := &State{Buffer: NewBuffer("a = #ff0000;\nb = 1\n")}
	var clicks []InlayClick
	st.OnInlayClick = func(c InlayClick) bool { clicks = append(clicks, c); return true }
	tt := ui.NewTester(func(c *ui.Context) { View(c, st).Fill() }, 600, 200)
	red := ui.RGB(255, 0, 0)
	st.SetInlays([]Inlay{{Offset: 0, Text: ": T"}, {Offset: 4, Swatch: red}})
	tt.Frame()
	ed := &st.ed
	cw := ed.view.charWidth
	y := float32(ed.view.heights.geometry(0).rows-ed.scrollY) + ed.view.lineHeight/2
	// ": T" "a = ": 7 characters, then the swatch's two.
	x := ed.textX + cw*(7+1)
	img := tt.Image()
	px := img.RGBAAt(int(x), int(y))
	if px.R < 200 || px.G > 60 || px.B > 60 {
		t.Fatalf("the swatch's middle is %v, want red", px)
	}
	if l := ed.view.line(0); !strings.Contains(l.disp, "a = \u00a0\u00a0#ff0000") {
		t.Fatalf("the swatch's room: %q", l.disp)
	}
	tt.ClickAt(x, y)
	if len(clicks) != 1 || clicks[0].Index != 1 {
		t.Fatalf("OnInlayClick told %+v, want index 1", clicks)
	}
	if c := st.Cursor(); c != 0 {
		t.Fatalf("a taken press moved the caret to %d", c)
	}
	// Past the swatch, on "#ff0000", the press is the text's.
	tt.ClickAt(ed.textX+cw*(9+3.5), y)
	if len(clicks) != 1 || st.Cursor() != 7 && st.Cursor() != 8 {
		t.Fatalf("a press on the text: told %+v, caret %d", clicks, st.Cursor())
	}
}
