package editor

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// F7-4: CaretCardShift places the caret card as at a caret that many
// DIPs right (zode's signature card at its caret.x - 40, that caret 6
// past the text's edge).
func TestCaretCardShift(t *testing.T) {
	for _, shift := range []float32{0, 6} {
		var card ui.Rect
		st := &State{Buffer: NewBuffer("hello world\nfoo(bar)\n"), CaretCardShift: shift}
		st.CaretCard = func(c *ui.Context, el ui.Element) {
			el.MarginEnd(40).Children(func() { ui.Text(c, "sig(a int)") })
			card = el.LayoutBounds()
		}
		st.Select(16, 16)
		tt := ui.NewTester(func(c *ui.Context) { View(c, st).Fill().AutoFocus() }, 600, 400)
		tt.Frame()
		tt.Frame()
		tt.Frame()
		cr, _ := st.CaretBox()
		if want := cr.X - 40 + shift; card.X < want-0.5 || card.X > want+0.5 {
			t.Errorf("shift %v: card at %v, want %v", shift, card.X, want)
		}
	}
}
