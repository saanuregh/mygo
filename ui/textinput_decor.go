package ui

import (
	"sort"

	"github.com/egoist/mygo/internal/text"
)

// A TextWidget draws runes Start up to End of a text input's text as one
// box, Width wide and Height high, Descent of it below the baseline, which
// Paint draws: a mention as a chip, a task's box, a menu button by a code
// block. The text keeps the runes: the caret stops only at the edges of
// the box, Backspace and Delete take the runes out whole, and copying
// copies them.
//
// OnClick, when set, takes a click on the box, called on the main thread
// as the click comes, between frames, as OnTextKey's fn: the caret stays
// where it was, and the pointer shows a hand over the box. Without it, a
// click there puts the caret at the nearer edge, as on text.
type TextWidget struct {
	Start, End             int
	Width, Height, Descent float32
	// Paint draws the box at r, in the painter's coordinates, after the
	// text of its line.
	Paint   func(p *Painter, r Rect)
	OnClick func()
}

// A TextBlock decorates the paragraphs of a text area holding runes Start
// through End, End included, as a Markdown editor frames a code block or
// draws a quote's bar: the lines of a block of whole lines end at its last
// line's end, before the newline.
type TextBlock struct {
	Start, End int
	// Indent moves the paragraphs' text that far from the start edge, the
	// lines they wrap into too, which wrap at the width less it.
	Indent float32
	// Background fills the block's box, across the text area's content
	// and from the top of its first paragraph to the bottom of its last,
	// with corners rounded by Radius; Border strokes the box's edge.
	Background, Border Color
	Radius             float32
	// Paint draws over the background and under the text, at the block's
	// box r: a quote's bar, a rule across a line.
	Paint func(p *Painter, r Rect)
	// Marker draws in the indent before the block's first line, at r, as
	// high as the line: a list's bullet, number or task box, which stay
	// out of the text, as CSS's ::marker. OnMarkerClick, when set, takes a
	// click there as a TextWidget's OnClick does: a task's box ticked.
	Marker        func(p *Painter, r Rect)
	OnMarkerClick func()
}

// TextWidgets draws runs of a text input's text as boxes of the app's, in
// the frames that call it, over the TextRanges that meet them. Widgets
// should not overlap; where they do, the last one given draws the runes
// they share. They follow the text as TextRanges do. A password shows
// none.
func (e *node) TextWidgets(widgets ...TextWidget) *node {
	ed := e.st.editor
	if ed == nil || e.flags&flagEditable == 0 {
		return e
	}
	ed.freshDecorations()
	for _, w := range widgets {
		if w.End > w.Start {
			ed.widgets = append(ed.widgets, w)
			ed.apart = false
		}
	}
	return e
}

// TextBlocks decorates paragraphs of a text area, in the frames that call
// it, behind their text. Where blocks given meet the same paragraph, the
// last one setting an Indent sets its indent; each paints its box. A
// single-line input shows none.
func (e *node) TextBlocks(blocks ...TextBlock) *node {
	ed := e.st.editor
	if ed == nil || e.flags&flagEditable == 0 || ed.area == nil {
		return e
	}
	ed.freshDecorations()
	for _, b := range blocks {
		if b.End >= b.Start {
			ed.blocks = append(ed.blocks, b)
		}
	}
	return e
}

// staleDecorations marks the ranges, widgets and blocks the last frame
// gave as the build's to replace: the input the build applies as the
// input's constructor returns, before the app gives this frame's, moves
// the caret through what the user saw.
func (ed *editor) staleDecorations() { ed.stale = true }

// freshDecorations drops the last frame's ranges, widgets and blocks
// once, as the first of this frame's comes, or as the input is laid out
// without any.
func (ed *editor) freshDecorations() {
	if !ed.stale {
		return
	}
	ed.stale = false
	ed.ranges, ed.widgets, ed.blocks = ed.ranges[:0], ed.widgets[:0], ed.blocks[:0]
	ed.apart = false
}

