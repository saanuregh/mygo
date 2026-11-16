package ui

import (
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/internal/scene"
	"github.com/egoist/mygo/internal/text"
)

func (ed *editor) lineHeight() float32 {
	if ed.area != nil && ed.area.line.Height > 0 {
		return ed.area.line.Height
	}
	if ed.layout != nil && len(ed.layout.Lines) > 0 {
		return ed.layout.Lines[0].Height
	}
	return 18
}

// caretRect returns the caret's box relative to the surface, where input
// methods show their candidates.
func (ed *editor) caretRect(st *state) Rect {
	if a := ed.area; a != nil && a.version != 0 {
		x, y, h := a.caretAt(ed, ed.caret, ed.composeCaret)
		return Rect{st.x + ed.originX + x, st.y + ed.originY + float32(y-a.scroll), 1, h}
	}
	if ed.layout == nil {
		return Rect{st.x, st.y, 1, st.h}
	}
	x, y, h := ed.layout.CaretAt(text.CaretPosition{Index: ed.displayIndex(ed.caret) + ed.composeCaret, Affinity: ed.caretAffinity})
	return Rect{st.x + ed.originX + x - ed.scrollX, st.y + ed.originY + y, 1, h}
}

// displayText returns what the input shows: the text with the
// composition at the caret, bullets for a password.
func (ed *editor) displayText() string {
	t := ed.buf.string()
	if ed.compose != "" {
		at := runeOffset(t, ed.caret)
		t = t[:at] + ed.compose + t[at:]
	}
	if ed.password {
		return strings.Repeat("•", utf8.RuneCountInString(t))
	}
	return t
}

func (e *node) inputParams(width float32) text.Params {
	p := e.textParams(width)
	ed := e.st.editor
	p.Text = ed.displayText()
	if !ed.password {
		p.Spans = ed.rangeKey(0, ed.buf.n)
	}
	p.KeepSpaces = true
	p.MaxLines = 0
	if !e.st.editor.multiline {
		p.Width = 0
	}
	return p
}

// Each control owns its current layout. Caching changing input strings in
// the system's display-text cache would retain old documents across edits.
func (ed *editor) shapeInput(params text.Params) *text.Layout {
	gen := ed.sys.Generation()
	if ed.layout == nil || ed.layout.Params != params || ed.layoutGeneration != gen {
		ed.layout = ed.sys.Shape(params)
		ed.layoutGeneration = gen
		ed.layoutVersion, ed.layoutCompose, ed.layoutPassword = ed.buf.version, ed.compose, ed.password
	}
	return ed.layout
}

func (e *node) inputHeight(cw float32) float32 {
	ed := e.st.editor
	ed.freshDecorations()
	if ed.area != nil {
		p := e.textParams(0)
		p.Text = ""
		line := ed.sys.Layout(p).Lines[0].Height
		if lo, hi := ed.lines[0], ed.lines[1]; hi > 0 {
			// As high as its text wrapped at the width it gets, between its
			// least and most lines.
			h := ed.area.wrappedHeight(e, ed, cw)
			return min(max(h, float32(lo)*line), float32(hi)*line)
		}
		// As high as its paragraphs unwrapped, without laying them out.
		return float32(max(len(ed.buf.paras), 3)) * line
	}
	l := ed.shapeInput(e.inputParams(0))
	return l.Lines[0].Height
}

