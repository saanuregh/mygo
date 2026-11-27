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

// A subsurface's frames are paced by its frame callbacks, faked here: a
// frame asked for while no callback is due draws at once, from the idle
// callback (soon); those asked for while one is due draw once, as it comes;
// one that showed nothing asks for none, and the next waits a refresh
// (later) rather than spin; a subsurface gone stops the wait.
func TestSubsurfaceFramesWaitForTheCallback(t *testing.T) {
	var soon, later []func()
	frames, show := 0, true
	var p *framePacer
	p = &framePacer{
		soon:  func() { soon = append(soon, p.run) },
		later: func() { later = append(later, p.run) },
		draw:  func() bool { frames++; return show },
	}
	runSoon := func() {
		t.Helper()
		if len(later) != 0 {
			t.Fatalf("%d frames wait a refresh", len(later))
		}
		q := soon
		soon = nil
		for _, run := range q {
			run()
		}
	}
	want := func(n int) {
		t.Helper()
		if frames != n {
			t.Fatalf("%d frames drawn, want %d", frames, n)
		}
	}

	p.request()
	if len(soon) != 1 {
		t.Fatalf("a frame asked for with no callback due queued %d runs, want 1", len(soon))
	}
	runSoon()
	want(1)

	// A callback is due: frames asked for meanwhile wait for it.
	p.request()
	p.request()
	if len(soon) != 0 {
		t.Fatalf("frames asked for while a callback is due queued %d runs", len(soon))
	}
	runSoon()
	want(1)
	p.done()
	runSoon()
	want(2)

	// Done with no frame asked for draws none; the next draws at once.
	p.done()
	runSoon()
	want(2)
	p.request()
	runSoon()
	want(3)

	// A frame asked for as one draws waits for that one's callback.
	p.done()
	p.draw = func() bool { frames++; p.request(); return true }
	p.request()
	runSoon()
	want(4)
	runSoon()
	want(4)
	p.draw = func() bool { frames++; return show }
	p.done()
	runSoon()
	want(5)

	// A frame that showed nothing, and asked for another: no callback
	// comes, the next waits a refresh.
	p.done()
	show = false
	p.draw = func() bool { frames++; p.request(); return show }
	p.request()
	q := soon
	soon = nil
	q[0]()
	want(6)
	if len(soon) != 0 || len(later) != 1 {
		t.Fatalf("after a frame that showed nothing: %d runs at once, %d after a refresh, want 0 and 1", len(soon), len(later))
	}
	p.draw = func() bool { frames++; return true }
	later[0]()
	later = nil
	want(7)

	// The subsurface goes with a callback due, which never comes: the next
	// frame draws at once.
	p.stop()
	p.request()
	runSoon()
	want(8)
}
