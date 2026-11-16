package editor

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// An excerpt shows only its lines, numbered as in the buffer, keeps the
// carets within them, refuses edits reaching outside, and grows with the
// lines typed at its end.
func TestExcerptShowsItsLines(t *testing.T) {
	ct := newEditorTest(t, "", "zero\none\ntwo\nthree\nfour")
	ct.st.SetExcerpt(1, 2)
	ct.tt.Frame()
	lh := float64(ct.st.ed.view.lineHeight)
	if got := ct.st.LineTop(1); got != 0 {
		t.Fatalf("the excerpt's first line at %v, want the top", got)
	}
	if got := ct.st.LineTop(5); got != 2*lh {
		t.Fatalf("the excerpt %v high, want two lines, %v", got, 2*lh)
	}
	ct.want("zero\n|one\ntwo\nthree\nfour")
	// The gutter numbers the lines shown, 1 and 2 of the buffer, as 2 and 3.
	if first, last := ct.st.VisibleLines(); first != 1 || last != 2 {
		t.Fatalf("lines %d to %d shown, want 1 to 2", first, last)
	}
	ct.key(0, ui.KeyUp)
	ct.want("zero\n|one\ntwo\nthree\nfour")
	// Backspace at its start would delete the line before.
	ct.key(0, ui.KeyBackspace)
	ct.want("zero\n|one\ntwo\nthree\nfour")
	ct.key(0, ui.KeyDown)
	ct.key(0, ui.KeyDown)
	ct.want("zero\none\ntwo|\nthree\nfour")
	ct.key(0, ui.KeyDelete)
	ct.want("zero\none\ntwo|\nthree\nfour")
	ct.typ("\n2b")
	ct.want("zero\none\ntwo\n2b|\nthree\nfour")
	if first, last, ok := ct.st.Excerpt(); !ok || first != 1 || last != 3 {
		t.Fatalf("excerpt %d to %d (%v), want 1 to 3 with the line typed", first, last, ok)
	}
	ct.run(SelectAll)
	if got := ct.st.SelectedText(); got != "one\ntwo\n2b" {
		t.Fatalf("Select All selected %q, want the excerpt", got)
	}
	ct.st.ClearExcerpt()
	ct.tt.Frame()
	if _, _, ok := ct.st.Excerpt(); ok || ct.st.LineTop(1) != lh {
		t.Fatal("the excerpt is still shown after ClearExcerpt")
	}
}

// Editors of one buffer show excerpts of it each, which edits in either
// move.
func TestExcerptsOfASharedBuffer(t *testing.T) {
	buf := NewBuffer("a\nb\nc\nd\ne")
	top, bottom := &State{Buffer: buf}, &State{Buffer: buf}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			View(c, top).FillWidth().Height(100).AutoFocus()
			View(c, bottom).FillWidth().Height(100)
		})
	}, 600, 400)
	top.SetExcerpt(0, 1)
	bottom.SetExcerpt(3, 4)
	tt.Frame()
	tt.Type("x\n")
	tt.Frame()
	if got := buf.String(); got != "x\na\nb\nc\nd\ne" {
		t.Fatalf("text %q", got)
	}
	if first, last, _ := bottom.Excerpt(); first != 4 || last != 5 {
		t.Fatalf("the other excerpt is lines %d to %d, want 4 to 5", first, last)
	}
	if first, last, _ := top.Excerpt(); first != 0 || last != 2 {
		t.Fatalf("the edited excerpt is lines %d to %d, want 0 to 2", first, last)
	}
}
