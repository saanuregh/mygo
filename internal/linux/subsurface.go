//go:build linux && !android && (amd64 || arm64)

package linux

import (
	"image"
	"log"
	"os"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"

	"github.com/egoist/mygo/internal/platform"
)

// The Wayland subsurface of a window's surface (MYGO_WAYLAND_SUBSURFACE=1):
// a wl_surface of mygo's own, a desynchronized wl_subsurface of the
// toplevel's, placed over the surface's widget, a GtkDrawingArea, which
// keeps every input, as the subsurface takes none (an empty input region).
// GDK makes the toplevel's wl_surface as the window shows and destroys it
// as it hides: the subsurface is made as the toplevel maps, and goes as it
// unmaps. A frame that moves or resizes it shows with the toplevel's
// commit, the subsurface synchronized for it (subsurface_sync.go).
//
// mygo's frames draw on it with OpenGL (subsurface_egl.go), paced by its
// own frame callbacks rather than GTK's frame clock, which then drives only
// GTK's widgets: a frame asked for while no callback is due draws at once,
// from an idle callback, and one asked for while a callback is due draws
// as it comes (framePacer). The drawing area under it paints nothing.
//
// Its buffers are at the compositor's preferred scale for it, fractional
// (wp_fractional_scale_v1), shown at its logical size (wp_viewport), and
// mygo draws at that scale (surface.Size); until the compositor sends one,
// as it does once the subsurface shows on an output, they are at GTK's
// integer scale, set as the buffers' (wl_surface.set_buffer_scale).
// Where the subsurface fails, the window draws as with MYGO_GPU=0, into
// the drawing area.

// Requests' opcodes, their order in wayland.xml.
const (
	wlCompositorCreateSurface  = 0
	wlCompositorCreateRegion   = 1
	wlSurfaceDestroy           = 0
	wlSurfaceFrame             = 3
	wlSurfaceSetInputRegion    = 5
	wlSurfaceSetBufferScale    = 8
	wlRegionDestroy            = 0
	wlSubcompositorGetSubsurf  = 1
	wlSubsurfaceDestroy        = 0
	wlSubsurfaceSetPosition    = 1
	wlSubsurfaceSetSync        = 4
	wlSubsurfaceSetDesync      = 5
	wpViewporterGetViewport    = 1
	wpViewportDestroy          = 0
	wpViewportSetDestination   = 2
	wpFractionManagerGetScale  = 1
	wpFractionalScaleDestroy   = 0
	wlMarshalFlagDestroy       = 1 << 0 // WL_MARSHAL_FLAG_DESTROY
	wlSurfaceBufferScaleSince  = 3
	wlSurfaceDamageBufferSince = 4
	gPriorityHighIdle          = 100 // G_PRIORITY_HIGH_IDLE
)

// subsurface is the Wayland subsurface of a window's surface.
type subsurface struct {
	s *surface
	c *waylandClient
	g *waylandGlobals
	// wlSurface and sub exist while the toplevel is mapped, with the
	// wlSurface's viewport and fractional scale, 0 where they failed.
	wlSurface, sub, viewport, fraction ptr
	egl                                *subsurfaceEGL
	// Where the subsurface is on the toplevel's wl_surface, and its size,
	// in logical pixels, and the buffers' scale, as last sent: 1 at a
	// fractional scale.
	x, y, logicalW, logicalH, scale int
	// destination is the viewport's destination as last sent, none until
	// the buffers are at a fractional scale; buffer the size of the
	// buffers, in pixels.
	destination, buffer image.Point
	// opaque is the opaque region as last sent, nil for none;
	// frameCallbacks tells that GDK asks for the toplevel's frame
	// callbacks on the wlSurface too (subsurface_opaque.go).
	opaque         []image.Rectangle
	frameCallbacks bool
	// scale120 is the compositor's preferred scale for the subsurface, in
	// 120ths, 0 until it sends one: kept as the subsurface is made anew,
	// as the best guess until the new one is sent.
	scale120 uint32
	mapped   bool
	// failed tells that it failed, which is logged once: the window draws
	// into the drawing area from then on.
	failed bool
	// pacer paces its frames; frameCB is the wl_callback of the last frame
	// presented, until done, 0 for none.
	pacer   framePacer
	frameCB ptr
	// drew tells that the frame drawing changed the framebuffer (DrewGL),
	// within damage, or all of it when whole; show that the next frame
	// shows even if it changes nothing, as the first on a new wl_surface.
	drew, show, whole bool
	damage            []image.Rectangle
	// top is the toplevel; sync follows whether the subsurface is
	// synchronized with it, resync tells that the next frame draws as GTK
	// paints the area, in step with it (subsurface_sync.go); stopWatch
	// stops following its after-paint, nil while unmapped.
	top       subsurfaceToplevel
	sync      subsurfaceSync
	resync    bool
	stopWatch func()
}

