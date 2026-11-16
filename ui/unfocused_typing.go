package ui

// unfocusedTyping is what was typed while no element that takes text had
// the keyboard focus, since the last pass: the text inserted, and whether anything came (text,
// a composition or an edit command).
type unfocusedTyping struct {
	text  string
	typed bool
}

// noteUnfocused keeps ev when no element with the focus takes text.
func (rt *engine) noteUnfocused(ev editEvent) {
	if receivesTyping(rt.states[rt.focused]) {
		return
	}
	switch ev.kind {
	case editInsert:
		rt.unfocused.text += ev.text
		rt.unfocused.typed = true
	case editCompose:
		rt.unfocused.typed = rt.unfocused.typed || ev.text != ""
	case editCommand:
		rt.unfocused.typed = true
	}
	if rt.unfocused.typed {
		rt.requestFrame()
	}
}

// UnfocusedTyping is what was typed since the last frame while no element
// that takes text (a text input or area, an editor, an element handling
// input) had the keyboard focus, which no element takes: the text inserted ("" for an
// edit command, as Backspace, or a composition), and whether anything was
// typed at all. The next frame forgets it. An app that gives a field the
// focus at any key (a chat's composer, resting) puts the text in it.
func (c *Context) UnfocusedTyping() (text string, typed bool) {
	rt := c.runtime()
	if rt == nil {
		return "", false
	}
	return rt.unfocused.text, rt.unfocused.typed
}

// receivesTyping reports whether s is an element typing reaches: a text
// field, an editable editor, a custom text client or an element handling
// input, which may take keys as text.
func receivesTyping(s *state) bool {
	editable := s != nil && s.editor != nil && s.flags&flagEditable != 0 && !s.editor.readOnly
	return s != nil && (s.textClient != nil || editable || s.input != nil)
}

// FocusCleared reports whether no element has the keyboard focus, or only
// read-only text inside an element that ClearsFocus, which holds it for
// copying its selection alone: what had the focus let it go, as a press on
// a control of nanoui's left the focus nowhere.
func (c *Context) FocusCleared() bool {
	rt := c.runtime()
	if rt == nil {
		return false
	}
	s := rt.states[rt.focused]
	if s == nil {
		return true
	}
	sc := rt.states[s.textScope]
	return s.textScope != 0 && s.flags&flagEditable == 0 && sc != nil && sc.clearsFocus
}
