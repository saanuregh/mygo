//go:build linux && !android && (amd64 || arm64)

package linux

import (
	"log"

	"github.com/egoist/mygo/internal/gpu/gl"
	"github.com/egoist/mygo/internal/platform"
)

// waylandSubsurface is what a window's surface reaches its Wayland
// subsurface (subsurface.go) through: the backend's methods are linked
// into every app, and call only this, which LinkWaylandSubsurface links.
type waylandSubsurface interface {
	platform.GLTarget
	// connect follows the toplevel's mapping, the surface's area made.
	connect()
	// active tells that frames draw on the subsurface.
	active() bool
	// drawScale is the scale frames draw at, 0 for GTK's.
	drawScale() float64
	// requestFrame asks for a frame, paced by the subsurface.
	requestFrame()
	// drewPixels runs present, which draws a frame drawn in memory into
	// the framebuffer, all of it, made current, and shows it (the software
	// fallback of PresentPixels).
	drewPixels(present func())
	// paint follows GTK painting the area, allocated its allocation and
	// place its allocation and scale, webViewsPlaced the content placing
	// its web views.
	paint()
	allocated()
	place()
	webViewsPlaced()
	// release lets go of it, as the window goes.
	release()
}

// waylandSubsurfaces is the code of Wayland subsurfaces, which
// LinkWaylandSubsurface links: find binds the globals they need as the
// backend starts, make makes a surface's, nil for none.
var waylandSubsurfaces struct {
	find func()
	make func(s *surface, gpu bool) waylandSubsurface
}

// subsurfaceGlobals are the globals Wayland subsurfaces need
// (wayland_client.go), nil unless they are linked,
// MYGO_WAYLAND_SUBSURFACE=1 and found. Not the Backend's: what its fields
// hold is linked with it.
var subsurfaceGlobals *waylandGlobals

// LinkWaylandSubsurface links the Wayland subsurface, with the GPU
// renderer drawing into it, which MYGO_WAYLAND_SUBSURFACE=1 then turns on: package
// plugins/waylandsubsurface calls it, as an app imports it, before the
// backend starts.
func LinkWaylandSubsurface() {
	gl.LinkTargets()
	waylandSubsurfaces.find = findWaylandGlobals
	waylandSubsurfaces.make = func(s *surface, gpu bool) waylandSubsurface {
		if u := newSubsurface(s, gpu); u != nil {
			return u
		}
		return nil
	}
}

// findWaylandGlobals binds what the Wayland subsurface needs on GDK's
// display, GDK's wl_compositor, or one of a later version, with the
// globals GDK keeps to itself, and says what it found.
func findWaylandGlobals() {
	var getDisplay, getCompositor func(display ptr) ptr
	if !bind(libGDK, &getDisplay, "gdk_wayland_display_get_wl_display") ||
		!bind(libGDK, &getCompositor, "gdk_wayland_display_get_wl_compositor") {
		log.Print("mygo: Wayland subsurface: GDK has no gdk_wayland_display_get_wl_display or _get_wl_compositor")
		return
	}
	c, err := loadWaylandClient()
	if err != nil {
		log.Printf("mygo: Wayland subsurface: %v", err)
		return
	}
	display := gdkDisplayGetDefault()
	g, err := c.findWaylandGlobals(getDisplay(display), getCompositor(display))
	if err != nil {
		log.Printf("mygo: Wayland subsurface: %v", err)
		return
	}
	subsurfaceGlobals = g
	log.Printf("mygo: Wayland subsurface: %s", g)
}

// subsurfaceOf returns the subsurface of the surface whose window has the
// id data, nil for none.
func subsurfaceOf(data ptr) *subsurface {
	s := theBackend.surfaceOf(data)
	if s == nil {
		return nil
	}
	u, _ := s.sub.(*subsurface)
	return u
}

// subActive tells that the surface's frames draw on its subsurface.
func (s *surface) subActive() bool { return s.sub != nil && s.sub.active() }
