package ui

import "testing"

// Theme.InputNoHover: a text input's border stays as it is under the
// pointer, as nanoui's text field's.
func TestInputNoHoverKeepsTheBorder(t *testing.T) {
	for _, noHover := range []bool{false, true} {
		var text string
		tt := NewTester(func(c *Context) {
			th := DarkTheme()
			th.InputNoHover = noHover
			c.SetTheme(th)
			Column(c).Padding(20).Children(func() {
				TextInput(c, &text).Label("Field").Width(120)
			})
		}, 200, 100)
		tt.Frame()
		r, _ := tt.Find("Field")
		x, y := int(r.X+r.W/2), int(r.Y)
		before := tt.Image().RGBAAt(x, y)
		tt.Move(r.X+r.W/2, r.Y+r.H/2)
		tt.Frame()
		after := tt.Image().RGBAAt(x, y)
		if (before == after) != noHover {
			t.Errorf("InputNoHover %v: border %v, under the pointer %v", noHover, before, after)
		}
		tt.rt.close()
	}
}
