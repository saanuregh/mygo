package ui

import "testing"

// FitLines: a box sized by its content within its MaxWidth is as wide as
// the widest line its text wraps to there, a paragraph filling its width
// inside it too; without FitLines it takes the whole MaxWidth.
func TestFitLinesHugsTheWidestWrappedLine(t *testing.T) {
	words := "alpha beta gamma delta epsilon zeta eta theta iota kappa lambda mu nu xi omicron"
	widths := map[bool]float32{}
	var line float32
	for _, fit := range []bool{false, true} {
		tt := NewTester(func(c *Context) {
			Row(c).AlignItems(Start).Children(func() {
				col := Column(c).MaxWidth(300).Padding(0, 12).Label("Card")
				if fit {
					col.FitLines()
				}
				col.Children(func() {
					Column(c).FillWidth().Children(func() {
						Text(c, words).Label("Words")
					})
				})
			})
		}, 800, 400)
		tt.Frame()
		tt.Frame()
		card, _ := tt.Find("Card")
		text, _ := tt.Find("Words")
		widths[fit] = card.W
		if fit {
			line = text.W
		}
		if text.H < 30 {
			t.Fatalf("fit %v: the words did not wrap: %v", fit, text)
		}
		tt.rt.close()
	}
	if widths[false] != 300 || widths[true] >= 300 || widths[true] < 200 {
		t.Fatalf("widths %v", widths)
	}
	if widths[true] != line+24 {
		t.Fatalf("fitted %v, its text %v and padding", widths[true], line)
	}
}
