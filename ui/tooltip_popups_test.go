package ui

import (
	"testing"
	"time"
)

// With TooltipsHiddenByPopups no tip shows while a popover is open, but
// those of elements inside it; without it they all show.
func TestTooltipsHiddenByPopups(t *testing.T) {
	for _, hidden := range []bool{false, true} {
		open := true
		tt := coreNewTester(func(c *context) {
			th := LightTheme()
			th.TooltipsHiddenByPopups = hidden
			c.SetTheme(th)
			coreRow(c).Gap(40).Padding(40).Children(func() {
				coreButton(c, "Outside").Tooltip("Outside's tip")
				menu := coreButton(c, "Menu")
				corePopover(c, menu, &open, func() { coreButton(c, "Inside").Tooltip("Inside's tip") })
			})
		}, 600, 400)
		rest := func(text string) bool {
			r, _ := tt.Find(text)
			x, y := center(r)
			tt.Move(x, y)
			tt.rt.tips.hoverSince = tt.now.Add(-time.Second)
			tt.Frame()
			tt.Frame()
			return tt.HasText(text + "'s tip")
		}
		if got := rest("Outside"); got == hidden {
			t.Errorf("hidden by popups %v: the tip outside the open popover shows %v", hidden, got)
		}
		if !rest("Inside") {
			t.Errorf("hidden by popups %v: no tip inside the popover", hidden)
		}
		open = false
		tt.Frame()
		if !rest("Outside") {
			t.Errorf("hidden by popups %v: no tip once the popover closed", hidden)
		}
	}
}

// TooltipAt puts the tip on the side asked, the theme's gap from it, and
// a tip's Border and Shadow default as before.
func TestTooltipAtASide(t *testing.T) {
	th := LightTheme()
	th.Tooltip = TooltipStyle{Gap: 6}
	if look := th.TooltipLook(); look.Shadow != RGBA(0, 0, 0, 0.2) || look.Border != Transparent {
		t.Fatalf("default shadow %v, border %v", look.Shadow, look.Border)
	}
	for _, side := range []Anchor{AnchorTop, AnchorBottom, AnchorLeft, AnchorRight} {
		tt := coreNewTester(func(c *context) {
			c.SetTheme(th)
			coreRow(c).Padding(150).Children(func() { coreButton(c, "Tool").TooltipAt("Tool's tip", side) })
		}, 500, 400)
		b, _ := tt.Find("Tool")
		x, y := center(b)
		tt.Move(x, y)
		tt.rt.tips.hoverSince = tt.now.Add(-time.Second)
		tt.Frame()
		tip, ok := tt.Find("Tool's tip")
		if !ok {
			t.Fatalf("side %v: no tip", side)
		}
		var ok2 bool
		switch side {
		case AnchorTop:
			ok2 = tip.Y+tip.H < b.Y
		case AnchorBottom:
			ok2 = tip.Y > b.Y+b.H
		case AnchorLeft:
			ok2 = tip.X+tip.W < b.X && tip.Y < y && tip.Y+tip.H > y
		case AnchorRight:
			ok2 = tip.X > b.X+b.W && tip.Y < y && tip.Y+tip.H > y
		}
		if !ok2 {
			t.Errorf("side %v: tip text %+v, button text %+v", side, tip, b)
		}
	}
}
