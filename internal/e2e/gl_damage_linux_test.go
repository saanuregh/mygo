//go:build linux && !android && (amd64 || arm64)

package e2e

import (
	"image"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/linux"
	"github.com/egoist/mygo/ui"
)

// GTK 3 tells a Wayland compositor what each paint of a window it
// composites with OpenGL changed in the window's coordinates unscaled,
// where EGL takes the buffer's pixels: at a scale of 2 the compositor
// learned of a change at the top of the window in its lower half, at half
// its size, and showed title buttons that flickered and a shadow missing
// on some sides (#1071). The window tells it what GTK repaints, in the
// surface's coordinates: a box that changes color within the GtkGLArea,
// below GTK's title bar and shadow, is in it. It runs on Wayland at a
// scale of 2 or more (weston --scale=2), with MYGO_GPU=1.
func TestGLDamageWayland(t *testing.T) {
	if os.Getenv("MYGO_GPU") != "1" {
		t.Skip("needs the GtkGLArea: MYGO_GPU=1")
	}
	var frames atomic.Int32
	var lit atomic.Bool
	box := ui.Rect{X: 300, Y: 200, W: 40, H: 30}
	view := func(c *ui.Context) {
		frames.Add(1)
		ui.Box(c).Fill().Background(ui.RGB(255, 255, 255)).Draw(func(pt *ui.Painter, r ui.Rect) {
			color := ui.RGB(200, 200, 200)
			if lit.Load() {
				color = ui.RGB(255, 0, 0)
			}
			pt.Fill(ui.Rect{X: r.X + box.X, Y: r.Y + box.Y, W: box.W, H: box.H}, color, 0)
		})
	}
	w := newWindow(t, mygo.WindowOptions{Title: "Damage", Width: 400, Height: 300, Content: ui.View(view)})
	eventually(t, "a frame with OpenGL", func() bool {
		how, _, _, _, _ := glSurface(w)
		return how == "opengl"
	})
	glDamage := func() (d []image.Rectangle, area image.Point, scale int, ok bool) {
		mygo.RunOnMain(func() { d, area, scale, ok = linux.TestGLDamage(w.NativeHandle()) })
		return d, area, scale, ok
	}
	_, area, scale, ok := glDamage()
	if !ok {
		t.Skip("GTK tells the compositor what changed itself but on Wayland")
	}
	if scale < 2 {
		t.Skipf("GTK tells the compositor what changed itself at a scale of %d", scale)
	}
	want := image.Rect(int(box.X), int(box.Y), int(box.X+box.W), int(box.Y+box.H)).Add(area)
	var damage []image.Rectangle
	before := frames.Load()
	w.Update(func() { lit.Store(true) })
	eventually(t, "the box's change told to the compositor", func() bool {
		d, _, _, _ := glDamage()
		damage = append(damage, d...)
		if frames.Load() == before {
			return false
		}
		for _, r := range damage {
			if want.In(r) {
				return true
			}
		}
		return false
	})
}

// A frame that changes a box repaints the box, not the window: GTK 3
// makes a CPU surface the size of what it repaints, and clears it, for
// every paint of a window it composites with OpenGL, and asking a
// GtkGLArea to render repaints all of it, which cost a large window
// milliseconds a frame. The window builds a frame before GTK paints and
// asks GTK to repaint what it changed; the area's framebuffer then holds
// the box in its new color. Wayland at a scale of 2 or more (weston
// --scale=2), with MYGO_GPU=1, where TestGLDamage sees each paint.
func TestGLRepaintsOnlyWhatChanged(t *testing.T) {
	if os.Getenv("MYGO_GPU") != "1" {
		t.Skip("needs the GtkGLArea: MYGO_GPU=1")
	}
	var frames atomic.Int32
	var lit atomic.Bool
	box := ui.Rect{X: 300, Y: 200, W: 40, H: 30}
	view := func(c *ui.Context) {
		frames.Add(1)
		ui.Box(c).Fill().Background(ui.RGB(255, 255, 255)).Draw(func(pt *ui.Painter, r ui.Rect) {
			color := ui.RGB(200, 200, 200)
			if lit.Load() {
				color = ui.RGB(255, 0, 0)
			}
			pt.Fill(ui.Rect{X: r.X + box.X, Y: r.Y + box.Y, W: box.W, H: box.H}, color, 0)
		})
	}
	// Frameless: GTK repaints its title bar of its own accord, which a paint
	// of the box then joins.
	w := newWindow(t, mygo.WindowOptions{Title: "Repaint", Width: 600, Height: 400, Frameless: true, Content: ui.View(view)})
	eventually(t, "a frame with OpenGL", func() bool {
		how, _, _, _, _ := glSurface(w)
		return how == "opengl"
	})
	glDamage := func() (d []image.Rectangle, area image.Point, scale int, ok bool) {
		mygo.RunOnMain(func() { d, area, scale, ok = linux.TestGLDamage(w.NativeHandle()) })
		return d, area, scale, ok
	}
	_, area, scale, ok := glDamage()
	if !ok || scale < 2 {
		t.Skipf("the paints are seen on Wayland at a scale of 2 or more (ok %v, scale %d)", ok, scale)
	}
	// What a frame's GPU damage grows by, rounded out to whole DIPs.
	near := image.Rect(int(box.X), int(box.Y), int(box.X+box.W), int(box.Y+box.H)).Add(area).Inset(-8)
	// The window settles: its first frames and paints repaint all of it.
	quiet := 0
	eventually(t, "the window at rest", func() bool {
		d, _, _, _ := glDamage()
		if len(d) > 0 {
			quiet = 0
			return false
		}
		quiet++
		time.Sleep(20 * time.Millisecond)
		return quiet >= 10
	})
	boxAt := image.Rect(int(box.X), int(box.Y), int(box.X+box.W), int(box.Y+box.H)).Add(area)
	// GTK repaints, with what a frame changed, what the frames before it
	// changed in the buffer it draws into again (EGL's buffer age): after
	// the window's whole first paints, the box's paints shrink to the box
	// within a few frames. GTK paints parts of the window of its own
	// accord too: the box's paints are those over it.
	var last image.Rectangle
	for i := range 5 {
		before := frames.Load()
		w.Update(func() { lit.Store(i%2 == 0) }) // red first and last
		var painted []image.Rectangle
		eventually(t, "the box's frame painted", func() bool {
			d, _, _, _ := glDamage()
			for _, r := range d {
				if r.Overlaps(boxAt) {
					painted = append(painted, r)
				}
			}
			return frames.Load() > before && len(painted) > 0
		})
		last = painted[len(painted)-1]
	}
	if !last.In(near) {
		t.Errorf("the box's last frame repainted %v, beyond the box's %v", last, near)
	}
	eventually(t, "the box red in the area's framebuffer", func() bool {
		_, pix, width, _, ok := glSurface(w)
		if !ok {
			return false
		}
		x, y := int(box.X+box.W/2)*scale, int(box.Y+box.H/2)*scale
		i := (y*width + x) * 4 // BGRA
		return i+3 < len(pix) && pix[i+2] > 200 && pix[i+1] < 50 && pix[i] < 50
	})
}
