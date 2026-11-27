//go:build linux && !android && (amd64 || arm64)

package linux

import (
	"fmt"
	"image"
	"slices"
	"testing"
	"unsafe"

	"github.com/egoist/mygo/internal/platform"
)

// A frame copies into a back buffer what changed since that buffer last
// showed: its own damage and that of the age-1 frames before it, within the
// buffer. A buffer of unknown age (0), one older than the frames
// remembered, a frame or one of those changing all of it, and any frame
// before one shows after the history started over (a new window surface,
// a resize, a new scale) copy all of it (nil).
func TestBlitRegionsByBufferAge(t *testing.T) {
	bounds := image.Rect(0, 0, 3087, 2100)
	key := []image.Rectangle{image.Rect(100, 200, 120, 240)}
	line := []image.Rectangle{image.Rect(0, 600, 3087, 644)}
	caret := []image.Rectangle{image.Rect(140, 200, 144, 240)}
	var h blitHistory
	want := func(name string, got []image.Rectangle, rects ...image.Rectangle) {
		t.Helper()
		if rects == nil && got != nil || !slices.Equal(got, rects) {
			t.Errorf("%s: copies %v, want %v", name, got, fmt.Sprint(rects))
		}
	}

	// Nothing shown yet: whatever the age, all of it.
	want("new history, age 1", h.copies(key, false, 1, bounds))
	h.reset()
	want("started over, age 2", h.copies(key, false, 2, bounds))
	h.shown(nil, true)

	want("age 1 after a whole frame", h.copies(key, false, 1, bounds), key...)
	want("age 2 after only a whole frame", h.copies(key, false, 2, bounds))
	want("unknown age", h.copies(key, false, 0, bounds))
	want("whole frame", h.copies(key, true, 1, bounds))
	h.shown(key, false)
	h.shown(line, false)
	h.shown(caret, false)

	// The frames before this one, the last first: caret, line, key, whole.
	want("age 1", h.copies(key, false, 1, bounds), key...)
	want("age 2", h.copies(key, false, 2, bounds), key[0], caret[0])
	want("age 3", h.copies(key, false, 3, bounds), key[0], caret[0], line[0])
	// The key's own damage again is within the first.
	want("age 4", h.copies(key, false, 4, bounds), key[0], caret[0], line[0])
	want("age 5 reaches the whole frame", h.copies(key, false, 5, bounds))
	want("age 6, older than the history", h.copies(key, false, 6, bounds))
	want("nil damage is all of it", h.copies(nil, false, 1, bounds))

	// Damage is cut to the buffer, which may be a pixel smaller than the
	// frame; damage only outside it copies all of it.
	edge := []image.Rectangle{image.Rect(3080, 2090, 3088, 2101)}
	want("cut to the buffer", h.copies(edge, false, 1, bounds), image.Rect(3080, 2090, 3087, 2100))
	want("only outside", h.copies([]image.Rectangle{image.Rect(3087, 0, 3088, 2101)}, false, 1, bounds))

	// The history holds the last few frames only.
	for range blitFrames + 2 {
		h.shown(caret, false)
	}
	want("age at the history's length", h.copies(key, false, blitFrames, bounds), key[0], caret[0])
	want("age past it", h.copies(key, false, blitFrames+1, bounds))

	// Many rectangles copy their bounds.
	var many []image.Rectangle
	for i := range blitMaxRects + 1 {
		many = append(many, image.Rect(10*i, 0, 10*i+5, 5))
	}
	want("many rectangles", h.copies(many, false, 1, bounds), image.Rect(0, 0, 10*blitMaxRects+5, 5))

	// A resize, a new scale or a sync transaction starts over.
	h.reset()
	want("after a reset, age 1", h.copies(key, false, 1, bounds))
	h.shown(key, false)
	want("one frame after it, age 1", h.copies(caret, false, 1, bounds), caret...)
	want("one frame after it, age 2", h.copies(caret, false, 2, bounds))

	// The damage given to EGL, from the buffer's bottom left.
	if got := eglDamageRects(key, 2100); !slices.Equal(got, []int32{100, 1860, 20, 40}) {
		t.Errorf("eglDamageRects = %v, want [100 1860 20 40]", got)
	}
}

