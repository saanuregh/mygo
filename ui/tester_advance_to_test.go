package ui

import (
	"testing"
	"time"
)

// AdvanceTo moves the clock on to a later time, running the frames due on
// the way, and leaves it at an earlier one.
func TestAdvanceToMovesTheClockOnToALaterTimeOnly(t *testing.T) {
	target := float32(0)
	var value float32
	tt := NewTester(func(c *Context) {
		value = Box(c).Animate("x", target, 100*time.Millisecond)
	}, 100, 50)
	start := tt.Now()
	target = 100
	tt.Frame()
	tt.AdvanceTo(start.Add(time.Second))
	if !tt.Now().Equal(start.Add(time.Second)) || value != 100 {
		t.Fatalf("the clock at %v after the start, the value at %v", tt.Now().Sub(start), value)
	}
	tt.AdvanceTo(start)
	if !tt.Now().Equal(start.Add(time.Second)) {
		t.Fatalf("an earlier time moved the clock back to %v after the start", tt.Now().Sub(start))
	}
}

// Frames show the Tester's time of day (Context.Wall), which Advance moves
// until AdvanceTo sets it: from then on Advance moves what animates and
// waits, not the ages a snapshot shows.
func TestAdvanceToSetsTheWallClockApart(t *testing.T) {
	var wall time.Time
	tt := NewTester(func(c *Context) { wall = c.Wall() }, 100, 50)
	start := tt.Now()
	tt.Advance(time.Minute)
	tt.Frame()
	if !wall.Equal(start.Add(time.Minute)) {
		t.Fatalf("before AdvanceTo the wall clock is %v after the start, want the Tester's minute", wall.Sub(start))
	}
	at := start.Add(2 * time.Minute)
	tt.AdvanceTo(at)
	tt.Advance(10 * time.Minute)
	tt.Frame()
	if !wall.Equal(at) || !tt.Wall().Equal(at) {
		t.Fatalf("after AdvanceTo and Advance the wall clock is %v after the start, want 2m", wall.Sub(start))
	}
	if !tt.Now().Equal(at.Add(10 * time.Minute)) {
		t.Fatalf("the animation clock is %v after the start, want 12m", tt.Now().Sub(start))
	}
}
