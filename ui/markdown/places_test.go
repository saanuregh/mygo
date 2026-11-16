package markdown

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestBlocksGiveTheirSourceLinesAndBoxes(t *testing.T) {
	src := "# Title\n\nA paragraph\nof two lines.\n\n```go\nx := 1\ny := 2\n```\n\n- one\n- two\n\nLast[^n].\n\n[^n]: A note.\n"
	var got []BlockPlace
	var view ui.Rect
	tt := ui.NewTester(func(c *ui.Context) {
		el := View(c, src, nil)
		got, view = Blocks(el), el.Bounds()
	}, 400, 600)
	type lines struct{ first, last int }
	want := []lines{{0, 0}, {2, 3}, {5, 8}, {10, 11}, {13, 13}}
	if len(got) != len(want) {
		t.Fatalf("got %d blocks, want %d: %+v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i].FirstLine != w.first || got[i].LastLine != w.last {
			t.Errorf("block %d spans lines %d to %d, want %d to %d", i, got[i].FirstLine, got[i].LastLine, w.first, w.last)
		}
	}
	// The boxes are the frame before's: the first frame has none yet.
	tt.Frame()
	for i, b := range got {
		if b.Bounds.H <= 0 {
			t.Fatalf("block %d has no box: %+v", i, b.Bounds)
		}
		if i > 0 && b.Bounds.Y <= got[i-1].Bounds.Y {
			t.Errorf("block %d is not below block %d: %g, %g", i, i-1, b.Bounds.Y, got[i-1].Bounds.Y)
		}
		if b.Bounds.Y < view.Y || b.Bounds.Y+b.Bounds.H > view.Y+view.H+0.5 {
			t.Errorf("block %d at %+v is outside the view at %+v", i, b.Bounds, view)
		}
	}
}

func TestBlocksOfAnotherElementAreNone(t *testing.T) {
	var got []BlockPlace
	ui.NewTester(func(c *ui.Context) {
		got = Blocks(ui.Column(c))
	}, 200, 100)
	if got != nil {
		t.Fatalf("got %+v", got)
	}
}
