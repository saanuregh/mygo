package ui

import "testing"

// The Tester's pointer carries the modifiers HoldModifiers left held, as a
// platform's events do: Ctrl held over a word stays held as the pointer
// moves, and Shift+wheel scrolls sideways.
func TestHoldModifiersRideThePointer(t *testing.T) {
	var mods Modifiers
	tt := coreNewTester(func(c *context) {
		mods = c.Modifiers()
		coreBox(c).Size(100, 100)
	}, 200, 200)
	tt.HoldModifiers(Ctrl)
	tt.Move(50, 50)
	tt.Frame()
	if mods != Ctrl {
		t.Fatalf("after a move, the frame saw %v, want Ctrl held", mods)
	}
	tt.Scroll(50, 50, 0, 5)
	tt.Frame()
	if mods != Ctrl {
		t.Fatalf("after a scroll, the frame saw %v, want Ctrl held", mods)
	}
	tt.HoldModifiers(0)
	tt.Move(60, 60)
	tt.Frame()
	if mods != 0 {
		t.Fatalf("let go, the frame saw %v", mods)
	}
}