// Frames copy into the window surface's back buffer what changed since it
// last showed, and tell the compositor what the frame changed, flipped to
// the buffer's bottom left; a resize starts over with a whole copy and
// whole damage. Without the extensions, every frame copies whole and swaps
// plainly.
func TestPresentCopiesByBufferAge(t *testing.T) {
	f := newFakeWayland()
	e := f.egl()
	var blits [][4]int32
	var swapped []int32
	plain := 0
	age := int32(0)
	e.l.blitFramebuffer = func(sx0, sy0, sx1, sy1, dx0, dy0, dx1, dy1 int32, mask, filter uint32) {
		// The framebuffer's top left goes to the buffer's.
		if sx0 != dx0 || sx1 != dx1 || sy1-sy0 != dy1-dy0 || sy0-dy0 != int32(e.h-2100) {
			t.Errorf("blit from %d,%d-%d,%d to %d,%d-%d,%d", sx0, sy0, sx1, sy1, dx0, dy0, dx1, dy1)
		}
		blits = append(blits, [4]int32{dx0, dy0, dx1, dy1})
	}
	swap := e.l.swapBuffers
	e.l.swapBuffers = func(display, surface ptr) uint32 { plain++; return swap(display, surface) }
	e.window, e.surface = 0x900, 0x901
	f.wlSurfaceOf[0x900] = "surface#1"
	present := func(damage []image.Rectangle, whole bool) {
		t.Helper()
		blits, swapped = nil, nil
		if err := e.present(3087, 2100, damage, whole); err != nil {
			t.Fatal(err)
		}
	}
	key := []image.Rectangle{image.Rect(100, 200, 120, 240)}

	// No extensions: whole copies, plain swaps.
	present(key, false)
	if len(blits) != 1 || blits[0] != [4]int32{0, 0, 3087, 2100} || plain != 1 {
		t.Fatalf("without the extensions: blits %v, %d plain swaps", blits, plain)
	}

	e.bufferAge = true
	e.l.querySurface = func(display, surface ptr, attrib int32, value *int32) uint32 {
		if attrib != eglBufferAgeEXT {
			t.Errorf("queried %#x", attrib)
		}
		*value = age
		return 1
	}
	e.swapWithDamage = func(display, surface ptr, rects *int32, n int32) uint32 {
		swapped = append(swapped, unsafe.Slice(rects, 4*int(n))...)
		return 1
	}
	e.blits.reset()
	// The first after the reset: whole, whatever the age.
	age = 2
	present(key, false)
	if len(blits) != 1 || !slices.Equal(swapped, []int32{0, 0, 3087, 2100}) {
		t.Fatalf("first frame: blits %v, damage %v", blits, swapped)
	}
	// Age 0 copies whole but tells the frame's damage.
	age = 0
	present(key, false)
	if len(blits) != 1 || blits[0] != [4]int32{0, 0, 3087, 2100} || !slices.Equal(swapped, []int32{100, 1860, 20, 40}) {
		t.Fatalf("age 0: blits %v, damage %v", blits, swapped)
	}
	// Age 1: the key only, flipped.
	age = 1
	present(key, false)
	if len(blits) != 1 || blits[0] != [4]int32{100, 1860, 120, 1900} || !slices.Equal(swapped, []int32{100, 1860, 20, 40}) {
		t.Fatalf("age 1: blits %v, damage %v", blits, swapped)
	}
	// A resize starts over.
	e.resize(1600, 1000)
	age = 1
	present(key, false)
	if len(blits) != 1 || !slices.Equal(swapped, []int32{0, 0, 3087, 2100}) {
		t.Fatalf("after a resize: blits %v, damage %v", blits, swapped)
	}
}

// frameHandler draws the content's frames with draw.
type frameHandler struct {
	platform.WindowHandler
	draw func()
}

func (h *frameHandler) SurfaceEvent(ev platform.SurfaceEvent) bool {
	if ev.Kind == platform.SurfaceFrame && h.draw != nil {
		h.draw()
	}
	return false
}

