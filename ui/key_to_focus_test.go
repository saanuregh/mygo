package ui

import "testing"

// KeyToFocus gives the key to the focused element alone, as nanoui's
// Tester.key: the window's shortcut for it does not run, and Key's does.
func TestKeyToFocusSkipsShortcuts(t *testing.T) {
	var ref Ref
	ref.RequestFocus()
	shortcuts, focused := 0, 0
	tt := NewTester(func(c *Context) {
		c.OnShortcut(0, KeyEscape, func() { shortcuts++ })
		Box(c).Size(100, 40).Focusable().Ref(&ref).HandleInput(func(ev InputEvent) bool {
			if ev.Kind == InputKeyDown && ev.Key == KeyEscape {
				focused++
				return true
			}
			return false
		})
	}, 200, 100)
	defer tt.rt.close()
	tt.Frame()
	tt.KeyToFocus(0, KeyEscape)
	if shortcuts != 0 || focused != 1 {
		t.Fatalf("KeyToFocus: shortcut ran %d times, the focus got %d keys", shortcuts, focused)
	}
	tt.Key(0, KeyEscape)
	if shortcuts != 1 || focused != 1 {
		t.Fatalf("Key: shortcut ran %d times, the focus got %d keys", shortcuts, focused)
	}
}
