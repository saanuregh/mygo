//go:build linux && !android && (amd64 || arm64)

package linux

import "image"

// A Wayland subsurface in step with its toplevel (subsurface.go). A
// desynchronized subsurface shows each of its commits at once, while what
// places it, its position, applies only as GDK commits the toplevel, in
// the frame clock's after-paint, once GTK painted it: a frame of a new
// size committed so would show for a frame against the toplevel's old
// size, stretched, cut off or beside a gap. Such a frame is committed with
// the subsurface synchronized instead, which holds it until the toplevel's
// next commit, and the subsurface goes back to desynchronized after that
// commit (syncOnce): a resize, a move, a new scale, and the first frame of
// a new subsurface, which is made synchronized and shows with the first
// commit after the toplevel's first configure, as GTK paints nothing
// before it.

// subsurfaceToplevel is what a subsurface reads of its toplevel and asks of
// GTK; tests fake it.
type subsurfaceToplevel struct {
	// surface returns the toplevel's wl_surface, 0 while it has none.
	surface func() ptr
	// allocation returns the area's offset in the toplevel widget, the
	// area's and the toplevel's allocations, and GTK's scale.
	allocation func() (in image.Point, area, toplevel gdkRectangle, scale int)
	// queuePaint asks GTK to paint the area, which GDK then commits.
	queuePaint func()
	// watchPaints calls the subsurface's afterPaint after GDK's own in each
	// after-paint of the toplevel's frame clock, until stop.
	watchPaints func() (stop func())
}

// gtkToplevel returns the subsurfaceToplevel of the surface s.
func gtkToplevel(s *surface) subsurfaceToplevel {
	return subsurfaceToplevel{
		surface: func() ptr { return s.w.b.waylandSurface(gtkWidgetGetWindow(s.w.win)) },
		allocation: func() (image.Point, gdkRectangle, gdkRectangle, int) {
			loadFlyouts() // binds gtk_widget_translate_coordinates
			var in image.Point
			var x, y int32
			if gtkWidgetTranslateCoordinates(s.area, s.w.win, 0, 0, &x, &y) {
				in = image.Pt(int(x), int(y))
			}
			var area, toplevel gdkRectangle
			gtkWidgetGetAllocation(s.area, &area)
			gtkWidgetGetAllocation(s.w.win, &toplevel)
			return in, area, toplevel, max(int(gtkWidgetGetScaleFactor(s.area)), 1)
		},
		queuePaint: func() { gtkWidgetQueueDraw(s.area) },
		watchPaints: func() func() {
			// GDK connects its handler as it makes the GdkWindow, before the
			// toplevel maps: this one runs after it.
			clock := gtkWidgetGetFrameClock(s.w.win)
			if clock == 0 {
				return func() {}
			}
			gObjectRef(clock)
			id := connect(clock, "after-paint", cbSubsurfaceAfterPaint, ptr(s.w.id))
			return func() {
				gSignalHandlerDisconnect(clock, id)
				gObjectUnref(clock)
			}
		},
	}
}

// subsurfaceSync follows whether a subsurface is synchronized with its
// toplevel, and until when.
type subsurfaceSync struct {
	// setSync and setDesync send wl_subsurface's requests; queuePaint asks
	// GTK to paint the toplevel.
	setSync, setDesync, queuePaint func()
	// synced tells that the subsurface is synchronized, painted that GTK
	// painted the area since, which GDK commits in the frame's after-paint,
	// and inPaint that GTK paints it now.
	synced, painted, inPaint bool
}

// begin synchronizes the subsurface until GDK next commits the toplevel
// after painting the area, now or as GTK paints it next.
func (t *subsurfaceSync) begin() {
	if !t.synced {
		t.setSync()
		t.synced, t.painted = true, false
	}
	if t.inPaint {
		t.painted = true
	} else {
		t.queuePaint()
	}
}

// paint runs f as GTK paints the area, which GDK commits in this frame's
// after-paint.
func (t *subsurfaceSync) paint(f func()) {
	t.inPaint = true
	if t.synced {
		t.painted = true
	}
	f()
	t.inPaint = false
}

// afterPaint follows GDK's after-paint, where it committed the toplevel if
// GTK painted it: a subsurface synchronized and painted since goes back to
// desynchronized, its frames showing at once again.
func (t *subsurfaceSync) afterPaint() {
	if t.synced && t.painted {
		t.setDesync()
		t.synced, t.painted = false, false
	}
}

// reset starts over with a new subsurface, synchronized as made, or none.
func (t *subsurfaceSync) reset(synced bool) {
	t.synced, t.painted, t.inPaint = synced, false, false
}

// syncOnce runs draw, which commits a frame, with the subsurface
// synchronized, so that the frame shows with GDK's next commit of the
// toplevel, after GTK paints the area, as what GTK changed in the toplevel
// meanwhile does; the subsurface is desynchronized again after that
// commit.
func (u *subsurface) syncOnce(draw func()) {
	if !u.mapped || u.failed {
		return
	}
	u.sync.begin()
	draw()
}

// resyncNext asks for the next frame to be drawn as GTK paints the area,
// in step with the toplevel (paint).
func (u *subsurface) resyncNext() {
	u.resync = true
	u.top.queuePaint()
}

// paint follows GTK painting the area, under the subsurface, in the frame
// clock's paint: a frame asked for in step with the toplevel draws now.
func (u *subsurface) paint() {
	if !u.mapped || u.failed {
		return
	}
	u.sync.paint(func() {
		if u.resync {
			u.resync = false
			u.syncOnce(u.pacer.now)
		}
	})
}

// afterPaint follows GDK's after-paint of the toplevel.
func (u *subsurface) afterPaint() {
	if u.mapped && !u.failed {
		u.sync.afterPaint()
	}
}
