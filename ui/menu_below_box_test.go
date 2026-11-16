package ui

import "testing"

// OpenPopoverMenuBelow opens below the element, and above it where the
// window has no room below: never over it.
func TestOpenPopoverMenuBelowNeverCoversTheElement(t *testing.T) {
	for _, top := range []float32{20, 360} {
		open := true
		var box Rect
		tt := coreNewTester(func(c *context) {
			b := coreBox(c).Absolute().Left(50).Top(top).Size(2, 18)
			build := func(m *Menu) { m.Item("Fix it"); m.Item("Refactor") }
			b.PopoverMenu(build)
			if open {
				open = false
				b.OpenPopoverMenuBelow(build)
			}
			box = b.Bounds()
		}, 400, 400)
		tt.Frame()
		tt.Frame()
		item, ok := tt.Find("Fix it")
		if !ok {
			t.Fatalf("top %v: no menu: %q", top, tt.Texts())
		}
		last, _ := tt.Find("Refactor")
		below := item.Y >= box.Y+box.H
		above := last.Y+last.H <= box.Y
		if !below && !above || (top == 20) != below {
			t.Fatalf("top %v: box %v, items %v %v", top, box, item, last)
		}
	}
}
