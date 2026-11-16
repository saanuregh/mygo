package ui

import (
	"slices"
	"testing"
)

// DragBare drags the value with no copy of the element under the
// pointer: the app draws what follows it (zode's tab chip).
func TestDragBarePaintsNoCopy(t *testing.T) {
	for _, bare := range []bool{false, true} {
		var dropped []string
		tt := coreNewTester(func(c *context) {
			coreColumn(c).Gap(20).Padding(20).Children(func() {
				src := coreBox(c).Size(100, 40).Label("Source").Background(RGB(255, 0, 0)).Drag("hello")
				if bare {
					src.DragBare()
				}
				target := coreBox(c).Size(200, 100).Label("Target")
				if v, ok := coreDrop[string](target); ok {
					dropped = append(dropped, v)
				}
			})
		}, 400, 400)
		src, _ := tt.Find("Source")
		target, _ := tt.Find("Target")
		tt.Press(src.X+10, src.Y+10)
		tt.Move(src.X+30, src.Y+30)
		tt.Move(target.X+20, target.Y+20)
		c := tt.Image().RGBAAt(int(target.X+40), int(target.Y+20))
		if copied := c.R > 150 && c.G < 100; copied == bare {
			t.Errorf("bare %v: a copy under the pointer %v (%v)", bare, copied, c)
		}
		tt.Release(target.X+20, target.Y+20)
		if !slices.Equal(dropped, []string{"hello"}) {
			t.Fatalf("bare %v: dropped %q", bare, dropped)
		}
		tt.rt.close()
	}
}
