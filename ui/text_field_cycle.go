package ui

// FocusNextTextField moves the keyboard focus to the next (back: the
// previous) element that takes text, in the order the view built them, as
// nanoui's focus cycle over its text fields; false, the focus unchanged,
// when no other such element can take it. A dialog on top keeps the cycle
// among its own elements, as Tab does.
func (c *Context) FocusNextTextField(back bool) bool {
	rt := c.runtime()
	if rt == nil {
		return false
	}
	var order []uint64
	for i, id := range rt.focusOrder {
		if (rt.modal == 0 || rt.focusScopes[i].modal == rt.modal) && receivesTyping(rt.states[id]) {
			order = append(order, id)
		}
	}
	n := len(order)
	i := -1
	for j, id := range order {
		if id == rt.focused {
			i = j
		}
	}
	if n == 0 || n == 1 && i == 0 {
		return false
	}
	next := 0
	switch {
	case i < 0 && back:
		next = n - 1
	case i >= 0 && back:
		next = (i + n - 1) % n
	case i >= 0:
		next = (i + 1) % n
	}
	rt.focused = order[next]
	rt.focusVisible = true
	rt.blinkStart = rt.now()
	rt.reveal(rt.focused)
	rt.requestFrame()
	return true
}
