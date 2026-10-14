package ui

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// A backend that does not count clicks, as Linux's and Windows', leaves
// the count to the engine: a press and its release carry it, and each
// button counts its own.
func TestEngineCountsClicks(t *testing.T) {
	var clicks []int
	tt := NewTester(func(c *Context) {
		Box(c).Size(100, 100).HandleInput(func(ev InputEvent) bool {
			if ev.Kind == InputPointerDown || ev.Kind == InputPointerUp {
				clicks = append(clicks, ev.Clicks)
			}
			return true
		})
	}, 100, 100)
	defer tt.rt.close()
	tt.Frame()
	tt.ClickAt(50, 50)
	tt.ClickAt(50, 50)
	tt.RightClickAt(50, 50)
	tt.ClickAt(50, 50)
	if want := []int{1, 1, 2, 2, 1, 1, 1, 1}; !slices.Equal(clicks, want) {
		t.Fatalf("presses and releases saw clicks %v, want %v", clicks, want)
	}
}

// A press a handler leaves is counted once, not again for the element
// under it.
func TestHandlerLeavesPressCountedOnce(t *testing.T) {
	doubles := 0
	tt := NewTester(func(c *Context) {
		b := Button(c, "Open").HandleInput(func(InputEvent) bool { return false })
		if b.DoubleClicked() {
			doubles++
		}
	}, 200, 100)
	defer tt.rt.close()
	tt.Frame()
	if err := tt.Click("Open"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if doubles != 0 {
		t.Fatal("one click was a double click")
	}
	if err := tt.Click("Open"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if doubles != 1 {
		t.Fatalf("%d double clicks after two clicks, want 1", doubles)
	}
}

// A release carries its press's count, though another button went down
// meanwhile.
func TestReleaseCarriesPressCount(t *testing.T) {
	var ups []int
	tt := NewTester(func(c *Context) {
		Box(c).Size(100, 100).HandleInput(func(ev InputEvent) bool {
			if ev.Kind == InputPointerUp && ev.Button == 0 {
				ups = append(ups, ev.Clicks)
			}
			return ev.Button == 0
		})
	}, 100, 100)
	defer tt.rt.close()
	tt.Frame()
	tt.ClickAt(50, 50)
	tt.Press(50, 50)
	tt.send(platform.SurfaceEvent{Kind: platform.PointerDown, X: 50, Y: 50, Button: 1})
	tt.Release(50, 50)
	if want := []int{1, 2}; !slices.Equal(ups, want) {
		t.Fatalf("releases saw clicks %v, want %v", ups, want)
	}
}

// A click that puts another element under the pointer does not make the
// next click on it a double click, inside a handler that leaves presses.
func TestClickOnNewElementIsNotDouble(t *testing.T) {
	opened := false
	doubles := 0
	tt := NewTester(func(c *Context) {
		Box(c).Size(200, 100).HandleInput(func(InputEvent) bool { return false }).Children(func() {
			if !opened {
				opened = Button(c.Key("open"), "Open").Size(100, 40).Clicked()
			} else if Button(c.Key("row"), "Row").Size(100, 40).DoubleClicked() {
				doubles++
			}
		})
	}, 200, 100)
	defer tt.rt.close()
	tt.Frame()
	tt.ClickAt(50, 20)
	tt.Frame()
	tt.Frame()
	if !opened {
		t.Fatal("the click did not open")
	}
	tt.ClickAt(50, 20)
	tt.Frame()
	if doubles != 0 {
		t.Fatal("the first click on the new element was a double click")
	}
}
