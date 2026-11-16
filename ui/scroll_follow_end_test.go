package ui

import "testing"

// TestScrollFollowEnd starts a FollowEnd container at its end, keeps it
// there as the content grows, stops when scrolled up and follows again
// once back at the end.
func TestScrollFollowEnd(t *testing.T) {
	var s ScrollState
	height := float32(1000)
	tt := coreNewTester(func(c *context) {
		coreScroll(c).Fill().TrackScroll(&s).FollowEnd().Children(func() {
			coreBox(c).Height(height).Shrink(0)
		})
	}, 200, 200)
	tt.Frame()
	if s.Y != 800 {
		t.Fatalf("at the start: %+v, want the end", s)
	}
	height = 1200
	tt.Frame()
	if s.Y != 1000 {
		t.Errorf("grown by 200: %+v, want the new end", s)
	}
	if tt.Frame(); tt.rt.animating {
		t.Errorf("still animating at rest")
	}
	tt.Move(100, 100)
	tt.Scroll(100, 100, 0, -300)
	height = 1400
	tt.Frame()
	if s.Y != 700 {
		t.Errorf("scrolled up, then grown: %+v, want it left at 700", s)
	}
	s.Y = 1200
	tt.Frame()
	height = 1500
	tt.Frame()
	if s.Y != 1300 {
		t.Errorf("back at the end, then grown: %+v, want 1300", s)
	}
}
