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
		// A bold arrow, its shaft a third of its height, its head half
		// its width.
		var path ui.Path
		mid, half := top+d/2, d/2
		path.MoveTo(at, mid-d/6).LineTo(at+half, mid-d/6).LineTo(at+half, top).LineTo(at+d, mid).
			LineTo(at+half, top+d).LineTo(at+half, mid+d/6).LineTo(at, mid+d/6).Close()
		p.FillPath(&path, m.Color)
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
