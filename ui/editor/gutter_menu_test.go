package editor

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// A right click on a line's number opens the GutterMenu of that line in
// place of the editor's menu, the caret where it was; on the text, the
// editor's own.
func TestGutterMenuOfTheLineClicked(t *testing.T) {
	ct := newEditorTest(t, "", "|one\ntwo\nthree\n")
	line := -1
	ct.st.GutterMenu = func(m *ui.Menu, l int) {
		line = l
		m.Item("Add Breakpoint")
	}
	ct.st.Menu = func(m *ui.Menu) { m.Item("Ask") }
	ct.tt.Frame()
	lh := float32(ct.st.ed.view.lineHeight)
	ct.tt.RightClickAt(4, 2*lh+lh/2)
	if got := strings.Join(ct.tt.Menu(), ","); got != "Add Breakpoint" || line != 2 {
		t.Fatalf("menu %q of line %d, want Add Breakpoint of line 2", got, line)
	}
	if h := ct.st.Selection().Head; h != 0 {
		t.Fatalf("the caret moved to %d", h)
	}
	ct.tt.CloseMenu()
	ct.tt.RightClickAt(200, lh/2)
	if got := strings.Join(ct.tt.Menu(), ","); got != "Ask" {
		t.Fatalf("menu on the text %q, want the editor's", got)
	}
	ct.tt.CloseMenu()
}

// NoContextMenu leaves a right click to the element around the editor.
func TestNoContextMenuLeavesTheClickToTheParent(t *testing.T) {
	st := &State{Buffer: NewBuffer("text")}
	st.NoContextMenu = true
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Size(400, 200).ContextMenu(func(m *ui.Menu) { m.Item("Window") }).Children(func() {
			View(c, st).Size(400, 200)
		})
	}, 400, 200)
	tt.Frame()
	tt.RightClickAt(200, 10)
	if got := strings.Join(tt.Menu(), ","); got != "Window" {
		t.Fatalf("menu %q, want the parent's", got)
	}
	tt.CloseMenu()
}

// LineTokens are a line's tokens from the buffer's highlighter: plain
// text without a language, the language's tokens with one.
func TestBufferLineTokens(t *testing.T) {
	var got []Token
	tt := ui.NewTester(func(c *ui.Context) {
		b := NewBuffer("x := 1\n")
		got, _ = b.LineTokens(c, 0, nil)
		if len(got) != 1 || got[0].End != 6 || got[0].Kind != TokenText {
			t.Errorf("plain tokens %v", got)
		}
		b.SetLanguage(LanguageFor("a.go"))
		got, _ = b.LineTokens(c, 0, nil)
	}, 100, 100)
	tt.Frame()
	if len(got) < 2 || got[len(got)-1].End != 6 {
		t.Fatalf("Go tokens %v, want several ending at 6", got)
	}
}

// CaretBox is the caret in the window: below a line's top, right of the
// text's start by the characters before it.
func TestCaretBoxIsInTheWindow(t *testing.T) {
	ct := newEditorTest(t, "", "one\ntw|o\n")
	ct.tt.Frame()
	r, ok := ct.st.CaretBox()
	lh := float32(ct.st.ed.view.lineHeight)
	if !ok || r.Y < lh-1 || r.Y > lh+1 || r.X <= ct.st.ed.textX || r.H != lh {
		t.Fatalf("caret box %+v (ok %v), line height %v, text left %v", r, ok, lh, ct.st.ed.textX)
	}
}
