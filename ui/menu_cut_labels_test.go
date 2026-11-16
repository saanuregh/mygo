package ui

import (
	"strings"
	"testing"
)

// TestMenuCutLabels checks a label too long for a menu whose style cuts
// labels stays one line, the items' height as ever, and ends at the
// menu's edge in its letters, with no "…": the label's last inked
// columns reach as high as its letters do, which an ellipsis's dots,
// on the baseline, do not.
func TestMenuCutLabels(t *testing.T) {
	long := strings.Repeat("W", 80)
	tt := coreNewTester(func(c *context) {
		th := LightTheme()
		th.Menu = MenuStyle{Width: 200, ItemHeight: 30, CutLabels: true}
		c.SetTheme(th)
		c.SetMenuPresentation(PopoverMenus)
		coreBox(c).Size(300, 300).Label("Here").PopoverMenu(func(m *Menu) {
			m.Item(long)
			m.Item("Short")
		})
	}, 600, 400)
	if err := tt.RightClick("Here"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	a, ok := tt.Find(long)
	b, _ := tt.Find("Short")
	if !ok || a.H > 30 || b.Y-a.Y != a.H {
		t.Fatalf("long %v, short %v: want one row of 30", a, b)
	}
	img := tt.Image()
	inked := func(x, y int) bool { return img.RGBAAt(x, y).R < 160 }
	// The panel past the row, its padding and border, cuts it.
	for x := int(a.X+a.W) + 6; x < int(a.X+a.W)+40; x++ {
		for y := int(a.Y); y < int(a.Y+a.H); y++ {
			if inked(x, y) {
				t.Fatalf("the long label draws at x=%d, past the menu's edge (its row ends at %v)", x, a.X+a.W)
			}
		}
	}
	right := -1
	for x := int(a.X + a.W); x >= int(a.X) && right < 0; x-- {
		for y := int(a.Y); y < int(a.Y+a.H); y++ {
			if inked(x, y) {
				right = x
				break
			}
		}
	}
	if right < 0 {
		t.Fatal("the long label draws nothing")
	}
	top := int(a.Y + a.H)
	for x := right - 3; x <= right; x++ {
		for y := int(a.Y); y < int(a.Y+a.H); y++ {
			if inked(x, y) {
				top = min(top, y)
				break
			}
		}
	}
	if mid := int(a.Y + a.H/2); top >= mid-2 {
		t.Errorf("the long label ends in ink from y=%d, under the row's middle %d: an ellipsis, not its letters", top, mid)
	}
}
