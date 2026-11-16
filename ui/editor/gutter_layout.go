package editor

import (
	"math"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

// The app's look for the gutter, notes after lines and inlay hints'
// chips (State.Gutter, LineNotes, Theme.InlayHint).

// gutterLayout is State.Gutter, for an editor not showing a diff.
func (ed *editor) gutterLayout() *GutterLayout {
	if ed.o.decor != nil {
		return nil
	}
	return ed.o.gutter
}

// paintChangeBar marks line against the DiffBase, its rows y to y+h, at
// x (GutterLayout): the lines of a change one bar, inset a DIP at its
// ends, or for lines deleted above it a short bar across its top edge.
func (ed *editor) paintChangeBar(p *ui.Painter, x, y, h float32, line int) {
	g := ed.gitDiff
	if g == nil {
		return
	}
	ct := ed.theme
	st := g.status(line)
	if st == 3 {
		p.Fill(ui.Rect{X: x - 2, Y: y - 2, W: 8, H: 4}, ct.Deleted, 2)
		return
	}
	c := ct.Added
	switch st {
	case 0:
		return
	case 2:
		c = ct.Modified
	}
	// Rounded only where the change starts and ends.
	top, bottom := g.status(line-1) != st || line == 0, g.status(line+1) != st
	r := ui.Rect{X: x, Y: y, W: 3, H: h}
	if top {
		r.Y++
		r.H--
	}
	if bottom {
		r.H--
	}
	if top && bottom {
		p.Fill(r, c, 1.5)
		return
	}
	if top {
		p.Fill(ui.Rect{X: r.X, Y: r.Y, W: 3, H: 3}, c, 1.5)
		r.Y, r.H = r.Y+1.5, r.H-1.5
	}
	if bottom {
		p.Fill(ui.Rect{X: r.X, Y: r.Y + r.H - 3, W: 3, H: 3}, c, 1.5)
		r.H -= 1.5
	}
	p.Fill(r, c, 0)
}

// textHeight is the height of the editor's font's line, short of the
// line height's spacing.
func (v *view) textHeight() float32 {
	return float32(math.Ceil(float64(v.ascent + v.descent)))
}

// paintInlayChips fills a chip behind each inlay hint of a row
// (Theme.InlayHintBackground), its spaces left out; not behind
// suggestions or trailing inlays.
func (ed *editor) paintInlayChips(p *ui.Painter, l *lineLayout, row *rowLayout, ox, y float32) {
	c := ed.theme.InlayHintBackground
	if c.A == 0 || len(l.segs) == 0 {
		return
	}
	v := &ed.view
	h := min(l.rowHeight, v.textHeight()+2)
	for i, sg := range l.segs {
		if sg.kind != segInlay || sg.ghost {
			continue
		}
		a, b := sg.disp, sg.disp+segLen(l, i)
		for a < b && l.disp[a] == ' ' {
			a++
		}
		for b > a && l.disp[b-1] == ' ' {
			b--
		}
		if a == b || a < row.from || b > row.to {
			continue
		}
		x0, x1 := v.xOf(l, row, a), v.xOf(l, row, b)
		p.Fill(ui.Rect{X: ox + x0 - 2, Y: y + (l.rowHeight-h)/2, W: x1 - x0 + 4, H: h}, c, 3)
	}
}

// noteOf is line's note (State.LineNotes), if it has one, in its color
// or the theme's.
func (ed *editor) noteOf(line int) (LineNote, bool) {
	for _, n := range ed.o.lineNotes {
		if n.Line == line {
			if n.Color.A == 0 {
				n.Color = ed.theme.Note
			}
			if n.Color.A == 0 {
				n.Color = ed.theme.LineNumber
			}
			return n, true
		}
	}
	return LineNote{}, false
}

// noteGlyphs are a note's text shaped, kept while notes stay the same.
type noteGlyphs struct {
	font   ui.Font
	text   string
	glyphs []ui.Glyph
	ascent float32
}

// paintNote draws line's note after a row ending at x, at y, lh high:
// four spaces after the text, its first line only, cut at the editor's
// edge.
func (ed *editor) paintNote(p *ui.Painter, line int, x, y, lh float32) {
	n, ok := ed.noteOf(line)
	if !ok || n.Text == "" {
		return
	}
	v := &ed.view
	f := ed.o.noteFont
	if f.Family == "" {
		f = v.font
		f.Size--
	}
	f.Italic = true
	text := n.Text
	if i := strings.IndexByte(text, '\n'); i >= 0 {
		text = text[:i]
	}
	if !utf8.ValidString(text) {
		text = strings.ToValidUTF8(text, "�")
	}
	g := &ed.note
	if g.font != f || g.text != text {
		g.font, g.text = f, text
		g.glyphs = ui.Shape(text, f)
		g.ascent = f.Metrics().Ascent
	}
	x += v.charWidth * 4
	baseline := y + (lh-v.textHeight())/2 + 1 + g.ascent
	p.Glyphs(g.glyphs, x, baseline, n.Color)
}
