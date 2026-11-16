package ui

import (
	"testing"
	"time"
)

// TestToastsShownAllAndHeldEach: with SetToastLimit(0) every toast shows,
// and with SetToastsHeldEach the pointer on one stops that one's time only.
func TestToastsShownAllAndHeldEach(t *testing.T) {
	SetToastLimit(0)
	SetToastsHeldEach(true)
	defer SetToastLimit(3)
	defer SetToastsHeldEach(false)
	var shown []Toast
	var c *context
	tt, now := clockTester(func(ctx *context) {
		c = ctx
		coreToastViewportBase(c, func(viewport *node, toasts []Toast) {
			shown = append(shown[:0], toasts...)
			viewport.Gap(4)
			for _, t := range toasts {
				coreToastBase(c, t).Root.Height(20).Children(func() { coreText(c, t.Title) })
			}
		})
	}, 400, 300)
	tick := ticker(tt, now)
	for _, id := range []string{"a", "b", "c", "d", "e"} {
		c.AddToast(Toast{ID: id, Title: "toast " + id})
	}
	tt.Frame()
	if len(shown) != 5 {
		t.Fatalf("%d toasts show, not 5", len(shown))
	}
	r, _ := tt.Find("toast e")
	tt.Move(r.X+1, r.Y+1)
	tick(time.Second)
	tick(5 * time.Second)
	if len(shown) != 1 || shown[0].ID != "e" {
		t.Fatalf("after 6 s with the pointer on e: %v", shown)
	}
}
