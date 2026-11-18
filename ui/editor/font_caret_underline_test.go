package editor

import (
	"image"
	"testing"

	"github.com/egoist/mygo/ui"
)

// caretBox is the box around the pixels of color c in img.
func caretBox(img *image.RGBA, c ui.Color) image.Rectangle {
	var box image.Rectangle
	r, g, b := int(c.R), int(c.G), int(c.B)
	for y := img.Rect.Min.Y; y < img.Rect.Max.Y; y++ {
		for x := img.Rect.Min.X; x < img.Rect.Max.X; x++ {
			px := img.RGBAAt(x, y)
			if near(int(px.R)-r) < 30 && near(int(px.G)-g) < 30 && near(int(px.B)-b) < 30 {
				box = box.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	return box
}

// With FontCarets, CursorUnderline draws a line under the character after
// the caret (wider than tall), as it does without; a bar stays a bar.
func TestFontCaretsDrawAnUnderline(t *testing.T) {
	theme := *DefaultTheme(ui.LightTheme())
	theme.Cursor = ui.RGB(230, 0, 0)
	for _, shape := range []CursorShape{CursorUnderline, CursorBar} {
		st := &State{Buffer: NewBuffer("MMMM"), Font: ui.Font{Family: "monospace", Size: 14}, Theme: &theme,
			FontCarets: true, NoCaretBlink: true, CursorShape: shape}
		tt := ui.NewTester(func(c *ui.Context) { View(c, st).Fill() }, 300, 100)
		tt.Frame()
		tt.ClickAt(2, 5)
		st.SetSelections(Selection{})
		tt.Frame()
		box := caretBox(tt.Image(), theme.Cursor)
		if box.Empty() {
			t.Fatalf("%v: no caret drawn", shape)
		}
		wide := box.Dx() > box.Dy()
		if wide != (shape == CursorUnderline) {
			t.Errorf("%v: caret drawn as %v", shape, box)
		}
	}
}

func near(n int) int { return max(n, -n) }
