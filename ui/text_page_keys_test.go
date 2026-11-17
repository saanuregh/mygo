package ui

import "testing"

// PageUp and PageDown with Ctrl or Alt are left to the window's shortcuts
// from a text field, as switching tabs with Ctrl+PageDown is in VS Code and
// Zed; without them, or with Shift alone, they page the text.
func TestTextFieldLeavesModifiedPageKeysToShortcuts(t *testing.T) {
	for _, multiline := range []bool{false, true} {
		v := "one\ntwo\nthree"
		ran := map[Modifiers]int{}
		tt := NewTester(func(c *Context) {
			for _, m := range []Modifiers{0, Shift, Ctrl, Ctrl | Shift, Alt} {
				c.OnShortcut(m, KeyPageDown, func() { ran[m]++ })
			}
			if multiline {
				TextArea(c, &v).AutoFocus().Label("field").Size(400, 200)
			} else {
				TextInput(c, &v).AutoFocus().Label("field").Width(400)
			}
		}, 500, 300)
		tt.Frame()
		for _, m := range []Modifiers{Ctrl, Ctrl | Shift, Alt} {
			tt.Key(m, KeyPageDown)
			if ran[m] != 1 {
				t.Errorf("multiline %v: the shortcut of %v+PageDown ran %d times", multiline, m, ran[m])
			}
		}
		if multiline {
			for _, m := range []Modifiers{0, Shift} {
				tt.Key(m, KeyPageDown)
				if ran[m] != 0 {
					t.Errorf("the text area left %v+PageDown to its shortcut", m)
				}
			}
		}
		tt.rt.close()
	}
}
