package ui

import "testing"

// TextKey's Selection is the selection as the key comes, with the keys
// before it applied in the same batch, and SetSelection moves it at once:
// a composer's Enter continues a list with the caret after the marker.
func TestTextKeySelection(t *testing.T) {
	text := ""
	var anchor, caret int
	tt := coreNewTester(func(c *context) {
		in := coreTextAreaBase(c, &text).Width(200).Label("draft")
		in.OnTextKey(func(k TextKey) bool {
			switch k.Key {
			case KeyTab:
				anchor, caret = k.Selection()
				return true
			case KeyEnter:
				k.SetText(k.Text() + "\n- ")
				n := len([]rune(k.Text()))
				k.SetSelection(n, n)
				return true
			}
			return false
		})
	}, 300, 80)
	tt.Click("draft")
	tt.Type("hello")
	tt.Key(0, KeyLeft)
	tt.Key(0, KeyLeft)
	tt.Key(Shift, KeyLeft)
	tt.Key(0, KeyTab)
	if anchor != 3 || caret != 2 {
		t.Fatalf("selection %d to %d as the key came, want 3 to 2", anchor, caret)
	}
	tt.Key(0, KeyEnter)
	tt.Frame()
	tt.Type("x")
	tt.Frame()
	if text != "hello\n- x" {
		t.Fatalf("text %q, want the caret after the marker SetSelection put it at", text)
	}
}
