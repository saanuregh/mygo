package editor

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// zode's editor jumps, find bar, Ctrl+click and copies.

// A jump puts its line at the middle of the view even when the line is
// in view already (zode's view.zig .center).
func TestJumpsCenterTheirLineInView(t *testing.T) {
	st := &State{Buffer: NewBuffer(strings.Repeat("x\n", 100))}
	tt := ui.NewTester(func(c *ui.Context) { View(c, st).Fill() }, 400, 400)
	tt.Frame()
	line := int(300 / st.ed.view.lineHeight)
	st.GoTo(Pos{Line: line})
	tt.Frame()
	want := st.LineTop(line) - float64(st.ed.viewH)/2
	if got := st.ScrollY(); abs(float32(got-want)) > 0.5 {
		t.Fatalf("going to line %d in view scrolled to %v, want %v", line, got, want)
	}
}

// Cmd (Ctrl) with any other modifier held clicks to the definition, ahead
// of Alt+Shift's column selection and Shift's extending; the link shows
// under it as well.
func TestCtrlWithOtherModifiersClicksToDefinition(t *testing.T) {
	var asked []int
	st := &State{Buffer: NewBuffer("first line\nsecond line\n"), OnDefinition: func(off int) { asked = append(asked, off) }}
	tt := ui.NewTester(func(c *ui.Context) { View(c, st).Fill() }, 600, 300)
	tt.Frame()
	cw := st.ed.view.charWidth
	x := func(col int) float32 { return st.ed.textX + cw*(float32(col)+0.5) }
	for i, mods := range []ui.Modifiers{ui.Cmd | ui.Shift, ui.Cmd | ui.Alt, ui.Cmd | ui.Alt | ui.Shift} {
		tt.Advance(time.Second)
		tt.ClickAtWith(mods, x(2+i), 5)
		tt.Frame()
		if len(asked) != i+1 || asked[i] != 2+i {
			t.Fatalf("mods %v: definitions asked at %v", mods, asked)
		}
	}
	tt.Advance(time.Second)
	tt.Move(x(7), 5)
	tt.HoldModifiers(ui.Cmd | ui.Shift)
	tt.Frame()
	if st.ed.link != [2]int{6, 10} {
		t.Fatalf("link %v with Cmd+Shift held", st.ed.link)
	}
	tt.HoldModifiers(0)
	tt.Frame()
	// Shift alone still extends the selection.
	st.Select(0, 0)
	tt.Frame()
	tt.ClickAtWith(ui.Shift, x(3), 5)
	tt.Frame()
	if s := st.Selection(); s.Anchor != 0 || s.Head != 3 || len(asked) != 3 {
		t.Fatalf("Shift+click: %+v, definitions %v", s, asked)
	}
}

// FindFromOrigin's bar: Find hides replacing, FindReplace over a query
// gives the replacement the keyboard, a selection of 1000 bytes or more
// seeds nothing, and a change of the options chooses the match from the
// origin again.
func TestFindFromOriginBar(t *testing.T) {
	ct := newEditorTest(t, "", "x a [x] a x a x\n"+strings.Repeat("y", 1200))
	ct.st.FindFromOrigin = true
	ct.tt.Frame()
	ct.st.Run(FindReplace)
	f := &ct.st.ed.find
	if !f.replacing || !f.focusReplace || f.focusQuery || f.query != "x" {
		t.Fatalf("FindReplace over x: replacing %v, focus replacement %v, query %v %q", f.replacing, f.focusReplace, f.focusQuery, f.query)
	}
	ct.tt.Frame()
	if f.focusReplace {
		t.Fatal("the bar's replacement field did not take the focus")
	}
	ct.st.Run(Find)
	if f.replacing || !f.focusQuery {
		t.Fatalf("Find: replacing %v, focus query %v", f.replacing, f.focusQuery)
	}
	ct.tt.Frame()
	if _, cur := ct.st.FindMatches(); cur != 1 {
		t.Fatalf("current %d, want the match at the origin (1)", cur)
	}
	ct.st.ed.find.step(&ct.st.ed, 1)
	ct.tt.Frame()
	if _, cur := ct.st.FindMatches(); cur != 2 {
		t.Fatalf("after Next, current %d", cur)
	}
	f.caseSens = true
	ct.tt.Frame()
	if _, cur := ct.st.FindMatches(); cur != 1 {
		t.Fatalf("an option changed: current %d, want the origin's again (1)", cur)
	}
	// A long selection seeds nothing.
	n := len("x a x a x a x\n")
	ct.st.Select(n, n+1100)
	ct.tt.Frame()
	ct.run(Find)
	if f.query != "x" {
		t.Fatalf("a selection of 1100 bytes seeded %d bytes", len(f.query))
	}
}

// SetCopied records an app's own copy for the editors' paste, which
// Copied reads back.
func TestSetCopied(t *testing.T) {
	SetCopied("a\nb", []string{"a", "b"}, []bool{false, false})
	defer SetCopied("", nil, nil)
	text, pieces, lines := Copied()
	if text != "a\nb" || len(pieces) != 2 || pieces[1] != "b" || len(lines) != 2 || lines[1] {
		t.Fatalf("Copied %q %q %v", text, pieces, lines)
	}
	ct := newEditorTest(t, "", "|1\n|2")
	ct.tt.SetClipboard("a\nb")
	ct.run(Paste)
	ct.want("a|1\nb|2")
}

// CompletionMenu.Anchor is the box at the caret the list attaches to.
func TestCompletionMenuAnchor(t *testing.T) {
	ct := newEditorTest(t, "", "fo|")
	var anchor ui.Element
	ct.st.RenderCompletions = func(c *ui.Context, panel ui.Element, m *CompletionMenu) {
		anchor = m.Anchor()
		DefaultCompletions(c, panel, m)
	}
	ct.st.ShowCompletions(0, []Completion{{Label: "foo"}})
	ct.tt.Frame()
	ct.tt.Frame()
	if anchor == (ui.Element{}) {
		t.Fatal("no anchor")
	}
}
