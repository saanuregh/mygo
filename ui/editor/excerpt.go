package editor

// excerpt is the lines an editor shows of its buffer (State.SetExcerpt),
// from the start of the first to the end of the last, as offsets that
// move with the edits: the start stays before text inserted at it, and
// the end goes after, so that lines typed at either end are the
// excerpt's.
type excerpt struct {
	on         bool
	start, end int
}

func (x *excerpt) moved(cs []change) {
	if x.on {
		x.start, x.end = mapOffset(x.start, cs, biasLeft), mapOffset(x.end, cs, biasRight)
		x.end = max(x.end, x.start)
	}
}

// excerptLines returns the first and the last line the editor shows, and
// false when it shows them all.
func (ed *editor) excerptLines() (first, last int, ok bool) {
	x := ed.excerpt
	if !x.on {
		return 0, 0, false
	}
	r := &ed.view.text
	n := r.Len()
	first = r.LineOf(min(x.start, n))
	return first, max(first, r.LineOf(min(x.end, n))), true
}

// excerptBounds returns the offsets of the start of the excerpt's first
// line and of the end of its last.
func (ed *editor) excerptBounds() (lo, hi int, ok bool) {
	first, last, ok := ed.excerptLines()
	if !ok {
		return 0, 0, false
	}
	r := &ed.view.text
	return r.LineStart(first), r.LineEnd(last), true
}

// excerptHidden returns the lines outside the excerpt.
func (ed *editor) excerptHidden() []lineSpan {
	first, last, ok := ed.excerptLines()
	if !ok {
		return nil
	}
	var spans []lineSpan
	if first > 0 {
		spans = append(spans, lineSpan{0, first - 1})
	}
	if n := ed.view.text.LineCount(); last < n-1 {
		spans = append(spans, lineSpan{last + 1, n - 1})
	}
	return spans
}

// inExcerpt reports whether edits, of the text the editor shows, stay
// within the excerpt.
func (ed *editor) inExcerpt(edits []Edit) bool {
	lo, hi, ok := ed.excerptBounds()
	if !ok {
		return true
	}
	for _, e := range edits {
		if e.Start < lo || e.End > hi {
			return false
		}
	}
	return true
}

// keepInExcerpt puts the ends of selections outside the excerpt at its
// edges.
func (ed *editor) keepInExcerpt() {
	lo, hi, ok := ed.excerptBounds()
	if !ok {
		return
	}
	moved := false
	for i := range ed.sels {
		s := &ed.sels[i]
		a, h := max(lo, min(s.anchor, hi)), max(lo, min(s.head, hi))
		if a != s.anchor || h != s.head {
			s.anchor, s.head, s.goal = a, h, -1
			moved = true
		}
	}
	if moved {
		ed.sels = mergeSels(ed.sels)
	}
}

// SetExcerpt shows only lines first to last, from 0, of the buffer, as
// an excerpt of a file in a multibuffer, where editors of one buffer
// show a part of it each: the other lines take no room, the carets stay
// within the excerpt, and an edit that would reach outside it is not
// made. The gutter numbers the lines as in the buffer. The excerpt moves
// with the edits of the text, and takes the lines typed at its first
// line's start and its last line's end.
func (st *State) SetExcerpt(first, last int) {
	st.ensure()
	ed := &st.ed
	r := &ed.view.text
	n := r.LineCount()
	first = max(0, min(first, n-1))
	last = max(first, min(last, n-1))
	ed.excerpt = excerpt{on: true, start: r.LineStart(first), end: r.LineEnd(last)}
	ed.keepInExcerpt()
	ed.needsBuild = true
}

// Excerpt returns the lines SetExcerpt shows, where the edits since moved
// them, and false when the editor shows every line; ClearExcerpt shows
// them all again.
func (st *State) Excerpt() (first, last int, ok bool) {
	st.ensure()
	return st.ed.excerptLines()
}

func (st *State) ClearExcerpt() {
	st.ensure()
	st.ed.excerpt = excerpt{}
	st.ed.needsBuild = true
}