func (e *node) layoutInput(cw, ch float32) {
	ed := e.st.editor
	ed.freshDecorations()
	if a := ed.area; a != nil {
		ed.contentW = cw
		ed.originX, ed.originY = e.contentX(), e.contentY()
		a.layout(e, cw, ch)
		return
	}
	l := ed.shapeInput(e.inputParams(cw))
	ed.layout = l
	ed.layoutVersion, ed.layoutCompose, ed.layoutPassword = ed.buf.version, ed.compose, ed.password
	ed.contentW = cw
	ed.originX, ed.originY = e.contentX(), e.contentY()
	// A single line, centered vertically in a taller box: text areas lay
	// out in area.
	if len(l.Lines) > 0 {
		ed.originY += max((ch-l.Lines[0].Height)/2, 0)
	}
	// Text narrower than the box goes where TextAlign puts it: Start is
	// the right of an input laid out right to left, whatever its text, as
	// that of Text. Its last column stays for the caret, as where the text
	// scrolls: a caret on the box's right edge would be clipped away.
	rtl := e.rtl()
	if room := cw - l.Width - 1; room > 0 {
		switch align := e.resolvedText().align; {
		case align == Center:
			ed.originX += room / 2
		case (align == End) != rtl:
			ed.originX += room
		}
	}
	// Keep the caret in view while the input has the focus. Without it, the
	// input shows the start of its text, as fields do on macOS and the web:
	// its right right to left.
	if e.c.rt.focused != e.id {
		ed.scrollX = 0
		if rtl {
			ed.scrollX = max(l.Width-cw+1, 0)
		}
	} else if x, _, _ := l.Caret(ed.displayIndex(ed.caret) + ed.composeCaret); x-ed.scrollX < 0 {
		ed.scrollX = x
	} else if x-ed.scrollX > cw-1 {
		ed.scrollX = x - cw + 1
	}
	ed.scrollX = max(0, min(ed.scrollX, max(l.Width-cw+1, 0)))
}

// placeholderParams lays out the placeholder of an empty input in a
// content box width wide: in the input's style, its line height too, fixed
// or not. A text area's placeholder wraps; a single-line input's stays on
// its line, cut off at the box.
func (e *node) placeholderParams(width float32) text.Params {
	ed := e.st.editor
	params := e.textParams(width)
	params.Text, params.Spans, params.MaxLines, params.NoWrap, params.Ellipsis = ed.placeholder, "", 0, !ed.multiline || ed.placeholderOneLine, ""
	return params
}

func (e *node) paintInput(p *Painter) {
	ed := e.st.editor
	l := ed.layout
	t := p.Theme()
	if l == nil && ed.area == nil {
		return
	}
	box := e.contentBox()
	clip := Rect{e.x + e.border[3], e.y + e.border[0], e.w - e.border[1] - e.border[3], e.h - e.border[0] - e.border[2]}
	saved := p.clip
	p.pushClip(clip, [4]float32{})
	ox, oy := e.x+ed.originX-ed.scrollX, e.y+ed.originY
	focused := e.Focused()
	ts := e.resolvedText()
	if ed.buf.n == 0 && ed.compose == "" && ed.placeholder != "" {
		pl := ed.sys.Layout(e.placeholderParams(box.W))
		color := t.TextMuted
		if ed.placeholderColorSet {
			color = ed.placeholderColor
		}
		// The placeholder aligns itself in the content box, as its layout has the box's width.
		p.textLayout(pl, e.x+e.contentX(), oy, color, ts, nil)
	}
	if ed.area != nil {
		ed.area.paint(e, p, e.x+ed.originX, e.y+ed.originY)
		p.popClip()
		p.clip = saved
		return
	}
	var sp *spanPaint
	if !ed.password {
		_, sp = ed.rangeSpans(0, ed.buf.n)
	}
	p.textBackgrounds(l, ox, oy, ts, sp)
	if focused {
		for _, selected := range ed.selectedRanges() {
			for _, r := range l.SelectionVisual(ed.displayIndex(selected.Start), ed.displayIndex(selected.End), false) {
				p.Fill(Rect{ox + r.X, oy + r.Y, r.W, r.H}, ts.selectionColor(t), t.InputSelection.Radius)
			}
		}
	}
	p.textForeground(l, ox, oy, ts.color, ts, sp)
	if !ed.password {
		ed.paintWidgets(p, l, ox, oy, ed.textIndex)
	}
	if ed.compose != "" {
		start := ed.caret
		end := start + utf8.RuneCountInString(ed.compose)
		for _, r := range l.Selection(start, end) {
			p.Fill(Rect{ox + r.X, oy + r.Y + r.H - 2, r.W, 1}, ts.color, 0)
		}
	}
	if focused && !ed.readOnly && !(t.InputSelection.NoCaret && ed.caret != ed.anchor) {
		rt := e.c.rt
		phase := e.c.now.Sub(rt.blinkStart)
		blink := t.CaretBlink
		if blink <= 0 {
			blink = 530 * time.Millisecond
		}
		if (phase/blink)%2 == 0 && ed.caretWidth > 0 {
			x, y, h := l.CaretAt(text.CaretPosition{Index: ed.displayIndex(ed.caret) + ed.composeCaret, Affinity: ed.caretAffinity})
			caret := ts.caretColor(t)
			p.add(scene.Op{Kind: scene.OpFill, Rect: p.snap(Rect{ox + x, oy + y, 0, h}), Color: caret.scene(), Wide: p.wide(caret, Color{}, Color{}), Opacity: p.opacity})
			op := &p.s.Ops[len(p.s.Ops)-1]
			op.Rect.W = max(round(ed.caretWidth*p.scale), 1)
		}
		if phase < 30*time.Second {
			e.c.After(blink - phase%blink)
		}
	}
	p.popClip()
	p.clip = saved
}