// newSubsurface returns the subsurface of s, with its EGL context, when
// MYGO_WAYLAND_SUBSURFACE=1 asked for one and OpenGL draws on a GPU
// (gpu: gpuGL, as for a GtkGLArea), or nil, saying once why it makes none.
func newSubsurface(s *surface, gpu bool) *subsurface {
	b := s.w.b
	if !waylandSubsurfaceAsked() || b.waylandSurface == nil {
		return nil
	}
	var why string
	switch {
	case os.Getenv("MYGO_GPU") == "0":
		why = "MYGO_GPU=0"
	case !gpu:
		why = "OpenGL does not draw on a GPU"
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
	if subsurfaceFrameListener == 0 || subsurfaceScaleListener == 0 {
		logSubsurfaceUnused("no memory for the listeners")
		e.release()
		return nil
	}
	u := &subsurface{s: s, c: c, g: b.wayland, egl: e, top: gtkToplevel(s)}
	u.sync = subsurfaceSync{
		setSync:    func() { u.request(u.sub, wlSubsurfaceSetSync) },
		setDesync:  func() { u.request(u.sub, wlSubsurfaceSetDesync) },
		queuePaint: func() { u.top.queuePaint() },
	}
	data := ptr(s.w.id)
	u.pacer = framePacer{
		soon: func() { gIdleAddFull(gPriorityHighIdle, cbSubsurfaceFrame, data, 0) },
		later: func() {
			ms := uint32(16)
			if mhz := s.refreshMilliHz(); mhz > 0 {
				ms = uint32(max(1_000_000/int64(mhz), 1))
			}
			gTimeoutAdd(ms, cbSubsurfaceFrame, data)
		},
		draw: u.frame,
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

// create makes the subsurface on the toplevel's wl_surface and places it;
// its first frame draws as GTK first paints the toplevel, which it does
// once the compositor configured it, and shows with GDK's commit of that
// paint, the subsurface synchronized until then, as made. A failure is
// logged once, and the content draws in the drawing area under it from
// then on.
func (u *subsurface) create() {
	if u.mapped || u.failed {
		return
	}
	toplevel := u.top.surface()
	if toplevel == 0 {
		u.broke("the window has no wl_surface")
		return
	}
	if u.wlSurface = u.newObject(u.g.compositor, wlCompositorCreateSurface, u.c.surfaceInterface, 0); u.wlSurface == 0 {
		u.broke("wl_compositor.create_surface failed")
		return
	}
	if u.sub = u.newObject(u.g.subcompositor, wlSubcompositorGetSubsurf, u.c.subsurfaceInterface, 0, uint64(u.wlSurface), uint64(toplevel)); u.sub == 0 {
		u.broke("wl_subcompositor.get_subsurface failed")
		return
	}
	// Every input reaches GTK's surface, under it.
	region := u.newObject(u.g.compositor, wlCompositorCreateRegion, u.c.regionInterface, 0)
	if region == 0 {
		u.broke("wl_compositor.create_region failed")
		return
	}
	u.request(u.wlSurface, wlSurfaceSetInputRegion, uint64(region))
	u.destroyProxy(region, wlRegionDestroy)
	// Without either, buffers stay at GTK's integer scale.
	if u.viewport = u.newObject(u.g.viewporter, wpViewporterGetViewport, u.c.viewportInterface, 0, uint64(u.wlSurface)); u.viewport != 0 {
		if u.fraction = u.newObject(u.g.fractionManager, wpFractionManagerGetScale, u.c.fractionInterface, 0, uint64(u.wlSurface)); u.fraction != 0 {
			u.c.addListener(u.fraction, subsurfaceScaleListener, ptr(u.s.w.id))
		}
	}
	if u.fraction == 0 {
		u.scale120 = 0
	}
	u.mapped = true
	u.frameCallbacks = u.top.frameCallbacks != nil && u.top.frameCallbacks(u.wlSurface, true)
	u.sync.reset(true)
	u.stopWatch = u.top.watchPaints()
	r, scale := u.placement()
	u.x, u.y = r.Min.X, r.Min.Y
	u.request(u.sub, wlSubsurfaceSetPosition, wlInt(u.x), wlInt(u.y))
	u.setSize(r.Dx(), r.Dy(), scale)
	u.setOpaque()
	bw, bh := u.bufferSize()
	u.buffer = image.Pt(bw, bh)
	if err := u.egl.attach(u.wlSurface, bw, bh); err != nil {
		u.broke(err.Error())
		return
	}
	// The framebuffer keeps the last frame, which the new wl_surface shows
	// even if the frame changes nothing.
	u.show = true
	u.resyncNext()
}

// broke says why the subsurface failed and lets go of it: the window draws
// into the drawing area under it from then on, with a renderer made anew.
func (u *subsurface) broke(why string) {
	if u.failed {
		return
	}
	u.failed = true
	log.Printf("mygo: Wayland subsurface failed, drawing under it: %s", why)
	u.destroy()
	u.s.send(platform.SurfaceEvent{Kind: platform.SurfaceRenew})
}

// active tells that frames draw on the subsurface.
func (u *subsurface) active() bool { return u != nil && !u.failed }

// destroy lets go of the subsurface, as the toplevel's wl_surface goes;
// its context stays for the next, with the last frame. No frame callback
// comes for it any more.
func (u *subsurface) destroy() {
	if u.stopWatch != nil {
		u.stopWatch()
		u.stopWatch = nil
	}
	u.sync.reset(false)
	u.resync = false
	if u.frameCB != 0 {
		u.c.proxyDestroy(u.frameCB)
		u.frameCB = 0
	}
	u.pacer.stop()
	u.egl.detach()
	if u.fraction != 0 {
		u.destroyProxy(u.fraction, wpFractionalScaleDestroy)
		u.fraction = 0
	}
	if u.viewport != 0 {
		u.destroyProxy(u.viewport, wpViewportDestroy)
		u.viewport = 0
	}
	if u.sub != 0 {
		u.destroyProxy(u.sub, wlSubsurfaceDestroy)
		u.sub = 0
	}
	if u.wlSurface != 0 {
		if u.frameCallbacks {
			u.top.frameCallbacks(u.wlSurface, false)
			u.frameCallbacks = false
		}
		u.destroyProxy(u.wlSurface, wlSurfaceDestroy)
		u.wlSurface = 0
	}
	u.mapped = false
	u.x, u.y, u.logicalW, u.logicalH, u.scale = 0, 0, 0, 0, 0
	u.destination, u.buffer = image.Point{}, image.Point{}
	u.opaque = nil
}

// release lets go of the subsurface and its context, as the window goes.
func (u *subsurface) release() {
	u.destroy()
	u.egl.release()
}

// placement returns where the area is on the toplevel's wl_surface, and
// the buffers' scale: 1 at a fractional scale, else GTK's.
func (u *subsurface) placement() (image.Rectangle, int) {
	in, area, toplevel, gtkScale := u.top.allocation()
	scale := 1
	if !u.fractional() && u.g.compositorVersion >= wlSurfaceBufferScaleSince {
		scale = gtkScale
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

// setSize keeps the subsurface's logical size and the buffers' scale,
// telling the compositor a new scale, and at a fractional scale the
// viewport's destination, the logical size. Both apply with the next
// buffer, which the next swap commits.
func (u *subsurface) setSize(w, h, scale int) {
	if scale != u.scale && u.g.compositorVersion >= wlSurfaceBufferScaleSince {
		u.request(u.wlSurface, wlSurfaceSetBufferScale, wlInt(scale))
	}
	u.logicalW, u.logicalH, u.scale = w, h, scale
	if _, d := fractionalSizes(image.Pt(w, h), u.scale120); u.fractional() && d != u.destination {
		u.request(u.viewport, wpViewportSetDestination, wlInt(d.X), wlInt(d.Y))
		u.destination = d
	}
}

// place moves and sizes the subsurface to the area, as it is allocated or
// its scale changes. The position applies as GDK commits the toplevel next,
// and the size with the next frame: one moved or of another size draws as
// GTK paints the area, in step with that commit (resyncNext).
func (u *subsurface) place() {
	if !u.mapped {
		return
	}
	r, scale := u.placement()
	was := image.Rect(u.x, u.y, u.x+u.logicalW, u.y+u.logicalH)
	if r.Min.X != u.x || r.Min.Y != u.y {
		u.x, u.y = r.Min.X, r.Min.Y
		u.request(u.sub, wlSubsurfaceSetPosition, wlInt(u.x), wlInt(u.y))
	}
	u.setSize(r.Dx(), r.Dy(), scale)
	u.setOpaque()
	resized := r != was
	if b := image.Pt(u.bufferSize()); b != u.buffer {
		u.buffer = b
		u.egl.resize(b.X, b.Y)
		resized = true
	}
	if resized {
		u.show = true
		u.egl.blits.reset()
		u.resyncNext()
	}
}

// MakeGLCurrent makes the subsurface's context current with the
// framebuffer frames are drawn into bound, as large as mygo's frames
// (platform.GLTarget).
func (u *subsurface) MakeGLCurrent() bool {
	if !u.mapped || u.failed {
		return false
	}
	fw, fh := u.frameSize()
	return u.egl.target(fw, fh)
}

// DrewGL tells that the frame drawing changed the framebuffer within
// damage, or all of it, which then shows (platform.GLTarget).
func (u *subsurface) DrewGL(damage []image.Rectangle, whole bool) {
	u.drew = true
	u.whole = u.whole || whole
	u.damage = append(u.damage, damage...)
}

var _ platform.GLTarget = (*subsurface)(nil)

// frame draws the frame asked for into the framebuffer, with the context
// current, the GPU renderer drawing into it (Native), and shows it, asking
// for the frame callback that paces the next; it reports whether it showed
// one. A frame that changed nothing shows none.
func (u *subsurface) frame() bool {
	s := u.s
	if !u.mapped || u.failed || s.w.closed {
		return false
	}
	if !u.MakeGLCurrent() {
		u.broke("EGL cannot make the context current with a framebuffer")
		return false
	}
	u.drew, u.whole, u.damage = false, false, u.damage[:0]
	s.rendering = true
	s.send(platform.SurfaceEvent{Kind: platform.SurfaceFrame})
	s.rendering = false
	if !u.mapped || u.failed || !u.drew && !u.show {
		return false
	}
	return u.present()
}

// present shows the frame drawn into the framebuffer, asking for the frame
// callback that paces the next; it reports whether it did.
func (u *subsurface) present() bool {
	// A frame drawn in step with the toplevel may draw with a callback
	// due, which then paces nothing: its done is let go of.
	if u.frameCB != 0 {
		u.c.proxyDestroy(u.frameCB)
		u.frameCB = 0
	}
	// The request goes in the commit of the swap.
	cb := u.newObject(u.wlSurface, wlSurfaceFrame, u.c.callbackInterface, 0)
	if cb == 0 {
		u.broke("wl_surface.frame failed")
		return false
	}
	u.c.addListener(cb, subsurfaceFrameListener, ptr(u.s.w.id))
	u.frameCB = cb
	// A frame that drew nothing shows as the first after a change, whole.
	bw, bh := u.bufferSize()
	if err := u.egl.present(bw, bh, u.damage, u.whole || !u.drew); err != nil {
		u.broke(err.Error())
		return false
	}
	u.show = false
	return true
}

// frameDone is the frame callback of the last frame shown: the compositor
// can take the next.
func (u *subsurface) frameDone(cb ptr) {
	if cb != u.frameCB {
		return
	}
	u.c.proxyDestroy(cb)
	u.frameCB = 0
	u.pacer.done()
}

var (
	subsurfaceCallbacksOnce            sync.Once
	cbSubsurfaceMap, cbSubsurfaceUnmap ptr
	// cbSubsurfaceAfterPaint follows the after-paint of a toplevel's frame
	// clock (subsurface_sync.go).
	cbSubsurfaceAfterPaint ptr
	// cbSubsurfaceFrame runs the frame asked for, from an idle callback or
	// a timeout; subsurfaceFrameListener is the wl_callback_listener of the
	// frame callbacks.
	cbSubsurfaceFrame       ptr
	subsurfaceFrameListener ptr
	// subsurfaceScaleListener is the wp_fractional_scale_v1_listener of
	// the subsurfaces' fractional scales.
	subsurfaceScaleListener ptr
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
		cbSubsurfaceAfterPaint = purego.NewCallback(func(clock, data ptr) {
			if s := theBackend.surfaceOf(data); s != nil && s.sub != nil {
				s.sub.afterPaint()
			}
		})
		cbSubsurfaceFrame = purego.NewCallback(func(data ptr) int32 {
			if s := theBackend.surfaceOf(data); s != nil && s.sub != nil {
				s.sub.pacer.run()
			}
			return 0 // G_SOURCE_REMOVE
		})
		done := purego.NewCallback(func(data, callback ptr, time uint32) {
			if s := theBackend.surfaceOf(data); s != nil && s.sub != nil {
				s.sub.frameDone(callback)
			}
		})
		// struct wl_callback_listener: done. In C memory, which libwayland
		// keeps a pointer to.
		if l, err := cAlloc(unsafe.Sizeof(ptr(0))); err == nil {
			*(*ptr)(cAt(l)) = done
			subsurfaceFrameListener = l
		}
		scale := purego.NewCallback(func(data, fraction ptr, scale120 uint32) {
			if s := theBackend.surfaceOf(data); s != nil && s.sub != nil {
				s.sub.preferredScale(fraction, scale120)
			}
		})
		// struct wp_fractional_scale_v1_listener: preferred_scale.
		if l, err := cAlloc(unsafe.Sizeof(ptr(0))); err == nil {
			*(*ptr)(cAt(l)) = scale
			subsurfaceScaleListener = l
		}
	})
}
