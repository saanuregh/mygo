package ui

import (
	"testing"
	"time"
)

// TestFadingScrollbars shows the bars only as the content scrolls: not
// on hover, opaque for 0.8 s after a scroll, gone 0.3 s later.
func TestFadingScrollbars(t *testing.T) {
	var sc *node
	tt := coreNewTester(func(c *context) {
		sc = coreScroll(c).Fill().FadingScrollbars().Children(func() {
			coreBox(c).Height(1000).Shrink(0)
		})
	}, 200, 200)
	tt.Move(100, 100)
	tt.Frame()
	if sc.st.barsShown {
		t.Fatal("shown on hover")
	}
	tt.Scroll(100, 100, 0, 60)
	tt.Frame()
	if !sc.st.barsShown {
		t.Fatal("not shown while scrolling")
	}
	tt.Advance(500 * time.Millisecond)
	tt.Frame()
	if !sc.st.barsShown {
		t.Fatal("gone before 0.8 s")
	}
	tt.Advance(time.Second)
	tt.Frame()
	if sc.st.barsShown {
		t.Fatal("still shown 1.5 s after")
	}
}

// ScrollbarStyle's FadeAfter and FadeOver are what FadingScrollbars sets;
// a theme's style fades every container's bars.
func TestScrollbarStyleFadeFromTheTheme(t *testing.T) {
	var sc *node
	tt := coreNewTester(func(c *context) {
		th := DarkTheme()
		th.ScrollbarStyle.FadeAfter, th.ScrollbarStyle.FadeOver = 200*time.Millisecond, 100*time.Millisecond
		c.SetTheme(th)
		sc = coreScroll(c).Fill().Children(func() {
			coreBox(c).Height(1000).Shrink(0)
		})
	}, 200, 200)
	tt.Move(100, 100)
	tt.Frame()
	if sc.st.barsShown {
		t.Fatal("shown on hover")
	}
	tt.Scroll(100, 100, 0, 60)
	tt.Frame()
	if !sc.st.barsShown {
		t.Fatal("not shown while scrolling")
	}
	tt.Advance(400 * time.Millisecond)
	tt.Frame()
	if sc.st.barsShown {
		t.Fatal("still shown past FadeAfter and FadeOver")
	}
}
