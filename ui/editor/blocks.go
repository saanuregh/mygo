package editor

import (
	"fmt"
	"slices"

	"github.com/egoist/mygo/ui"
)

// Block is a row of an editor between its lines that is not text, as
// the lines an inline diff deleted or a diagnostic's message under its
// line, Height DIPs high, above or below line Line, from 0, showing what
// State.RenderBlock builds there. Key identifies it among the others: a
// comparable value, as a string or a number, or nil to identify it by
// its line and place.
type Block struct {
	Line   int
	Below  bool
	Height float32
	Key    any
}

// appBlocks are the app's blocks as the display places them, made again
// when they change.
type appBlocks struct {
	src []Block
	out []block
	// built is the geometry the blocks' elements were built for, which a
	// frame painting another builds them again for.
	built blocksGeom
}

type blocksGeom struct {
	y, total float64
	w        float32
}

// appBlocks returns the app's blocks for the display: the slice changes
// when they do, which the display compares by identity.
func (ed *editor) appBlocks() []block {
	c := &ed.blocks
	if !slices.Equal(c.src, ed.o.blocks) {
		c.src = append(c.src[:0], ed.o.blocks...)
		c.out = make([]block, len(c.src))
		for i, b := range c.src {
			c.out[i] = block{line: b.Line, below: b.Below, height: max(0, b.Height), kind: blockApp, from: i}
		}
	}
	return c.out
}

// buildBlocks builds the elements of the app's blocks in view, where the
// last layout put them.
func (ed *editor) buildBlocks(c *ui.Context) {
	if ed.o.renderBlock == nil || len(ed.o.blocks) == 0 || ed.o.decor != nil || ed.r.W <= 0 {
		return
	}
	ed.blocks.built = ed.blocksGeom()
	w := ed.r.W - ed.trackWidth()
	ed.blocksInView(func(b *block, top float64) {
		if b.kind != blockApp || b.from >= len(ed.blocks.src) {
			return
		}
		ab := ed.blocks.src[b.from]
		key := ab.Key
		if key == nil {
			key = [3]any{ab.Line, ab.Below, b.from}
		}
		box := ui.Box(c).Key(fmt.Sprint("code block ", key)).Absolute().Left(0).Top(ed.top + float32(top-ed.scrollY)).
			Width(w).Height(b.height).HandleInput(ed.wheelOnly)
		ed.o.renderBlock(c, box, ab)
	})
}

func (ed *editor) blocksGeom() blocksGeom {
	return blocksGeom{y: ed.scrollY, total: ed.view.heights.total(), w: ed.r.W}
}

// rebuildBlocks builds the view again when a frame painted the blocks
// elsewhere than they were built, as scrolled or laid out anew.
func (ed *editor) rebuildBlocks() {
	if ed.o.renderBlock == nil || len(ed.o.blocks) == 0 || ed.o.decor != nil {
		return
	}
	if ed.blocksGeom() != ed.blocks.built {
		ed.services.Invalidate()
	}
}

// TextLeft returns where the text starts in the editor, past its
// gutter, in DIPs from its left edge, as the last frame laid it out: a
// block's content lines up with the text from there.
func (st *State) TextLeft() float32 { return st.ed.textX }
