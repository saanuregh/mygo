//go:build linux && !android && (amd64 || arm64)

package linux

import (
	"image"
	"log"
	"os"
	"sync"

	"github.com/ebitengine/purego"
)

// The Wayland subsurface of a window's surface (MYGO_WAYLAND_SUBSURFACE=1):
// a wl_surface of mygo's own, a desynchronized wl_subsurface of the
// toplevel's, placed over the surface's widget, a GtkDrawingArea, which
// keeps every input, as the subsurface takes none (an empty input region).
// GDK makes the toplevel's wl_surface as the window shows and destroys it
// as it hides: the subsurface is made as the toplevel maps, and goes as it
// unmaps.
//
// For now (stage 2) it shows one colour, transparent, or with
// MYGO_WAYLAND_SUBSURFACE_TEST_COLOR set a translucent magenta that shows
// where it is: mygo's frames are still drawn in memory and painted into
// the drawing area, under it, as with MYGO_GPU=0. Stage 3 draws them here.

// Requests' opcodes, their order in wayland.xml.
const (
	wlCompositorCreateSurface  = 0
	wlCompositorCreateRegion   = 1
	wlSurfaceDestroy           = 0
	wlSurfaceSetInputRegion    = 5
	wlSurfaceSetBufferScale    = 8
	wlRegionDestroy            = 0
	wlSubcompositorGetSubsurf  = 1
	wlSubsurfaceDestroy        = 0
	wlSubsurfaceSetPosition    = 1
	wlSubsurfaceSetDesync      = 5
	wlMarshalFlagDestroy       = 1 << 0 // WL_MARSHAL_FLAG_DESTROY
	wlSurfaceBufferScaleSince  = 3
	subsurfaceTestColorEnvName = "MYGO_WAYLAND_SUBSURFACE_TEST_COLOR"
)

// subsurface is the Wayland subsurface of a window's surface.
type subsurface struct {
	s *surface
	c *waylandClient
	g *waylandGlobals
	// wlSurface and sub exist while the toplevel is mapped.
	wlSurface, sub ptr
	egl            *subsurfaceEGL
	// Where the subsurface is on the toplevel's wl_surface, and its size,
	// in logical pixels, and the scale of its buffers, as last sent.
	x, y, logicalW, logicalH, scale int
	mapped                          bool
	// color is what it shows, premultiplied.
	color [4]float32
	// failed tells that making it failed once, which is logged once.
	failed bool
}

// newSubsurface returns the subsurface of s, with its EGL context, when
// MYGO_WAYLAND_SUBSURFACE=1 asked for one, or nil, saying once why it
// makes none.
func newSubsurface(s *surface) *subsurface {
	b := s.w.b
	if !waylandSubsurfaceAsked() || b.waylandSurface == nil {
		return nil
	}
	var why string
	switch {
	case os.Getenv("MYGO_GPU") == "0":
		why = "MYGO_GPU=0"
	case b.wayland == nil:
		why = "the globals it needs were not bound"
	case !b.wayland.complete():
		why = "the compositor lacks some globals (" + b.wayland.String() + ")"
	}
	if why != "" {
		logSubsurfaceUnused(why)
		return nil
	}
	c, err := loadWaylandClient()
	if err != nil {
		logSubsurfaceUnused(err.Error())
		return nil
	}
	e, err := newSubsurfaceEGL(b.wayland.display)
	if err != nil {
		logSubsurfaceUnused(err.Error())
		return nil
	}
	initSubsurfaceCallbacks()
	u := &subsurface{s: s, c: c, g: b.wayland, egl: e}
	if os.Getenv(subsurfaceTestColorEnvName) != "" {
		u.color = [4]float32{0.5, 0, 0.5, 0.5}
	}
	return u
}

var subsurfaceUnusedOnce sync.Once

// logSubsurfaceUnused says, once, why windows draw as without the switch.
func logSubsurfaceUnused(why string) {
	subsurfaceUnusedOnce.Do(func() { log.Printf("mygo: Wayland subsurface not used: %s", why) })
}

