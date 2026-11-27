//go:build linux && !android && (amd64 || arm64)

package linux

import (
	"image"
	"slices"

	"github.com/egoist/mygo/internal/platform"
)

// A Wayland subsurface's opaque region (subsurface.go): where its frames
// are opaque, which the compositor need not blend with, nor draw, what is
// under them. Frames are opaque but in their holes, where web views show
// from the toplevel under them, as a GtkGLArea's are, which has no alpha
// channel then; a transparent window's are not, and have none.
//
// A compositor may send no frame callbacks for a surface it shows none of,
// as sway does for the toplevel once an opaque subsurface covers it all:
// GDK, which waits for one after each commit, would stop painting. GDK
// asks for the toplevel's callbacks on the subsurface too, then
// (gdk_wayland_window_add_frame_callback_surface), in its after-paint,
// which the subsurface commits right after it (commitFrameRequest). Where
// GDK cannot, the subsurface has no opaque region.

// Requests' opcodes, their order in wayland.xml.
const (
	wlSurfaceSetOpaqueRegion = 4
	wlSurfaceCommit          = 6
	wlRegionAdd              = 1
)

// opaqueRegion returns the opaque region of a subsurface w×h logical
// pixels large, the frames' holes for web views placed cut out: the
// pixels their clips cover at all, what the content painted over them
// included; nil for none.
func opaqueRegion(w, h int, placed []platform.WebViewPlacement, transparent bool) []image.Rectangle {
	if transparent || w <= 0 || h <= 0 {
		return nil
	}
	region := []image.Rectangle{image.Rect(0, 0, w, h)}
	for i := range placed {
		r := gdkRectOut(placed[i].Clip)
		hole := image.Rect(int(r.X), int(r.Y), int(r.X+r.Width), int(r.Y+r.Height))
		if hole.Empty() {
			continue
		}
		var cut []image.Rectangle
		for _, o := range region {
			cut = append(cut, subtractRect(o, hole)...)
		}
		region = cut
	}
	return region
}

// subtractRect returns the rectangles of r outside hole: those above and
// below it as wide as r, those beside it as high as hole.
func subtractRect(r, hole image.Rectangle) []image.Rectangle {
	in := r.Intersect(hole)
	if in.Empty() {
		return []image.Rectangle{r}
	}
	var out []image.Rectangle
	for _, p := range []image.Rectangle{
		image.Rect(r.Min.X, r.Min.Y, r.Max.X, in.Min.Y),
		image.Rect(r.Min.X, in.Max.Y, r.Max.X, r.Max.Y),
		image.Rect(r.Min.X, in.Min.Y, in.Min.X, in.Max.Y),
		image.Rect(in.Max.X, in.Min.Y, r.Max.X, in.Max.Y),
	} {
		if !p.Empty() {
			out = append(out, p)
		}
	}
	return out
}

// setOpaque tells the compositor the subsurface's opaque region for its
// size and the web views placed, when it changed: it applies with the next
// frame the subsurface commits, synchronized with the toplevel's commit as
// that frame is.
func (u *subsurface) setOpaque() {
	if !u.mapped || u.failed {
		return
	}
	transparent := u.s.w.opts != nil && u.s.w.opts.Transparent
	var region []image.Rectangle
	if u.frameCallbacks {
		region = opaqueRegion(u.logicalW, u.logicalH, u.s.placed, transparent)
	}
	if slices.Equal(region, u.opaque) {
		return
	}
	u.opaque = region
	var wlRegion ptr // none, for an empty region
	if len(region) > 0 {
		if wlRegion = u.newObject(u.g.compositor, wlCompositorCreateRegion, u.c.regionInterface, 0); wlRegion == 0 {
			return
		}
		for _, r := range region {
			u.request(wlRegion, wlRegionAdd, wlInt(r.Min.X), wlInt(r.Min.Y), wlInt(r.Dx()), wlInt(r.Dy()))
		}
	}
	u.request(u.wlSurface, wlSurfaceSetOpaqueRegion, uint64(wlRegion))
	if wlRegion != 0 {
		u.destroyProxy(wlRegion, wlRegionDestroy)
	}
}

// commitFrameRequest commits the subsurface after GDK's after-paint, in
// which GDK may have asked for the toplevel's frame callback on it, which
// only a commit sends, while it has an opaque region: the commit changes
// nothing else, as every other request the subsurface sends before a
// frame is drawn in the same frame clock cycle, before the after-paint.
func (u *subsurface) commitFrameRequest() {
	if u.frameCallbacks && u.opaque != nil {
		u.request(u.wlSurface, wlSurfaceCommit)
	}
}
