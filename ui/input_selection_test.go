package ui

import "testing"

// Theme.InputSelection.NoCaret: a one-line input with text selected
// shows no caret, as nanoui's text field.
func TestInputSelectionHidesTheCaret(t *testing.T) {
	for _, noCaret := range []bool{false, true} {
		text := "abc"
		sel := false
		tt := NewTester(func(c *Context) {
			th := LightTheme()
			th.Accent = RGB(255, 0, 0)
			th.InputSelection = InputSelectionStyle{Radius: 2, NoCaret: noCaret}
			c.SetTheme(th)
			Column(c).Padding(20).Children(func() {
				in := TextInput(c, &text).Label("Field").Width(120).AutoFocus()
				if sel {
					in.SetTextSelection(0, 3)
					sel = false
				}
			})
		}, 200, 100)
		tt.Frame()
		sel = true
		tt.Frame()
		tt.Frame()
		caret, ok := tt.TextCaret()
		if !ok {
			t.Fatal("no caret rect")
		}
		img := tt.Image()
		s := float32(img.Bounds().Dx()) / 200
		red := false
		for dx := -1; dx <= 2; dx++ {
			c := img.RGBAAt(int(caret.X*s)+dx, int((caret.Y+caret.H/2)*s))
			if c.R > 200 && c.G < 60 && c.B < 60 {
				red = true
			}
		}
		if red == noCaret {
			t.Errorf("NoCaret %v: caret drawn %v", noCaret, red)
		}
		tt.rt.close()
	}
}
