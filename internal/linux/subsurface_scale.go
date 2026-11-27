//go:build linux && !android && (amd64 || arm64)

package linux

import (
	"image"
	"math"

	"github.com/egoist/mygo/internal/platform"
)

// A Wayland subsurface's fractional scale (subsurface.go): the scale the
// compositor prefers for it (wp_fractional_scale_v1), which mygo draws at,
// and the sizes of its buffers, shown at its logical size (wp_viewport).

// fractional tells that the buffers are at the compositor's preferred
// scale, which it sent, with a viewport showing them.
func (u *subsurface) fractional() bool { return u.scale120 != 0 && u.viewport != 0 }

// drawScale is the scale mygo draws the subsurface's frames at
// (surface.Size): the compositor's preferred, or 0 until it sends one, as
// it may be sent only with a viewport to show it.
func (u *subsurface) drawScale() float64 {
	if !u.active() || u.scale120 == 0 {
		return 0
	}
	return float64(u.scale120) / 120
}

// bufferSize returns the size of the buffers, in pixels.
func (u *subsurface) bufferSize() (int, int) {
	if u.fractional() {
		b, _ := fractionalSizes(image.Pt(u.logicalW, u.logicalH), u.scale120)
		return b.X, b.Y
	}
	return u.logicalW * u.scale, u.logicalH * u.scale
}

// frameSize returns the size of mygo's frames, which the framebuffer they
// are drawn into has: the buffers', but at a fractional scale, where mygo
// rounds up and the buffers to the nearest, up to a pixel larger.
func (u *subsurface) frameSize() (int, int) {
	if f := u.drawScale(); f > 0 {
		return framePixels(u.logicalW, f), framePixels(u.logicalH, f)
	}
	return u.bufferSize()
}

// fractionalSizes returns the size of the buffers of a surface logical
// pixels large at the preferred scale scale120/120, rounded half away from
// zero (fractional-scale-v1 rounds a toplevel's so and leaves a
// subsurface's undefined), and the viewport's destination, its logical
// size.
func fractionalSizes(logical image.Point, scale120 uint32) (buffer, destination image.Point) {
	round := func(v int) int { return (v*int(scale120) + 60) / 120 }
	return image.Pt(round(logical.X), round(logical.Y)), logical
}

// framePixels is how many pixels mygo draws for logical DIPs at scale, as
// package ui computes a frame's size (engine.pixels): in float32, rounded
// up.
func framePixels(logical int, scale float64) int {
	return int(math.Ceil(float64(float32(logical) * float32(scale))))
}

// preferredScale is the compositor's preferred scale for the subsurface,
// in 120ths: a new one resizes its buffers, and mygo's frames, which draw
// whole at the new scale (SurfaceResize).
func (u *subsurface) preferredScale(fraction ptr, scale120 uint32) {
	if fraction != u.fraction || scale120 == 0 || scale120 == u.scale120 {
		return
	}
	u.scale120 = scale120
	if !u.mapped || u.failed {
		return
	}
	u.place()
	u.s.send(platform.SurfaceEvent{Kind: platform.SurfaceResize})
}
