package editor

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// A press on a trailing inlay's text is told to OnTrailingInlayClick with
// its index in SetInlays' list, never to OnInlayClick; taken, it leaves
// the caret where it was. Past its text, or with no hook, the press is the
// text's.
func TestTrailingInlayClickIsItsOwn(t *testing.T) {
	st := &State{Buffer: NewBuffer("let x = 1\nlet y = 2\n")}
	var hints []InlayClick
	var trailing []int
	st.OnInlayClick = func(c InlayClick) bool { hints = append(hints, c); return true }
	st.OnTrailingInlayClick = func(i int) bool { trailing = append(trailing, i); return true }
	tt := ui.NewTester(func(c *ui.Context) { View(c, st).Fill() }, 600, 200)
	st.SetInlays([]Inlay{{Offset: 5, Text: ": number"}, {Offset: 9, Text: "  run", Trailing: true}, {Offset: 9, Text: " · test", Trailing: true}})
	tt.Frame()
	ed := &st.ed
	cw := ed.view.charWidth
	y := float32(ed.view.heights.geometry(0).rows-ed.scrollY) + ed.view.lineHeight/2
	// "let x" ": number" " = 1": 17 characters, then "  run" and " · test".
	tt.ClickAt(ed.textX+cw*(17+3.5), y)
	tt.ClickAt(ed.textX+cw*(17+5+4.5), y)
	if len(hints) != 0 {
		t.Fatalf("a press on a trailing inlay reached OnInlayClick: %+v", hints)
	}
	if len(trailing) != 2 || trailing[0] != 1 || trailing[1] != 2 {
		t.Fatalf("OnTrailingInlayClick told %v, want [1 2]", trailing)
	}
	if c := st.Cursor(); c != 0 {
		t.Fatalf("a taken press moved the caret to %d", c)
	}
	tt.Move(ed.textX+cw*(17+3.5), y)
	tt.Frame()
	if tt.Cursor() != ui.CursorPointer {
		t.Fatal("over a trailing inlay that takes presses, the cursor is not a pointer")
	}
	// Past the text, the press places the caret at the line's end.
	tt.ClickAt(ed.textX+cw*40, y)
	if len(trailing) != 2 || st.Cursor() != 9 {
		t.Fatalf("a press past the inlay: told %v, caret %d", trailing, st.Cursor())
	}
	st.OnTrailingInlayClick = nil
	tt.Frame()
	tt.ClickAt(ed.textX+cw*(17+3.5), y)
	if c := st.Cursor(); c != 9 {
		t.Fatalf("with no hook, a press on the inlay put the caret at %d, want 9", c)
	}
}