// rangeSpans returns the TextRanges in runes start up to end of the text,
// from start: the spans of their weights for the layout ("" when none
// sets one), and how to paint their colors (nil when none sets one). An
// input method's composition at the caret, between start and end, sits
// unstyled between the runs, and those after it move past it. What it
// returns to paint is the editor's, good until the next call.
func (ed *editor) rangeSpans(start, end int) (string, *spanPaint) {
	return ed.spansOf(ed.inReach(start), start, end)
}

// rangeKey returns the spans of the weights of the TextRanges in runes
// start up to end, as rangeSpans does, without how to paint them: a
// layout needs no more, and a text without weights costs nothing.
func (ed *editor) rangeKey(start, end int) string {
	key, _ := ed.styleRanges(ed.inReach(start), start, end, false)
	return key
}

// inReach returns the ranges from the first that can reach rune start.
func (ed *editor) inReach(start int) []TextRange {
	if !ed.apart {
		ed.sortRanges()
	}
	// The ranges are sorted and apart: of those starting before start, only
	// the last can reach in. A paragraph of a long text skips the others
	// rather than walking them all.
	first := sort.Search(len(ed.ranges), func(i int) bool { return ed.ranges[i].Start >= start })
	if first > 0 && ed.ranges[first-1].End > start {
		first--
	}
	return ed.ranges[first:]
}

// spansOf is rangeSpans of ranges, sorted and apart: those from the first
// starting at end or later are out of reach.
func (ed *editor) spansOf(ranges []TextRange, start, end int) (string, *spanPaint) {
	return ed.styleRanges(ranges, start, end, true)
}

