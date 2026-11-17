package editor

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// PrimarySelection: what the focused editor selects goes to the primary
// selection once the selection settles, never to the clipboard; without
// the option, or unfocused, nothing goes there.
func TestSelectionGoesToPrimary(t *testing.T) {
	for _, on := range []bool{false, true} {
		st := &State{Buffer: NewBuffer("abcdef\nghijkl"), Font: ui.Font{Family: "monospace", Size: 14}, PrimarySelection: on}
		tt := ui.NewTester(func(c *ui.Context) { View(c, st).Fill().AutoFocus() }, 400, 200)
		tt.Frame()
		tt.Key(ui.Shift, ui.KeyRight)
		tt.Key(ui.Shift, ui.KeyRight)
		tt.Frame()
		want := ""
		if on {
			want = "ab"
		}
		if got := tt.Primary(); got != want {
			t.Errorf("PrimarySelection %v: primary %q, want %q", on, got, want)
		}
		if got := tt.Clipboard(); got != "" {
			t.Errorf("PrimarySelection %v: clipboard %q", on, got)
		}
	}

	// A drag goes there when the button is let go, and a caret leaves it.
	st := &State{Buffer: NewBuffer("abcdef\nghijkl"), Font: ui.Font{Family: "monospace", Size: 14}, PrimarySelection: true}
	tt := ui.NewTester(func(c *ui.Context) { View(c, st).Fill().AutoFocus() }, 400, 200)
	tt.Frame()
	at := func(off int) (float32, float32) {
		r, _ := st.ed.caretRect(off)
		return st.ed.textX + r.X - st.ed.scrollX, r.Y + r.H/2
	}
	x0, y0 := at(7)
	x1, y1 := at(10)
	tt.Press(x0, y0)
	tt.Move(x1, y1)
	tt.Frame()
	if got := tt.Primary(); got != "" {
		t.Errorf("mid-drag primary %q", got)
	}
	tt.Release(x1, y1)
	tt.Frame()
	if got := tt.Primary(); got != "ghi" {
		t.Errorf("after the drag primary %q, want %q", got, "ghi")
	}
	tt.Key(0, ui.KeyRight)
	tt.Frame()
	if got := tt.Primary(); got != "ghi" {
		t.Errorf("a caret changed primary to %q", got)
	}

	// An editor without the focus leaves it to the one that has it.
	st.SetSelections(Selection{Anchor: 0, Head: 3})
	tt.SetFocused(false)
	tt.Frame()
	if got := tt.Primary(); got != "ghi" {
		t.Errorf("an unfocused editor wrote primary %q", got)
	}
}

// A middle press puts the caret at the pointer and pastes the primary
// selection there, not the clipboard; without PrimarySelection, or read
// only, it pastes nothing.
func TestMiddlePressPastesPrimaryAtThePointer(t *testing.T) {
	for _, c := range []struct {
		on, readOnly bool
		want         string
	}{
		{true, false, "abcdef\nghXYijkl"},
		{false, false, "abcdef\nghijkl"},
		{true, true, "abcdef\nghijkl"},
	} {
		st := &State{Buffer: NewBuffer("abcdef\nghijkl"), Font: ui.Font{Family: "monospace", Size: 14},
			PrimarySelection: c.on, ReadOnly: c.readOnly}
		tt := ui.NewTester(func(ctx *ui.Context) { View(ctx, st).Fill().AutoFocus() }, 400, 200)
		tt.Frame()
		tt.SetClipboard("CLIP")
		tt.SetPrimary("XY")
		r, _ := st.ed.caretRect(9)
		x, y := st.ed.textX+r.X-st.ed.scrollX, r.Y+r.H/2
		tt.PressButton(2, x, y)
		tt.ReleaseButton(2, x, y)
		tt.Frame()
		if got := st.Buffer.String(); got != c.want {
			t.Errorf("%+v: text %q, want %q", c, got, c.want)
		}
		if c.on && !c.readOnly {
			if s := st.Selection(); s.Anchor != 11 || s.Head != 11 {
				t.Errorf("%+v: caret %+v, want after the paste at 11", c, s)
			}
		}
		if got := tt.Clipboard(); got != "CLIP" {
			t.Errorf("%+v: clipboard became %q", c, got)
		}
	}
}
