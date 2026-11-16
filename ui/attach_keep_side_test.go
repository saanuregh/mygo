package ui

import "testing"

// KeepSide: a panel below its target stays below, though the window ends
// first.
func TestAttachToKeepSide(t *testing.T) {
	keep := false
	tt := coreNewTester(func(c *context) {
		b := coreButtonBase(c).Size(40, 20).Label("anchor").Absolute().Top(100).Left(100)
		coreOverlay(c, func() {
			p := coreBox(c).Size(60, 30).Label("panel").AttachTo(b, AnchorBottomLeft, AnchorTopLeft).Margin(4, 0, 0, 0)
			if keep {
				p.KeepSide()
			}
		})
	}, 400, 140)
	tt.Frame()
	if r, _ := tt.Find("panel"); r.Y != 66 {
		t.Fatalf("without KeepSide: %v, want above", r)
	}
	keep = true
	tt.Frame()
	if r, _ := tt.Find("panel"); r.Y != 124 {
		t.Fatalf("KeepSide: %v, want below", r)
	}
}
