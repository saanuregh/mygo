package ui

import "testing"

// OnCut sees a cut, by its key or the Cut menu item, before the text goes.
func TestOnCutSeesACut(t *testing.T) {
	text := "hello world"
	var cuts []string
	tt := NewTester(func(c *Context) {
		TextInput(c, &text).Label("Field").OnCut(func() { cuts = append(cuts, text) })
	}, 300, 100)
	defer tt.rt.close()
	tt.Frame()
	if err := tt.Click("Field"); err != nil {
		t.Fatal(err)
	}
	tt.Key(Cmd, KeyA)
	tt.Key(Cmd, KeyX)
	tt.Frame()
	if len(cuts) != 1 || cuts[0] != "hello world" || text != "" || tt.Clipboard() != "hello world" {
		t.Fatalf("cuts %q, text %q, clipboard %q", cuts, text, tt.Clipboard())
	}
	text = "again"
	tt.Frame()
	tt.Key(Cmd, KeyA)
	tt.Command("cut")
	tt.Frame()
	if len(cuts) != 2 || text != "" {
		t.Fatalf("by the command: cuts %q, text %q", cuts, text)
	}
}
