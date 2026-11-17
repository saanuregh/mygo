package ui

import (
	"image/color"
	"testing"
)

// Theme.InputFocus draws a focused input's accent the width asked within
// its box, its text staying where it was, and a glow around it; its zero
// value keeps the one-DIP border.
func TestInputFocusRing(t *testing.T) {
	for _, width := range []float32{0, 2} {
		value := "words"
		tt := coreNewTester(func(c *context) {
			th := LightTheme()
			th.InputFocus = InputFocusStyle{Width: width, Glow: 6, Opacity: 0.25}
			c.SetTheme(th)
			coreColumn(c).Padding(40).Children(func() { coreTextInput(c, &value).Width(200).Label("Field") })
		}, 300, 140)
		r, ok := tt.Find("Field")
		if !ok {
			t.Fatal("no input")
		}
		unfocused := tt.Image().RGBAAt(int(r.X)+20, int(r.Y+r.H/2))
		if err := tt.Click("Field"); err != nil {
			t.Fatal(err)
		}
		tt.Frame()
		if after, _ := tt.Find("Field"); after != r {
			t.Errorf("width %v: the box moved from %+v to %+v", width, r, after)
		}
		// The text stays: the same pixel of its first letter.
		if got := tt.Image().RGBAAt(int(r.X)+20, int(r.Y+r.H/2)); got != unfocused {
			t.Errorf("width %v: the text changed under the ring: %v, was %v", width, got, unfocused)
		}
		img := tt.Image()
		accent := LightTheme().Accent
		y := int(r.Y + r.H/2)
		inner := img.RGBAAt(int(r.X)+1, y)
		isAccent := func(c color.RGBA) bool {
			return absDiff(c.R, accent.R) < 40 && absDiff(c.G, accent.G) < 40 && absDiff(c.B, accent.B) < 40
		}
		if got := isAccent(inner); got != (width == 2) {
			t.Errorf("width %v: the second pixel in is %v, accent %v", width, inner, got)
		}
		if glow := img.RGBAAt(int(r.X)-3, y); width == 2 && glow == (color.RGBA{255, 255, 255, 255}) {
			t.Errorf("width %v: no glow outside the box", width)
		}
	}
}