// atomAround returns the run that rune offset i is inside of, past its
// start and before its end, that the caret does not stop in: a widget's
// runes, or runes hidden now, those hidden next to each other as one.
func (ed *editor) atomAround(i int) (start, end int, ok bool) {
	if len(ed.ranges) == 0 || ed.password {
		return 0, 0, false
	}
	if !ed.apart {
		ed.sortRanges()
	}
	rs := ed.ranges
	// atom is what makes run k one: a widget's number, -1 hidden, 0 none.
	atom := func(k int) int {
		if r := &rs[k]; r.widget != 0 {
			return r.widget
		} else if ed.hides(r) {
			return -1
		}
		return 0
	}
	// The run holding i, or the one ending at i when one starts there.
	k := sort.Search(len(rs), func(k int) bool { return rs[k].End > i })
	switch {
	case k < len(rs) && rs[k].Start < i:
	case k < len(rs) && rs[k].Start == i && k > 0 && rs[k-1].End == i && atom(k-1) != 0 && atom(k-1) == atom(k):
	default:
		return 0, 0, false
	}
	a := atom(k)
	if a == 0 {
		return 0, 0, false
	}
	lo, hi := k, k
	for lo > 0 && rs[lo-1].End == rs[lo].Start && atom(lo-1) == a {
		lo--
	}
	for hi+1 < len(rs) && rs[hi+1].Start == rs[hi].End && atom(hi+1) == a {
		hi++
	}
	return rs[lo].Start, rs[hi].End, true
}

// outsideAtom returns i, or the edge of the atom it is inside of
// (atomAround): its start going back (dir < 0), else its end, where the
// offsets inside it are laid out, as its runes past the first take no
// room: a click past the middle of a widget's box hits one of them.
func (ed *editor) outsideAtom(i, dir int) int {
	s, e, ok := ed.atomAround(i)
	switch {
	case !ok:
		return i
	case dir < 0:
		return s
	}
	return e
}

// selectAtomsWhole widens the ranges of a visual selection to the atoms
// they end in, and drops those left empty, as a selection from an edge of
// an atom to the same edge, which the layout may take for the runes past
// its first, laid out where it ends.
func (ed *editor) selectAtomsWhole() {
	if ed.anchor == ed.caret {
		ed.visualSelection, ed.selected = false, nil
		return
	}
	out := ed.selected[:0]
	for _, r := range ed.selected {
		r.Start, r.End = ed.outsideAtom(r.Start, -1), ed.outsideAtom(r.End, 1)
		if r.End > r.Start {
			out = append(out, r)
		}
	}
	ed.selected = out
}

// stepOverAtoms is visualNext through text with atoms (atomAround): an
// atom at the caret is crossed in one step, and a step stops at the first
// edge of one it reaches, which the layout does not tell from the runes
// beside it, as the atom's runes past the first take no room.
func (ed *editor) stepOverAtoms(direction int) text.CaretPosition {
	if len(ed.ranges) == 0 || ed.password {
		return ed.visualNext(direction)
	}
	forward := direction > 0
	if ed.caretParagraphRTL() {
		forward = !forward
	}
	from := ed.caret
	if forward {
		if s, e, ok := ed.atomAround(from + 1); ok && s == from {
			return text.CaretPosition{Index: e, Affinity: text.Upstream}
		}
	} else if s, e, ok := ed.atomAround(from - 1); ok && e == from {
		return text.CaretPosition{Index: s}
	}
	p := ed.visualNext(direction)
	if p.Index == from {
		return p
	}
	// The first edge of an atom from the caret on, the way the step went.
	edge, found := 0, false
	for i := range ed.ranges {
		r := &ed.ranges[i]
		if r.widget == 0 && !ed.hides(r) {
			continue
		}
		for _, at := range [2]int{r.Start, r.End} {
			switch {
			case p.Index > from && at > from && at < p.Index && (!found || at < edge),
				p.Index < from && at < from && at > p.Index && (!found || at > edge):
				edge, found = at, true
			}
		}
	}
	if found {
		p = text.CaretPosition{Index: edge}
	}
	return p
}

// widgetBox returns the box of widget w relative to the element: from
// where its first rune is laid out, as wide as it is and as high as its
// line.
func (ed *editor) widgetBox(w *TextWidget) (Rect, bool) {
	if w.Start > ed.buf.n {
		return Rect{}, false
	}
	if a := ed.area; a != nil && a.version != 0 {
		x, y, h := a.caretAt(ed, w.Start, 0)
		return Rect{ed.originX + x, ed.originY + float32(y-a.scroll), w.Width, h}, true
	}
	if ed.layout == nil {
		return Rect{}, false
	}
	x, y, h := ed.layout.CaretAt(text.CaretPosition{Index: ed.displayIndex(w.Start)})
	return Rect{ed.originX + x - ed.scrollX, ed.originY + y, w.Width, h}, true
}

