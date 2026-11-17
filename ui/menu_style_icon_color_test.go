package ui

import (
	"image/color"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// TestMenuStyleIconColor draws a template icon in its menu style's
// IconColor, highlighted under a Hover fill too, while a Danger item's icon
// keeps the Danger color and an item's own IconColor wins.
func TestMenuStyleIconColor(t *testing.T) {
	tmpl := &platform.MenuIcon{PNG: menuIconPNG(t, color.NRGBA{A: 255}), Template: true}
	red, blue := RGB(220, 20, 20), RGB(20, 20, 220)
	view := func(c *context) {
		th := LightTheme()
		th.Menu = MenuStyle{IconColor: red, Hover: RGB(230, 230, 230)}
		c.SetTheme(th)
		coreColumn(c).Fill().Children(func() {
			coreText(c, "Row").PopoverMenu(func(m *Menu) {
				m.Item("Plain").Icon(testMenuIcon{tmpl})
				m.Item("Own").Icon(testMenuIcon{tmpl}).IconColor(blue)
				m.Item("Trash").Icon(testMenuIcon{tmpl}).Danger()
			})
		})
	}
	tt := coreNewTester(view, 400, 300)
	if err := tt.RightClick("Row"); err != nil {
		t.Fatal(err)
	}
	at := func(s string) (int, int, int) {
		var r Rect
		for _, n := range tt.rt.labels {
			if n.text == s {
				r = n.r
			}
		}
		x := int(r.X) - 8 - platform.MenuIconSize/2
		cr, cg, cb, _ := tt.Image().At(x, int(r.Y+r.H/2)).RGBA()
		return int(cr >> 8), int(cg >> 8), int(cb >> 8)
	}
	isRed := func(r, g, b int) bool { return r > 180 && g < 70 && b < 70 }
	if r, g, b := at("Plain"); !isRed(r, g, b) {
		t.Errorf("the icon is (%d, %d, %d), want the style's red", r, g, b)
	}
	if r, g, b := at("Own"); r > 70 || b < 180 {
		t.Errorf("the icon with its own IconColor is (%d, %d, %d), want blue", r, g, b)
	}
	danger := LightTheme().Danger.SRGB()
	near := func(v int, w uint8) bool { return v-int(w) <= 30 && int(w)-v <= 30 }
	if r, g, b := at("Trash"); !near(r, danger.R) || !near(g, danger.G) || !near(b, danger.B) {
		t.Errorf("the Danger item's icon is (%d, %d, %d), want the Danger color %v", r, g, b, danger)
	}

	// Highlighted under a Hover fill, the icon keeps the style's color.
	tt.Key(0, KeyDown)
	if r, g, b := at("Plain"); !isRed(r, g, b) {
		t.Errorf("the highlighted icon is (%d, %d, %d), want the style's red", r, g, b)
	}
}
