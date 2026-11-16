package editor

import "github.com/egoist/mygo/ui"

// GutterShape is how a GutterMark is drawn.
type GutterShape uint8

const (
	// GutterDot is a dot, as a breakpoint.
	GutterDot GutterShape = iota
	// GutterSquare is a square with rounded corners, as a logpoint.
	GutterSquare
	// GutterArrow is an arrow pointing at the line, as where a debugged
	// program is stopped.
	GutterArrow
)

// paintGutterMark draws m on the row at y, lh high: at the gutter's left
// (x), or Beside the number starting at numLeft.
func (ed *editor) paintGutterMark(p *ui.Painter, m GutterMark, x, numLeft, y, lh float32) {
	d := m.Size
	if d <= 0 {
		d = lh * 0.45
	}
	at := x + 3
	if m.Beside {
		at = numLeft - 4 - d
	}
	top := y + (lh-d)/2
	switch m.Shape {
	case GutterSquare:
		p.Fill(ui.Rect{X: at, Y: top, W: d, H: d}, m.Color, 2)
	case GutterArrow:
		// A line arrow in bold strokes with round ends, as SF Symbols'
		// arrow.right at a bold weight (zode's editor/view.zig), 1 point
		// further right than a box of its size.
		var path ui.Path
		at++
		mid, w := top+d/2, d*0.13
		path.MoveTo(at+d*0.1, mid).LineTo(at+d*0.9, mid)
		path.MoveTo(at+d*0.52, top+d*0.16).LineTo(at+d*0.9, mid).LineTo(at+d*0.52, top+d*0.84)
		p.StrokePath(&path, w, m.Color)
	default:
		p.Fill(ui.Rect{X: at, Y: top, W: d, H: d}, m.Color, d/2)
	}
}

// hoverGutterLine is the line under the pointer, as last moved over the
// editor.
func (ed *editor) hoverGutterLine() int {
	_, cy := ed.toContent(ed.pointer.X, ed.pointer.Y)
	return ed.view.heights.lineAt(cy)
}
