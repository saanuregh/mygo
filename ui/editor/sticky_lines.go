package editor

import (
	"strconv"

	"github.com/egoist/mygo/ui"
)

// maxStickyLines is how many of State.StickyLines are pinned.
const maxStickyLines = 5

// stickyCount returns how many sticky lines are pinned: no more than
// fit, leaving a row of text below them.
func (ed *editor) stickyCount() int {
	n := min(len(ed.o.stickyLines), maxStickyLines)
	if lh := ed.view.lineHeight; lh > 0 {
		n = min(n, max(0, int(ed.r.H/lh)-1))
	}
	return n
}

// stickyLineAt returns the sticky line pinned at y, from the top of the
// text's area.
func (ed *editor) stickyLineAt(y float32) (int, bool) {
	lh := ed.view.lineHeight
	if y < 0 || lh <= 0 {
		return 0, false
	}
	i := int(y / lh)
	if i >= ed.stickyCount() {
		return 0, false
	}
	line := ed.o.stickyLines[i]
	if line < 0 || line >= ed.view.text.LineCount() {
		return 0, false
	}
	return line, true
}

// paintStickyLines paints the sticky lines over the top rows of the
// text and gutter: each line's first row and its number.
func (ed *editor) paintStickyLines(p *ui.Painter, r ui.Rect, f *frame) {
	n := ed.stickyCount()
	if n == 0 {
		return
	}
	v := &ed.view
	ct := ed.theme
	lh := v.lineHeight
	w := r.W
	if vertical, _ := ed.scrollbars(); vertical {
		w -= ed.barWidth()
	}
	// Opaque, so the text scrolled under them does not show through.
	if bg := ct.Background; bg.A > 0 {
		bg.A = 255
		p.Fill(ui.Rect{X: r.X, Y: r.Y, W: w, H: float32(n) * lh}, bg, 0)
	}
	text := ui.Rect{X: r.X + ed.gutterW, Y: r.Y, W: w - ed.gutterW, H: float32(n) * lh}
	numRight := r.X + ed.gutterW - 4 - v.charWidth*0.5
	if ed.o.folding && ed.o.decor == nil {
		numRight -= lh * 0.8
	}
	if ed.gitDiff != nil {
		numRight -= 6
	}
	if gl := ed.gutterLayout(); gl != nil {
		numRight = r.X + ed.gutterW - gl.Right
	}
	var cc runColors
	for i := 0; i < n; i++ {
		line := ed.o.stickyLines[i]
		if line < 0 || line >= v.text.LineCount() {
			continue
		}
		y := r.Y + float32(i)*lh
		l := v.line(line)
		if len(l.rows) > 0 {
			v.colorsOf(l, &f.colors)
			p.Clip(text, 0, func() {
				v.paintRow(p, l, &l.rows[0], r.X+ed.textX-ed.scrollX, y, &f.colors, nil)
			})
		}
		if ed.gutterW > 0 && ed.numbersOn() {
			s := strconv.Itoa(line + 1)
			lay := v.shaped.get(s, v.style)
			row := rowLayout{pieces: []rowPiece{{to: len(s), l: lay, ascii: true, width: lay.Width}}, to: len(s)}
			col := ct.LineNumber
			v.paintRow(p, &lineLayout{disp: s}, &row, numRight-lay.Width, y, &cc, func(int, ui.Color) ui.Color { return col })
		}
	}
	p.Fill(ui.Rect{X: r.X, Y: r.Y + float32(n)*lh - 1, W: w, H: 1}, ct.LineNumber, 0)
}

// pressStickyLine scrolls the sticky line pressed to the top and puts
// the caret at its start.
func (ed *editor) pressStickyLine(ev ui.InputEvent) bool {
	line, ok := ed.stickyLineAt(ev.Y)
	if !ok {
		return false
	}
	off := ed.view.text.LineStart(line)
	ed.setSels([]sel{ed.newSel(off, off)}, revealNone)
	ed.setScroll(ed.view.heights.top(line))
	ed.services.Invalidate()
	return true
}
