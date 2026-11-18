package editor

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestCodeEditorDefinitionLinkAsked checks that, with OnDefinitionLink
// set, Cmd held over the text asks the app once per place and shows a link
// only over the range it answers: none until it answers, none where it
// answers nothing, none once the text changes.
func TestCodeEditorDefinitionLinkAsked(t *testing.T) {
	var asked []int
	st := &State{Buffer: NewBuffer("func first() {}\nsecond line\n"),
		OnDefinition: func(int) {}, OnDefinitionLink: func(off int) { asked = append(asked, off) }}
	tt := ui.NewTester(func(c *ui.Context) { View(c, st).Fill() }, 600, 300)
	tt.Frame()
	at := func(col float32) float32 { return st.ed.textX + st.ed.view.charWidth*(col+0.5) }

	// Over "func": asked once, no link while it is not answered.
	tt.Move(at(1), 5)
	tt.HoldModifiers(ui.Cmd)
	tt.Frame()
	tt.Frame()
	if len(asked) != 1 || asked[0] != 1 {
		t.Fatalf("asked %v with Cmd held over offset 1", asked)
	}
	if st.ed.link != ([2]int{}) || tt.Cursor() == ui.CursorPointer {
		t.Fatalf("link %v, cursor %v before the app answered", st.ed.link, tt.Cursor())
	}
	// Answered with nothing there: still no link.
	st.SetDefinitionLink(1, 1)
	tt.Frame()
	if st.ed.link != ([2]int{}) || tt.Cursor() == ui.CursorPointer {
		t.Fatalf("link %v, cursor %v where the app found nothing", st.ed.link, tt.Cursor())
	}

	// Over "first": asked, then shown over the range answered.
	tt.Move(at(6), 5)
	tt.HoldModifiers(ui.Cmd)
	tt.Frame()
	if len(asked) != 2 || asked[1] != 6 {
		t.Fatalf("asked %v over \"first\"", asked)
	}
	st.SetDefinitionLink(5, 10)
	tt.Frame()
	if st.ed.link != [2]int{5, 10} || tt.Cursor() != ui.CursorPointer {
		t.Fatalf("link %v, cursor %v once answered with 5-10", st.ed.link, tt.Cursor())
	}
	// Elsewhere in the range answered: not asked again.
	tt.Move(at(8), 5)
	tt.HoldModifiers(ui.Cmd)
	tt.Frame()
	if len(asked) != 2 || st.ed.link != [2]int{5, 10} {
		t.Fatalf("asked %v, link %v inside the range answered", asked, st.ed.link)
	}

	// Cmd let go: no link; held again at the same place, the answer stands.
	tt.HoldModifiers(0)
	tt.Frame()
	if st.ed.link != ([2]int{}) {
		t.Fatalf("link %v after Cmd was let go", st.ed.link)
	}
	tt.HoldModifiers(ui.Cmd)
	tt.Frame()
	if len(asked) != 2 || st.ed.link != [2]int{5, 10} {
		t.Fatalf("asked %v, link %v with Cmd held again", asked, st.ed.link)
	}

	// An edit forgets the answer, and the place is asked about again.
	st.Buffer.Edit(Edit{Start: 0, End: 0, Text: "x"})
	tt.Frame()
	if st.ed.link != ([2]int{}) {
		t.Fatalf("link %v after an edit", st.ed.link)
	}
	if len(asked) != 3 {
		t.Fatalf("asked %v after an edit", asked)
	}
}
