package editor

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// zode's options for its editor's clicks, carets, paging and find bar.

// PlainClicks: Alt+Shift+click extends the selection as Shift+click, and
// Alt+click on a caret adds none and takes none away.
func TestPlainClicks(t *testing.T) {
	for _, plain := range []bool{false, true} {
		st := &State{Buffer: NewBuffer("abcdef\nghijkl"), Font: ui.Font{Family: "monospace", Size: 14}, PlainClicks: plain}
		tt := ui.NewTester(func(c *ui.Context) { View(c, st).Fill() }, 400, 200)
		tt.Frame()
		st.Select(1, 1)
		tt.Frame()
		r, _ := st.ed.caretRect(4)
		x, y := st.ed.textX+r.X-st.ed.scrollX, r.Y+r.H/2
		tt.ClickAtWith(ui.Alt|ui.Shift, x, y)
		tt.Frame()
		sels := st.Selections()
		if plain && (len(sels) != 1 || sels[0].Anchor != 1 || sels[0].Head != 4) {
			t.Errorf("plain Alt+Shift+click: %+v, want 1 to 4", sels)
		}
		if !plain && len(sels) == 1 && sels[0].Anchor == 1 && sels[0].Head == 4 {
			t.Errorf("Alt+Shift+click extended as Shift+click without PlainClicks")
		}
	}
}

// GutterLayout's FoldStripClicksText: a press in the fold strip of a line
// that does not fold places the caret at its start, not OnGutterClick.
func TestFoldStripClicksText(t *testing.T) {
	gutterClicks := 0
	st := &State{Buffer: NewBuffer("one\ntwo\nthree"), Font: ui.Font{Family: "monospace", Size: 14},
		Gutter: &GutterLayout{Left: 18, Right: 14, MinDigits: 3, FoldStripClicksText: true}, OnGutterClick: func(int) { gutterClicks++ }}
	tt := ui.NewTester(func(c *ui.Context) { View(c, st).Fill() }, 400, 200)
	tt.Frame()
	lh := st.ed.view.lineHeight
	tt.ClickAt(5, lh*1.5)
	tt.Frame()
	if s := st.Selection(); gutterClicks != 0 || s.Anchor != 4 || s.Head != 4 {
		t.Errorf("fold strip click: gutter clicks %d, selection %+v", gutterClicks, s)
	}
	// On the numbers, the gutter's.
	tt.ClickAt(30, lh*1.5)
	tt.Frame()
	if gutterClicks != 1 {
		t.Errorf("numbers' click: %d gutter clicks", gutterClicks)
	}
	// A right click there has no menu.
	st.GutterMenu = func(m *ui.Menu, line int) { m.Item("Gutter") }
	tt.Frame()
	tt.RightClickAt(5, lh*1.5)
	tt.Frame()
	if m := tt.Menu(); len(m) != 0 {
		t.Errorf("fold strip's menu %q", m)
	}
	tt.CloseMenu()
	tt.RightClickAt(30, lh*1.5)
	tt.Frame()
	if m := tt.Menu(); len(m) != 1 || m[0] != "Gutter" {
		t.Errorf("numbers' menu %q", m)
	}
}

// BlockCaretWhenEmpty: a block for a caret, a bar for a selection's.
func TestBlockCaretWhenEmpty(t *testing.T) {
	st := &State{Buffer: NewBuffer("abc"), CursorShape: CursorBlock, BlockCaretWhenEmpty: true}
	tt := ui.NewTester(func(c *ui.Context) { View(c, st).Fill() }, 400, 200)
	tt.Frame()
	if got := st.ed.caretShape(sel{anchor: 1, head: 1}); got != CursorBlock {
		t.Errorf("a caret: %v", got)
	}
	if got := st.ed.caretShape(sel{anchor: 0, head: 2}); got != CursorBar {
		t.Errorf("a selection: %v", got)
	}
}

