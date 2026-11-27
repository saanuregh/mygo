//go:build linux && !android && (amd64 || arm64)

package linux

import (
	"image"
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// The opaque region is the subsurface less the holes web views show in:
// the pixels their clips cover at all, cut out of what was left by the
// ones before; none in a transparent window.
func TestOpaqueRegionMinusHoles(t *testing.T) {
	place := func(clips ...platform.RectF) []platform.WebViewPlacement {
		var p []platform.WebViewPlacement
		for _, c := range clips {
			p = append(p, platform.WebViewPlacement{Frame: c, Clip: c})
		}
		return p
	}
	cases := []struct {
		name        string
		placed      []platform.WebViewPlacement
		transparent bool
		want        []image.Rectangle
	}{
		{name: "no web views", want: []image.Rectangle{image.Rect(0, 0, 2058, 1400)}},
		{name: "transparent", transparent: true},
		{
			name:   "a page in the middle",
			placed: place(platform.RectF{X: 1126, Y: 122, W: 892, H: 1251}),
			want: []image.Rectangle{
				image.Rect(0, 0, 2058, 122), image.Rect(0, 1373, 2058, 1400),
				image.Rect(0, 122, 1126, 1373), image.Rect(2018, 122, 2058, 1373),
			},
		},
		{
			// Pixels the clip covers in part are left out.
			name:   "a fractional clip at the right edge",
			placed: place(platform.RectF{X: 1126.4, Y: 122.5, W: 1000, H: 1250.2}),
			want: []image.Rectangle{
				image.Rect(0, 0, 2058, 122), image.Rect(0, 1373, 2058, 1400),
				image.Rect(0, 122, 1126, 1373),
			},
		},
		{
			name:   "two pages",
			placed: place(platform.RectF{X: 0, Y: 0, W: 1029, H: 1400}, platform.RectF{X: 1029, Y: 700, W: 1029, H: 700}),
			want:   []image.Rectangle{image.Rect(1029, 0, 2058, 700)},
		},
		{name: "a page over all of it", placed: place(platform.RectF{X: -5, Y: -5, W: 3000, H: 3000})},
	}
	for _, c := range cases {
		got := opaqueRegion(2058, 1400, c.placed, c.transparent)
		if !slices.Equal(got, c.want) {
			t.Errorf("%s: opaqueRegion = %v, want %v", c.name, got, c.want)
		}
		// No rectangle overlaps another or a hole.
		for i, r := range got {
			for _, o := range got[i+1:] {
				if r.Overlaps(o) {
					t.Errorf("%s: %v overlaps %v", c.name, r, o)
				}
			}
			for _, p := range c.placed {
				g := gdkRectOut(p.Clip)
				if hole := image.Rect(int(g.X), int(g.Y), int(g.X+g.Width), int(g.Y+g.Height)); r.Overlaps(hole) {
					t.Errorf("%s: %v overlaps the hole %v", c.name, r, hole)
				}
			}
		}
	}
}

// A subsurface whose frames are opaque tells the compositor so, the holes
// of its web views cut out, in the commit of the frame that places them,
// within that frame's sync transaction. GDK asks for the toplevel's frame
// callbacks on it too, as the compositor may send none for a toplevel it
// covers, and the subsurface commits that request after each after-paint;
// it stops before the subsurface goes. Where GDK cannot, it has no opaque
// region.
func TestOpaqueRegionKeepsGDKsFrameCallbacks(t *testing.T) {
	f := newFakeWayland()
	f.names[0xA00] = "toplevel"
	area := gdkRectangle{Width: 2058, Height: 1400}
	var soon []func()
	var places []platform.WebViewPlacement
	moves := false
	u := &subsurface{
		s:   &surface{w: &window{id: 7, h: fakeSurfaceHandler{}}, layers: 1},
		c:   f.client(),
		g:   &waylandGlobals{compositor: 1, subcompositor: 2, viewporter: 3, fractionManager: 4, compositorVersion: 4},
		egl: f.egl(),
		top: subsurfaceToplevel{
			surface: func() ptr { return 0xA00 },
			allocation: func() (image.Point, gdkRectangle, gdkRectangle, int) {
				return image.Point{}, area, area, 2
			},
			queuePaint:    func() {},
			watchPaints:   func() func() { return func() {} },
			requestLayout: func() {},
			frameCallbacks: func(surface ptr, add bool) bool {
				what := "GDK stops asking for frames on "
				if add {
					what = "GDK asks for frames on "
				}
				f.log = append(f.log, what+f.name(surface))
				return true
			},
		},
	}
	u.sync = subsurfaceSync{
		setSync:    func() { u.request(u.sub, wlSubsurfaceSetSync) },
		setDesync:  func() { u.request(u.sub, wlSubsurfaceSetDesync) },
		queuePaint: func() {},
	}
	u.pacer = framePacer{
		soon:  func() { soon = append(soon, u.pacer.run) },
		later: func() { t.Fatal("a frame waits a refresh") },
		draw: func() bool {
			if moves {
				moves = false
				u.s.placed = places
				u.webViewsPlaced()
			}
			return u.present()
		},
	}
	frameClock := func() {
		u.paint()
		f.log = append(f.log, "commit toplevel")
		u.afterPaint()
	}
	want := func(steps ...string) {
		t.Helper()
		got := f.take()
		if !slices.Equal(got, steps) {
			t.Fatalf("requests:\n  %s\nwant:\n  %s", strings.Join(got, "\n  "), strings.Join(steps, "\n  "))
		}
	}

	u.create()
	got := f.take()
	if i := slices.Index(got, "GDK asks for frames on surface#1"); i < 0 || !slices.Contains(got[i:], "surface#1.set_opaque_region(region#2)") {
		t.Fatalf("made:\n  %s", strings.Join(got, "\n  "))
	}
	if !slices.Contains(got, "region#2.add(0, 0, 2058, 1400)") {
		t.Fatalf("made without the whole area opaque:\n  %s", strings.Join(got, "\n  "))
	}
	frameClock()
	want(
		"callback#1 = surface#1.frame()",
		"commit surface#1",
		"commit toplevel",
		"subsurface#1.set_desync()",
		"surface#1.commit()",
	)
	u.frameDone(u.frameCB)
	f.take()

	// A page placed: the region less its hole goes with the frame, in its
	// sync transaction.
	places = []platform.WebViewPlacement{{Frame: platform.RectF{X: 1126, Y: 122, W: 892, H: 1251}, Clip: platform.RectF{X: 1126, Y: 122, W: 892, H: 1251}}}
	moves = true
	u.pacer.request()
	for len(soon) > 0 {
		run := soon[0]
		soon = soon[1:]
		run()
	}
	want(
		"region#3 = compositor.create_region()",
		"region#3.add(0, 0, 2058, 122)",
		"region#3.add(0, 1373, 2058, 27)",
		"region#3.add(0, 122, 1126, 1251)",
		"region#3.add(2018, 122, 40, 1251)",
		"surface#1.set_opaque_region(region#3)",
		"region#3.destroy()",
		"subsurface#1.set_sync()",
		"callback#2 = surface#1.frame()",
		"commit surface#1",
	)
	// The same placement again, in the same transaction, sends nothing.
	u.webViewsPlaced()
	want()
	frameClock()
	want("commit toplevel", "subsurface#1.set_desync()", "surface#1.commit()")

	// A resize sends the new region with the frame drawn for it.
	area = gdkRectangle{Width: 1600, Height: 1000}
	u.s.placed = nil
	u.place()
	want(
		"region#4 = compositor.create_region()",
		"region#4.add(0, 0, 1600, 1000)",
		"surface#1.set_opaque_region(region#4)",
		"region#4.destroy()",
		"wl_egl_window_resize(egl_window#1, 3200, 2000)",
	)

	// GDK stops before the wl_surface goes.
	u.destroy()
	got = f.take()
	stop, gone := slices.Index(got, "GDK stops asking for frames on surface#1"), slices.Index(got, "surface#1.destroy()")
	if stop < 0 || gone < stop {
		t.Fatalf("destroyed:\n  %s", strings.Join(got, "\n  "))
	}

	// Without GDK's help: no opaque region, no extra commit.
	u.top.frameCallbacks = func(surface ptr, add bool) bool { return false }
	u.failed = false
	u.create()
	for _, r := range f.take() {
		if strings.Contains(r, "set_opaque_region") {
			t.Fatalf("an opaque region without GDK's frame callbacks: %s", r)
		}
	}
	frameClock()
	if got := f.take(); slices.Contains(got, "surface#2.commit()") {
		t.Fatalf("committed after the after-paint without an opaque region:\n  %s", strings.Join(got, "\n  "))
	}
}

// A subsurface synchronized for a frame while GTK cannot paint the area (it
// has no size, say) goes back to desynchronized after GDK's next commit of
// the toplevel, before it commits GDK's request for the toplevel's frame
// callback on it: held synchronized, that request would wait for a commit
// of the toplevel, which GDK makes only once that callback came, and GTK's
// frame clock would stop. With nothing painted, GDK commits nothing, and
// it stays synchronized.
func TestSubsurfaceLeavesSyncAfterAnyToplevelCommit(t *testing.T) {
	f := newFakeWayland()
	f.names[0xA00] = "toplevel"
	area := gdkRectangle{Width: 2058, Height: 1400}
	u := &subsurface{
		s:   &surface{w: &window{id: 7, h: fakeSurfaceHandler{}}},
		c:   f.client(),
		g:   &waylandGlobals{compositor: 1, subcompositor: 2, viewporter: 3, fractionManager: 4, compositorVersion: 4},
		egl: f.egl(),
		top: subsurfaceToplevel{
			surface: func() ptr { return 0xA00 },
			allocation: func() (image.Point, gdkRectangle, gdkRectangle, int) {
				return image.Point{}, area, area, 2
			},
			queuePaint:     func() { f.log = append(f.log, "queue_paint") },
			watchPaints:    func() func() { return func() {} },
			frameCallbacks: func(surface ptr, add bool) bool { return true },
		},
	}
	u.sync = subsurfaceSync{
		setSync:    func() { u.request(u.sub, wlSubsurfaceSetSync) },
		setDesync:  func() { u.request(u.sub, wlSubsurfaceSetDesync) },
		queuePaint: func() { u.top.queuePaint() },
	}
	u.pacer = framePacer{soon: func() {}, later: func() {}, draw: u.present}
	want := func(steps ...string) {
		t.Helper()
		got := f.take()
		if !slices.Equal(got, steps) {
			t.Fatalf("requests:\n  %s\nwant:\n  %s", strings.Join(got, "\n  "), strings.Join(steps, "\n  "))
		}
	}
	u.create()
	f.take()

	// Made synchronized; GTK paints the toplevel, not the area.
	u.toplevelPainted()
	f.log = append(f.log, "commit toplevel")
	u.afterPaint()
	want("commit toplevel", "subsurface#1.set_desync()", "surface#1.commit()")

	// A frame that places web views, held for the toplevel's commit: an
	// after-paint with nothing painted keeps it held; the next, after a
	// paint of the toplevel alone, lets it go.
	u.webViewsPlaced()
	f.take()
	u.afterPaint()
	want("surface#1.commit()")
	if !u.sync.synced {
		t.Fatal("desynchronized without a commit of the toplevel")
	}
	u.toplevelPainted()
	f.log = append(f.log, "commit toplevel")
	u.afterPaint()
	want("commit toplevel", "subsurface#1.set_desync()", "surface#1.commit()")
}
