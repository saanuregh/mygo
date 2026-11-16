package ui

import "testing"

// A text area's long placeholder wraps; PlaceholderOneLine keeps it on
// one line, cut at the edge, as nanoui's multiline fields drew it.
func TestPlaceholderOneLineKeepsALongPlaceholderOnOneLine(t *testing.T) {
	s := ""
	oneLine := false
	var area *node
	tt := coreNewTester(func(c *context) {
		area = coreTextArea(c, &s).Placeholder("a placeholder far longer than the area is wide").Width(80)
		if oneLine {
			area.PlaceholderOneLine()
		}
	}, 300, 200)
	tt.Frame()
	if area.placeholderParams(80).NoWrap {
		t.Fatal("a text area's placeholder kept on one line by default")
	}
	oneLine = true
	tt.Frame()
	if !area.placeholderParams(80).NoWrap {
		t.Fatal("PlaceholderOneLine wraps it")
	}
}