// Each frame a subsurface shows copies what changed into the back buffer
// by its age, whichever renderer drew it: a frame the software renderer
// draws after the GPU renderer failed, all of the framebuffer, copies
// whole into a buffer of age 2, which would otherwise keep what the GPU
// frame before left there. A move, and a frame held for the toplevel's
// commit, change nothing a back buffer holds: the next frames still copy
// only what changed.
func TestSubsurfaceFramesCopyWhatChangedByAge(t *testing.T) {
	f := newFakeWayland()
	f.names[0xA00] = "toplevel"
	area := gdkRectangle{Width: 1000, Height: 750}
	at := image.Point{}
	h := &frameHandler{}
	u := &subsurface{
		s:   &surface{w: &window{id: 7, h: h}},
		c:   f.client(),
		g:   &waylandGlobals{compositor: 1, subcompositor: 2, viewporter: 3, fractionManager: 4, compositorVersion: 4},
		egl: f.egl(),
		top: subsurfaceToplevel{
			surface: func() ptr { return 0xA00 },
			allocation: func() (image.Point, gdkRectangle, gdkRectangle, int) {
				return at, area, area, 2
			},
			queuePaint:  func() {},
			watchPaints: func() func() { return func() {} },
		},
	}
	u.sync = subsurfaceSync{setSync: func() {}, setDesync: func() {}, queuePaint: func() {}}
	u.pacer = framePacer{soon: func() {}, later: func() {}, draw: u.frame}
	e := u.egl
	age := int32(0)
	e.bufferAge = true
	e.l.querySurface = func(display, surface ptr, attrib int32, value *int32) uint32 {
		*value = age
		return 1
	}
	e.swapWithDamage = func(display, surface ptr, rects *int32, n int32) uint32 { return 1 }
	var blits []image.Rectangle
	e.l.blitFramebuffer = func(sx0, sy0, sx1, sy1, dx0, dy0, dx1, dy1 int32, mask, filter uint32) {
		blits = append(blits, image.Rect(int(dx0), 1500-int(dy1), int(dx1), 1500-int(dy0)))
	}
	u.create()
	e.w, e.h = u.frameSize()
	whole := []image.Rectangle{image.Rect(0, 0, 2000, 1500)}
	key := image.Rect(100, 200, 120, 240)
	frame := func(n int32, draw func()) []image.Rectangle {
		t.Helper()
		age, blits, h.draw = n, nil, draw
		if !u.frame() {
			t.Fatal("the frame showed nothing")
		}
		return blits
	}
	gpu := func(damage ...image.Rectangle) func() {
		return func() { u.DrewGL(damage, false) }
	}
	software := func() { u.drewPixels(func() {}) }

	if got := frame(0, gpu(whole[0])); !slices.Equal(got, whole) {
		t.Fatalf("first frame: copies %v", got)
	}
	if got := frame(1, gpu(key)); !slices.Equal(got, []image.Rectangle{key}) {
		t.Fatalf("a key, age 1: copies %v", got)
	}
	if got := frame(2, software); !slices.Equal(got, whole) {
		t.Fatalf("the software renderer's frame, age 2: copies %v, want all of it", got)
	}
	if got := frame(2, gpu(key)); !slices.Equal(got, whole) {
		t.Fatalf("a key after the software frame, age 2: copies %v, want all of it", got)
	}
	if got := frame(1, gpu(key)); !slices.Equal(got, []image.Rectangle{key}) {
		t.Fatalf("a key, age 1: copies %v", got)
	}

	// Moved, the same size.
	at = image.Pt(0, 40)
	u.place()
	if got := frame(1, gpu(key)); !slices.Equal(got, []image.Rectangle{key}) {
		t.Fatalf("a key after a move, age 1: copies %v", got)
	}
	// Held for the toplevel's commit.
	var held []image.Rectangle
	u.syncOnce(func() { held = frame(1, gpu(key)) })
	if !slices.Equal(held, []image.Rectangle{key}) {
		t.Fatalf("a key held for the toplevel's commit, age 1: copies %v", held)
	}
	// Resized: all of it.
	area = gdkRectangle{Width: 800, Height: 600}
	u.place()
	e.w, e.h = u.frameSize()
	if got := frame(1, gpu(key)); len(got) != 1 || got[0].Dx() != 1600 {
		t.Fatalf("a key after a resize: copies %v, want all of it", got)
	}
}
