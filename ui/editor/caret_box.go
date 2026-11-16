package editor

import "github.com/egoist/mygo/ui"

// CaretBox returns the newest caret's rectangle in the window, as the
// editor was last painted, and whether its line shows: where an app opens
// a menu of its own at the caret (OpenPopoverMenu), as code actions.
func (st *State) CaretBox() (ui.Rect, bool) {
	st.ensure()
	ed := &st.ed
	cr, ok := ed.caretRect(ed.primary().head)
	if !ok || ed.r.W <= 0 {
		return ui.Rect{}, false
	}
	return ui.Rect{X: ed.r.X + ed.textX + cr.X - ed.scrollX, Y: ed.r.Y + float32(float64(cr.Y)-ed.scrollY), W: cr.W, H: cr.H}, true
}

// CaretX returns where the newest caret is across its row, from the
// text's left edge, as the editor was last laid out: the x Up and Down
// keep (its goal) once they moved it.
func (st *State) CaretX() float32 {
	st.ensure()
	ed := &st.ed
	p := ed.primary()
	if p.goal >= 0 {
		return p.goal
	}
	cr, _ := ed.caretRect(p.head)
	return cr.X
}

// PlaceCaret puts one caret on line, nearest x across its first row (its
// last with lastRow) from the text's left edge, and keeps x for Up and
// Down to move along: where a caret coming from another editor lands, as
// from one excerpt to the next. It is false, and changes nothing, for an
// editor not yet laid out.
func (st *State) PlaceCaret(line int, x float32, lastRow bool) bool {
	st.ensure()
	ed := &st.ed
	v := &ed.view
	if ed.r.W <= 0 || line < 0 || line >= v.text.LineCount() {
		return false
	}
	l := v.line(line)
	ri := 0
	if lastRow {
		ri = len(l.rows) - 1
	}
	at := l.start + l.fromDisp(v.dispAt(l, &l.rows[ri], x))
	st.SetSelections(Selection{Anchor: at, Head: at})
	ed.sels[newest(ed.sels)].goal = x
	return true
}
