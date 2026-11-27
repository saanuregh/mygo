//go:build linux && !android && (amd64 || arm64)

package linux

import (
	"image"
	"sync"

	"github.com/ebitengine/purego"
)

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
	// after-paint of the toplevel's frame clock, and its layout in each of
	// the clock's layouts, until stop.
	watchPaints func() (stop func())
	// requestLayout asks the frame clock for a layout, another if it lays
	// out now.
	requestLayout func()
	// frameCallbacks has GDK ask for the toplevel's frame callbacks on
	// surface too, or no more, and reports whether it does; nil where GDK
	// cannot (subsurface_opaque.go).
	frameCallbacks func(surface ptr, add bool) bool
}

var (
	subsurfaceGTKOnce sync.Once
	// cbSubsurfaceLayout follows the layout of a toplevel's frame clock.
	cbSubsurfaceLayout        ptr
	gdkFrameClockRequestPhase func(clock ptr, phase uint32)
	// GDK 3.24.25's.
	gdkWaylandWindowAddFrameCallbackSurface    func(window, surface ptr)
	gdkWaylandWindowRemoveFrameCallbackSurface func(window, surface ptr)
)

// gdkFrameClockPhaseLayout is GDK_FRAME_CLOCK_PHASE_LAYOUT.
const gdkFrameClockPhaseLayout = 1 << 3

func loadSubsurfaceGTK() {
	subsurfaceGTKOnce.Do(func() {
		mustBind(libGDK, &gdkFrameClockRequestPhase, "gdk_frame_clock_request_phase")
		if !bind(libGDK, &gdkWaylandWindowAddFrameCallbackSurface, "gdk_wayland_window_add_frame_callback_surface") ||
			!bind(libGDK, &gdkWaylandWindowRemoveFrameCallbackSurface, "gdk_wayland_window_remove_frame_callback_surface") {
			gdkWaylandWindowAddFrameCallbackSurface, gdkWaylandWindowRemoveFrameCallbackSurface = nil, nil
		}
		cbSubsurfaceLayout = purego.NewCallback(func(clock, data ptr) {
			if s := theBackend.surfaceOf(data); s != nil && s.sub != nil {
				s.sub.layout()
			}
		})
	})
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
			loadSubsurfaceGTK()
			gObjectRef(clock)
			id := connect(clock, "after-paint", cbSubsurfaceAfterPaint, ptr(s.w.id))
			layout := connect(clock, "layout", cbSubsurfaceLayout, ptr(s.w.id))
			return func() {
				gSignalHandlerDisconnect(clock, id)
				gSignalHandlerDisconnect(clock, layout)
				gObjectUnref(clock)
			}
		},
		requestLayout: func() {
			if clock := gtkWidgetGetFrameClock(s.w.win); clock != 0 {
				loadSubsurfaceGTK()
				gdkFrameClockRequestPhase(clock, gdkFrameClockPhaseLayout)
			}
		},
		frameCallbacks: func(surface ptr, add bool) bool {
			loadSubsurfaceGTK()
			win := gtkWidgetGetWindow(s.w.win)
			if gdkWaylandWindowAddFrameCallbackSurface == nil || win == 0 {
				return false
			}
			if add {
				gdkWaylandWindowAddFrameCallbackSurface(win, surface)
			} else {
				gdkWaylandWindowRemoveFrameCallbackSurface(win, surface)
			}
			return true
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
	// The compositor may apply the frame with others held with it: it
	// shows whole, and frames copy whole into each back buffer until it
	// holds one shown since.
	u.egl.blits.reset()
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

// allocated follows GTK allocating the area, after place, in the frame
// clock's layout. In a window with web views, a frame asked for in step
// with the toplevel, as a resize asks, draws in a layout of the clock's
// once GTK allocated the window (layout), rather than as GTK paints the
// area: GTK then moves the web views the frame places (webViewsPlaced) in
// a layout more, which the clock runs before it paints, and paints them
// there for the commit that shows the frame. Drawn in the paint, the
// frame would show with the web views where they were, GTK moving them
// only for the next commit; drawn in the area's allocation, GTK would not
// move them at all, as an allocation forgets what asks for one in it.
func (u *subsurface) allocated() {
	if u.mapped && !u.failed && u.resync && u.s.layers != 0 {
		u.top.requestLayout()
	}
}

// layout follows the layout of the toplevel's frame clock, which runs
// again for as long as GTK asks for one, up to 4 times: a frame asked for
// in step with the toplevel, in a window with web views, draws now.
func (u *subsurface) layout() {
	if !u.mapped || u.failed || !u.resync || u.s.layers == 0 {
		return
	}
	u.resync = false
	u.syncOnce(u.pacer.now)
}

// webViewsPlaced follows the content placing its web views anew, as it
// draws a frame (PlaceWebViews): GTK moves them as it next lays out and
// paints the toplevel, which GDK then commits, and the frame, whose holes
// show them at their new places, commits synchronized to show with that
// commit, with the opaque region less those holes. The frame draws
// already: syncOnce's draw is the rest of it. One drawn in a paint shows
// with this paint's commit, a frame before the web views move.
func (u *subsurface) webViewsPlaced() {
	u.setOpaque()
	u.syncOnce(func() {})
}

// afterPaint follows GDK's after-paint of the toplevel.
func (u *subsurface) afterPaint() {
	if u.mapped && !u.failed {
		u.sync.afterPaint()
		u.commitFrameRequest()
	}
}
