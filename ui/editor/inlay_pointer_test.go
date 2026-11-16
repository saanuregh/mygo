package editor

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// The pointer over an inlay hint is told with the hint's index and box,
// leaving it is told, and resting on it asks for no hover of the text
// beside it.
func TestInlayHoverTellsTheHintUnderThePointer(t *testing.T) {
	st := &State{Buffer: NewBuffer("let x = compute(1, 2)\n")}
	var told []InlayHover
	var asked []int
	st.OnInlayHover = func(h InlayHover) { told = append(told, h) }
	st.OnHover = func(r HoverRequest) { asked = append(asked, r.Offset) }
	tt := ui.NewTester(func(c *ui.Context) { View(c, st).Fill() }, 600, 200)
	// The type after x, and a parameter's name before 2.
	st.SetInlays([]Inlay{{Offset: 5, Text: ": number"}, {Offset: 19, Text: "b: "}})
	tt.Frame()
	ed := &st.ed
	y := float32(ed.view.heights.geometry(0).rows-ed.scrollY) + ed.view.lineHeight/2
	// The first hint starts after "let x", five characters in.
	cw := ed.view.charWidth
	tt.Move(ed.textX+cw*7, y)
	if len(told) != 1 || told[0].Index != 0 {
		t.Fatalf("over the first hint: told %+v", told)
	}
	box := told[0].Box
	if box.X > ed.textX+cw*5+0.5 || box.X+box.W < ed.textX+cw*13-0.5 || box.H != ed.view.lineHeight {
		t.Fatalf("the first hint's box is %+v, its text from x %g", box, ed.textX+cw*5)
	}
	tt.Advance(time.Hour)
	if len(asked) != 0 {
		t.Fatalf("resting on a hint asked for the hover of %v", asked)
	}
	// Over the text: the hint is left, and the text's hover is asked.
	tt.Move(ed.textX+cw*1.5, y)
	if len(told) != 2 || told[1].Index != -1 {
		t.Fatalf("off the hints: told %+v", told)
	}
	tt.Advance(time.Hour)
	if len(asked) != 1 {
		t.Fatalf("resting on the text asked %v", asked)
	}
	// The second hint is "b: " before 2: after "let x", ": number" and
	// " = compute(1, " (5 + 8 + 14 characters).
	tt.Move(ed.textX+cw*28, y)
	if n := len(told); told[n-1].Index != 1 {
		t.Fatalf("over the second hint: told %+v", told)
	}
	// Leaving the editor is leaving the hint.
	tt.Move(599, 199)
	tt.Move(ed.textX+cw*28, y)
	before := len(told)
	st.SetInlays([]Inlay{{Offset: 19, Text: "b: "}})
	tt.Frame()
	tt.Move(ed.textX+cw*20.5, y)
	if len(told) <= before || told[len(told)-1].Index != 0 {
		t.Fatalf("after SetInlays, the hint is told by its new index: %+v", told[before:])
	}
}

// A press on a hint goes to OnInlayClick with its modifiers; one it takes
// moves no caret, one it leaves does as on the text.
func TestInlayClickTellsTheHintAndItsModifiers(t *testing.T) {
	st := &State{Buffer: NewBuffer("let x = 1\n")}
	var clicks []InlayClick
	take := true
	st.OnInlayClick = func(c InlayClick) bool { clicks = append(clicks, c); return take }
	defined := -1
	st.OnDefinition = func(off int) { defined = off }
	tt := ui.NewTester(func(c *ui.Context) { View(c, st).Fill() }, 600, 200)
	st.SetInlays([]Inlay{{Offset: 5, Text: ": number"}})
	tt.Frame()
	ed := &st.ed
	y := float32(ed.view.heights.geometry(0).rows-ed.scrollY) + ed.view.lineHeight/2
	x := ed.textX + ed.view.charWidth*8
	tt.ClickAtWith(ui.Cmd, x, y)
	if len(clicks) != 1 || clicks[0].Index != 0 || clicks[0].Mods != ui.Cmd || clicks[0].Clicks != 1 {
		t.Fatalf("clicks %+v", clicks)
	}
	if defined != -1 || st.Cursor() != 0 {
		t.Fatalf("a press taken went on: definition at %d, caret at %d", defined, st.Cursor())
	}
	take = false
	tt.Advance(time.Second)
	tt.ClickAt(x, y)
	if len(clicks) != 2 || clicks[1].Mods != 0 {
		t.Fatalf("clicks %+v", clicks)
	}
	if st.Cursor() != 5 {
		t.Fatalf("a press left to the editor put the caret at %d, not at the hint", st.Cursor())
	}
	// Off the hints, OnInlayClick hears nothing.
	tt.Advance(time.Second)
	tt.ClickAt(ed.textX+ed.view.charWidth*1.5, y)
	if len(clicks) != 2 {
		t.Fatalf("a press on the text was told as a hint's: %+v", clicks)
	}
}

// Trailing inlays (Inlay.Trailing) are not hints to the pointer: neither
// OnInlayHover nor OnInlayClick hears of them, the text's hover is asked
// over them as upstream has it, and the hints after them in the list keep
// their indexes in it.
func TestTrailingInlaysAreNotHintsToThePointer(t *testing.T) {
	st := &State{Buffer: NewBuffer("let x = 1\n"), HoverDelay: time.Second}
	var told []InlayHover
	var clicks []InlayClick
	var asked []int
	st.OnInlayHover = func(h InlayHover) { told = append(told, h) }
	st.OnInlayClick = func(c InlayClick) bool { clicks = append(clicks, c); return true }
	st.OnHover = func(r HoverRequest) { asked = append(asked, r.Offset) }
	tt := ui.NewTester(func(c *ui.Context) { View(c, st).Fill() }, 600, 200)
	st.SetInlays([]Inlay{{Offset: 9, Text: "  You, 2 months ago", Trailing: true}, {Offset: 5, Text: ": number"}})
	tt.Frame()
	ed := &st.ed
	cw := ed.view.charWidth
	y := float32(ed.view.heights.geometry(0).rows-ed.scrollY) + ed.view.lineHeight/2
	// "let x" ": number" " = 1" then the trailing text: 5 + 8 + 4 characters.
	trailing := ed.textX + cw*(17+6)
	tt.Move(trailing, y)
	tt.Advance(1500 * time.Millisecond)
	tt.Frame()
	if len(told) != 0 {
		t.Fatalf("over a trailing inlay, OnInlayHover was told %+v", told)
	}
	if len(asked) != 1 || asked[0] != 9 {
		t.Fatalf("over a trailing inlay, the text's hover asked %v, want [9]", asked)
	}
	tt.Advance(time.Second)
	tt.ClickAt(trailing, y)
	if len(clicks) != 0 {
		t.Fatalf("a press on a trailing inlay was told as a hint's: %+v", clicks)
	}
	// The hint is the second of the list given to SetInlays.
	tt.Move(ed.textX+cw*8, y)
	if len(told) != 1 || told[0].Index != 1 {
		t.Fatalf("over the hint after a trailing inlay in the list: told %+v", told)
	}
}
