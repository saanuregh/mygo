//go:build linux && !android && (amd64 || arm64)

package linux

import (
	"errors"
	"fmt"
	"image"
	"slices"
	"strings"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

// EGL on the Wayland subsurface (subsurface.go): a context of mygo's own,
// on the EGLDisplay of GDK's wl_display, drawing into a wl_egl_window of
// the subsurface's wl_surface. mygo's renderer draws only what changed
// since the last frame, and Mesa's window surfaces keep no frame from one
// swap to the next: frames are drawn into a framebuffer of the context's
// own, a texture as large as the subsurface's buffers, which keeps the
// last, and copied into the window's for each swap: where EGL tells the
// back buffer's age, only what changed since that buffer showed
// (subsurface_blit.go), else all of it. At a fractional scale, the
// framebuffer is mygo's frame, which may be a pixel larger than the
// window's buffers: its top left is copied.

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
	eglExtensions            = 0x3055
	eglBufferAgeEXT          = 0x313D
	glColorBufferBit         = 0x4000
	glFramebuffer            = 0x8D40
	glReadFramebuffer        = 0x8CA8
	glDrawFramebuffer        = 0x8CA9
	glFramebufferComplete    = 0x8CD5
	glColorAttachment0       = 0x8CE0
	glTexture2D              = 0x0DE1
	glTextureMinFilter       = 0x2801
	glTextureMagFilter       = 0x2800
	glNearest                = 0x2600
	glRGBA8                  = 0x8058
	glRGBA                   = 0x1908
	glUnsignedByte           = 0x1401
	glScissorTest            = 0x0C11
)

// eglLib is libEGL, libwayland-egl, the GL functions of the framebuffer
// frames are drawn into, through libepoxy, as mygo's renderer calls GL,
// and GDK's gdk_gl_context_clear_current.
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
	querySurface        func(display, surface ptr, attrib int32, value *int32) uint32
	queryString         func(display ptr, name int32) string
	getProcAddress      func(name *byte) ptr

	windowCreate  func(surface ptr, width, height int32) ptr // wl_egl_window_create
	windowResize  func(window ptr, width, height, dx, dy int32)
	windowDestroy func(window ptr)

	clearColor           func(r, g, b, a float32)
	clear                func(mask uint32)
	disable              func(cap uint32)
	genTextures          func(n int32, out *uint32)
	deleteTextures       func(n int32, textures *uint32)
	bindTexture          func(target, texture uint32)
	texImage2D           func(target uint32, level, internalFormat, w, h, border int32, format, typ uint32, pix ptr)
	texParameteri        func(target, name uint32, v int32)
	genFramebuffers      func(n int32, out *uint32)
	bindFramebuffer      func(target, fb uint32)
	framebufferTexture2D func(target, attachment, texTarget, texture uint32, level int32)
	checkFramebuffer     func(target uint32) uint32
	blitFramebuffer      func(sx0, sy0, sx1, sy1, dx0, dy0, dx1, dy1 int32, mask, filter uint32)
	glGetError           func() uint32

	// clearCurrent is gdk_gl_context_clear_current: GDK remembers the
	// context it made current last and makes it current again only when it
	// thinks another is, so it must forget it before mygo's is.
	clearCurrent func()
}

