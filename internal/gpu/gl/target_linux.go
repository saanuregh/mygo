//go:build linux && !android && (amd64 || arm64)

package gl

import (
	"errors"
	"image"

	"github.com/egoist/mygo/internal/scene"
)

// Target is a framebuffer of the backend's own that frames draw into,
// which keeps the last frame drawn (platform.GLTarget).
type Target interface {
	// MakeGLCurrent makes the target's context current with its
	// framebuffer bound, and reports whether it did.
	MakeGLCurrent() bool
	// DrewGL tells what the frame changed: damage, or all of it when whole.
	DrewGL(damage []image.Rectangle, whole bool)
}

// FrameRenderer is a renderer drawing frames into a Target.
type FrameRenderer interface {
	Render(s *scene.Scene) error
	Release()
}

// newTarget is NewTargetRenderer's renderer, which LinkTargets links: apps
// whose backend makes no Target (a Wayland subsurface's) do not link it.
var newTarget func(t Target) (FrameRenderer, error)

// LinkTargets links the renderer of Targets, as the backend links what
// makes them.
func LinkTargets() { newTarget = newTargetRenderer }

// NewTargetRenderer returns a renderer drawing into t, which keeps the last
// frame (partial redraws) but as the surface resizes.
func NewTargetRenderer(t Target) (FrameRenderer, error) {
	if newTarget == nil {
		return nil, errors.New("gl: no renderer of targets is linked (LinkTargets)")
	}
	return newTarget(t)
}

func newTargetRenderer(t Target) (FrameRenderer, error) {
	if !t.MakeGLCurrent() {
		return nil, errors.New("gl: the surface's OpenGL context cannot be made current")
	}
	r, err := New()
	if err != nil {
		return nil, err
	}
	r.Partial = true
	return &targetRenderer{Renderer: r, target: t}, nil
}

// targetRenderer draws into a Target, made current again for each frame
// as a GtkGLArea's context is (MakeAreaCurrent), and tells the target what
// each frame drew, which it shows. Frames are paced by the backend, which
// sends them only once the display can take one.
type targetRenderer struct {
	*Renderer
	target Target
}

func (r *targetRenderer) Render(s *scene.Scene) error {
	if !r.target.MakeGLCurrent() {
		return errors.New("gl: the surface's OpenGL context cannot be made current")
	}
	damage, whole, err := r.Renderer.Prepare(s)
	if err != nil {
		return err
	}
	if err := r.Renderer.DrawPrepared(); err != nil {
		return err
	}
	if whole || len(damage) > 0 {
		r.target.DrewGL(damage, whole)
	}
	return nil
}

// Release deletes the renderer's objects with its context current, when
// it can be.
func (r *targetRenderer) Release() {
	r.target.MakeGLCurrent()
	r.Renderer.Release()
}
