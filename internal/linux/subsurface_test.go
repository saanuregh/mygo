//go:build linux && !android && (amd64 || arm64)

package linux

import (
	"image"
	"testing"
)

func TestSubsurfacePlacement(t *testing.T) {
	cases := []struct {
		name       string
		inToplevel image.Point
		area       gdkRectangle
		toplevel   gdkRectangle
		want       image.Rectangle
	}{
		{
			name:     "no client-side decorations",
			area:     gdkRectangle{Width: 720, Height: 520},
			toplevel: gdkRectangle{Width: 720, Height: 520},
			want:     image.Rect(0, 0, 720, 520),
		},
		{
			// GTK's title bar above the area, the shadow around the window.
			name:       "client-side decorations with a shadow",
			inToplevel: image.Pt(0, 37),
			area:       gdkRectangle{X: 23, Y: 60, Width: 360, Height: 260},
			toplevel:   gdkRectangle{X: 23, Y: 23, Width: 360, Height: 297},
			want:       image.Rect(23, 60, 383, 320),
		},
		{
			name:       "maximized, without a shadow",
			inToplevel: image.Pt(0, 37),
			area:       gdkRectangle{Y: 37, Width: 1920, Height: 1043},
			toplevel:   gdkRectangle{Width: 1920, Height: 1080},
			want:       image.Rect(0, 37, 1920, 1080),
		},
		{
			name:       "a menu bar and a side shadow",
			inToplevel: image.Pt(0, 64),
			area:       gdkRectangle{X: 12, Y: 76, Width: 400, Height: 300},
			toplevel:   gdkRectangle{X: 12, Y: 12, Width: 400, Height: 364},
			want:       image.Rect(12, 76, 412, 376),
		},
		{
			name:     "not allocated yet",
			area:     gdkRectangle{Width: 0, Height: -1},
			toplevel: gdkRectangle{},
			want:     image.Rect(0, 0, 1, 1),
		},
	}
	for _, c := range cases {
		if got := subsurfacePlacement(c.inToplevel, c.area, c.toplevel); got != c.want {
			t.Errorf("%s: subsurfacePlacement = %v, want %v", c.name, got, c.want)
		}
	}
}
