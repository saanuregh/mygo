package editor

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// With RoundRanges, an empty match of the find bar shows as a bar 2 wide
// where it is (zode's view.zig: at least 2 wide).
func TestAnEmptyMatchIsABar(t *testing.T) {
	ct := newEditorTest(t, "", "ab\n  cd|")
	ct.st.Theme = &Theme{Background: ui.RGB(255, 255, 255), Match: ui.RGB(0, 200, 0), CurrentMatch: ui.RGB(0, 200, 0)}
	ct.st.RoundRanges, ct.st.NoActiveLine, ct.st.FindFromOrigin = true, true, true
	ct.tt.Frame()
	ct.st.ed.find.regex = true
	ct.st.Find("^")
	ct.tt.Frame()
	ed := &ct.st.ed
	y := float32(ed.view.heights.geometry(1).rows-ed.scrollY) + ed.view.lineHeight/2
	if r, g, b := ct.pixel(ed.textX+1, y); r > 50 || g < 150 || b > 50 {
		t.Fatalf("at the empty match on line 2: %v,%v,%v", r, g, b)
	}
}

// StraightUnderlines: a diagnostic of no width underlines the character
// after it (zode's pane_editor_parts.zig widens it to one); at a line's
// end, a mark 2 wide.
func TestAnEmptyDiagnosticUnderlinesTheNextCharacter(t *testing.T) {
	ct := newEditorTest(t, "", "abcd\nx|")
	ct.st.Theme = &Theme{Background: ui.RGB(255, 255, 255)}
	ct.st.StraightUnderlines, ct.st.NoActiveLine = true, true
	ct.st.SetDiagnostics(ct.st.Buffer.Version(), []Diagnostic{{Start: 1, End: 1, Severity: SeverityError, Message: "here"}})
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
	if !marked(ed.textX+cw*1.7) || marked(ed.textX+cw*2.6) {
		t.Fatalf("the underline is not under the second character alone")
	}
}

// FilledBrackets with FitContent (zode's excerpts): the brackets at the
// caret are marked only while the editor has the keyboard.
func TestFilledBracketsInAPageOnlyWithTheKeyboard(t *testing.T) {
	st := &State{Buffer: NewBuffer("f(x)\n"), FilledBrackets: true, FitContent: true}
	st.Select(1, 1)
	tt := ui.NewTester(func(c *ui.Context) { View(c, st).FillWidth() }, 400, 200)
	tt.Frame()
	var f frame
	st.ed.prepare(&f)
	if st.ed.focused || f.bracketsOK {
		t.Fatalf("unfocused: focused %v, brackets marked %v", st.ed.focused, f.bracketsOK)
	}
	st.ed.focused = true
	f = frame{}
	st.ed.prepare(&f)
	if !f.bracketsOK {
		t.Fatal("focused: no brackets marked")
	}
}
