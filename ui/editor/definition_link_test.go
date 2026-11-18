package editor

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// linkTester is an editor whose OnDefinitionLink records its questions.
type linkTester struct {
	st    *State
	tt    *ui.Tester
	asked []DefinitionLinkRequest
}

func newLinkTester(text string) *linkTester {
	l := &linkTester{}
	l.st = &State{Buffer: NewBuffer(text), OnDefinition: func(int) {},
		OnDefinitionLink: func(r DefinitionLinkRequest) { l.asked = append(l.asked, r) }}
	l.tt = ui.NewTester(func(c *ui.Context) { View(c, l.st).Fill() }, 600, 300)
	l.tt.Frame()
	return l
}

// hover moves the pointer onto column col of the first line with mods held.
func (l *linkTester) hover(col float32, mods ui.Modifiers) {
	l.tt.Move(l.st.ed.textX+l.st.ed.view.charWidth*(col+0.5), 5)
	l.tt.HoldModifiers(mods)
	l.tt.Frame()
}

func (l *linkTester) lastAsked() DefinitionLinkRequest { return l.asked[len(l.asked)-1] }

// TestCodeEditorDefinitionLinkAsked checks that, with OnDefinitionLink
// set, Cmd held over the text asks the app once per place and shows a link
// only over the range it answers: none where it answers nothing, none once
// the text changes.
func TestCodeEditorDefinitionLinkAsked(t *testing.T) {
	l := newLinkTester("func first() {}\nsecond line\n")
	st, tt := l.st, l.tt

	// Over "func": asked once, no link while it is not answered.
	l.hover(1, ui.Cmd)
	tt.Frame()
	if len(l.asked) != 1 || l.asked[0] != (DefinitionLinkRequest{Offset: 1}) {
		t.Fatalf("asked %v with Cmd held over offset 1", l.asked)
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
	l.hover(6, ui.Cmd)
	if len(l.asked) != 2 || l.asked[1].Offset != 6 {
		t.Fatalf("asked %v over \"first\"", l.asked)
	}
	st.SetDefinitionLink(5, 10)
	tt.Frame()
	if st.ed.link != [2]int{5, 10} || tt.Cursor() != ui.CursorPointer {
		t.Fatalf("link %v, cursor %v once answered with 5-10", st.ed.link, tt.Cursor())
	}
	// Elsewhere in the range answered: not asked again.
	l.hover(8, ui.Cmd)
	if len(l.asked) != 2 || st.ed.link != [2]int{5, 10} {
		t.Fatalf("asked %v, link %v inside the range answered", l.asked, st.ed.link)
	}

	// An edit forgets the answer, and the place is asked about again.
	st.Buffer.Edit(Edit{Start: 0, End: 0, Text: "x"})
	tt.Frame()
	if st.ed.link != ([2]int{}) {
		t.Fatalf("link %v after an edit", st.ed.link)
	}
	if len(l.asked) != 3 {
		t.Fatalf("asked %v after an edit", l.asked)
	}
}

// TestCodeEditorDefinitionLinkKeepsTheLastWhileAsking checks that the
// last link still shows while the next place waits for its answer, as
// Zed's does, and goes when that answer is that nothing is there.
func TestCodeEditorDefinitionLinkKeepsTheLastWhileAsking(t *testing.T) {
	l := newLinkTester("first second\n")
	l.hover(1, ui.Cmd)
	l.st.SetDefinitionLink(0, 5)
	l.tt.Frame()
	l.hover(8, ui.Cmd)
	if l.lastAsked().Offset != 8 {
		t.Fatalf("asked %v over \"second\"", l.asked)
	}
	if l.st.ed.link != [2]int{0, 5} {
		t.Fatalf("link %v while \"second\" waits", l.st.ed.link)
	}
	l.st.SetDefinitionLink(8, 8)
	l.tt.Frame()
	if l.st.ed.link != ([2]int{}) {
		t.Fatalf("link %v once \"second\" has nothing", l.st.ed.link)
	}
}

// TestCodeEditorDefinitionLinkShiftAsksForTheType checks that Shift held
// with Cmd asks for the type's definition, the other kind's link gone at
// once.
func TestCodeEditorDefinitionLinkShiftAsksForTheType(t *testing.T) {
	l := newLinkTester("first second\n")
	l.hover(1, ui.Cmd)
	l.st.SetDefinitionLink(0, 5)
	l.tt.Frame()
	l.hover(1, ui.Cmd|ui.Shift)
	if l.lastAsked() != (DefinitionLinkRequest{Offset: 1, Type: true}) {
		t.Fatalf("asked %v with Shift", l.asked)
	}
	if l.st.ed.link != ([2]int{}) {
		t.Fatalf("link %v of the definition with Shift held", l.st.ed.link)
	}
	l.st.SetDefinitionLink(0, 5)
	l.tt.Frame()
	if l.st.ed.link != [2]int{0, 5} {
		t.Fatalf("link %v of the type", l.st.ed.link)
	}
}

// TestCodeEditorDefinitionLinkLetGoTellsTheApp checks that letting go of
// Cmd forgets the link and tells the app once, and that Cmd held again
// asks again.
func TestCodeEditorDefinitionLinkLetGoTellsTheApp(t *testing.T) {
	l := newLinkTester("first second\n")
	l.hover(1, ui.Cmd)
	l.st.SetDefinitionLink(0, 5)
	l.tt.Frame()
	l.tt.HoldModifiers(0)
	l.tt.Frame()
	l.tt.Frame()
	if l.st.ed.link != ([2]int{}) {
		t.Fatalf("link %v after Cmd was let go", l.st.ed.link)
	}
	if len(l.asked) != 2 || l.asked[1].Offset != -1 {
		t.Fatalf("asked %v: want the letting go told once", l.asked)
	}
	l.tt.HoldModifiers(ui.Cmd)
	l.tt.Frame()
	if l.lastAsked().Offset != 1 {
		t.Fatalf("asked %v with Cmd held again", l.asked)
	}
}

// TestCodeEditorCmdShiftClickGoesToTheType checks that Cmd+Shift+click
// calls OnTypeDefinition, and Cmd+click OnDefinition.
func TestCodeEditorCmdShiftClickGoesToTheType(t *testing.T) {
	var definition, typ []int
	st := &State{Buffer: NewBuffer("first second\n"),
		OnDefinition:     func(off int) { definition = append(definition, off) },
		OnTypeDefinition: func(off int) { typ = append(typ, off) }}
	tt := ui.NewTester(func(c *ui.Context) { View(c, st).Fill() }, 600, 300)
	tt.Frame()
	x := st.ed.textX + st.ed.view.charWidth*2.5
	tt.ClickAtWith(ui.Cmd|ui.Shift, x, 5)
	tt.Frame()
	if len(typ) != 1 || typ[0] != 2 || len(definition) != 0 {
		t.Fatalf("Cmd+Shift+click: type %v, definition %v", typ, definition)
	}
}
