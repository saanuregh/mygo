package editor

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// A press on a mark with OnClick runs it and not OnGutterClick; a press
// beside it, or on a mark without one, stays OnGutterClick's, and a press
// on a change bar stays OnChangeBarClick's.
func TestGutterMarkClickRunsBeforeGutterClick(t *testing.T) {
	var order []string
	st := &State{Buffer: NewBuffer("one\nTWO\nthree\nfour"), DiffBase: NewBuffer("one\ntwo\nthree\nfour"), DiffPlain: true,
		Font:          ui.Font{Family: "monospace", Size: 14},
		OnGutterClick: func(line int) { order = append(order, "gutter") },
		OnChangeBarClick: func(DiffHunk) bool {
			order = append(order, "bar")
			return true
		},
		GutterMarks: []GutterMark{
			{Line: 1, Color: ui.RGB(0, 200, 0), Shape: GutterPlay, Size: 10, Tip: "Run", OnClick: func() { order = append(order, "mark") }},
			{Line: 2, Color: ui.RGB(200, 0, 0), Size: 10},
		}}
	tt := ui.NewTester(func(c *ui.Context) { View(c, st).Fill() }, 400, 200)
	tt.Frame()
	lh := st.ed.view.lineHeight
	markX := float32(3 + 5)
	tt.ClickAt(markX, lh*1.5)
	tt.Frame()
	if len(order) != 1 || order[0] != "mark" {
		t.Fatalf("a press on the mark: %v", order)
	}
	// The pointer over it is a hand.
	tt.Move(markX, lh*1.5)
	tt.Frame()
	if c := tt.Cursor(); c != ui.CursorPointer {
		t.Errorf("over a clickable mark the cursor is %v", c)
	}
	// Resting there shows its Tip, and a press still runs it.
	tt.Advance(time.Second)
	tt.Frame()
	if !tt.HasText("Run") {
		t.Errorf("no tip over the mark: %v", tt.Texts())
	}
	tt.ClickAt(markX, lh*1.5)
	tt.Frame()
	if len(order) != 2 || order[1] != "mark" {
		t.Fatalf("a press on the mark with its tip: %v", order)
	}
	order = order[:1]
	// Below the mark's box, on the same line's number, the gutter's.
	tt.ClickAt(st.ed.gutterW-12, lh*1.5)
	tt.Frame()
	// A mark without OnClick is the gutter's.
	tt.ClickAt(markX, lh*2.5)
	tt.Frame()
	// The change bar on the mark's line is the bar's.
	tt.ClickAt(st.ed.changeBarX()+1.5, lh*1.5)
	tt.Frame()
	want := []string{"mark", "gutter", "gutter", "bar"}
	if len(order) != len(want) {
		t.Fatalf("presses: %v, want %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("presses: %v, want %v", order, want)
		}
	}
}

// GutterPlay draws a filled triangle pointing right: its middle and the
// left of its middle row are the mark's color, its top right corner is not.
func TestGutterPlayDraws(t *testing.T) {
	green := ui.RGB(0, 200, 0)
	st := &State{Buffer: NewBuffer("one\ntwo"), Font: ui.Font{Family: "monospace", Size: 14},
		GutterMarks: []GutterMark{{Line: 0, Color: green, Shape: GutterPlay, Size: 10}}}
	tt := ui.NewTester(func(c *ui.Context) { View(c, st).Fill() }, 400, 200)
	tt.Frame()
	lh := st.ed.view.lineHeight
	img := tt.Image()
	isGreen := func(x, y float32) bool {
		px := img.RGBAAt(int(x), int(y))
		return px.G > 150 && px.R < 80 && px.B < 80
	}
	top := (lh - 10) / 2
	if !isGreen(3+4, lh/2) || !isGreen(3+2, lh/2) {
		t.Errorf("the triangle's middle is not drawn")
	}
	if isGreen(3+9, top+1) {
		t.Errorf("the triangle's top right corner is filled: a square, not a triangle")
	}
}
