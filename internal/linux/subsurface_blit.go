//go:build linux && !android && (amd64 || arm64)

package linux

import "image"

// What a Wayland subsurface's frame copies into the window surface's back
// buffer (subsurface_egl.go). Mesa's window surfaces keep no frame from
// one swap to the next, but EGL_EXT_buffer_age tells which frame a back
// buffer last held: N frames ago, 0 for none known. Such a buffer needs
// only what changed since, the damage of this frame and of the N-1 before
// it, copied from the framebuffer frames are drawn into, which holds the
// last; one of unknown age, or older than the frames remembered, needs all
// of it.

// blitFrames is how many frames' damage blitHistory remembers: more than
// the back buffers Mesa's Wayland surfaces keep.
const blitFrames = 5

// blitMaxRects is how many rectangles a copy takes before it copies their
// bounds instead.
const blitMaxRects = 16

// blitHistory remembers the damage of the last frames shown on a window
// surface, in its framebuffer's pixels from the top left, since it last
// started over (reset).
type blitHistory struct {
	// frames holds each frame's damage, the last first, nil for a frame
	// that changed all of it.
	frames [][]image.Rectangle
	// fresh tells that no frame showed since the history started over:
	// the next changes all of the surface, for the compositor too.
	fresh bool
}

// reset starts over, as the window surface, its size or the frames' size
// changed: no back buffer is known to hold a frame of the framebuffer's,
// and the next frame shows whole. Neither a move nor a frame held for the
// toplevel's commit (subsurface_sync.go) changes what a back buffer holds:
// EGL counts its age by swaps, and the compositor adds up the damage of
// the commits it applies together.
func (h *blitHistory) reset() {
	h.frames = h.frames[:0]
	h.fresh = true
}

// copies returns what a frame with damage, or changing all of it when
// whole or nil, copies into a back buffer of age, within bounds: nil for
// all of it. Empty damage, not nil, is a frame that changed nothing: the
// copy is what earlier frames changed. Rectangles overlap at most where
// the damage did.
func (h *blitHistory) copies(damage []image.Rectangle, whole bool, age int, bounds image.Rectangle) []image.Rectangle {
	if whole || h.fresh || age <= 0 || age > len(h.frames) {
		return nil
	}
	var out []image.Rectangle
	add := func(rects []image.Rectangle) bool {
		if rects == nil {
			return false
		}
		for _, r := range rects {
			out = addBlitRect(out, r.Intersect(bounds))
		}
		return true
	}
	if !add(damage) {
		return nil
	}
	for _, d := range h.frames[:age-1] {
		if !add(d) {
			return nil
		}
	}
	if len(out) > blitMaxRects {
		u := out[0]
		for _, r := range out[1:] {
			u = u.Union(r)
		}
		out = []image.Rectangle{u}
	}
	if len(out) == 0 {
		// Damage only outside the buffer, as a frame a pixel larger than
		// it has: all of it, as EGL takes no rectangles for all of it too.
		return nil
	}
	return out
}

// shown remembers the damage of the frame just shown, nil or whole for
// all of it, copying the rectangles.
func (h *blitHistory) shown(damage []image.Rectangle, whole bool) {
	var d []image.Rectangle
	if !whole && damage != nil {
		d = append(make([]image.Rectangle, 0, len(damage)), damage...)
	}
	if len(h.frames) < blitFrames {
		h.frames = append(h.frames, nil)
	}
	copy(h.frames[1:], h.frames[:len(h.frames)-1])
	h.frames[0] = d
	h.fresh = false
}

// addBlitRect adds r to rects, unless it is empty or within one of them,
// dropping those within it.
func addBlitRect(rects []image.Rectangle, r image.Rectangle) []image.Rectangle {
	if r.Empty() {
		return rects
	}
	for _, o := range rects {
		if r.In(o) {
			return rects
		}
	}
	out := rects[:0]
	for _, o := range rects {
		if !o.In(r) {
			out = append(out, o)
		}
	}
	return append(out, r)
}

// eglDamageRects returns rects, of a buffer height pixels high, as EGL
// takes them for eglSwapBuffersWithDamage: x, y, width and height each,
// from the bottom left.
func eglDamageRects(rects []image.Rectangle, height int) []int32 {
	out := make([]int32, 0, 4*len(rects))
	for _, r := range rects {
		out = append(out, int32(r.Min.X), int32(height-r.Max.Y), int32(r.Dx()), int32(r.Dy()))
	}
	return out
}
