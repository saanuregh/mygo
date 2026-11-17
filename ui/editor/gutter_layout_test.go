package editor

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// An app's gutter sizes (State.Gutter) place the text, and the app's
// margin (MarginRight) narrows the wrap.
func TestGutterLayoutAndMargin(t *testing.T) {
	st := &State{Buffer: NewBuffer("one\ntwo"), Font: ui.Font{Family: "monospace", Size: 14},
		Gutter: &GutterLayout{Left: 18, Right: 14, TextGap: 6, MinDigits: 3}}
	tt := ui.NewTester(func(c *ui.Context) { View(c, st).Fill() }, 400, 200)
	tt.Frame()
	cw := st.ed.view.charWidth
	if got, want := st.TextLeft(), 18+3*cw+14+6; got != want {
		t.Fatalf("text at %v, want %v", got, want)
	}
	st.MarginRight = 100
	st.NoScrollbars = true
	tt.Frame()
	if got := st.ed.trackWidth(); got != 100 {
		t.Fatalf("room at the right %v, want 100", got)
	}
}

// A line's note (LineNotes) is drawn after its text in its color, and is
// no part of the line: it neither wraps it nor is an inlay of it. A note
// without a color of its own takes Theme.Note.
func TestLineNotes(t *testing.T) {
	red := ui.RGB(255, 0, 0)
	ct := newEditorTest(t, "", "one\ntwo|")
	ct.st.Theme = &Theme{Background: ui.RGB(255, 255, 255), Note: ui.RGB(0, 0, 255)}
	ct.st.NoActiveLine, ct.st.SoftWrap = true, true
	ct.st.LineNotes = []LineNote{{Line: 0, Text: "broken", Color: red}, {Line: 0, Text: "second"}, {Line: 1, Text: "blame"}}
	ct.tt.Frame()
	ed := &ct.st.ed
	if n, ok := ed.noteOf(0); !ok || n.Text != "broken" || n.Color != red {
		t.Fatalf("line 0's note %+v", n)
	}
	if n, ok := ed.noteOf(1); !ok || n.Color != ct.st.Theme.Note {
		t.Fatalf("line 1's note %+v takes Theme.Note", n)
	}
	if len(ed.view.inlays) != 0 || ed.view.heights.geometry(0).below-ed.view.heights.geometry(0).rows != float64(ed.view.lineHeight) {
		t.Fatal("a note is laid out in its line")
	}
	cw := ed.view.charWidth
	y := float32(ed.view.heights.geometry(0).rows-ed.scrollY) + ed.view.lineHeight/2
	found := func(want func(r, g, b uint8) bool) bool {
		for x := ed.textX + cw*7; x < ed.textX+cw*14; x++ {
			for dy := float32(-3); dy <= 3; dy++ {
				if want(ct.pixel(x, y+dy)) {
					return true
				}
			}
		}
		return false
	}
	if !found(func(r, g, b uint8) bool { return r > 150 && g < 120 && b < 120 }) {
		t.Fatal("no note drawn after line 0's text")
	}
	y = float32(ed.view.heights.geometry(1).rows-ed.scrollY) + ed.view.lineHeight/2
	if !found(func(r, g, b uint8) bool { return b > 150 && r < 120 && g < 120 }) {
		t.Fatal("no note in Theme.Note after line 1's text")
	}
}

// FindFromOrigin: the bar seeds no query from the word at the caret, and
// each query typed selects its match at or after where the caret was as
// the bar opened, not after the match last selected.
func TestFindFromOrigin(t *testing.T) {
	for _, origin := range []bool{false, true} {
		ct := newEditorTest(t, "", "abc ab|c abc")
		ct.st.FindFromOrigin = origin
		ct.tt.Frame()
		ct.st.Run(Find)
		ct.tt.Frame()
		f := &ct.st.ed.find
		if seeded := f.query != ""; seeded == origin {
			t.Fatalf("FindFromOrigin %v: query %q", origin, f.query)
		}
		f.query = "a"
		ct.tt.Frame()
		f.query = "c"
		ct.tt.Frame()
		want := 10
		if origin {
			want = 6
		}
		if s := ct.st.Selections(); len(s) != 1 || s[0].Anchor != want {
			t.Fatalf("FindFromOrigin %v: selections %+v, want the c at %d", origin, s, want)
		}
	}
}

// E7-9, zode's view.zig: with a gutter layout, the mark a click would
// put shows only with the pointer right of the fold strip (Left), whose
// press folds.
func TestGutterHoverMarkRightOfTheFoldStrip(t *testing.T) {
	ct := newEditorTest(t, "", "one\ntwo|")
	ct.st.Theme = &Theme{Background: ui.RGB(255, 255, 255)}
	ct.st.Gutter = &GutterLayout{Left: 18, Right: 14, MinDigits: 3}
	ct.st.GutterHoverMark = GutterMark{Color: ui.RGB(255, 0, 0), Size: 8}
	ct.st.NoActiveLine = true
	ct.tt.Frame()
	y := ct.st.ed.view.lineHeight / 2
	red := func() bool {
		for x := float32(2); x < 14; x++ {
			if r, g, b := ct.pixel(x, y); r > 200 && g < 80 && b < 80 {
				return true
			}
		}
		return false
	}
	ct.tt.Move(8, y)
	ct.tt.Frame()
	if red() {
		t.Fatal("a mark with the pointer over the fold strip")
	}
	ct.tt.Move(24, y)
	ct.tt.Frame()
	if !red() {
		t.Fatal("no mark with the pointer right of the fold strip")
	}
}
