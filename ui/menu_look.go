package ui

import "github.com/egoist/mygo/internal/platform"

// MenuStyle is how the menus MyGo draws in the window look, set as the
// Theme's Menu: those of PopoverMenu and PopoverMenuButton, and of
// ContextMenu and Menu in a window that draws its menus
// (SetMenuPresentation). Its zero value is MyGo's own look, and each field
// left zero keeps that part of it. A menu's PopoverMenuOptions and
// PopoverMenuButtonOptions style it after its theme's MenuStyle:
//
//	th.Menu = ui.MenuStyle{Width: 220, ItemHeight: 30, Hover: th.SurfaceHover}
type MenuStyle struct {
	// Width is the width of a menu whose function sets no MaxWidth, and
	// of its submenus; 0 fits the items, at least 45 units of the theme's
	// Spacing.
	Width float32
	// Background fills the panel (0: the Theme's Background); Shadow,
	// ShadowBlur and ShadowY are its shadow's color, blur and offset down
	// (0 color: MyGo's soft black one).
	Background          Color
	Shadow              Color
	ShadowBlur, ShadowY float32
	Radius              float32 // the panel's corners; 0 is the Theme's Radius+2
	// Padding is the room between the panel's edge and its items, its
	// border's width in it; 0 is Space(1) inside the border.
	Padding float32
	Gap     float32 // between items
	// Offset is the room between the menu and where it opens: below the
	// point it opened at, or apart from its menu button (0: Space(1) from
	// the button, none from the point). Menu.Offset sets one menu's.
	Offset float32
	// ShiftIntoWindow moves a menu opened at a point, whose width is known
	// (Width, Menu.MaxWidth), back into the window where it would pass the
	// window's end, rather than opening it before the point.
	ShiftIntoWindow bool
	// ItemHeight is each item's height; 0 is its text's, with Space(1.5)
	// above and below.
	ItemHeight  float32
	ItemPadding float32 // at an item's start and end; 0 is Space(2)
	ItemGap     float32 // between an item's mark, icon, label and keys; 0 is Space(2)
	ItemRadius  float32 // an item's corners; 0 is the Theme's Radius
	// FontSize is the labels' size, and ShortcutSize that of the shortcuts
	// and Trailing texts (0: the Theme's FontSize).
	FontSize, ShortcutSize float32
	// IconSize is the size an item's icon (MenuItem.Icon) is drawn at; 0
	// is 16.
	IconSize float32
	// MarkSize is the width of the room for check marks; 0 is Space(4).
	MarkSize float32
	// SeparatorSpace is the room above and below a separator, whose line
	// then spans the panel; 0 is Space(1), the line inset by Space(2).
	SeparatorSpace float32
	// Hover fills the item highlighted, which keeps its colors, a Danger
	// one's filled with the Danger color, faint; 0 fills it with the
	// Accent, AccentText on it.
	Hover Color
	// DisabledOpacity fades a disabled item whole, to that opacity in
	// all, its label in its usual color; 0 grays its text and dims it as
	// any disabled element.
	DisabledOpacity float32
}

// Offset sets the room between the menu, as PopoverMenu and
// PopoverMenuButton draw it, and where it opens, d DIPs, for this menu
// alone, in place of its theme's MenuStyle.Offset: below the point it
// opened at, or apart from its menu button. 0 keeps the style's. The
// system's menus take no offset from it.
//
//	m.Offset(7)
func (m *Menu) Offset(d float32) *Menu {
	if r := m.run; r.collect && m.menu != nil {
		if r.offsets == nil {
			r.offsets = map[*platform.Menu]float32{}
		}
		r.offsets[m.menu] = d
	}
	return m
}

// or returns v, or by when v is zero.
func or[T comparable](v, by T) T {
	var zero T
	if v == zero {
		return by
	}
	return v
}

// TestMenuItem is an item of the menu drawn in the window that shows, as
// Tester.PopoverMenu tells it.
type TestMenuItem struct {
	Label     string
	Separator bool
	Disabled  bool
	Checked   bool // a check box or radio item, checked
	Submenu   bool
	// Shortcut is the item's keys as the menu writes them (MenuItem.Shortcut),
	// and Trailing its Trailing text.
	Shortcut string
	Trailing string
	Icon     MenuIcon // MenuItem.Icon
	Danger   bool     // MenuItem.Danger
}

// PopoverMenu returns the items of the menu drawn in the window that
// shows, as the last frame built it: PopoverMenu's and
// PopoverMenuButton's, and ContextMenu's and Menu's in a window that draws
// its menus (SetMenuPresentation). It tells the menu itself, not its
// submenus, and returns nil while none shows.
func (t *Tester) PopoverMenu() []TestMenuItem {
	d := t.rt.menu.drawn
	if d == nil || d.items == nil {
		return nil
	}
	out := []TestMenuItem{}
	for _, it := range d.items.Items {
		if !it.Visible {
			continue
		}
		ti := TestMenuItem{Label: it.Label, Separator: it.Type == platform.MenuItemSeparator}
		if !ti.Separator {
			ti.Disabled = !it.Enabled
			ti.Checked = it.Checked
			ti.Submenu = it.Type == platform.MenuItemSubmenu
			ti.Shortcut = shortcutText(it.Accelerator)
			if mi := d.looks[it.ID-1]; mi != nil {
				ti.Trailing, ti.Icon, ti.Danger = mi.trailing, mi.icon, mi.danger
			}
		}
		out = append(out, ti)
	}
	return out
}
