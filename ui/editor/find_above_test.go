package editor

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestCodeEditorFindBarAbove checks that a find bar placed above the text
// pushes the text down while it shows: the lines paint below it, a click
// below it lands on the line under the pointer, the hover at the
// text and the caret input methods are told of move with the text, and
// the text comes back up once it closes.
func TestCodeEditorFindBarAbove(t *testing.T) {
	ct := newEditorTest(t, "", "one\ntwo\nthree\nfour")
	st := ct.st
	const barH = 40
	st.FindBarAbove = true
	st.RenderFindBar = func(c *ui.Context, bar ui.Element, f *FindBar) {
		bar.Height(barH).Children(func() { ui.Text(c, "the bar") })
	}
	st.ShowHover(0, 3, "onerous", nil)
	ct.tt.Frame()
	textTop := func() float32 {
		r, _ := ct.tt.Find("onerous")
		return r.Y
	}
	caretTop := func() float32 {
		r, _ := ct.tt.TextCaret()
		return r.Y
	}
	before, caretBefore := textTop(), caretTop()
	lh := st.ed.view.lineHeight
	if !st.Run(Find) {
		t.Fatal("find did not open")
	}
	// Opening the bar selects the word, which may close a popup; shown
	// again, the hover is at its text.
	st.ShowHover(0, 3, "onerous", nil)
	ct.tt.Frame()
	ct.tt.Frame()
	bar, ok := ct.tt.Find("the bar")
	if !ok || bar.Y > 1 {
		t.Fatalf("bar %v, %v: want it at the top", bar, ok)
	}
	if st.ed.top != barH || st.ed.r.Y != barH || st.ViewportHeight() != 400-barH {
		t.Fatalf("top %v, text box %v, viewport %v: want the text below the bar", st.ed.top, st.ed.r, st.ViewportHeight())
	}
	// The hover at the first line's text moved down with the text, and
	// so did the caret.
	if got := textTop(); got != before+barH {
		t.Errorf("hover at %v, were %v: want them %v lower", got, before, barH)
	}
	if got := caretTop(); got != caretBefore+barH {
		t.Errorf("caret at %v, was %v: want it %v lower", got, caretBefore, barH)
	}
	// A click half a line below the bar is on the first line; one and a
	// half lines below it, on the second (the hover gone).
	st.ed.hover.close()
	ct.tt.Frame()
	ct.tt.ClickAt(st.ed.textX+st.ed.view.charWidth*2+1, barH+lh*1.5)
	ct.tt.Frame()
	if got := st.CursorPos(); got.Line != 1 {
		t.Errorf("click below the bar put the caret at %+v, want line 1", got)
	}
	ct.tt.ClickAt(st.ed.textX+1, barH+lh*0.5)
	ct.tt.Frame()
	if got := st.CursorPos(); got.Line != 0 {
		t.Errorf("click under the bar's edge put the caret at %+v, want line 0", got)
	}
	// Closing it gives the room back.
	st.ed.find.open = false
	ct.tt.Frame()
	ct.tt.Frame()
	if st.ed.top != 0 || st.ed.r.Y != 0 || caretTop() != caretBefore {
		t.Errorf("after closing: top %v, text box %v, caret %v", st.ed.top, st.ed.r, caretTop())
	}
}

// TestCodeEditorFindBarOverText checks that without FindBarAbove the bar
// floats over the text, which keeps its place.
func TestCodeEditorFindBarOverText(t *testing.T) {
	ct := newEditorTest(t, "", "one\ntwo")
	ct.st.RenderFindBar = func(c *ui.Context, bar ui.Element, f *FindBar) {
		bar.Height(40).Children(func() { ui.Text(c, "the bar") })
	}
	ct.st.Run(Find)
	ct.tt.Frame()
	ct.tt.Frame()
	if !ct.tt.HasText("the bar") || ct.st.ed.top != 0 || ct.st.ed.r.Y != 0 {
		t.Fatalf("top %v, text box %v", ct.st.ed.top, ct.st.ed.r)
	}
}

// TestFindBarAboveMovesTheBlocks checks that an app's block between the
// lines moves down with the text while the bar's strip shows.
func TestFindBarAboveMovesTheBlocks(t *testing.T) {
	ct := newEditorTest(t, "", "one\ntwo")
	st := ct.st
	const barH = 40
	st.FindBarAbove = true
	st.RenderFindBar = func(c *ui.Context, bar ui.Element, f *FindBar) {
		bar.Height(barH).Children(func() { ui.Text(c, "the bar") })
	}
	st.Blocks = []Block{{Line: 0, Below: true, Height: 20}}
	st.RenderBlock = func(c *ui.Context, box ui.Element, b Block) {
		box.Children(func() { ui.Text(c, "the block") })
	}
	ct.tt.Frame()
	before, ok := ct.tt.Find("the block")
	if !ok {
		t.Fatal("no block")
	}
	st.Run(Find)
	ct.tt.Frame()
	ct.tt.Frame()
	if got, _ := ct.tt.Find("the block"); got.Y != before.Y+barH {
		t.Fatalf("block at %v, was %v: want it %v lower", got.Y, before.Y, barH)
	}
}

// TestPaddingTopMovesTheTextAndBlocks checks that the room above the text
// (PaddingTop) moves the lines, an app's block and a click by it, and
// adds to the find bar's strip.
func TestPaddingTopMovesTheTextAndBlocks(t *testing.T) {
	ct := newEditorTest(t, "", "one\ntwo")
	st := ct.st
	st.Blocks = []Block{{Line: 0, Below: true, Height: 20}}
	st.RenderBlock = func(c *ui.Context, box ui.Element, b Block) {
		box.Children(func() { ui.Text(c, "the block") })
	}
	ct.tt.Frame()
	before, _ := ct.tt.Find("the block")
	caretBefore, _ := ct.tt.TextCaret()
	st.PaddingTop = 6
	ct.tt.Frame()
	ct.tt.Frame()
	if got, _ := ct.tt.Find("the block"); got.Y != before.Y+6 {
		t.Errorf("block at %v, was %v: want it 6 lower", got.Y, before.Y)
	}
	if got, _ := ct.tt.TextCaret(); got.Y != caretBefore.Y+6 {
		t.Errorf("caret at %v, was %v: want it 6 lower", got.Y, caretBefore.Y)
	}
	lh := st.ed.view.lineHeight
	// The second line is below the block.
	ct.tt.ClickAt(st.ed.textX+1, 6+lh*1.5+20)
	ct.tt.Frame()
	if got := st.CursorPos(); got.Line != 1 {
		t.Errorf("click on the second line put the caret at %+v", got)
	}
	ct.tt.ClickAt(st.ed.textX+1, 6+lh*0.5)
	ct.tt.Frame()
	if got := st.CursorPos(); got.Line != 0 {
		t.Errorf("click on the first line put the caret at %+v", got)
	}
	st.FindBarAbove = true
	st.RenderFindBar = func(c *ui.Context, bar ui.Element, f *FindBar) {
		bar.Height(40).Children(func() { ui.Text(c, "the bar") })
	}
	st.Run(Find)
	ct.tt.Frame()
	ct.tt.Frame()
	if st.ed.top != 46 || st.ViewportHeight() != 400-46 {
		t.Errorf("top %v, viewport %v: want the padding below the bar", st.ed.top, st.ViewportHeight())
	}
	if bar, _ := ct.tt.Find("the bar"); bar.Y > 1 {
		t.Errorf("bar at %v, want it at the top", bar.Y)
	}
}
