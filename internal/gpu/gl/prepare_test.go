//go:build linux && !android && (amd64 || arm64)

package gl

import (
	"bytes"
	"image"
	"testing"

	"github.com/egoist/mygo/internal/gpu"
	"github.com/egoist/mygo/internal/gpu/gputest"
)

// TestPreparedFrameDrawsWhatRenderDrew draws changing scenes two ways,
// with Render and with Prepare then DrawPrepared, as a GtkGLArea's
// window does when it learns a frame's damage before GTK paints: the
// bytes must be the same, and the damage Prepare tells must hold every
// pixel the frame redraws, the whole frame when it is drawn whole, and
// none when nothing changed.
func TestPreparedFrameDrawsWhatRenderDrew(t *testing.T) {
	for _, api := range apis {
		t.Run(api.name, func(t *testing.T) {
			read := offscreen(t, api.es, 320, 240)
			var outer int32
			glGetIntegerv(glFramebufferBinding, &outer)
			// The prepared frames go into a framebuffer of their own, which
			// keeps what they drew as a GtkGLArea's does.
			tex := newTexture(glRGBA8, glRGBA, 320, 240, nil, 0)
			var fb uint32
			glGenFramebuffers(1, &fb)
			glBindFramebuffer(glFramebuffer, fb)
			glFramebufferTexture2D(glFramebuffer, glColorAttachment0, glTexture2D, tex, 0)
			defer func() {
				glDeleteFramebuffers(1, &fb)
				glDeleteTextures(1, &tex)
			}()
			seen := map[redrawn]int{}
			for seed := range uint64(4) {
				m := gputest.NewChanging(seed)
				s := m.Scene()
				rendered, err := New()
				if err != nil {
					t.Fatal(err)
				}
				rendered.Partial = true
				prepared, err := New()
				if err != nil {
					t.Fatal(err)
				}
				prepared.Partial = true
				for step := range 20 {
					what := "first"
					switch {
					case step == 7:
						what = "unchanged" // nothing to redraw
					case step > 0:
						what = m.Change(s)
					}
					glBindFramebuffer(glFramebuffer, uint32(outer))
					if err := rendered.Render(s); err != nil {
						t.Fatal(err)
					}
					want := read()
					// Prepared with no framebuffer of its own bound, as in a
					// frame clock's tick, before GTK binds the area's.
					glBindFramebuffer(glFramebuffer, uint32(outer))
					damage, whole, err := prepared.Prepare(s)
					if err != nil {
						t.Fatal(err)
					}
					glBindFramebuffer(glFramebuffer, fb)
					if err := prepared.DrawPrepared(); err != nil {
						t.Fatal(err)
					}
					got := read()
					if !bytes.Equal(got, want) {
						t.Fatalf("seed %d, step %d (%s): prepared frame differs from Render's", seed, step, what)
					}
					seen[prepared.redrew]++
					switch prepared.redrew {
					case redrewWhole:
						if !whole {
							t.Errorf("seed %d, step %d (%s): drawn whole, damage %v not whole", seed, step, what, damage)
						}
					case redrewNothing:
						if whole || len(damage) != 0 {
							t.Errorf("seed %d, step %d (%s): nothing redrawn, damage %v whole %v", seed, step, what, damage, whole)
						}
					case redrewPart:
						if whole || !sameRects(damage, prepared.part.Rects) {
							t.Errorf("seed %d, step %d (%s): redrew %s, damage %v whole %v", seed, step, what, prepared.part.String(), damage, whole)
						}
					}
				}
				rendered.Release()
				prepared.Release()
			}
			if seen[redrewWhole] == 0 || seen[redrewPart] == 0 || seen[redrewNothing] == 0 {
				t.Errorf("frames drawn whole %d, in part %d, unchanged %d: not a test of each", seen[redrewWhole], seen[redrewPart], seen[redrewNothing])
			}
		})
	}
}

func sameRects(damage []image.Rectangle, part []gpu.Scissor) bool {
	if len(damage) != len(part) {
		return false
	}
	for i, d := range damage {
		p := part[i]
		if d != image.Rect(int(p.Left), int(p.Top), int(p.Right), int(p.Bottom)) {
			return false
		}
	}
	return true
}