// connect follows the toplevel's mapping, the surface's area made already.
func (u *subsurface) connect() {
	data := ptr(u.s.w.id)
	connectAfter(u.s.w.win, "map", cbSubsurfaceMap, data)
	connect(u.s.w.win, "unmap", cbSubsurfaceUnmap, data)
	if gtkWidgetGetMapped(u.s.w.win) {
		u.create()
	}
}

// request sends a request with no new object; its args, up to 4.
func (u *subsurface) request(proxy ptr, opcode uint32, args ...uint64) {
	var a [4]uint64
	copy(a[:], args)
	u.c.marshal(proxy, opcode, 0, u.c.proxyGetVersion(proxy), 0, &a[0])
}

// newObject sends a request making an object of iface: args holds 0
// where its new_id goes.
func (u *subsurface) newObject(proxy ptr, opcode uint32, iface ptr, args ...uint64) ptr {
	var a [4]uint64
	copy(a[:], args)
	return u.c.marshal(proxy, opcode, iface, u.c.proxyGetVersion(proxy), 0, &a[0])
}

// destroyProxy sends a destructor request and destroys the proxy.
func (u *subsurface) destroyProxy(proxy ptr, opcode uint32) {
	var a [4]uint64
	u.c.marshal(proxy, opcode, 0, u.c.proxyGetVersion(proxy), wlMarshalFlagDestroy, &a[0])
}

// wlInt is an int32 in its union wl_argument.
func wlInt(v int) uint64 { return uint64(uint32(int32(v))) }

// create makes the subsurface on the toplevel's wl_surface, places it and
// shows its colour; a failure is logged once, and the content shows as it
// is painted under it.
func (u *subsurface) create() {
	if u.mapped {
		return
	}
	toplevel := u.s.w.b.waylandSurface(gtkWidgetGetWindow(u.s.w.win))
	if toplevel == 0 {
		u.fail("the window has no wl_surface")
		return
	}
	if u.wlSurface = u.newObject(u.g.compositor, wlCompositorCreateSurface, u.c.surfaceInterface, 0); u.wlSurface == 0 {
		u.fail("wl_compositor.create_surface failed")
		return
	}
	if u.sub = u.newObject(u.g.subcompositor, wlSubcompositorGetSubsurf, u.c.subsurfaceInterface, 0, uint64(u.wlSurface), uint64(toplevel)); u.sub == 0 {
		u.fail("wl_subcompositor.get_subsurface failed")
		u.destroy()
		return
	}
	u.request(u.sub, wlSubsurfaceSetDesync)
	// Every input reaches GTK's surface, under it.
	region := u.newObject(u.g.compositor, wlCompositorCreateRegion, u.c.regionInterface, 0)
	if region == 0 {
		u.fail("wl_compositor.create_region failed")
		u.destroy()
		return
	}
	u.request(u.wlSurface, wlSurfaceSetInputRegion, uint64(region))
	u.destroyProxy(region, wlRegionDestroy)
	u.mapped = true
	r, scale := u.placement()
	u.x, u.y = r.Min.X, r.Min.Y
	u.request(u.sub, wlSubsurfaceSetPosition, wlInt(u.x), wlInt(u.y))
	u.setSize(r.Dx(), r.Dy(), scale)
	if err := u.egl.attach(u.wlSurface, u.logicalW*u.scale, u.logicalH*u.scale); err != nil {
		u.fail(err.Error())
		u.destroy()
		return
	}
	u.draw()
}

// fail says, once, why the subsurface shows nothing.
func (u *subsurface) fail(why string) {
	if !u.failed {
		u.failed = true
		log.Printf("mygo: Wayland subsurface shows nothing: %s", why)
	}
}

// destroy lets go of the subsurface, as the toplevel's wl_surface goes;
// its context stays for the next.
func (u *subsurface) destroy() {
	u.egl.detach()
	if u.sub != 0 {
		u.destroyProxy(u.sub, wlSubsurfaceDestroy)
		u.sub = 0
	}
	if u.wlSurface != 0 {
		u.destroyProxy(u.wlSurface, wlSurfaceDestroy)
		u.wlSurface = 0
	}
	u.mapped = false
	u.x, u.y, u.logicalW, u.logicalH, u.scale = 0, 0, 0, 0, 0
}

