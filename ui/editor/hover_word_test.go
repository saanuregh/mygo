package editor

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// With HoverAppOnly (zode's hover_card.zig): the wait (HoverDelay, zode's
// 450 ms) runs on across a name's characters, starting again on another
// name only, and nothing rests on punctuation.
func TestHoverWaitsByNameAppOnly(t *testing.T) {
	st := &State{Buffer: NewBuffer("alpha (beta)\n"), HoverAppOnly: true, HoverDelay: 450 * time.Millisecond}
	var asked []int
	st.OnHover = func(r HoverRequest) { asked = append(asked, r.Offset) }
	tt := ui.NewTester(func(c *ui.Context) { View(c, st).Fill().AutoFocus() }, 600, 300)
	tt.Frame()
	ed := &st.ed
	y := float32(ed.view.heights.geometry(0).rows-ed.scrollY) + ed.view.lineHeight/2
	cw := ed.view.charWidth
	at := func(col int) float32 { return ed.textX + cw*float32(col) + cw/2 }
	tt.Move(at(1), y)
	tt.Advance(300 * time.Millisecond)
	tt.Move(at(3), y) // the same name: the wait goes on
	tt.Advance(200 * time.Millisecond)
	if len(asked) != 1 || asked[0] != 0 {
		t.Fatalf("asked %v, want [0] (alpha's start) after 500 ms on alpha", asked)
	}
	tt.Move(at(6), y) // on "(": nothing rests
	tt.Advance(time.Second)
	if len(asked) != 1 {
		t.Fatalf("asked %v on punctuation", asked)
	}
}
