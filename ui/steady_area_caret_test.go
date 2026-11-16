package ui_test

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// With Theme.SteadyTextAreaCaret a text area's caret stays as the blink
// would hide it and goes while text is selected; a one-line input's still
// blinks.
func TestSteadyTextAreaCaret(t *testing.T) {
	for _, area := range []bool{false, true} {
		name := "input"
		if area {
			name = "area"
		}
		t.Run(name, func(t *testing.T) {
			value := "ab"
			tt := ui.NewTester(func(c *ui.Context) {
				theme := ui.LightTheme()
				theme.Background, theme.Text = ui.RGB(245, 245, 245), ui.RGB(15, 15, 15)
				theme.Accent, theme.Selection = ui.RGB(255, 0, 220), ui.RGB(80, 150, 240)
				theme.SteadyTextAreaCaret = true
				c.SetTheme(theme)
				c.Root().FontSize(20)
				ui.Column(c).Fill().Padding(16).Children(func() {
					var target ui.Element
					if area {
						target = ui.TextAreaBase(c, &value).AutoFocus()
					} else {
						target = ui.TextInputBase(c, &value).AutoFocus()
					}
					target.Size(240, 60).Padding(8).Shrink(0).TextColor(theme.Text).CaretWidth(2).Label("caret target")
				})
			}, 320, 180)
			tt.SetScale(2)
			caretWidthWantPixels(t, tt, 4)
			off := 0
			if area {
				off = 4
			}
			tt.Advance(530 * time.Millisecond)
			caretWidthWantPixels(t, tt, off)
			tt.Advance(530 * time.Millisecond)
			caretWidthWantPixels(t, tt, 4)
			tt.Key(ui.Shift, ui.KeyLeft)
			if area {
				caretWidthWantPixels(t, tt, 0)
			}
		})
	}
}
