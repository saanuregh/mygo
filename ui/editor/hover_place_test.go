package editor

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// hoverFixture is a zode editor (HoverBelow, HoverAppOnly) answering a
// hover with text, the pointer resting at column col of line.
func hoverFixture(t *testing.T, text string, w, h int) (*State, *ui.Tester, func(col, line int) (float32, float32)) {
	t.Helper()
	st := &State{Buffer: NewBuffer(text), HoverAppOnly: true, HoverBelow: true}
	st.OnHover = func(r HoverRequest) { st.ShowHover(r.Offset, r.Offset+3, "a card", nil) }
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, st).Fill()
		st.RenderHover = func(c *ui.Context, panel ui.Element, card *HoverCard) {
			panel.Label("Hover card").Padding(10, 12).Children(func() { ui.Text(c, card.Text()).Label("Card") })
		}
	}, w, h)
	tt.Frame()
	ed := &st.ed
	at := func(col, line int) (float32, float32) {
		return ed.textX + ed.view.charWidth*(float32(col)+0.5), float32(ed.view.heights.geometry(line).rows-ed.scrollY) + ed.view.lineHeight/2
	}
	return st, tt, at
}

// hover_card.zig's place: from 10 above the pointer, 26 high, the card 4
// below that (20 below the pointer), 12 left of it, kept 8 inside the
// window; where below has too little room, above.
func TestHoverBelowIsPlacedAsZodesCard(t *testing.T) {
	st, tt, at := hoverFixture(t, strings.Repeat("one two\n", 40), 1200, 300)
	x, y := at(1, 0)
	tt.Move(x, y)
	tt.Advance(time.Second)
	tt.Frame()
	card, ok := tt.Find("Hover card")
	if !st.ed.hover.shown || !ok {
		t.Fatalf("no card: %q", tt.Texts())
	}
	// x clamped as if the card were its widest: min(600, window − 16).
	wantX := min(max(8, x-12), 1200-600-8)
	if abs(card.Y-(y+20)) > 0.5 || abs(card.X-wantX) > 0.5 {
		t.Fatalf("card at %v; pointer %v,%v: want x %v y %v", card, x, y, wantX, y+20)
	}
	// Near the window's bottom: above the pointer's rect, 4 apart.
	line := 0
	for ; ; line++ {
		if _, ly := at(1, line); ly > 250 {
			break
		}
	}
	x, y = at(1, line)
	tt.Move(x, y)
	tt.Advance(time.Second)
	tt.Frame()
	tt.Frame()
	if card, _ := tt.Find("Hover card"); !st.ed.hover.shown || abs(card.Y+card.H-(y-10-4)) > 0.5 {
		t.Fatalf("near the bottom (pointer at %v) the card is at %v, shown %v, %q", y, card, st.ed.hover.shown, tt.Texts())
	}
}

// A press on the word hovered keeps its card (hover_card.zig trackPointer:
// it goes only as the word under the pointer changes).
func TestAClickOnTheWordHoveredKeepsTheCard(t *testing.T) {
	st, tt, at := hoverFixture(t, "alpha beta\n", 400, 300)
	x, y := at(2, 0)
	tt.Move(x, y)
	tt.Advance(time.Second)
	tt.Frame()
	if !st.ed.hover.shown {
		t.Fatal("no card")
	}
	tt.ClickAt(x, y)
	tt.Frame()
	if !st.ed.hover.shown {
		t.Fatal("a click on the word closed its card")
	}
	x2, _ := at(8, 0)
	tt.ClickAt(x2, y)
	tt.Frame()
	tt.Frame()
	if st.ed.hover.shown {
		t.Fatal("a click on another word kept the card")
	}
}

// Asked by the keyboard (ShowHover at the caret), the card goes below the
// caret's line, not where the pointer was.
func TestHoverBelowFromTheCaret(t *testing.T) {
	st, tt, at := hoverFixture(t, strings.Repeat("one two\n", 40), 1200, 300)
	st.Select(4, 4)
	tt.Frame()
	st.Run(ShowHover)
	tt.Frame()
	tt.Frame()
	card, ok := tt.Find("Hover card")
	_, y := at(4, 0)
	bottom := y + st.ed.view.lineHeight/2
	if !ok || abs(card.Y-(bottom+4)) > 0.5 {
		t.Fatalf("card %v, shown %v; want its top 4 below the caret's line (%v)", card, ok, bottom+4)
	}
}

// With HoverAppOnly, a rest on the next name goes on as the card of the
// one before goes: no new wait starts from the card's going.
func TestHoverRestOnTheNextNameGoesOnAsTheCardGoes(t *testing.T) {
	st, tt, at := hoverFixture(t, "alpha beta\n", 600, 300)
	asked := 0
	st.OnHover = func(r HoverRequest) { asked++; st.ShowHover(r.Offset, r.Offset+4, "a card", nil) }
	x, y := at(1, 0)
	tt.Move(x, y)
	tt.Advance(time.Second)
	tt.Frame()
	if !st.ed.hover.shown || asked != 1 {
		t.Fatalf("no card on alpha: asked %d", asked)
	}
	x, _ = at(7, 0)
	tt.Move(x, y)
	tt.Frame()
	tt.Frame()
	if st.ed.hover.shown {
		t.Fatal("alpha's card stayed off alpha")
	}
	tt.Advance(time.Second)
	tt.Frame()
	if asked != 2 || !st.ed.hover.shown || st.ed.hover.start != 6 {
		t.Fatalf("on beta: asked %d, shown %v at %d", asked, st.ed.hover.shown, st.ed.hover.start)
	}
}
