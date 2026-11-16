package ui

import "testing"

// With MenuStyle.ClickLeavesFocusNowhere, an item of a drawn menu chosen
// by a click leaves the focus nowhere, as a press on a control of nanoui's
// did; Escape gives it back to the field that had it (zode's composer
// under a sidebar row's menu).
func TestAMenuItemClickedLeavesTheFocusNowhere(t *testing.T) {
	var draft string
	chosen := 0
	tt := coreNewTester(func(c *context) {
		th := LightTheme()
		th.Menu.ClickLeavesFocusNowhere = true
		c.SetTheme(th)
		coreColumn(c).Fill().Gap(10).Children(func() {
			coreTextInput(c, &draft).Label("Message")
			coreRow(c).Padding(8).Label("Thread").Children(func() {
				coreText(c, "A thread")
			}).PopoverMenu(func(m *Menu) {
				if m.Item("Rename").Chosen() {
					chosen++
				}
			})
		})
	}, 400, 300)
	defer tt.rt.close()
	tt.Frame()
	if err := tt.Click("Message"); err != nil {
		t.Fatal(err)
	}
	field := tt.rt.focused

	tt.RightClick("A thread")
	tt.Frame()
	tt.Key(0, KeyEscape)
	tt.Frame()
	tt.Frame()
	if tt.rt.focused != field {
		t.Fatal("Escape did not give the focus back")
	}

	tt.RightClick("A thread")
	tt.Frame()
	if err := tt.Click("Rename"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	tt.Frame()
	if chosen != 1 {
		t.Fatalf("chosen %d times", chosen)
	}
	if tt.rt.focused != 0 {
		t.Fatalf("the focus went to %d (the field is %d)", tt.rt.focused, field)
	}
	tt.TypeKey(0, KeyA, "a")
	if draft != "" {
		t.Fatalf("typing reached the field: %q", draft)
	}
}

// With MenuStyle.ClickLeavesFocusNowhere, an item clicked that opens a
// NextMenu leaves the field no focus under it, nor after it closes, even
// by Escape: the click left the focus nowhere (zode's Snooze… in a sidebar
// row's menu, as nanoui's).
func TestANextMenuOpenedByAClickLeavesTheFieldUnfocused(t *testing.T) {
	var draft string
	under := false
	tt := coreNewTester(func(c *context) {
		th := LightTheme()
		th.Menu.ClickLeavesFocusNowhere = true
		c.SetTheme(th)
		coreColumn(c).Fill().Gap(10).Children(func() {
			in := coreTextInput(c, &draft).Label("Message")
			under = in.FocusedUnderOverlay() || in.FocusedUnderMenu()
			coreRow(c).Padding(8).Label("Thread").Children(func() {
				coreText(c, "A thread")
			}).PopoverMenu(func(m *Menu) {
				m.NextMenu("Snooze…", func(m *Menu) { m.Item("In an Hour") })
			})
		})
	}, 400, 300)
	defer tt.rt.close()
	tt.Frame()
	if err := tt.Click("Message"); err != nil {
		t.Fatal(err)
	}
	tt.RightClick("A thread")
	tt.Frame()
	if !under {
		t.Fatal("the field shows no focus under the first menu")
	}
	if err := tt.Click("Snooze…"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText("In an Hour") {
		t.Fatalf("no next menu: %q", tt.Texts())
	}
	if under {
		t.Fatal("the field shows its focus under the next menu")
	}
	tt.Key(0, KeyEscape)
	tt.Frame()
	tt.Frame()
	if tt.rt.states[tt.rt.focused] != nil {
		t.Fatal("the focus came back after the next menu")
	}
	tt.TypeKey(0, KeyA, "a")
	if draft != "" {
		t.Fatalf("typing reached the field: %q", draft)
	}
}

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