// openEGL loads what the subsurface's EGL calls (loadEGL, once).
func openEGL() (*eglLib, error) {
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
		{&l.querySurface, "eglQuerySurface"}, {&l.queryString, "eglQueryString"},
		{&l.getProcAddress, "eglGetProcAddress"},
	}
	for _, f := range funcs {
		if !bind(egl, f.fn, f.name) {
			return nil, fmt.Errorf("libEGL has no %s", f.name)
		}
	}
	// EGL 1.5's, else EXT_platform_base's, whose attributes are EGLint
	// rather than EGLAttrib: none are passed. GDK looks them up so too.
	if !bind(egl, &l.getPlatformDisplay, "eglGetPlatformDisplay") {
		if p := l.getProcAddress(cs("eglGetPlatformDisplayEXT")); p != 0 {
			purego.RegisterFunc(&l.getPlatformDisplay, p)
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
	}{
		{&l.clearColor, "glClearColor"}, {&l.clear, "glClear"}, {&l.disable, "glDisable"},
		{&l.genTextures, "glGenTextures"}, {&l.deleteTextures, "glDeleteTextures"}, {&l.bindTexture, "glBindTexture"},
		{&l.texImage2D, "glTexImage2D"}, {&l.texParameteri, "glTexParameteri"},
		{&l.genFramebuffers, "glGenFramebuffers"}, {&l.bindFramebuffer, "glBindFramebuffer"},
		{&l.framebufferTexture2D, "glFramebufferTexture2D"}, {&l.checkFramebuffer, "glCheckFramebufferStatus"},
		{&l.blitFramebuffer, "glBlitFramebuffer"}, {&l.glGetError, "glGetError"},
	} {
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
}

// eglLoaded is what the subsurface's EGL calls, as loaded once (as
// waylandClientLoaded is).
var eglLoaded struct {
	once sync.Once
	l    *eglLib
	err  error
}

func loadEGL() (*eglLib, error) {
	e := &eglLoaded
	e.once.Do(func() { e.l, e.err = openEGL() })
	return e.l, e.err
}

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
	// bufferAge tells that EGL tells the back buffer's age
	// (EGL_EXT_buffer_age) and swapWithDamage takes what a frame changed
	// (EGL_KHR_swap_buffers_with_damage, or EXT's): frames are copied in
	// part then. blits remembers what the last frames changed.
	bufferAge      bool
	swapWithDamage func(display, surface ptr, rects *int32, n int32) uint32
	blits          blitHistory
	// fb is the framebuffer frames are drawn into, of tex, w×h pixels;
	// 0 until the first.
	fb, tex uint32
	w, h    int
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
			e.partialCopies()
			return e, nil
		}
	}
	return nil, fmt.Errorf("EGL makes no OpenGL 3.3 or OpenGL ES 3.0 context for a window (%#x)", l.getError())
}

// partialCopies finds whether the display tells back buffers' ages and
// takes what a swap changed, which copying only that needs: without
// either, frames are copied whole.
func (e *subsurfaceEGL) partialCopies() {
	l := e.l
	exts := strings.Fields(l.queryString(e.display, eglExtensions))
	if !slices.Contains(exts, "EGL_EXT_buffer_age") {
		return
	}
	for _, f := range []struct{ ext, name string }{
		{"EGL_KHR_swap_buffers_with_damage", "eglSwapBuffersWithDamageKHR"},
		{"EGL_EXT_swap_buffers_with_damage", "eglSwapBuffersWithDamageEXT"},
	} {
		if !slices.Contains(exts, f.ext) {
			continue
		}
		if p := l.getProcAddress(cs(f.name)); p != 0 {
			purego.RegisterFunc(&e.swapWithDamage, p)
			e.bufferAge = true
			return
		}
	}
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
	e.blits.reset()
	return nil
}

