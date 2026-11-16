package ui

import "testing"

// FocusNextTextField goes from field to field, past what takes no text,
// both ways and round; a field alone keeps the focus (nanoui's focus
// cycle over its text fields).
func TestFocusNextTextFieldCyclesTheTextFields(t *testing.T) {
	var one, two string
	var c0 *Context
	tt := NewTester(func(c *Context) {
		c0 = c
		Column(c).Fill().Children(func() {
			TextInput(c, &one).Label("One")
			Box(c).Size(40, 20).Focusable().Label("Box")
			TextInput(c, &two).Label("Two")
		})
	}, 300, 200)
	defer tt.rt.close()
	tt.Frame()
	tt.Click("One")
	tt.Frame()
	for _, step := range []struct {
		back bool
		want string
	}{{false, "Two"}, {false, "One"}, {true, "Two"}, {true, "One"}} {
		moved := c0.FocusNextTextField(step.back)
		tt.Frame()
		if !moved || !tt.Focused(step.want) {
			t.Fatalf("back %v: moved %v, %q not focused", step.back, moved, step.want)
		}
	}
}

func TestFocusNextTextFieldKeepsAFieldAlone(t *testing.T) {
	var one string
	var c0 *Context
	tt := NewTester(func(c *Context) {
		c0 = c
		Column(c).Fill().Children(func() {
			TextInput(c, &one).Label("One")
			Box(c).Size(40, 20).Focusable().Label("Box")
		})
	}, 300, 200)
	defer tt.rt.close()
	tt.Frame()
	tt.Click("One")
	tt.Frame()
	if c0.FocusNextTextField(false) || c0.FocusNextTextField(true) {
		t.Fatal("moved from the only field")
	}
	tt.Frame()
	if !tt.Focused("One") {
		t.Fatal("the field lost the focus")
	}
}
