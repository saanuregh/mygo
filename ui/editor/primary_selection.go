package editor

import (
	"strings"

	"github.com/egoist/mygo/ui"
)

// offerPrimary puts the selected text in the primary selection when the
// selections changed since it last looked and the editor has the
// keyboard, once a drag has let go (State.PrimarySelection). Selections
// that change while another editor has the keyboard are noted, not
// offered, so focusing the editor again leaves the primary selection to
// what was selected last.
func (ed *editor) offerPrimary() {
	if !ed.o.primarySelection || ed.gesture.active || ed.samePrimarySels() {
		return
	}
	ed.primarySels = ed.primarySels[:0]
	for _, s := range ed.sels {
		ed.primarySels = append(ed.primarySels, [2]int{s.anchor, s.head})
	}
	if !ed.focused {
		return
	}
	if text := ed.selectedText(); text != "" {
		ed.services.WritePrimary(text)
	}
}

func (ed *editor) samePrimarySels() bool {
	if len(ed.primarySels) != len(ed.sels) {
		return false
	}
	for i, s := range ed.sels {
		if ed.primarySels[i] != [2]int{s.anchor, s.head} {
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
