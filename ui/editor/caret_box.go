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
