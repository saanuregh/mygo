package ui

import "testing"

func TestPixelScaleIsTheWindowsScaleZoomed(t *testing.T) {
	var got float32
	zoom := float32(1)
	tt := NewTester(func(c *Context) {
		c.SetZoom(zoom)
		got = c.PixelScale()
		Text(c, "x")
	}, 100, 100)
	tt.SetScale(2)
	tt.Frame()
	if got != 2 {
		t.Fatalf("PixelScale at scale 2: %v", got)
	}
	zoom = 1.5
	tt.Frame()
	tt.Frame()
	if got != 3 {
		t.Fatalf("PixelScale at scale 2 zoomed 1.5: %v", got)
	}
	var outside Context
	if s := outside.PixelScale(); s != 1 {
		t.Fatalf("PixelScale outside a build: %v", s)
	}
}
