package editor

import "github.com/egoist/mygo/ui"

// The pointer over inlay hints: which one it is on (State.OnInlayHover)
// and a press on one (State.OnInlayClick), as Zed's hints show their
// tooltips and go where their labels point. Trailing inlays
// (Inlay.Trailing) are not hints here: the text's hover covers them.

// InlayHover tells which inlay hint the pointer is over, as
// State.OnInlayHover is told.
type InlayHover struct {
	// Index is the hint's in the list last given to SetInlays, -1 once
	// the pointer left the hints.
	Index int
	// Box is where the hint shows, in the editor's DIPs from its top
	// left corner.
	Box ui.Rect
}

// InlayClick is a press of the primary button on an inlay hint, as
// State.OnInlayClick is told.
type InlayClick struct {
	// Index is the hint's in the list last given to SetInlays.
	Index int
	// Mods are the modifiers held, as ui.Cmd for Cmd+click (Ctrl+click
	// on Linux and Windows); Clicks counts the presses in a row.
	Mods   ui.Modifiers
	Clicks int
}

// inlayAt returns the hint under x, y of the content, not a suggestion,
// a trailing inlay or a diff's mark: its index in the app's list and its
// box in the content.
func (ed *editor) inlayAt(x float32, y float64) (int, ui.Rect, bool) {
	v := &ed.view
	if len(v.inlays) == 0 || v.text.LineCount() == 0 {
		return -1, ui.Rect{}, false
	}
	line := v.heights.lineAt(y)
	g := v.heights.geometry(line)
	if g.hidden || y < g.rows || y >= g.below {
		return -1, ui.Rect{}, false
	}
	l := v.line(line)
	if l.segs == nil || len(l.rows) == 0 {
		return -1, ui.Rect{}, false
	}
	ri := min(int((y-g.rows)/float64(l.rowHeight)), len(l.rows)-1)
	r := &l.rows[ri]
	inl := v.inlaysIn(l.start, l.start+len(l.text))
	for i, sg := range l.segs {
		if sg.kind != segInlay || sg.inlay >= len(inl) || inl[sg.inlay].kind != 0 || inl[sg.inlay].index < 0 {
			continue
		}
		a, b := sg.disp, sg.disp+segLen(l, i)
		if b <= r.from || a >= r.to {
			continue
		}
		// Its label's glyphs, not the room around it, and 2 past each
		// side (zode's view.zig inlaysOnRow).
		a, b = max(a, r.from), min(b, r.to)
		for a < b && l.disp[a] == ' ' {
			a++
		}
		for b > a && l.disp[b-1] == ' ' {
			b--
		}
		if a == b {
			continue
		}
		x0, x1 := v.xOf(l, r, a)-2, v.xOf(l, r, b)+2
		if x >= x0 && x < x1 {
			top := g.rows + float64(ri)*float64(l.rowHeight)
			return inl[sg.inlay].index, ui.Rect{X: x0, Y: float32(top), W: x1 - x0, H: l.rowHeight}, true
		}
	}
	return -1, ui.Rect{}, false
}

// pointerOnInlay follows the pointer at x, y of the editor over the hints,
// and tells the app when the hint under it changes; whether it is on one,
// and whether that changed.
func (ed *editor) pointerOnInlay(x, y float32, overText bool) (on, changed bool) {
	if ed.o.onInlayHover == nil && ed.o.onInlayClick == nil {
		return false, false
	}
	index, box := -1, ui.Rect{}
	if overText {
		cx, cy := ed.toContent(x, y)
		if i, r, ok := ed.inlayAt(cx, cy); ok {
			index = i
			box = ui.Rect{X: ed.textX + r.X - ed.scrollX, Y: ed.top + float32(float64(r.Y)-ed.scrollY), W: r.W, H: r.H}
		}
	}
	changed = index != ed.inlayHovered
	ed.tellInlayHover(index, box)
	return index >= 0, changed
}

// tellInlayHover tells the app that the pointer is on hint index (-1:
// none), once per change.
func (ed *editor) tellInlayHover(index int, box ui.Rect) {
	if index == ed.inlayHovered {
		return
	}
	ed.inlayHovered = index
	if ed.o.onInlayHover != nil {
		ed.o.onInlayHover(InlayHover{Index: index, Box: box})
	}
}

// pressOnInlay offers a press at content x, y to the app if it is on a
// hint; whether the app took it.
func (ed *editor) pressOnInlay(x float32, y float64, ev ui.InputEvent) bool {
	if ed.o.onInlayClick == nil {
		return false
	}
	i, _, ok := ed.inlayAt(x, y)
	return ok && ed.o.onInlayClick(InlayClick{Index: i, Mods: ev.Mods, Clicks: ev.Clicks})
}
