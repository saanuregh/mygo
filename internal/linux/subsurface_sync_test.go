//go:build linux && !android && (amd64 || arm64)

package linux

import (
	"fmt"
	"image"
	"slices"
	"strings"
	"testing"
	"unsafe"

	"github.com/egoist/mygo/internal/platform"
)

// A subsurface synchronized for a frame shows that frame with GDK's next
// commit of the toplevel after GTK painted the area: it stays synchronized
// through an after-paint with nothing painted, as GDK then committed none
// of the area, or none at all, and goes back to desynchronized after the
// one following a paint. A frame asked for before the paint asks GTK for
// one; a frame drawn in the paint, as resizes are, does not.
func TestSyncOnceShowsWithTheToplevelsCommit(t *testing.T) {
	var log []string
	sync := subsurfaceSync{
		setSync:    func() { log = append(log, "set_sync") },
		setDesync:  func() { log = append(log, "set_desync") },
		queuePaint: func() { log = append(log, "queue_paint") },
	}
	commit := func(what string) { log = append(log, "commit "+what) }
	// GDK's after-paint commits the toplevel if GTK painted it; the
	// subsurface's runs after it.
	afterPaint := func(painted bool) {
		if painted {
			commit("toplevel")
		}
		sync.afterPaint()
	}
	want := func(steps ...string) {
		t.Helper()
		if !slices.Equal(log, steps) {
			t.Fatalf("requests:\n  %s\nwant:\n  %s", strings.Join(log, "\n  "), strings.Join(steps, "\n  "))
		}
		log = nil
	}

	// A resize: the frame draws as GTK paints the area.
	sync.paint(func() {
		sync.begin()
		commit("subsurface")
	})
	afterPaint(true)
	want("set_sync", "commit subsurface", "commit toplevel", "set_desync")

	// A frame asked for outside a paint: GTK is asked for one. An
	// after-paint in between, with nothing painted, keeps it synchronized;
	// so does a second frame, which replaces the first.
	sync.begin()
	commit("subsurface")
	afterPaint(false)
	sync.begin()
	commit("subsurface")
	sync.paint(func() {})
	afterPaint(true)
	want("set_sync", "queue_paint", "commit subsurface", "queue_paint", "commit subsurface", "commit toplevel", "set_desync")

	// Desynchronized, a paint changes nothing.
	sync.paint(func() {})
	afterPaint(true)
	want("commit toplevel")

	// A new subsurface is synchronized as made: its first frame shows with
	// the toplevel's first commit after a paint.
	sync.reset(true)
	afterPaint(false)
	sync.paint(func() {
		sync.begin()
		commit("subsurface")
	})
	afterPaint(true)
	want("commit subsurface", "commit toplevel", "set_desync")

	// Gone, it sends nothing more.
	sync.reset(true)
	sync.reset(false)
	sync.paint(func() {})
	afterPaint(true)
	want("commit toplevel")
}

// fakeWayland records the requests a subsurface sends, naming each proxy
// it makes by its interface, and what EGL does with the window surfaces.
type fakeWayland struct {
	names map[ptr]string
	kinds map[ptr]string
	count map[string]int
	next  ptr
	log   []string
	// wlSurfaceOf names the wl_surface of each wl_egl_window.
	wlSurfaceOf map[ptr]string
}

// Requests' names by the kind of proxy and opcode.
var fakeRequests = map[string][]string{
	"compositor":    {"create_surface", "create_region"},
	"surface":       {"destroy", "", "", "frame", "", "set_input_region", "", "", "set_buffer_scale"},
	"region":        {"destroy"},
	"subcompositor": {"", "get_subsurface"},
	"subsurface":    {"destroy", "set_position", "", "", "set_sync", "set_desync"},
	"viewporter":    {"", "get_viewport"},
	"viewport":      {"destroy", "", "set_destination"},
	"fractionMgr":   {"", "get_fractional_scale"},
	"fraction":      {"destroy"},
}

func newFakeWayland() *fakeWayland {
	f := &fakeWayland{names: map[ptr]string{}, kinds: map[ptr]string{}, count: map[string]int{}, next: 0x1000, wlSurfaceOf: map[ptr]string{}}
	for p, kind := range map[ptr]string{1: "compositor", 2: "subcompositor", 3: "viewporter", 4: "fractionMgr"} {
		f.names[p], f.kinds[p] = kind, kind
	}
	return f
}

