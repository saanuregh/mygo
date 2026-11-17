package ui

import "testing"

// ScrollbarStyle.HoverWidth: a vertical thumb widens to it with the
// pointer within 10 DIPs of the container's right edge (nanoui's 4 to 7).
func TestScrollbarHoverWidthUnderThePointerAtTheEdge(t *testing.T) {
	var sc *node
	tt := coreNewTester(func(c *context) {
		th := DarkTheme()
		th.ScrollbarWidth = 4
		th.ScrollbarStyle.HoverWidth = 7
		c.SetTheme(th)
		sc = coreScroll(c).Size(100, 200).KeepScrollbars(true).Children(func() {
			coreBox(c).Size(100, 800)
		})
	}, 200, 200)
	tt.Move(50, 100)
	tt.Frame()
	if sc.st.barWide {
		t.Fatal("wide with the pointer away from the edge")
	}
	tt.Move(95, 100)
	tt.Frame()
	if !sc.st.barWide {
		t.Fatal("not wide with the pointer at the edge")
	}
}
