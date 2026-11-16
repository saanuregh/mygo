package ui

import "testing"

// A PopoverMenuButton with OnRelease opens its menu as the click ends,
// not as the press starts, and takes no focus.
func TestPopoverMenuOnClickOpensOnTheRelease(t *testing.T) {
	chosen := false
	tt := NewTester(func(c *Context) {
		Column(c).Padding(20).Children(func() {
			b := Box(c).Size(60, 24).Label("More")
			b.PopoverMenuButton(func(m *Menu) {
				if m.Item("Rename").Chosen() {
					chosen = true
				}
			}, PopoverMenuButtonOptions{OnRelease: true})
		})
	}, 300, 200)
	defer tt.rt.close()
	tt.Frame()
	r, _ := tt.Find("More")
	x, y := r.X+r.W/2, r.Y+r.H/2
	tt.Press(x, y)
	tt.Frame()
	if tt.PopoverMenu() != nil {
		t.Fatal("the menu opened on the press")
	}
	tt.Release(x, y)
	tt.Frame()
	if tt.PopoverMenu() == nil {
		t.Fatalf("no menu after the click: %q", tt.Texts())
	}
	if tt.Focused("More") {
		t.Fatal("the button took the focus")
	}
	if err := tt.Click("Rename"); err != nil {
		t.Fatal(err)
	}
	if !chosen {
		t.Fatal("the item chosen did not report it")
	}
}