func (f *fakeWayland) name(p ptr) string {
	if n, ok := f.names[p]; ok {
		return n
	}
	return fmt.Sprintf("%#x", p)
}

// add names a new proxy of kind.
func (f *fakeWayland) add(kind string) ptr {
	f.next += 0x10
	f.count[kind]++
	f.names[f.next] = fmt.Sprintf("%s#%d", kind, f.count[kind])
	f.kinds[f.next] = kind
	return f.next
}

func (f *fakeWayland) take() []string {
	l := f.log
	f.log = nil
	return l
}

// client returns the waylandClient of the fake, its interfaces telling
// which kind of proxy a request makes.
func (f *fakeWayland) client() *waylandClient {
	ifaces := map[ptr]string{11: "surface", 12: "subsurface", 13: "region", 14: "callback", 15: "viewport", 16: "fraction"}
	c := &waylandClient{
		surfaceInterface: 11, subsurfaceInterface: 12, regionInterface: 13, callbackInterface: 14,
		viewportInterface: 15, fractionInterface: 16,
	}
	c.marshal = func(proxy ptr, opcode uint32, iface ptr, version, flags uint32, args *uint64) ptr {
		a := (*[4]uint64)(unsafe.Pointer(args))
		op := fmt.Sprint(opcode)
		if names := fakeRequests[f.kinds[proxy]]; int(opcode) < len(names) && names[opcode] != "" {
			op = names[opcode]
		}
		var made ptr
		var shown []string
		switch {
		case iface != 0:
			made = f.add(ifaces[iface])
			for _, v := range a[1:] {
				if v != 0 {
					shown = append(shown, f.name(ptr(v)))
				}
			}
		case op == "set_input_region":
			shown = append(shown, f.name(ptr(a[0])))
		case op == "set_position" || op == "set_destination":
			shown = append(shown, fmt.Sprint(int32(a[0])), fmt.Sprint(int32(a[1])))
		case op == "set_buffer_scale":
			shown = append(shown, fmt.Sprint(int32(a[0])))
		}
		line := f.name(proxy) + "." + op + "(" + strings.Join(shown, ", ") + ")"
		if made != 0 {
			line = f.name(made) + " = " + line
		}
		f.log = append(f.log, line)
		return made
	}
	c.addListener = func(proxy, implementation, data ptr) int32 { return 0 }
	c.proxyGetVersion = func(proxy ptr) uint32 { return 4 }
	c.proxyDestroy = func(proxy ptr) { f.log = append(f.log, f.name(proxy)+" let go of") }
	return c
}

// egl returns a subsurfaceEGL whose window surfaces are the fake's, with a
// framebuffer of frames already made.
func (f *fakeWayland) egl() *subsurfaceEGL {
	var current ptr
	l := &eglLib{
		windowCreate: func(surface ptr, w, h int32) ptr {
			p := f.add("egl_window")
			f.wlSurfaceOf[p] = f.name(surface)
			f.log = append(f.log, fmt.Sprintf("%s = wl_egl_window_create(%s, %d, %d)", f.name(p), f.name(surface), w, h))
			return p
		},
		windowResize: func(window ptr, w, h, dx, dy int32) {
			f.log = append(f.log, fmt.Sprintf("wl_egl_window_resize(%s, %d, %d)", f.name(window), w, h))
		},
		windowDestroy: func(window ptr) { f.log = append(f.log, "wl_egl_window_destroy("+f.name(window)+")") },
		createWindowSurface: func(display, config, window ptr, attribs *int32) ptr {
			return window + 1
		},
		destroySurface:    func(display, surface ptr) uint32 { return 1 },
		getCurrentContext: func() ptr { return current },
		makeCurrent: func(display, draw, read, context ptr) uint32 {
			current = context
			return 1
		},
		swapInterval:    func(display ptr, interval int32) uint32 { return 1 },
		destroyContext:  func(display, context ptr) uint32 { f.log = append(f.log, "eglDestroyContext"); return 1 },
		getError:        func() int32 { return 0x3000 },
		clearCurrent:    func() {},
		bindFramebuffer: func(target, fb uint32) {},
		disable:         func(cap uint32) {},
		blitFramebuffer: func(sx0, sy0, sx1, sy1, dx0, dy0, dx1, dy1 int32, mask, filter uint32) {},
		glGetError:      func() uint32 { return 0 },
	}
	e := &subsurfaceEGL{l: l, display: 0x50, config: 0x51, context: 0x52, fb: 1, tex: 2, w: 4000, h: 3000}
	l.swapBuffers = func(display, surface ptr) uint32 {
		f.log = append(f.log, "commit "+f.wlSurfaceOf[surface-1])
		return 1
	}
	return e
}

