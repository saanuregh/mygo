//go:build linux && !android && (amd64 || arm64)

package linux

// framePacer paces a subsurface's frames by its frame callbacks. A frame
// asked for (request) while none is due draws at once, from soon; one
// asked for while a callback is due, as many as are asked meanwhile, draws
// once it comes (done). A frame that showed nothing asks for no callback:
// the next waits about a refresh (later), rather than spin.
type framePacer struct {
	// soon has run called at once from the main loop, later after about a
	// refresh; draw draws a frame and reports whether it showed one, with
	// a frame callback asked for.
	soon, later func()
	draw        func() bool
	// wanted tells that a frame is asked for, queued that run is, drawing
	// that draw runs, and waiting that a frame callback is due.
	wanted, queued, drawing, waiting bool
}

// request asks for a frame.
func (p *framePacer) request() {
	p.wanted = true
	if !p.queued && !p.drawing && !p.waiting {
		p.queued = true
		p.soon()
	}
}

// run draws the frame asked for, unless a frame callback is due.
func (p *framePacer) run() {
	p.queued = false
	if p.waiting || !p.wanted {
		return
	}
	p.now()
}

// now draws a frame at once, even with a frame callback due, as one in
// step with the toplevel does: the callback of the frame it shows paces
// the next.
func (p *framePacer) now() {
	p.wanted, p.drawing = false, true
	shown := p.draw()
	p.drawing = false
	switch {
	case shown:
		p.waiting = true
	case p.wanted && !p.queued && !p.waiting:
		// The frame asked for another as it drew.
		p.queued = true
		p.later()
	}
}

// done tells that the frame callback came: a frame asked for meanwhile
// draws now.
func (p *framePacer) done() {
	p.waiting = false
	if p.wanted && !p.queued {
		p.queued = true
		p.soon()
	}
}

// stop tells that no frame callback is coming, as the subsurface went: the
// next frame asked for draws at once.
func (p *framePacer) stop() { p.waiting = false }
