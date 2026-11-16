package ui

import "testing"

// FocusedUnderOverlay: a field that had the focus as a dialog opened,
// which gives it back as it closes, says so while the dialog shows and
// no text input in it has the keyboard, as nanoui's dialogs left the
// focus.
func TestFocusedUnderOverlay(t *testing.T) {
	var field Ref
	field.RequestFocus()
	open, under := false, false
	text := ""
	tt := NewTester(func(c *Context) {
		in := TextInput(c, &text).Label("Field").Ref(&field)
		under = in.FocusedUnderOverlay()
		DialogBase(c, &open, func(back, panel Element) {
			panel.Size(100, 60).Children(func() { Button(c, "OK") })
		})
	}, 300, 200)
	tt.Frame()
	if !tt.Focused("Field") || under {
		t.Fatalf("before: focused %v, under %v", tt.Focused("Field"), under)
	}
	open = true
	tt.Frame()
	tt.Frame()
	if !under {
		t.Fatalf("under the dialog: %v", under)
	}
	open = false
	tt.Frame()
	tt.Frame()
	if under || !tt.Focused("Field") {
		t.Fatalf("after: focused %v, under %v", tt.Focused("Field"), under)
	}
}