// A window hidden and shown again draws again: GDK made its toplevel a new
// wl_surface, which gets a new subsurface, placed again, with the scale
// asked for again and the last known one's buffers, a viewport, and frames
// paced afresh; the callback due on the old one is let go of, and what
// comes for the old one changes nothing. The context, its framebuffer with
// the last frame and the renderer drawing into it are kept, as nothing of
// them belonged to the wl_surface gone. Each subsurface's first frame, and
// a resize's, shows with the toplevel's commit.
func TestSubsurfaceRebuiltOnRemap(t *testing.T) {
	f := newFakeWayland()
	toplevel := ptr(0xA00)
	f.names[0xA00], f.names[0xB00] = "toplevel A", "toplevel B"
	area := gdkRectangle{Width: 2058, Height: 1400}
	paints, watching, watched := 0, 0, 0
	var soon []func()
	u := &subsurface{
		s:   &surface{w: &window{id: 7, h: fakeSurfaceHandler{}}},
		c:   f.client(),
		g:   &waylandGlobals{compositor: 1, subcompositor: 2, viewporter: 3, fractionManager: 4, compositorVersion: 4},
		egl: f.egl(),
		top: subsurfaceToplevel{
			surface: func() ptr { return toplevel },
			allocation: func() (image.Point, gdkRectangle, gdkRectangle, int) {
				return image.Point{}, area, area, 2
			},
			queuePaint: func() { paints++ },
			watchPaints: func() func() {
				watching++
				watched++
				return func() { watching-- }
			},
		},
	}
	u.sync = subsurfaceSync{
		setSync:    func() { u.request(u.sub, wlSubsurfaceSetSync) },
		setDesync:  func() { u.request(u.sub, wlSubsurfaceSetDesync) },
		queuePaint: func() { u.top.queuePaint() },
	}
	var drewSynced []bool
	u.pacer = framePacer{
		soon:  func() { soon = append(soon, u.pacer.run) },
		later: func() { t.Fatal("a frame waits a refresh") },
		draw: func() bool {
			drewSynced = append(drewSynced, u.sync.synced)
			return u.present()
		},
	}
	want := func(steps ...string) {
		t.Helper()
		got := f.take()
		if !slices.Equal(got, steps) {
			t.Fatalf("requests:\n  %s\nwant:\n  %s", strings.Join(got, "\n  "), strings.Join(steps, "\n  "))
		}
	}
	// A frame clock's paint and after-paint, GDK committing the toplevel
	// after GTK painted it.
	frameClock := func() {
		t.Helper()
		u.paint()
		f.log = append(f.log, "commit "+f.name(toplevel))
		u.afterPaint()
	}

	u.create()
	want(
		"surface#1 = compositor.create_surface()",
		"subsurface#1 = subcompositor.get_subsurface(surface#1, toplevel A)",
		"region#1 = compositor.create_region()",
		"surface#1.set_input_region(region#1)",
		"region#1.destroy()",
		"viewport#1 = viewporter.get_viewport(surface#1)",
		"fraction#1 = fractionMgr.get_fractional_scale(surface#1)",
		"subsurface#1.set_position(0, 0)",
		"surface#1.set_buffer_scale(2)",
		"egl_window#1 = wl_egl_window_create(surface#1, 4116, 2800)",
	)
	if paints != 1 || watching != 1 || !u.sync.synced {
		t.Fatalf("made: %d paints asked for, %d after-paints followed, synchronized %v; want 1, 1, true", paints, watching, u.sync.synced)
	}
	// The first frame draws as GTK first paints, after the configure, and
	// shows with that commit.
	frameClock()
	want(
		"callback#1 = surface#1.frame()",
		"commit surface#1",
		"commit toplevel A",
		"subsurface#1.set_desync()",
	)
	u.preferredScale(u.fraction, 180)
	want(
		"surface#1.set_buffer_scale(1)",
		"viewport#1.set_destination(2058, 1400)",
		"wl_egl_window_resize(egl_window#1, 3087, 2100)",
	)
	frameClock()
	want(
		"subsurface#1.set_sync()",
		"callback#1 let go of",
		"callback#2 = surface#1.frame()",
		"commit surface#1",
		"commit toplevel A",
		"subsurface#1.set_desync()",
	)

	// A resize, the callback of the last frame due: the frame of the new
	// size draws as GTK paints, in step.
	area = gdkRectangle{Width: 1600, Height: 1000}
	u.place()
	frameClock()
	want(
		"viewport#1.set_destination(1600, 1000)",
		"wl_egl_window_resize(egl_window#1, 2400, 1500)",
		"subsurface#1.set_sync()",
		"callback#2 let go of",
		"callback#3 = surface#1.frame()",
		"commit surface#1",
		"commit toplevel A",
		"subsurface#1.set_desync()",
	)
	// Asked for while its callback is due, a frame waits.
	u.pacer.request()
	if len(soon) != 0 {
		t.Fatal("a frame asked for with a callback due drew at once")
	}
	oldCallback, oldFraction := u.frameCB, u.fraction

	// Hidden: the toplevel's wl_surface goes.
	u.destroy()
	want(
		"callback#3 let go of",
		"wl_egl_window_destroy(egl_window#1)",
		"fraction#1.destroy()",
		"viewport#1.destroy()",
		"subsurface#1.destroy()",
		"surface#1.destroy()",
	)
	if watching != 0 || u.pacer.waiting || u.frameCB != 0 || u.sync.synced || u.resync {
		t.Fatalf("gone: watching %d, waiting %v, callback %#x, synchronized %v, resync %v", watching, u.pacer.waiting, u.frameCB, u.sync.synced, u.resync)
	}
	// What comes for the old one changes nothing.
	u.frameDone(oldCallback)
	u.preferredScale(oldFraction, 240)
	u.paint()
	u.afterPaint()
	want()
	// The frame asked for draws on nothing.
	for _, run := range soon {
		run()
	}
	soon = nil
	want()

	// Shown: a new wl_surface. Its buffers are at the last scale known,
	// until the compositor sends one for the new subsurface.
	toplevel = 0xB00
	u.create()
	want(
		"surface#2 = compositor.create_surface()",
		"subsurface#2 = subcompositor.get_subsurface(surface#2, toplevel B)",
		"region#2 = compositor.create_region()",
		"surface#2.set_input_region(region#2)",
		"region#2.destroy()",
		"viewport#2 = viewporter.get_viewport(surface#2)",
		"fraction#2 = fractionMgr.get_fractional_scale(surface#2)",
		"subsurface#2.set_position(0, 0)",
		"surface#2.set_buffer_scale(1)",
		"viewport#2.set_destination(1600, 1000)",
		"egl_window#2 = wl_egl_window_create(surface#2, 2400, 1500)",
	)
	if paints < 2 || watching != 1 || watched != 2 || !u.show {
		t.Fatalf("made again: %d paints asked for, watching %d (%d times), show %v", paints, watching, watched, u.show)
	}
	frameClock()
	want(
		"callback#4 = surface#2.frame()",
		"commit surface#2",
		"commit toplevel B",
		"subsurface#2.set_desync()",
	)
	// Its frames are paced by its own callbacks.
	u.frameDone(u.frameCB)
	want("callback#4 let go of")
	u.pacer.request()
	for _, run := range soon {
		run()
	}
	want("callback#5 = surface#2.frame()", "commit surface#2")
	if !slices.Equal(drewSynced, []bool{true, true, true, true, false}) {
		t.Fatalf("frames drawn synchronized: %v", drewSynced)
	}
	if u.egl.context != 0x52 || u.egl.tex == 0 {
		t.Fatal("the context or its framebuffer went with the wl_surface")
	}

	// A preferred scale for the new subsurface, the same: nothing changes.
	u.preferredScale(u.fraction, 180)
	want()
}

// fakeSurfaceHandler takes a surface's events and nothing else.
type fakeSurfaceHandler struct{ platform.WindowHandler }

func (fakeSurfaceHandler) SurfaceEvent(platform.SurfaceEvent) bool { return false }
