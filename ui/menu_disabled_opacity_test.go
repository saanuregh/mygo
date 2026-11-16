package ui

import "testing"

// TestMenuStyleDisabledOpacity checks a disabled item of a menu whose
// style has a DisabledOpacity draws its label in the Text color at that
// opacity in all, not at half of it again as a disabled element is.
func TestMenuStyleDisabledOpacity(t *testing.T) {
	a := &lookApp{style: MenuStyle{DisabledOpacity: 0.45}, disabled: true}
	tt := coreNewTester(a.view, 600, 400)
	r, _ := tt.Find("Report.pdf")
	tt.RightClickAt(r.X+10, r.Y+r.H/2)
	row, ok := tt.Find("Snooze…")
	if !ok {
		t.Fatal("no Snooze… item")
	}
	img := tt.Image()
	darkest := 255
	for y := int(row.Y); y < int(row.Y+row.H); y++ {
		for x := int(row.X); x < int(row.X+row.W); x++ {
			if v := int(img.RGBAAt(x, y).R); v < darkest {
				darkest = v
			}
		}
	}
	th := LightTheme()
	bg, fg := float32(th.Background.R), float32(th.Text.R)
	want := int(bg + 0.45*(fg-bg))
	if darkest < want-12 || darkest > want+12 {
		t.Errorf("the disabled label's darkest red is %d, want about %d (Text at 0.45)", darkest, want)
	}
}
