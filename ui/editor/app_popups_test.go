package editor

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// The app's own ranking (CompletionScore) orders the list, equal scores by
// SortText; PageDown and PageUp move the choice a page of rows, around the
// ends.
func TestCompletionScoreSortTextAndPages(t *testing.T) {
	ct := newEditorTest(t, "go", "|")
	var items []Completion
	for i, label := range []string{"alpha", "alpine", "alps", "albatross", "alder", "algae", "alibi", "alley", "allot", "alloy", "almond"} {
		items = append(items, Completion{Label: label, SortText: string(rune('k' - i))})
	}
	ct.st.CompletionScore = func(query, text string) (int, bool) {
		return 0, strings.HasPrefix(text, query)
	}
	ct.st.OnCompletion = func(r CompletionRequest) { ct.st.ShowCompletions(r.Offset-len(r.Prefix), items) }
	ct.typ("al")
	cc := &ct.st.ed.complete
	if got := cc.items[cc.filtered[0]].Label; got != "almond" {
		t.Fatalf("first %q, want the lowest SortText (almond)", got)
	}
	ct.key(0, ui.KeyPageDown)
	if cc.selected != 9 {
		t.Fatalf("PageDown chose row %d, want 9", cc.selected)
	}
	ct.key(0, ui.KeyPageDown)
	if cc.selected != (9+9)%len(cc.filtered) {
		t.Fatalf("a second PageDown chose row %d", cc.selected)
	}
	ct.key(0, ui.KeyPageUp)
	if cc.selected != 9 {
		t.Fatalf("PageUp chose row %d, want 9", cc.selected)
	}
}

// Asked for again (Ctrl+Space) while the list shows, the app is asked
// again; typing only filters it.
func TestAskedCompletionsAskTheAppAgain(t *testing.T) {
	ct := newEditorTest(t, "go", "|")
	asked := 0
	ct.st.OnCompletion = func(r CompletionRequest) {
		asked++
		ct.st.ShowCompletions(r.Offset-len(r.Prefix), []Completion{{Label: "alpha"}, {Label: "alpine"}})
	}
	ct.typ("a")
	ct.typ("l")
	n := asked
	ct.run(ShowCompletions)
	if asked != n+1 {
		t.Fatalf("asked %d times after Ctrl+Space, want %d", asked, n+1)
	}
}

// HoverAppOnly: the hover holds only what the app gives, nothing for a
// diagnostic before it answers, and goes as soon as the pointer leaves
// the text hovered.
func TestHoverAppOnly(t *testing.T) {
	st := &State{Buffer: NewBuffer("foo := bar()\n"), HoverAppOnly: true}
	answer := true
	st.OnHover = func(r HoverRequest) {
		if answer {
			st.ShowHover(0, 3, "foo is a name", nil)
		}
	}
	tt := ui.NewTester(func(c *ui.Context) { View(c, st).Fill() }, 600, 400)
	st.SetDiagnostics(st.Buffer.Version(), []Diagnostic{{Start: 7, End: 10, Message: "bar is broken"}})
	tt.Frame()
	ed := &st.ed
	y := float32(ed.view.heights.geometry(0).rows-ed.scrollY) + ed.view.lineHeight/2
	// On the diagnostic, with no answer: no hover.
	answer = false
	tt.Move(ed.textX+ed.view.charWidth*8.5, y)
	tt.Advance(time.Hour)
	if ed.hover.shown || tt.HasText("bar is broken") {
		t.Fatal("a hover for the diagnostic alone")
	}
	answer = true
	tt.Move(ed.textX+ed.view.charWidth*1.5, y)
	tt.Advance(time.Hour)
	if !tt.HasText("foo is a name") || tt.HasText("bar is broken") {
		t.Fatalf("hover %q", tt.Texts())
	}
	// Off the word: gone at once.
	tt.Move(ed.textX+ed.view.charWidth*5.5, y)
	tt.Frame()
	tt.Frame()
	if ed.hover.shown {
		t.Fatal("the hover stayed after the pointer left its word")
	}
}

// CaretX and PlaceCaret: a caret placed at an x lands on the character
// there, and Up and Down keep that x.
func TestPlaceCaretKeepsTheX(t *testing.T) {
	ct := newEditorTest(t, "", "abcdefgh\nab\nabcdefgh\n|")
	cw := ct.st.ed.view.charWidth
	if !ct.st.PlaceCaret(0, 5*cw, false) {
		t.Fatal("not placed")
	}
	if got := ct.st.Cursor(); got != 5 {
		t.Fatalf("caret at %d, want 5", got)
	}
	if x := ct.st.CaretX(); x != 5*cw {
		t.Fatalf("CaretX %v, want %v", x, 5*cw)
	}
	ct.key(0, ui.KeyDown)
	ct.key(0, ui.KeyDown)
	if got := ct.st.CursorPos(); got.Line != 2 || got.Col != 5 {
		t.Fatalf("after Down twice: %+v, want line 2 column 5", got)
	}
}