// markerBox returns the box of block b's marker relative to the element:
// the indent before its first line.
func (ed *editor) markerBox(b *TextBlock) (Rect, bool) {
	a := ed.area
	if a == nil || a.version == 0 || b.Start > ed.buf.n {
		return Rect{}, false
	}
	p := ed.buf.para(b.Start)
	indent := ed.paraIndent(p)
	l := a.paraLayout(ed, p)
	if indent <= 0 || len(l.Lines) == 0 {
		return Rect{}, false
	}
	x := ed.originX
	if a.rtl {
		x += ed.contentW - indent
	}
	return Rect{x, ed.originY + float32(a.hs.top(p)-a.scroll), indent, l.Lines[0].Height}, true
}

// clickableAt returns what takes a click at (x, y), relative to the
// element: widget i with an OnClick as i, the marker of block j with an
// OnMarkerClick as len(widgets)+j, else -1.
func (ed *editor) clickableAt(x, y float32) int {
	if ed.password || !ed.laidOut() {
		return -1
	}
	in := func(r Rect) bool { return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H }
	for i := len(ed.widgets) - 1; i >= 0; i-- {
		w := &ed.widgets[i]
		if w.OnClick == nil {
			continue
		}
		if r, ok := ed.widgetBox(w); ok && in(r) {
			return i
		}
	}
	for j := len(ed.blocks) - 1; j >= 0; j-- {
		b := &ed.blocks[j]
		if b.OnMarkerClick == nil {
			continue
		}
		if r, ok := ed.markerBox(b); ok && in(r) {
			return len(ed.widgets) + j
		}
	}
	return -1
}

// click calls the OnClick of what clickableAt returned as k.
func (ed *editor) click(k int) {
	switch {
	case k < len(ed.widgets):
		ed.widgets[k].OnClick()
	default:
		ed.blocks[k-len(ed.widgets)].OnMarkerClick()
	}
}

// paintWidgets paints the widgets whose first rune is laid out in l,
// laid out from (x, y); global maps an index of l to the rune of the text.
func (ed *editor) paintWidgets(p *Painter, l *text.Layout, x, y float32, global func(int) int) {
	if len(ed.widgets) == 0 || l == nil {
		return
	}
	for li := range l.Lines {
		line := &l.Lines[li]
		for _, g := range line.Glyphs {
			if !g.Object || g.Advance <= hiddenWidth {
				continue
			}
			i := global(g.Cluster)
			k := sort.Search(len(ed.ranges), func(k int) bool { return ed.ranges[k].End > i })
			if k == len(ed.ranges) || ed.ranges[k].widget == 0 {
				continue
			}
			w := &ed.widgets[ed.ranges[k].widget-1]
			if w.Start != i || w.Paint == nil {
				continue
			}
			r := Rect{x + g.X, y + line.Baseline - (w.Height - w.Descent), g.Advance, w.Height}
			if p.visible(r, 0) {
				w.Paint(p, r)
			}
		}
	}
}

// paraIndent returns the indent of paragraph p (TextBlock.Indent).
func (ed *editor) paraIndent(p int) float32 {
	if len(ed.blocks) == 0 {
		return 0
	}
	start, end := ed.buf.start(p), ed.buf.end(p)
	var indent float32
	for i := range ed.blocks {
		b := &ed.blocks[i]
		if b.Indent != 0 && b.Start <= end && b.End >= start {
			indent = max(b.Indent, 0)
		}
	}
	return indent
}

// paintBlocks paints the blocks of a text area meeting the paragraphs in
// view, the content's left at ox and its top at oy.
func (a *area) paintBlocks(ed *editor, p *Painter, ox, oy float32) {
	b := &ed.buf
	for i := range ed.blocks {
		blk := &ed.blocks[i]
		first, last := b.para(min(blk.Start, b.n)), b.para(min(blk.End, b.n))
		if last < a.first || first > a.last {
			continue
		}
		top := oy + float32(a.hs.top(first)-a.scroll)
		bottom := oy + float32(a.hs.top(last+1)-a.scroll)
		r := Rect{ox, top, ed.contentW, bottom - top}
		if blk.Background.A > 0 {
			p.Fill(r, blk.Background, blk.Radius)
		}
		if blk.Border.A > 0 {
			p.Stroke(r, blk.Border, blk.Radius, 1)
		}
		if blk.Paint != nil {
			blk.Paint(p, r)
		}
		if blk.Marker != nil && first >= a.first {
			if m, ok := ed.markerBox(blk); ok {
				m.X, m.Y = m.X-ed.originX+ox, m.Y-ed.originY+oy // the element's to the painter's
				blk.Marker(p, m)
			}
		}
	}
}
