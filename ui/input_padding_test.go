package ui

import "testing"

// Theme.InputPadding is the room inside a text input's border; zero keeps
// Space(1.5) down and Space(2.5) across.
func TestInputPadding(t *testing.T) {
	for _, c := range []struct {
		set          [2]float32
		wantY, wantX float32
	}{{[2]float32{}, 6, 10}, {[2]float32{6, 9}, 6, 9}} {
		var in *node
		value := ""
		tt := coreNewTester(func(ctx *context) {
			th := LightTheme()
			th.Spacing = 4
			th.InputPadding = c.set
			ctx.SetTheme(th)
			in = coreTextInput(ctx, &value)
		}, 300, 100)
		tt.Frame()
		if in.pad[0] != c.wantY || in.pad[1] != c.wantX || in.pad[3] != c.wantX {
			t.Errorf("InputPadding %v: padding %v", c.set, in.pad)
		}
	}
}
