package ui

import "math"

// WarmUp has view built, laid out and painted once off screen as this
// frame ends, at the window's size and scale, in its appearance, with the
// text system the window draws with: the fonts view's text needs, its
// text shaped and its glyphs rasterized are then ready when the window
// first shows them, rather than stalling that frame. Nothing view draws
// shows, and what it does is dropped. Ask once, as when the window idles
// after it first shows.
func (c *Context) WarmUp(view func(c *Context)) {
	if b := c.build(); b != nil {
		b.rt.warmUps = append(b.rt.warmUps, view)
	}
}

// warmUp builds, lays out and paints view off screen (WarmUp).
func (rt *engine) warmUp(view func(c *Context)) {
	w, h, scale := rt.painted[0], rt.painted[1], rt.painted[2]
	if w <= 0 || h <= 0 || scale <= 0 {
		return
	}
	o := RecorderOptions{Width: int(math.Ceil(float64(w * scale))), Height: int(math.Ceil(float64(h * scale))), Scale: scale, Dark: rt.host.isDark()}
	r := newRecorderIn(func(c *context) { view(makeContext(c)) }, o, rt.text)
	// As the window's: text sizes may follow the desktop's settings.
	r.h.prefs, r.h.lang, r.h.rtl = rt.host.preferences(), rt.host.locale(), rt.host.rightToLeft()
	r.Frame(0)
	r.Close()
}
