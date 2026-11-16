package markdown

import (
	"strings"

	"github.com/egoist/mygo/ui"
)

// BlockPlace is where a block of a view's document showed, as Blocks
// returns it.
type BlockPlace struct {
	// Index is the block's in the Document.
	Index int
	// FirstLine and LastLine are the lines of the source the block
	// spans, from 0, LastLine included.
	FirstLine, LastLine int
	// Bounds is its box in the last frame, in window DIPs, as
	// ui.Element.Bounds returns it: empty for one the last frame lacked.
	Bounds ui.Rect
}

// Blocks returns the top-level blocks of the view el, which View or
// Stream returned in this frame, in order: the lines of the source each
// spans and where each showed in the last frame. A preview beside an
// editor scrolls in step with it by them, to the block at the editor's
// top line. Blocks an Options.Block hook built are among them; footnotes
// are not. Measure the boxes against the view's own (el.Bounds), from the
// same frame, rather than against a scroll container's offset, which the
// user may have moved since.
//
//	md := markdown.View(c, src, &app.md)
//	for _, b := range markdown.Blocks(md) {
//		if b.FirstLine <= top && top <= b.LastLine { /* scroll to b.Bounds */ }
//	}
func Blocks(el ui.Element) []BlockPlace {
	st := *ui.Local(el, stateKey{}, newState)
	if st.doc == nil || len(st.placed) == 0 {
		return nil
	}
	src := st.doc.Source()
	out := make([]BlockPlace, 0, len(st.placed))
	at, line := 0, 0
	for _, p := range st.placed {
		b := st.doc.Block(p.block)
		start, end := min(b.Start, len(src)), min(b.End, len(src))
		if start < at {
			// Not after the last: counted from the start again.
			at, line = 0, 0
		}
		line += strings.Count(src[at:start], "\n")
		at = start
		last := line + strings.Count(src[start:end], "\n")
		if end > start && src[end-1] == '\n' {
			last--
		}
		out = append(out, BlockPlace{Index: p.block, FirstLine: line, LastLine: max(line, last), Bounds: p.bounds})
	}
	return out
}

// placed is a top-level block built in the frame, and its last box.
type placed struct {
	block  int
	bounds ui.Rect
}
