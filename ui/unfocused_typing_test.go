package ui

import "testing"

// Text typed with nothing focused is told to the next frame, then
// forgotten; typed into a focused input, it is the input's.
func TestUnfocusedTyping(t *testing.T) {
	var got []string
	var nowhere bool
	value := ""
	var field Handle
	focus := false
	tt := NewTester(func(c *Context) {
		if text, typed := c.UnfocusedTyping(); typed {
			got = append(got, text)
		}
		nowhere = !c.FocusTakesText()
		TextInput(c, &value).Bind(&field)
		if focus {
			focus = false
			field.Focus(c)
		}
	}, 200, 100)
	tt.Frame()
	if !nowhere {
		t.Fatal("something has the focus")
	}
	tt.Type("é")
	tt.Frame()
	tt.Frame()
	if len(got) != 1 || got[0] != "é" || value != "" {
		t.Fatalf("got %q, the input has %q", got, value)
	}
	focus = true
	tt.Frame()
	tt.Frame()
	tt.Type("x")
	tt.Frame()
	if len(got) != 1 || value != "x" || nowhere {
		t.Fatalf("got %q, the input has %q", got, value)
	}
}
