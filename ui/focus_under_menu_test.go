package ui

import "testing"

// A text area keeps its caret under a menu another element opened from
// code (a list row's ⋯), and a view drawing its own caret reads that it
// had the focus, under its own menu too (FocusedUnderMenu).
func TestFocusUnderAMenuOpenedFromCode(t *testing.T) {
	text := ""
	open, own := false, false
	var area, row *node
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Padding(20).Gap(10).Children(func() {
			area = coreTextArea(c, &text).Label("Message")
			area.PopoverMenu(func(m *Menu) { m.Item("Own") })
			row = coreRow(c).Padding(8).Label("Row").Children(func() { coreText(c, "Row one") })
			row.PopoverMenu(func(m *Menu) { m.Item("Choose") })
			if open {
				open = false
				row.OpenPopoverMenu(30, 80, func(m *Menu) { m.Item("Choose") })
			}
			if own {
				own = false
				area.OpenPopoverMenu(30, 30, func(m *Menu) { m.Item("Own") })
			}
		})
	}, 400, 300)
	tt.Click("Message")
	tt.Frame()
	if area.FocusedUnderMenu() || tt.rt.caretUnderMenu(area.id) {
		t.Fatal("under a menu with none")
	}
	open = true
	tt.Frame()
	tt.Frame()
	if !tt.HasText("Choose") || !tt.rt.caretUnderMenu(area.id) || !area.FocusedUnderMenu() || row.FocusedUnderMenu() {
		t.Fatal("the area's caret went with the row's menu")
	}
	tt.Key(0, KeyEscape)
	tt.Frame()
	tt.Click("Message")
	tt.Frame()
	own = true
	tt.Frame()
	tt.Frame()
	if !tt.HasText("Own") || !area.FocusedUnderMenu() {
		t.Fatal("the area had the focus as its own menu opened")
	}
}
