package editor

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestCodeEditorCaretCard checks that the app's card shows above the
// caret's line, and below it while completions show.
func TestCodeEditorCaretCard(t *testing.T) {
	ct := newEditorTest(t, "go", "a\nb\nc\nd\ne\nf|(x)")
	ct.st.CaretCard = func(c *ui.Context, card ui.Element) {
		card.MarginEnd(40).Padding(4).Children(func() { ui.Text(c, "f(x int)") })
	}
	ct.tt.Frame()
	ct.tt.Frame()
	card, ok := ct.tt.Find("f(x int)")
	caret, _ := ct.st.ed.caretRect(ct.st.Cursor())
	if !ok {
		t.Fatal("no card")
	}
	// The text inside its padding of 4, 40 left of the caret.
	if x := ct.st.ed.textX + caret.X; card.Y+card.H+4 > caret.Y || card.X-4 != x-40 {
		t.Errorf("card %v, caret %v: want above the line, left of the caret", card, caret)
	}
	ct.st.ShowCompletions(ct.st.Cursor(), []Completion{{Label: "foo"}})
	ct.tt.Frame()
	ct.tt.Frame()
	if card, _ = ct.tt.Find("f(x int)"); card.Y < caret.Y+caret.H {
		t.Errorf("card %v, caret %v: want below the line with completions", card, caret)
	}
	ct.st.CaretCard = nil
	ct.tt.Frame()
	if ct.tt.HasText("f(x int)") {
		t.Error("the card stays")
	}
}