// styleRanges is spansOf, building how to paint the ranges only if paint
// is set. Without it, it builds nothing unless a range changes how the
// runes are laid out: a weight, font, size or slant, runes hidden, a
// widget.
func (ed *editor) styleRanges(ranges []TextRange, start, end int, paint bool) (string, *spanPaint) {
	reach := len(ranges)
	if !paint {
		shapes := false
		for i := range ranges {
			r := &ranges[i]
			if r.Start >= end {
				reach = i
				break
			}
			shapes = shapes || max(r.Start, start) < min(r.End, end) && ed.shapes(r)
		}
		if !shapes {
			return "", nil
		}
	}
	caret, n := -1, 0
	if ed.compose != "" && start <= ed.caret && ed.caret <= end {
		caret, n = ed.caret, utf8.RuneCountInString(ed.compose)
	}
	// local is where rune i is in the layout: a run that starts at the
	// caret starts after the composition, one that ends there ends before it.
	local := func(i int, starts bool) int {
		if caret >= 0 && (i > caret || i == caret && starts) {
			i += n
		}
		return i - start
	}
	// Each range adds a run, a gap before it, and one more where the caret
	// splits it. To paint, the editor's buffers, which the painter holds
	// only while it paints the text.
	var styles []text.Span
	var spans []Span
	if paint {
		styles, spans = ed.paintStyles[:0], ed.paintSpans[:0]
	} else {
		styles = make([]text.Span, 0, 2*reach+1)
	}
	at := 0
	add := func(from, to int, st text.Span, ps Span) {
		if from > at {
			styles = append(styles, text.Span{End: from})
			if paint {
				spans = append(spans, Span{})
			}
		}
		st.End = to
		styles = append(styles, st)
		if paint {
			spans = append(spans, ps)
		}
		at = to
	}
	piece := func(from, to int, st text.Span, ps Span) {
		if from >= to {
			return
		}
		if from < caret && caret < to {
			add(local(from, true), local(caret, false), st, ps)
			add(local(caret, true), local(to, false), st, ps)
		} else {
			add(local(from, true), local(to, false), st, ps)
		}
	}
	shaped, painted := false, false
	for i := range ranges[:reach] {
		r := &ranges[i]
		if r.Start >= end {
			break
		}
		from, to := max(r.Start, start), min(r.End, end)
		if from >= to {
			continue
		}
		st, ps := ed.runStyle(r)
		if r.widget != 0 {
			// The widget's box is its first rune; the others take no room.
			w := &ed.widgets[r.widget-1]
			if from == w.Start {
				first := st
				first.ObjectWidth, first.ObjectHeight, first.ObjectDescent = max(w.Width, hiddenWidth), max(w.Height-w.Descent, 0), w.Descent
				piece(from, from+1, first, ps)
				from++
			}
		}
		piece(from, to, st, ps)
		shaped = shaped || st != text.Span{}
		painted = painted || ps.Color.A > 0 || ps.Strikethrough || ps.Background.A > 0
	}
	key := ""
	if shaped {
		key = text.EncodeSpans(styles)
	}
	var sp *spanPaint
	if paint {
		ed.paintStyles, ed.paintSpans = styles, spans
		if painted {
			ed.painted = spanPaint{spans: spans, styles: styles}
			sp = &ed.painted
		}
	}
	return key, sp
}

// hiddenWidth is the width of a rune hidden (TextRange.Hidden) or under a
// widget past its first: an inline object as narrow as the text engines
// measure, one Pango unit, which draws nothing.
const hiddenWidth = 1.0 / 1024

// hiddenSize is the font size of runes hidden: Pango leaves the spaces of
// an inline object's run their font's width, which this makes none.
const hiddenSize = 1.0 / 64

// shapes reports whether r changes how its runes are laid out.
func (ed *editor) shapes(r *TextRange) bool {
	return r.Weight > 0 || r.Font != "" || r.Size > 0 || r.Italic || r.widget != 0 || ed.hides(r)
}

// hides reports whether r hides its runes now: it hides them, and neither
// the caret nor an end of the selection is within its reveal range.
func (ed *editor) hides(r *TextRange) bool {
	if !r.Hidden {
		return false
	}
	if r.RevealEnd <= r.RevealStart {
		return true
	}
	lo, hi := min(ed.anchor, ed.caret), max(ed.anchor, ed.caret)
	return lo > r.RevealEnd || hi < r.RevealStart
}

// runStyle returns how a run styled by r is laid out and painted, past
// the first rune of a widget.
func (ed *editor) runStyle(r *TextRange) (text.Span, Span) {
	if r.widget != 0 || ed.hides(r) {
		return text.Span{ObjectWidth: hiddenWidth, Size: hiddenSize}, Span{}
	}
	weight := max(r.Weight, 0)
	return text.Span{Weight: weight, Family: r.Font, Size: r.Size, Italic: r.Italic},
		Span{Color: r.Color, Weight: weight, Strikethrough: r.Strikethrough, Background: r.Background, Radius: r.Radius}
}
