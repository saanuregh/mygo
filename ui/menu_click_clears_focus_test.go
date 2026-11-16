package ui

import "testing"

// A press on a control that ClearsFocus, outside a drawn menu, closes the
// menu and leaves the focus nowhere: the field that had it as the menu
// opened does not take it back (zode's sidebar row's Snooze button under
// the row's menu, as nanoui's controls).
func TestAPressOnAClearsFocusControlPastAMenuLeavesTheFocusNowhere(t *testing.T) {
	var draft string
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Gap(10).Children(func() {
			coreTextInput(c, &draft).Label("Message")
			coreRow(c).Padding(8).Label("Thread").Children(func() {
				coreText(c, "A thread")
			}).PopoverMenu(func(m *Menu) { m.Item("Rename") })
			coreRow(c).Size(60, 24).Label("Snooze").ClearsFocus()
		})
	}, 400, 300)
	defer tt.rt.close()
	tt.Frame()
	if err := tt.Click("Message"); err != nil {
		t.Fatal(err)
	}
	tt.RightClick("A thread")
	tt.Frame()
	if !tt.HasText("Rename") {
		t.Fatalf("no menu: %q", tt.Texts())
	}
	if err := tt.Click("Snooze"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	tt.Frame()
	if tt.HasText("Rename") || tt.Focused("Message") {
		t.Fatalf("menu %v, the field focused %v", tt.HasText("Rename"), tt.Focused("Message"))
	}
}
