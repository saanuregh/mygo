package ui

import "github.com/egoist/mygo/internal/platform"

// NextMenu adds an item showing label which, chosen, closes the menu and
// opens another where it was, which build fills as the menu's function
// fills the menu, its items' Chosen reporting their choice as there: a
// second step of a choice, as a thread's Snooze… opening the times it can
// sleep until. Chosen by the keys, the next menu starts on its first item.
// In a menu the system shows, it adds build's items as a submenu instead.
// Its Chosen reports false; the item is found by its place and label.
//
//	m.NextMenu("Snooze…", func(m *ui.Menu) {
//		if m.Item("Tomorrow").Chosen() {
//			app.snooze(tomorrow)
//		}
//	})
func (m *Menu) NextMenu(label string, build func(m *Menu)) *MenuItem {
	r := m.run
	if !r.drawn {
		return m.Submenu(label, build)
	}
	it := m.add(label, platform.MenuItemNormal)
	if !r.collect && r.choice == it.seq && r.label == label {
		r.chosen, r.next = true, build
	}
	return it
}

// openNextMenu opens the menu build fills where the drawn menu d was, as d
// closes for the NextMenu item chosen from it: on its first item when the
// keys chose it, as they show the focus, and else on the item under the
// pointer, which has not moved.
func (rt *engine) openNextMenu(d *drawnMenu, build func(m *Menu)) {
	// Built in this pass, as d was: it stays once the pass is done.
	next := &drawnMenu{id: d.id, text: d.text, built: rt.frame, pass: rt.pass, build: build}
	pointer := menuPointerNotYet
	if rt.focusVisible {
		pointer = [2]float32{rt.pointerX, rt.pointerY}
	}
	next.menu = menuShown{below: d.menu.below, dx: d.menu.dx, dy: d.menu.dy, attach: d.menu.attach, opened: rt.frame,
		levels: []menuLevel{{first: rt.focusVisible}}, pointer: pointer}
	rt.menu.drawn = next
	rt.consumed = true
}

// OpenPopoverMenu opens the element's menu drawn in the window at (x, y)
// in the window, as a right-click there would, filled by build in place of
// the element's own function: for a menu that a control opens on the
// element, as a row's button that shows only while the row is hovered
// opening a menu that stays as the pointer leaves the row. Call it on an
// element with PopoverMenu or PopoverMenuButton built in this pass, or on
// one with ContextMenu or Menu in a window that draws its menus
// (SetMenuPresentation); build's items report Chosen as the element's own
// do, and the menu goes as the element goes. It takes the focus as
// PopoverMenu's does, and notes the element that had it, as a menu the
// pointer opens does (FocusedUnderMenu).
func (e *node) OpenPopoverMenu(x, y float32, build func(m *Menu)) {
	if !e.hasState() || e.disabled() || build == nil {
		return
	}
	rt := e.c.rt
	b := e.Bounds()
	d := &drawnMenu{id: e.id, built: rt.frame, pass: rt.pass, build: build, back: rt.focused}
	d.menu = menuShown{dx: x - b.X, dy: y - b.Y, opened: rt.frame, levels: []menuLevel{{}}, pointer: menuPointerNotYet}
	rt.menu.drawn = d
	rt.consumed = true
}

// OpenPopoverMenuBelow opens the element's menu drawn in the window below
// the element, or above it where the window has no room, lined up with its
// start, as a menu button's: for a menu anchored on a box, as a code
// editor's actions on the caret's, which never covers the box. As
// OpenPopoverMenu, build fills it in place of the element's own function.
func (e *node) OpenPopoverMenuBelow(build func(m *Menu)) {
	if !e.hasState() || e.disabled() || build == nil {
		return
	}
	rt := e.c.rt
	d := &drawnMenu{id: e.id, built: rt.frame, pass: rt.pass, build: build, back: rt.focused}
	d.menu = menuShown{below: true, opened: rt.frame, levels: []menuLevel{{}}, pointer: menuPointerNotYet}
	rt.menu.drawn = d
	rt.consumed = true
}

// MenuShows reports whether the element's menu drawn in the window shows:
// its PopoverMenu or PopoverMenuButton, its ContextMenu or Menu in a
// window that draws its menus, or one OpenPopoverMenu opened on it.
func (e *node) MenuShows() bool {
	d := e.c.rt.menu.drawn
	return d != nil && d.id == e.id && e.hasState()
}

// FocusedUnderMenu reports whether the element had the keyboard focus as
// the drawn menu that shows opened, its own or another element's: a view
// that draws its own caret or cursor keeps it, as nanoui's menus left the
// focus where it was.
func (e *node) FocusedUnderMenu() bool {
	d := e.c.rt.menu.drawn
	return d != nil && e.hasState() && d.back == e.id
}

// PopupShown reports whether a popover, a select's options or a drawn
// menu showed anywhere in the window in the last frame, as nanoui's
// Context.popovers_last: a view that rests while nothing needs it stays
// awake under one.
func (c *context) PopupShown() bool { return c.rt.tips.popupsLast }

// FocusedUnderOverlay reports whether an overlay open over the window, as
// a dialog, opened while the element had the keyboard focus, which it
// gives back as it closes, and no text input in the overlay has taken the
// focus since: an element that shows its focus as nanoui's did under a
// dialog, whose buttons took no focus of their own.
func (e *node) FocusedUnderOverlay() bool {
	if !e.hasState() {
		return false
	}
	rt := e.c.rt
	for id, opener := range rt.openers {
		if opener != e.id || rt.states[id] == nil {
			continue
		}
		f := rt.states[rt.focused]
		return f == nil || f.flags&flagEditable == 0
	}
	return false
}
