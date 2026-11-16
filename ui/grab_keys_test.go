package ui

import "testing"

// GrabKeys sees every key before the focus and the shortcuts, swallows the
// text of a key it took, and leaves the pointer alone.
func TestGrabKeysTakesKeysFirstNotClicks(t *testing.T) {
	grab := true
	var grabbed []Key
	var draft string
	clicks, shortcuts := 0, 0
	tt := NewTester(func(c *Context) {
		if grab {
			c.GrabKeys(func(ev InputEvent) bool {
				grabbed = append(grabbed, ev.Key)
				grab = false
				return true
			})
		}
		c.OnShortcut(Ctrl, KeyK, func() { shortcuts++ })
		Column(c).Children(func() {
			TextInput(c, &draft).Label("Field")
			Button(c, "Press").OnClick(func() { clicks++ })
		})
	}, 300, 200)
	defer tt.rt.close()
	if err := tt.Click("Field"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Press"); err != nil {
		t.Fatal(err)
	}
	if clicks != 1 {
		t.Fatalf("the click under a grab went nowhere: %d", clicks)
	}
	tt.Click("Field")
	tt.TypeKey(0, KeyV, "v")
	if len(grabbed) != 1 || grabbed[0] != KeyV || draft != "" {
		t.Fatalf("grabbed %v, the field typed %q", grabbed, draft)
	}
	tt.TypeKey(0, KeyW, "w")
	tt.Key(Ctrl, KeyK)
	if draft != "w" || shortcuts != 1 || len(grabbed) != 1 {
		t.Fatalf("after the grab: field %q, shortcuts %d, grabbed %v", draft, shortcuts, grabbed)
	}
}

// A key pressed before the one GrabKeys took comes up types its own text.
func TestGrabKeysLeavesTheNextKeysText(t *testing.T) {
	grab := true
	var draft string
	tt := NewTester(func(c *Context) {
		if grab {
			c.GrabKeys(func(ev InputEvent) bool {
				grab = false
				return true
			})
		}
		TextInput(c, &draft).Label("Field")
	}, 300, 200)
	defer tt.rt.close()
	if err := tt.Click("Field"); err != nil {
		t.Fatal(err)
	}
	tt.KeyDown(KeyV)
	tt.Type("v")
	tt.KeyDown(KeyW)
	tt.Type("w")
	tt.KeyUp(KeyV)
	tt.KeyUp(KeyW)
	if draft != "w" {
		t.Fatalf("the field typed %q, want the second key's w", draft)
	}
}
