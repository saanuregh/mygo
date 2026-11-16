package ui

import (
	"strings"
	"testing"
	"time"
)

// TooltipStyle.OneLine keeps a long tip on one line, moved into the window
// 6 DIPs from its edge (nanoui's kit.tooltip); without it the tip wraps.
func TestTooltipOneLineKeepsALongTipOnOneLine(t *testing.T) {
	long := strings.Repeat("words ", 30) + "end"
	for _, oneLine := range []bool{false, true} {
		th := LightTheme()
		th.Tooltip = TooltipStyle{Gap: 6, OneLine: oneLine}
		tt := coreNewTester(func(c *context) {
			c.SetTheme(th)
			coreRow(c).Padding(60, 10).Children(func() { coreButton(c, "Save").Tooltip(long) })
		}, 1400, 200)
		r, _ := tt.Find("Save")
		x, y := center(r)
		tt.Move(x, y)
		tt.rt.tips.hoverSince = tt.now.Add(-time.Second)
		tt.Frame()
		tip, ok := tt.Find(long)
		if !ok {
			t.Fatal("no tooltip")
		}
		lines := tip.H / (th.FontSize * 1.2)
		if oneLine && (lines > 1.5 || tip.X < 5.5) {
			t.Errorf("one line: the tip is %v", tip)
		}
		if !oneLine && lines < 1.5 {
			t.Errorf("wrapping: the tip is %v, one line", tip)
		}
	}
}
