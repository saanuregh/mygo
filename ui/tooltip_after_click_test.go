package ui

import (
	"testing"
	"time"
)

// With TooltipAfterClick a press hides the tip only while the pointer is
// down: it shows again as the pointer comes up and stays, and Escape still
// hides it until the pointer comes back.
func TestTooltipShowsAgainAfterAClickWithTooltipAfterClick(t *testing.T) {
	clicks := 0
	tt := coreNewTester(func(c *context) {
		th := LightTheme()
		th.TooltipAfterClick = true
		c.SetTheme(th)
		coreRow(c).Gap(40).Padding(40).Children(func() {
			if coreButton(c, "Save").Tooltip("Save the file").Clicked() {
				clicks++
			}
		})
	}, 400, 200)
	rest := func(x, y float32) {
		tt.Move(x, y)
		tt.rt.tips.hoverSince = tt.now.Add(-time.Second)
		tt.Frame()
	}
	r, _ := tt.Find("Save")
	x, y := center(r)
	rest(x, y)
	if !tt.HasText("Save the file") {
		t.Fatal("no tooltip over the button")
	}
	tt.Press(x, y)
	tt.Frame()
	if tt.HasText("Save the file") {
		t.Error("the tooltip shows while the button is pressed")
	}
	tt.Release(x, y)
	tt.Frame()
	if clicks != 1 {
		t.Fatalf("clicks %d", clicks)
	}
	rest(x, y)
	if !tt.HasText("Save the file") {
		t.Error("no tooltip after the click while the pointer stays")
	}
	tt.Key(0, KeyEscape)
	rest(x, y)
	if tt.HasText("Save the file") {
		t.Error("the tooltip shows after Escape")
	}
}

// Theme.Tooltip sets the look of Tooltip's tips; its zero fields keep
// MyGo's, which TooltipLook fills in for a tip of one's own.
func TestTooltipStyleSetsTheTipsLook(t *testing.T) {
	th := LightTheme()
	def := th.TooltipLook()
	if def.FontSize != th.FontSize-1 || def.PaddingX != th.Space(2) || def.PaddingY != th.Space(1.25) || def.Radius != th.Space(1.25) || def.Gap != th.Space(1.5) || def.Background != th.Text || def.Text != th.Background {
		t.Fatalf("MyGo's look: %+v", def)
	}
	th.Tooltip = TooltipStyle{FontSize: 11.5, PaddingX: 8, PaddingY: 4, Radius: 6, Gap: 6}
	tt := coreNewTester(func(c *context) {
		c.SetTheme(th)
		coreRow(c).Padding(60).Children(func() { coreButton(c, "Save").Tooltip("Save the file") })
	}, 400, 200)
	r, _ := tt.Find("Save")
	x, y := center(r)
	tt.Move(x, y)
	tt.rt.tips.hoverSince = tt.now.Add(-time.Second)
	tt.Frame()
	tip, ok := tt.Find("Save the file")
	if !ok {
		t.Fatal("no tooltip")
	}
	// The text's foot: the gap and the padding over the button's top (the
	// row's padding, 60).
	if gap := 60 - (tip.Y + tip.H); gap < 9.5 || gap > 10.5 {
		t.Errorf("the text %v over the button, want 6 + 4", gap)
	}
}
