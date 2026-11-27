//go:build linux && !android && (amd64 || arm64) && !android

package ui

import (
	"errors"

	"github.com/egoist/mygo/internal/gpu/gl"
	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/internal/scene"
	"image"
)

// newGPURenderer returns an OpenGL renderer for a surface drawing in a
// GtkGLArea's render signal, or into the framebuffer of a Wayland
// subsurface. Surfaces draw in memory where OpenGL does not run on a GPU,
// which the backend tells.
func newGPURenderer(n platform.SurfaceNative) (gpuRenderer, error) {
	if n.GLTarget != nil {
		return newTargetRenderer(n.GLTarget)
	}
	if n.GLArea == 0 {
		return nil, nil
	}
	// The renderer reads the version of the context current.
	gl.MakeAreaCurrent(n.GLArea)
	r, err := gl.New()
	if err != nil {
		return nil, err
	}
	// GTK keeps what the area's framebuffer holds from render to render,
	// but as it allocates the area (SurfaceResize, see invalidator).
	r.Partial = true
	return &areaRenderer{Renderer: r, area: n.GLArea}, nil
}

// areaRenderer draws into a GtkGLArea with the area's context current: the
// frame, as a window a button's action creates or hides, may have left
// another current, or none, since GTK made it current (gl.MakeAreaCurrent).
type areaRenderer struct {
	*gl.Renderer
	area uintptr
}

func (r *areaRenderer) Render(s *scene.Scene) error {
	gl.MakeAreaCurrent(r.area)
	return r.Renderer.Render(s)
}

func (r *areaRenderer) Prepare(s *scene.Scene) ([]image.Rectangle, bool, error) {
	gl.MakeAreaCurrent(r.area)
	return r.Renderer.Prepare(s)
}

func (r *areaRenderer) DrawPrepared() error {
	gl.MakeAreaCurrent(r.area)
	return r.Renderer.DrawPrepared()
}

// newTargetRenderer returns an OpenGL renderer drawing into t, which keeps
// the last frame (partial redraws) but as the surface resizes.
func newTargetRenderer(t platform.GLTarget) (gpuRenderer, error) {
	if !t.MakeGLCurrent() {
		return nil, errors.New("ui: the surface's OpenGL context cannot be made current")
	}
	r, err := gl.New()
	if err != nil {
		return nil, err
	}
	r.Partial = true
	return &targetRenderer{Renderer: r, target: t}, nil
}

// targetRenderer draws into a GLTarget, made current again for each frame
// as areaRenderer's area is, and tells the target what each frame drew,
// which it shows. Frames are paced by the backend, which sends them only
// once the display can take one (no displayWaiter).
type targetRenderer struct {
	*gl.Renderer
	target platform.GLTarget
}

func (r *targetRenderer) Render(s *scene.Scene) error {
	if !r.target.MakeGLCurrent() {
		return errors.New("ui: the surface's OpenGL context cannot be made current")
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
