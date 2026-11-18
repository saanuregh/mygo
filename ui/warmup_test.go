package ui

import "testing"

// A view warms up what it will show: the frame after asks for no layout
// of the text warmed, which the window's text system already has.
func TestWarmUpLaysOutWhatAViewWillShow(t *testing.T) {
	show, asked := false, false
	warm := func(c *Context) { Text(c, "a dialog's text, warmed before it opens").FontSize(15) }
	tt := NewTester(func(c *Context) {
		if !asked {
			asked = true
			c.WarmUp(warm)
		}
		if show {
			warm(c)
		}
	}, 400, 300)
	tt.Frame()
	made := tt.rt.text.LayoutsMade()
	show = true
	tt.Frame()
	if n := tt.rt.text.LayoutsMade() - made; n != 0 {
		t.Errorf("showing the warmed text made %d layouts", n)
	}
	if !tt.HasText("a dialog's text, warmed before it opens") {
		t.Errorf("texts %q", tt.Texts())
	}
}
