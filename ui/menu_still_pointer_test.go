package ui

import "testing"

// TestAMenuUnderAStillPointerHighlightsTheItemThere opens a menu over the
// pointer, which does not move: the item under it is highlighted at once,
// as nanoui's menus do.
func TestAMenuUnderAStillPointerHighlightsTheItemThere(t *testing.T) {
	open := false
	build := func(m *Menu) {
		m.Item("One")
		m.Item("Two")
		m.Item("Three")
	}
	tt := coreNewTester(func(c *context) {
		b := coreBox(c).Size(400, 300)
		b.PopoverMenu(build)
		if open {
			open = false
			b.OpenPopoverMenu(100, 100, build)
		}
	}, 400, 300)
	tt.Move(140, 140)
	tt.Frame()
	open = true
	tt.Frame()
	tt.Frame()
	tt.Frame()
	d := tt.rt.menu.drawn
	if d == nil {
		t.Fatal("no menu")
	}
	if len(d.highlighted) == 0 {
		t.Fatalf("nothing highlighted under the pointer; items %+v", tt.PopoverMenu())
	}
}

// TestANextMenuUnderAStillPointerHighlightsTheItemThere: a NextMenu item
// clicked opens its menu where the menu was; the item under the pointer,
// which has not moved, is highlighted, as nanoui's menus did.
func TestANextMenuUnderAStillPointerHighlightsTheItemThere(t *testing.T) {
	open := false
	next := func(m *Menu) {
		for _, s := range []string{"N1", "N2", "N3", "N4", "N5", "N6", "N7", "N8"} {
			m.Item(s)
		}
	}
	build := func(m *Menu) {
		for _, s := range []string{"One", "Two", "Three", "Four", "Five"} {
			m.Item(s)
		}
		m.NextMenu("Next", next)
	}
	tt := coreNewTester(func(c *context) {
		b := coreBox(c).Size(400, 400)
		b.PopoverMenu(build)
		if open {
			open = false
			b.OpenPopoverMenu(20, 20, build)
		}
	}, 400, 400)
	tt.Frame()
	open = true
	tt.Frame()
	tt.Frame()
	tt.Frame()
	r, _ := tt.Find("Next")
	if err := tt.Click("Next"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	tt.Frame()
	d := tt.rt.menu.drawn
	if d == nil || !tt.HasText("N1") {
		t.Fatalf("no next menu: %q", tt.Texts())
	}
	under, ok := tt.Find("N6")
	if !ok || under.Y > r.Y+r.H/2 || under.Y+under.H < r.Y+r.H/2 {
		t.Fatalf("N6 at %v is not under the pointer at %v", under, r)
	}
	if len(d.highlighted) == 0 || d.highlighted[0].label != "N6" {
		t.Fatalf("highlighted %+v; items %+v", d.highlighted, tt.PopoverMenu())
	}
}

// A NextMenu laid out unlike the menu it replaces (separators above the
// item clicked, none in the new one) highlights the item under the still
// pointer in its own layout, not the row the old one had there.
func TestANextMenuOfAnotherLayoutHighlightsTheItemUnderThePointer(t *testing.T) {
	open := false
	next := func(m *Menu) {
		for _, s := range []string{"N1", "N2", "N3", "N4", "N5", "N6", "N7", "N8"} {
			m.Item(s)
		}
	}
	build := func(m *Menu) {
		m.Item("One")
		m.Separator()
		m.Separator()
		m.Separator()
		m.Separator()
		m.NextMenu("Next", next)
	}
	tt := coreNewTester(func(c *context) {
		th := LightTheme()
		th.Menu = MenuStyle{SeparatorSpace: 6}
		c.SetTheme(th)
		b := coreBox(c).Size(400, 400)
		b.PopoverMenu(build)
		if open {
			open = false
			b.OpenPopoverMenu(20, 20, build)
		}
	}, 400, 400)
	tt.Frame()
	open = true
	tt.Frame()
	tt.Frame()
	r, _ := tt.Find("Next")
	if err := tt.Click("Next"); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		tt.Frame()
	}
	y := r.Y + r.H/2
	want := ""
	for _, s := range []string{"N1", "N2", "N3", "N4", "N5", "N6", "N7", "N8"} {
		if b, ok := tt.Find(s); ok && b.Y <= y && y < b.Y+b.H {
			want = s
		}
	}
	d := tt.rt.menu.drawn
	if want == "" || d == nil || len(d.highlighted) == 0 || d.highlighted[0].label != want {
		t.Fatalf("under the pointer %q; highlighted %+v of %+v", want, d, tt.PopoverMenu())
	}
}
