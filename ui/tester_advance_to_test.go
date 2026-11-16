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
