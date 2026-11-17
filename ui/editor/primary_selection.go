package editor

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// primaryIdle is how long a selection rests before it is offered: copying
// a large selection on every key of Shift+arrow would cost its size each.
const primaryIdle = 150 * time.Millisecond

// offerPrimary puts the selected text in the primary selection once the
// selections have rested primaryIdle since they last changed, or at once
// when a drag lets go, while the editor has the keyboard
// (State.PrimarySelection). Selections already offered are not offered
// again. Selections that change while another editor has the keyboard are
// noted, not offered, so focusing the editor again leaves the primary
// selection to what was selected last.
func (ed *editor) offerPrimary(c *ui.Context) {
	if !ed.o.primarySelection {
		return
	}
	if ed.gesture.active {
		ed.primaryDragged = true
		return
	}
	now := c.Now()
	if !sameSels(ed.primaryRest, ed.sels) {
		ed.primaryRest = snapshotSels(ed.primaryRest, ed.sels)
		ed.primaryRestAt = now
	}
	if sameSels(ed.primarySels, ed.sels) {
		ed.primaryDragged = false
		return
	}
	if ed.focused && !ed.primaryDragged {
		if wait := primaryIdle - now.Sub(ed.primaryRestAt); wait > 0 {
			c.After(wait)
			return
		}
	}
	ed.primaryDragged = false
	ed.primarySels = snapshotSels(ed.primarySels, ed.sels)
	if !ed.focused {
		return
	}
	if text := ed.selectedText(); text != "" {
		ed.services.WritePrimary(text)
	}
}

func snapshotSels(into [][2]int, sels []sel) [][2]int {
	into = into[:0]
	for _, s := range sels {
		into = append(into, [2]int{s.anchor, s.head})
	}
	return into
}

func sameSels(seen [][2]int, sels []sel) bool {
	if len(seen) != len(sels) {
		return false
	}
	for i, s := range sels {
		if seen[i] != [2]int{s.anchor, s.head} {
			return false
		}
	}
	return true
}

// pastePrimary is a middle press on the text: the caret goes to the
// pointer and the primary selection's text is pasted there, whole.
func (ed *editor) pastePrimary(ev ui.InputEvent) bool {
	if ed.o.readOnly || ed.region != regionText {
		return false
	}
	cx, cy := ed.toContent(ev.X, ev.Y)
	off := ed.offsetAt(cx, cy)
	text := strings.ReplaceAll(ed.services.ReadPrimary(), "\r\n", "\n")
	ed.endGroup()
	caret := ed.newSel(off, off)
	ed.setSels([]sel{caret}, revealNone)
	if text == "" {
		return true
	}
	ed.apply([]Edit{{Start: off, End: off, Text: text}}, editOther, func(move func(int, bias) int) []sel {
		caret.collapse(move(off, biasRight))
		return []sel{caret}
	})
	return true
}
