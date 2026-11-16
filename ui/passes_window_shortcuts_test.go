package ui

import "testing"

// The window's shortcuts wait behind a dialog, unless its backdrop
// PassesWindowShortcuts: then they run as though it were not there.
func TestDialogPassesWindowShortcuts(t *testing.T) {
	for _, passes := range []bool{false, true} {
		open := true
		var draft string
		shortcuts := 0
		tt := NewTester(func(c *Context) {
			c.OnShortcut(Ctrl, KeyK, func() { shortcuts++ })
			DialogBase(c, &open, func(backdrop, panel Element) {
				if passes {
					backdrop.PassesWindowShortcuts()
				}
				TextInput(c, &draft).Label("Field")
			})
		}, 300, 200)
		tt.Frame()
		if err := tt.Click("Field"); err != nil {
			t.Fatal(err)
		}
		tt.Key(Ctrl, KeyK)
		tt.TypeKey(0, KeyA, "a")
		want := 0
		if passes {
			want = 1
		}
		if shortcuts != want || draft != "a" || !open {
			t.Errorf("passes %v: shortcuts %d (want %d), field %q, open %v", passes, shortcuts, want, draft, open)
		}
		tt.rt.close()
	}
}
