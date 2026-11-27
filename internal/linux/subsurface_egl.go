//go:build linux && !android && (amd64 || arm64)

package linux

import (
	"errors"
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

// EGL on the Wayland subsurface (subsurface.go): a context of mygo's own,
// on the EGLDisplay of GDK's wl_display, drawing into a wl_egl_window of
// the subsurface's wl_surface.

// EGL's and GL's enumerations.
const (
	eglPlatformWaylandKHR    = 0x31D8
	eglNone                  = 0x3038
	eglSurfaceType           = 0x3033
	eglWindowBit             = 0x0004
	eglRenderableType        = 0x3040
	eglOpenGLBit             = 0x0008
	eglOpenGLES3Bit          = 0x0040 // EGL_OPENGL_ES3_BIT_KHR
	eglRedSize               = 0x3024
	eglGreenSize             = 0x3023
	eglBlueSize              = 0x3022
	eglAlphaSize             = 0x3021
	eglOpenGLAPI             = 0x30A2
	eglOpenGLESAPI           = 0x30A0
	eglContextMajorVersion   = 0x3098
	eglContextMinorVersion   = 0x30FB
	eglContextProfileMask    = 0x30FD
	eglContextCoreProfileBit = 0x1
	glColorBufferBit         = 0x4000
)

// eglLib is libEGL, libwayland-egl, the GL functions of a clear through
// libepoxy, as mygo's renderer calls GL, and GDK's
// gdk_gl_context_clear_current.
type eglLib struct {
	getPlatformDisplay  func(platform uint32, native, attribs ptr) ptr
	getDisplay          func(native ptr) ptr
	initialize          func(display ptr, major, minor *int32) uint32
	bindAPI             func(api uint32) uint32
	queryAPI            func() uint32
	chooseConfig        func(display ptr, attribs *int32, configs *ptr, size int32, n *int32) uint32
	getConfigAttrib     func(display, config ptr, attrib int32, value *int32) uint32
	createContext       func(display, config, share ptr, attribs *int32) ptr
	createWindowSurface func(display, config, window ptr, attribs *int32) ptr
	makeCurrent         func(display, draw, read, context ptr) uint32
	swapBuffers         func(display, surface ptr) uint32
	swapInterval        func(display ptr, interval int32) uint32
	destroySurface      func(display, surface ptr) uint32
	destroyContext      func(display, context ptr) uint32
	getCurrentContext   func() ptr
	getError            func() int32

	windowCreate  func(surface ptr, width, height int32) ptr // wl_egl_window_create
	windowResize  func(window ptr, width, height, dx, dy int32)
	windowDestroy func(window ptr)

	viewport   func(x, y, width, height int32)
	clearColor func(r, g, b, a float32)
	clear      func(mask uint32)

	// clearCurrent is gdk_gl_context_clear_current: GDK remembers the
	// context it made current last and makes it current again only when it
	// thinks another is, so it must forget it before mygo's is.
	clearCurrent func()
}

// loadEGL loads what the subsurface's EGL calls, once.
var loadEGL = sync.OnceValues(func() (*eglLib, error) {
	l := &eglLib{}
	egl, err := open("libEGL.so.1")
	if err != nil {
		return nil, err
	}
	funcs := []struct {
		fn   any
		name string
	}{
		{&l.getDisplay, "eglGetDisplay"}, {&l.initialize, "eglInitialize"}, {&l.bindAPI, "eglBindAPI"},
		{&l.queryAPI, "eglQueryAPI"}, {&l.chooseConfig, "eglChooseConfig"}, {&l.getConfigAttrib, "eglGetConfigAttrib"},
		{&l.createContext, "eglCreateContext"}, {&l.createWindowSurface, "eglCreateWindowSurface"},
		{&l.makeCurrent, "eglMakeCurrent"}, {&l.swapBuffers, "eglSwapBuffers"}, {&l.swapInterval, "eglSwapInterval"},
		{&l.destroySurface, "eglDestroySurface"}, {&l.destroyContext, "eglDestroyContext"},
		{&l.getCurrentContext, "eglGetCurrentContext"}, {&l.getError, "eglGetError"},
	}
	for _, f := range funcs {
		if !bind(egl, f.fn, f.name) {
			return nil, fmt.Errorf("libEGL has no %s", f.name)
		}
	}
	// EGL 1.5's, else EXT_platform_base's, whose attributes are EGLint
	// rather than EGLAttrib: none are passed. GDK looks them up so too.
	if !bind(egl, &l.getPlatformDisplay, "eglGetPlatformDisplay") {
		var getProcAddress func(name *byte) ptr
		if bind(egl, &getProcAddress, "eglGetProcAddress") {
			if p := getProcAddress(cs("eglGetPlatformDisplayEXT")); p != 0 {
				purego.RegisterFunc(&l.getPlatformDisplay, p)
			}
		}
	}

	wl, err := open("libwayland-egl.so.1")
	if err != nil {
		return nil, err
	}
	for _, f := range []struct {
		fn   any
		name string
	}{{&l.windowCreate, "wl_egl_window_create"}, {&l.windowResize, "wl_egl_window_resize"}, {&l.windowDestroy, "wl_egl_window_destroy"}} {
		if !bind(wl, f.fn, f.name) {
			return nil, fmt.Errorf("libwayland-egl has no %s", f.name)
		}
	}

	epoxy, err := open("libepoxy.so.0")
	if err != nil {
		return nil, err
	}
	// libepoxy exports a pointer for each GL function, to a stub that finds
	// the function of the current context when first called.
	for _, f := range []struct {
		fn   any
		name string
	}{{&l.viewport, "glViewport"}, {&l.clearColor, "glClearColor"}, {&l.clear, "glClear"}} {
		sym, err := purego.Dlsym(epoxy, "epoxy_"+f.name)
		if err != nil || sym == 0 {
			return nil, fmt.Errorf("libepoxy has no %s", f.name)
		}
		purego.RegisterFunc(f.fn, **(**ptr)(unsafe.Pointer(&sym)))
	}
	if !bind(libGDK, &l.clearCurrent, "gdk_gl_context_clear_current") {
		return nil, errors.New("GDK has no gdk_gl_context_clear_current")
	}
	return l, nil
})

// subsurfaceEGL is the subsurface's context, and the window surface it
// draws into while the subsurface's wl_surface exists.
type subsurfaceEGL struct {
	l                        *eglLib
	display, config, context ptr
	// window is the wl_egl_window of the wl_surface, surface its
	// EGLSurface; 0 while the subsurface has none (attach).
	window, surface ptr
	// unsynced tells that the surface's swaps wait for no frame callback.
	unsynced bool
}

// newSubsurfaceEGL makes a context on the EGLDisplay of wlDisplay, which
// is GDK's: OpenGL 3.3 core, else OpenGL ES 3.0, as a GtkGLArea's
// (glContext), with an 8-bit ARGB configuration for windows. The display
// is never terminated, as GDK may be using it.
func newSubsurfaceEGL(wlDisplay ptr) (*subsurfaceEGL, error) {
	l, err := loadEGL()
	if err != nil {
		return nil, err
	}
	e := &subsurfaceEGL{l: l}
	if l.getPlatformDisplay != nil {
		e.display = l.getPlatformDisplay(eglPlatformWaylandKHR, wlDisplay, 0)
	}
	if e.display == 0 {
		e.display = l.getDisplay(wlDisplay)
	}
	if e.display == 0 || l.initialize(e.display, nil, nil) == 0 {
		return nil, fmt.Errorf("EGL has no display for Wayland (%#x)", l.getError())
	}
	// The API is the thread's: GDK binds its own before making a context.
	api := l.queryAPI()
	defer l.bindAPI(api)
	attempts := []struct {
		api     uint32
		bit     int32
		version []int32
	}{
		{eglOpenGLAPI, eglOpenGLBit, []int32{eglContextMajorVersion, 3, eglContextMinorVersion, 3, eglContextProfileMask, eglContextCoreProfileBit, eglNone}},
		{eglOpenGLESAPI, eglOpenGLES3Bit, []int32{eglContextMajorVersion, 3, eglContextMinorVersion, 0, eglNone}},
	}
	for _, a := range attempts {
		if l.bindAPI(a.api) == 0 {
			continue
		}
		if e.config = e.chooseARGB8888(a.bit); e.config == 0 {
			continue
		}
		if e.context = l.createContext(e.display, e.config, 0, &a.version[0]); e.context != 0 {
			return e, nil
		}
	}
	return nil, fmt.Errorf("EGL makes no OpenGL 3.3 or OpenGL ES 3.0 context for a window (%#x)", l.getError())
}

// chooseARGB8888 returns a configuration for windows rendering with the
// API of bit whose channels are 8 bits each, alpha included, or 0: EGL
// sorts deeper ones first.
func (e *subsurfaceEGL) chooseARGB8888(bit int32) ptr {
	attribs := []int32{eglSurfaceType, eglWindowBit, eglRenderableType, bit,
		eglRedSize, 8, eglGreenSize, 8, eglBlueSize, 8, eglAlphaSize, 8, eglNone}
	var configs [64]ptr
	var n int32
	if e.l.chooseConfig(e.display, &attribs[0], &configs[0], int32(len(configs)), &n) == 0 {
		return 0
	}
	for _, c := range configs[:n] {
		exact := true
		for _, a := range []int32{eglRedSize, eglGreenSize, eglBlueSize, eglAlphaSize} {
			var v int32
			if e.l.getConfigAttrib(e.display, c, a, &v) == 0 || v != 8 {
				exact = false
			}
		}
		if exact {
			return c
		}
	}
	return 0
}

// attach makes the window surface on wlSurface, of bw×bh pixels.
func (e *subsurfaceEGL) attach(wlSurface ptr, bw, bh int) error {
	if e.window = e.l.windowCreate(wlSurface, int32(bw), int32(bh)); e.window == 0 {
		return errors.New("wl_egl_window_create failed")
	}
	if e.surface = e.l.createWindowSurface(e.display, e.config, e.window, nil); e.surface == 0 {
		err := fmt.Errorf("EGL makes no window surface (%#x)", e.l.getError())
		e.l.windowDestroy(e.window)
		e.window = 0
		return err
	}
	e.unsynced = false
	return nil
}

// resize sizes the window's next buffer bw×bh pixels.
func (e *subsurfaceEGL) resize(bw, bh int) {
	if e.window != 0 {
		e.l.windowResize(e.window, int32(bw), int32(bh), 0, 0)
	}
}

// detach lets go of the window surface, before its wl_surface goes.
func (e *subsurfaceEGL) detach() {
	if e.surface != 0 {
		if e.l.getCurrentContext() == e.context {
			e.l.makeCurrent(e.display, 0, 0, 0)
		}
		e.l.destroySurface(e.display, e.surface)
		e.surface = 0
	}
	if e.window != 0 {
		e.l.windowDestroy(e.window)
		e.window = 0
	}
}

// release lets go of the window surface and the context.
func (e *subsurfaceEGL) release() {
	e.detach()
	if e.context != 0 {
		e.l.destroyContext(e.display, e.context)
		e.context = 0
	}
}

// makeCurrent makes the context current on the window surface, past GDK's
// memory of its own.
func (e *subsurfaceEGL) makeCurrent() bool {
	if e.surface == 0 {
		return false
	}
	e.l.clearCurrent()
	if e.l.makeCurrent(e.display, e.surface, e.surface, e.context) == 0 {
		return false
	}
	if !e.unsynced {
		// Mesa's swaps otherwise wait for the compositor's frame callback
		// of the last, which a covered window never gets.
		e.l.swapInterval(e.display, 0)
		e.unsynced = true
	}
	return true
}

// fill shows a buffer of bw×bh pixels of one colour, premultiplied.
func (e *subsurfaceEGL) fill(c [4]float32, bw, bh int) error {
	if !e.makeCurrent() {
		return fmt.Errorf("EGL cannot make the context current (%#x)", e.l.getError())
	}
	e.l.viewport(0, 0, int32(bw), int32(bh))
	e.l.clearColor(c[0], c[1], c[2], c[3])
	e.l.clear(glColorBufferBit)
	if e.l.swapBuffers(e.display, e.surface) == 0 {
		return fmt.Errorf("eglSwapBuffers failed (%#x)", e.l.getError())
	}
	return nil
}
