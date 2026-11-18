package web

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

func TestIframeClipPath(t *testing.T) {
	r := func(x, y, w, h float64) platform.RectF { return platform.RectF{X: x, Y: y, W: w, H: h} }
	for _, tc := range []struct {
		name string
		p    platform.WebViewPlacement
		want string
	}{
		{"whole", platform.WebViewPlacement{Frame: r(5, 5, 100, 100), Clip: r(5, 5, 100, 100)}, "none"},
		{
			// Two menus over the page: the rows around them, in the page's
			// coordinates, from the frame's corner.
			"two covers",
			platform.WebViewPlacement{Frame: r(10, 20, 100, 100), Clip: r(10, 20, 100, 100),
				Covers: []platform.RectF{r(20, 30, 20, 20), r(70, 80, 20, 20)}},
			`path("M0 0H100V10H0Z M0 10H10V30H0Z M30 10H100V30H30Z M0 30H100V60H0Z M0 60H60V80H0Z M80 60H100V80H80Z M0 80H100V100H0Z")`,
		},
		{
			// Overlapping covers leave no gap and cut nothing twice.
			"overlapping covers",
			platform.WebViewPlacement{Frame: r(0, 0, 100, 100), Clip: r(0, 0, 100, 100),
				Covers: []platform.RectF{r(0, 0, 60, 100), r(40, 0, 60, 50)}},
			`path("M60 50H100V100H60Z")`,
		},
		{
			// Scaled to half: the clip is in the surface's DIPs, the path in
			// the page's.
			"scaled",
			platform.WebViewPlacement{Frame: r(10, 20, 50, 50), Clip: r(10, 20, 50, 25), Width: 100, Height: 100},
			`path("M0 0H100V50H0Z")`,
		},
		{"covered", platform.WebViewPlacement{Frame: r(0, 0, 10, 10), Clip: r(0, 0, 10, 10), Covers: []platform.RectF{r(-5, -5, 20, 20)}}, "polygon(0 0,0 0,0 0)"},
	} {
		if got := iframeClipPath(&tc.p); got != tc.want {
			t.Errorf("%s:\n got %s\nwant %s", tc.name, got, tc.want)
		}
	}
}