// release lets go of the subsurface and its context, as the window goes.
func (u *subsurface) release() {
	u.destroy()
	u.egl.release()
}

// placement returns where the area is on the toplevel's wl_surface, and
// the scale of its buffers.
func (u *subsurface) placement() (image.Rectangle, int) {
	loadFlyouts() // binds gtk_widget_translate_coordinates
	w := u.s.w
	var in image.Point
	var x, y int32
	if gtkWidgetTranslateCoordinates(u.s.area, w.win, 0, 0, &x, &y) {
		in = image.Pt(int(x), int(y))
	}
	var area, toplevel gdkRectangle
	gtkWidgetGetAllocation(u.s.area, &area)
	gtkWidgetGetAllocation(w.win, &toplevel)
	scale := 1
	if u.g.compositorVersion >= wlSurfaceBufferScaleSince {
		scale = max(int(gtkWidgetGetScaleFactor(u.s.area)), 1)
	}
	return subsurfacePlacement(in, area, toplevel), scale
}

// subsurfacePlacement returns where the subsurface goes on the toplevel's
// wl_surface, in its logical pixels, which are those of the toplevel's
// GdkWindow: the area's offset in the toplevel widget
// (gtk_widget_translate_coordinates), moved by the toplevel's allocation,
// which starts inside the shadow GTK draws around a window it decorates,
// and the area's allocated size, at least 1×1, as a buffer has pixels.
func subsurfacePlacement(inToplevel image.Point, area, toplevel gdkRectangle) image.Rectangle {
	at := inToplevel.Add(image.Pt(int(toplevel.X), int(toplevel.Y)))
	return image.Rectangle{Min: at, Max: at.Add(image.Pt(max(int(area.Width), 1), max(int(area.Height), 1)))}
}

// setSize keeps the subsurface's logical size and scale, telling the
// compositor a new scale.
func (u *subsurface) setSize(w, h, scale int) {
	if scale != u.scale && u.g.compositorVersion >= wlSurfaceBufferScaleSince {
		u.request(u.wlSurface, wlSurfaceSetBufferScale, wlInt(scale))
	}
	u.logicalW, u.logicalH, u.scale = w, h, scale
}

// place moves and sizes the subsurface to the area, as it is allocated or
// its scale changes, and shows its colour at the new size. The position
// applies as GDK commits the toplevel next.
func (u *subsurface) place() {
	if !u.mapped {
		return
	}
	r, scale := u.placement()
	if r.Min.X != u.x || r.Min.Y != u.y {
		u.x, u.y = r.Min.X, r.Min.Y
		u.request(u.sub, wlSubsurfaceSetPosition, wlInt(u.x), wlInt(u.y))
	}
	if r.Dx() == u.logicalW && r.Dy() == u.logicalH && scale == u.scale {
		return
	}
	u.setSize(r.Dx(), r.Dy(), scale)
	u.egl.resize(u.logicalW*u.scale, u.logicalH*u.scale)
	u.draw()
}

// draw shows the subsurface's colour: the swap attaches and commits its
// buffer.
func (u *subsurface) draw() {
	if err := u.egl.fill(u.color, u.logicalW*u.scale, u.logicalH*u.scale); err != nil {
		u.fail(err.Error())
	}
}

var (
	subsurfaceCallbacksOnce            sync.Once
	cbSubsurfaceMap, cbSubsurfaceUnmap ptr
)

func initSubsurfaceCallbacks() {
	subsurfaceCallbacksOnce.Do(func() {
		// After GTK's own handler, which shows the GdkWindow, and GDK makes
		// its wl_surface.
		cbSubsurfaceMap = purego.NewCallback(func(widget, data ptr) {
			if s := theBackend.surfaceOf(data); s != nil && s.sub != nil {
				s.sub.create()
			}
		})
		// GTK's handler runs first and GDK has destroyed the toplevel's
		// wl_surface: the subsurface, inert since, goes too.
		cbSubsurfaceUnmap = purego.NewCallback(func(widget, data ptr) {
			if s := theBackend.surfaceOf(data); s != nil && s.sub != nil {
				s.sub.destroy()
			}
		})
	})
}