// PageMovesCarets: Page Down moves the caret a page of the text below
// PaddingTop less two rows, and scrolls only to keep it in view.
func TestPageMovesCarets(t *testing.T) {
	text := "0"
	for i := 1; i < 100; i++ {
		text += "\nx"
	}
	st := &State{Buffer: NewBuffer(text), Font: ui.Font{Family: "monospace", Size: 14}, PaddingTop: 6, PageMovesCarets: true}
	tt := ui.NewTester(func(c *ui.Context) { View(c, st).Fill() }, 400, 300)
	tt.Frame()
	lh := st.ed.view.lineHeight
	st.Run(MovePageDown)
	tt.Frame()
	want := int((st.ed.viewH-6)/lh) - 2
	if line := st.ed.view.text.LineOf(st.Selection().Head); line != want {
		t.Errorf("page down to line %d, want %d", line, want)
	}
	if st.ScrollY() > float64(3*lh) {
		t.Errorf("scrolled %v, more than the margin", st.ScrollY())
	}
}

// FindFromOrigin's bar opened again on the matches it had counts them
// with none current (Matches' -1), and Next chooses one.
func TestFindBarMatchesWithoutACurrentOne(t *testing.T) {
	ct := newEditorTest(t, "", "x a x a x|")
	ct.st.FindFromOrigin = true
	var bar *FindBar
	ct.st.RenderFindBar = func(c *ui.Context, el ui.Element, f *FindBar) { bar = f; DefaultFindBar(c, el, f) }
	ct.tt.Frame()
	ct.st.Find("x")
	ct.tt.Frame()
	if n, cur := bar.Matches(); n != 3 || cur < 0 {
		t.Fatalf("first open: %d matches, current %d", n, cur)
	}
	ct.key(0, ui.KeyEscape)
	ct.run(Find)
	if n, cur := bar.Matches(); n != 3 || cur != -1 {
		t.Fatalf("opened again: %d matches, current %d, want 3 and -1", n, cur)
	}
	bar.Next()
	ct.tt.Frame()
	if _, cur := bar.Matches(); cur < 0 {
		t.Fatal("Next chose no match")
	}
}

// FindFromOrigin's search stops at 20000 matches, as zode's find bar.
func TestFindFromOriginFindsAtMost20000(t *testing.T) {
	ct := newEditorTest(t, "", strings.Repeat("x", 30000))
	ct.st.FindFromOrigin = true
	ct.tt.Frame()
	ct.st.Find("x")
	if n, _ := ct.st.FindMatches(); n != 20000 {
		t.Fatalf("%d matches, want 20000", n)
	}
}

// StraightUnderlines: a diagnostic of no width at a line's end is a mark
// 2 DIPs wide where it is.
func TestStraightUnderlinesMarkAnEmptyDiagnosticAtTheEnd(t *testing.T) {
	ct := newEditorTest(t, "", "abcd\nx|")
	ct.st.Theme = &Theme{Background: ui.RGB(255, 255, 255)}
	ct.st.StraightUnderlines, ct.st.NoActiveLine = true, true
	ct.st.SetDiagnostics(ct.st.Buffer.Version(), []Diagnostic{{Start: 4, End: 4, Severity: SeverityError, Message: "here"}})
	ct.tt.Frame()
	ed := &ct.st.ed
	cw := ed.view.charWidth
	bottom := float32(ed.view.heights.geometry(0).rows-ed.scrollY) + ed.view.lineHeight
	marked := func(x float32) bool {
		for dy := float32(1); dy < 6; dy++ {
			if r, g, b := ct.pixel(x, bottom-dy); r > 150 && g < 120 && b < 120 {
				return true
			}
		}
		return false
	}
	if !marked(ed.textX+cw*4+1) || marked(ed.textX+cw*4+4) || marked(ed.textX+cw*3.5) {
		t.Fatal("no 2-DIP mark at the line's end alone")
	}
}

// BandsInset: the caret line's band ends BandsInset DIPs short of the
// editor's right edge.
func TestBandsInset(t *testing.T) {
	ct := newEditorTest(t, "", "ab|c\n")
	ct.st.Theme = &Theme{Background: ui.RGB(255, 255, 255), ActiveLine: ui.RGB(0, 0, 255)}
	ct.st.BandsInset, ct.st.NoScrollbars = 6, true
	ct.tt.Frame()
	ed := &ct.st.ed
	y := float32(ed.view.heights.geometry(0).rows-ed.scrollY) + ed.view.lineHeight/2
	w := ed.r.W
	if _, _, b := ct.pixel(w-8, y); b < 200 {
		t.Fatal("no band 8 DIPs from the edge")
	}
	if r, _, _ := ct.pixel(w-3, y); r < 200 {
		t.Fatal("the band reaches into the inset")
	}
}
