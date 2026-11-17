package ui

import "testing"

// Span.Chip sets the box of a span's rounded background: ChipBox.X past
// its glyphs on each side, from Above over the baseline to Below under
// it, rather than hugging the glyphs.
func TestSpanChip(t *testing.T) {
	red := RGB(255, 0, 0)
	left := func(chip ChipBox) bool {
		tt := NewTester(func(c *Context) {
			Box(c).Padding(20).Background(RGB(255, 255, 255)).Children(func() {
				RichText(c, Span{Text: "abc", Size: 14, Background: red, Radius: 2, Chip: chip})
			})
		}, 200, 100)
		r, ok := tt.Find("abc")
		if !ok {
			t.Fatal("no text")
		}
		c := tt.Image().RGBAAt(int(r.X)-5, int(r.Y+r.H/2))
		return c.R > 200 && c.G < 60 && c.B < 60
	}
	if left(ChipBox{}) {
		t.Error("the background reaches 5 DIPs left of the glyphs without a chip")
	}
	if !left(ChipBox{X: 7, Above: 12, Below: 6}) {
		t.Error("the chip does not reach 5 DIPs left of the glyphs")
	}
}