// resize sizes the window's next buffer bw×bh pixels.
func (e *subsurfaceEGL) resize(bw, bh int) {
	if e.window != 0 {
		e.l.windowResize(e.window, int32(bw), int32(bh), 0, 0)
	}
	e.blits.reset()
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

// release lets go of the window surface and the context, which takes the
// framebuffer frames are drawn into with it.
func (e *subsurfaceEGL) release() {
	e.detach()
	if e.context != 0 {
		e.l.destroyContext(e.display, e.context)
		e.context = 0
	}
	e.fb, e.tex, e.w, e.h = 0, 0, 0, 0
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

// target makes the context current with the framebuffer frames are drawn
// into bound, bw×bh pixels, the size of mygo's frames: made anew,
// transparent, when it is another size, as the renderer then draws a
// frame whole (it does after every SurfaceResize).
func (e *subsurfaceEGL) target(bw, bh int) bool {
	if !e.makeCurrent() {
		return false
	}
	l := e.l
	if e.fb == 0 {
		l.genFramebuffers(1, &e.fb)
	}
	l.bindFramebuffer(glFramebuffer, e.fb)
	if e.tex != 0 && e.w == bw && e.h == bh {
		return true
	}
	if e.tex != 0 {
		l.deleteTextures(1, &e.tex)
	}
	l.genTextures(1, &e.tex)
	l.bindTexture(glTexture2D, e.tex)
	l.texParameteri(glTexture2D, glTextureMinFilter, glNearest)
	l.texParameteri(glTexture2D, glTextureMagFilter, glNearest)
	l.texImage2D(glTexture2D, 0, glRGBA8, int32(bw), int32(bh), 0, glRGBA, glUnsignedByte, 0)
	l.bindTexture(glTexture2D, 0)
	l.framebufferTexture2D(glFramebuffer, glColorAttachment0, glTexture2D, e.tex, 0)
	e.w, e.h = bw, bh
	e.blits.reset()
	if l.checkFramebuffer(glFramebuffer) != glFramebufferComplete {
		l.deleteTextures(1, &e.tex)
		e.tex, e.w, e.h = 0, 0, 0
		return false
	}
	l.disable(glScissorTest)
	l.clearColor(0, 0, 0, 0)
	l.clear(glColorBufferBit)
	return true
}

// present copies the frame from the framebuffer it was drawn into to the
// window surface's, of bw×bh pixels, and swaps: the swap attaches the buffer and
// commits the wl_surface. damage is what the frame changed, in the
// framebuffer's pixels from the top left, nil or whole for all of it: the
// copy takes what changed since the back buffer last showed, where EGL
// tells its age, else all of it, and the swap tells the compositor what
// changed. Both framebuffers keep GL's rows, which go up, as a
// GtkGLArea's does: where the window's is smaller, the framebuffer's top
// rows are copied, from the left.
func (e *subsurfaceEGL) present(bw, bh int, damage []image.Rectangle, whole bool) error {
	if e.tex == 0 || !e.makeCurrent() {
		return fmt.Errorf("EGL cannot make the context current (%#x)", e.l.getError())
	}
	l := e.l
	bounds := image.Rect(0, 0, min(bw, e.w), min(bh, e.h))
	all := []image.Rectangle{bounds}
	copies, shown := all, all
	if e.bufferAge {
		// The query picks the back buffer the frame draws into.
		var age int32
		if l.querySurface(e.display, e.surface, eglBufferAgeEXT, &age) == 0 {
			age = 0
		}
		if c := e.blits.copies(damage, whole, int(age), bounds); c != nil {
			copies = c
		}
		if c := e.blits.copies(damage, whole, 1, bounds); c != nil {
			shown = c
		}
	}
	l.bindFramebuffer(glReadFramebuffer, e.fb)
	l.bindFramebuffer(glDrawFramebuffer, 0)
	l.disable(glScissorTest)
	for _, r := range copies {
		sy0, sy1 := int32(e.h-r.Max.Y), int32(e.h-r.Min.Y)
		dy0, dy1 := int32(bh-r.Max.Y), int32(bh-r.Min.Y)
		l.blitFramebuffer(int32(r.Min.X), sy0, int32(r.Max.X), sy1, int32(r.Min.X), dy0, int32(r.Max.X), dy1, glColorBufferBit, glNearest)
	}
	l.bindFramebuffer(glFramebuffer, e.fb)
	if err := l.glGetError(); err != 0 {
		e.blits.reset()
		return fmt.Errorf("copying the frame failed (GL error %#x)", err)
	}
	var swapped uint32
	if e.bufferAge && e.swapWithDamage != nil {
		rects := eglDamageRects(shown, bh)
		swapped = e.swapWithDamage(e.display, e.surface, &rects[0], int32(len(shown)))
	} else {
		swapped = l.swapBuffers(e.display, e.surface)
	}
	if swapped == 0 {
		e.blits.reset()
		return fmt.Errorf("eglSwapBuffers failed (%#x)", l.getError())
	}
	if e.bufferAge {
		e.blits.shown(damage, whole)
	}
	return nil
}
