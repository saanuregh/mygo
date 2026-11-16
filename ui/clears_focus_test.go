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

// A right press on read-only text inside an element that ClearsFocus
// opens its menu and leaves the focus where it was: the field keeps its
// caret, and takes the typing once the menu closes (zode's conversation
// under its composer, as nanoui's).
func TestARightPressInClearsFocusTextKeepsTheFocus(t *testing.T) {
	var draft string
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Children(func() {
			coreColumn(c).Size(300, 100).Selectable().ClearsFocus().ContextMenu(func(m *Menu) { m.EditItems() }).Children(func() {
				coreText(c, "hello world")
			})
			coreTextInput(c, &draft).Label("Message")
		})
	}, 400, 300)
	defer tt.rt.close()
	tt.Frame()
	if err := tt.Click("Message"); err != nil {
		t.Fatal(err)
	}
	field := tt.rt.focused
	if err := tt.RightClick("hello world"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if tt.rt.focused != field {
		t.Error("the text took the focus")
	}
	tt.Key(0, KeyEscape)
	tt.Frame()
	tt.TypeKey(0, KeyA, "a")
	if draft != "a" {
		t.Errorf("typed %q after the menu", draft)
	}
}

// FocusCleared: nothing has the focus, or the text of an element that
// ClearsFocus holds it for Copy; a field is no such thing.
func TestFocusClearedLooksPastTheTextOfAClearsFocusElement(t *testing.T) {
	var draft string
	cleared := false
	tt := NewTester(func(c *Context) {
		cleared = c.FocusCleared()
		Column(c).Fill().Children(func() {
			Column(c).Size(300, 100).Selectable().ClearsFocus().Children(func() {
				Text(c, "hello world")
			})
			TextInput(c, &draft).Label("Message")
		})
	}, 400, 300)
	defer tt.rt.close()
	tt.Frame()
	if !cleared {
		t.Fatal("nothing focused is not cleared")
	}
	tt.Click("Message")
	tt.Frame()
	if cleared {
		t.Fatal("a focused field is cleared")
	}
	tt.Click("hello world")
	tt.Frame()
	if !cleared || tt.rt.focused == 0 {
		t.Fatalf("the text focused %v: cleared %v", tt.rt.focused != 0, cleared)
	}
	tt.Key(Cmd, KeyA)
	tt.Key(Cmd, KeyC)
	if tt.Clipboard() != "hello world" {
		t.Fatalf("copied %q", tt.Clipboard())
	}
}
