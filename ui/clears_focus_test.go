package ui

import "testing"

// A press on an element that ClearsFocus leaves the keyboard focus
// nowhere, though the element around it takes the focus: a control over a
// code editor, as nanoui's controls were.
func TestClearsFocusLeavesTheFocusNowhere(t *testing.T) {
	var field Ref
	field.RequestFocus()
	clicks := 0
	tt := NewTester(func(c *Context) {
		Column(c).Size(200, 100).Focusable().Label("Outer").Children(func() {
			Box(c).Size(100, 30).Focusable().Label("Field").Ref(&field)
			b := Row(c).Size(40, 30).Label("Next").ClearsFocus()
			if b.Clicked() {
				clicks++
			}
		})
	}, 200, 100)
	defer tt.rt.close()
	tt.Frame()
	if !tt.Focused("Field") {
		t.Fatal("the field is not focused")
	}
	if err := tt.Click("Next"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if clicks != 1 || tt.Focused("Outer") {
		t.Fatalf("clicks %d, field focused %v, outer focused %v", clicks, tt.Focused("Field"), tt.Focused("Outer"))
	}
}
