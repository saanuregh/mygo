package ui

import "testing"

// ScrollToOffset puts that many DIPs of a list's rows above its top.
func TestListScrollToOffset(t *testing.T) {
	var s ListState
	tt := NewTester(func(c *Context) {
		List(c, &s, 100, func(i int) { Box(c).Height(26) }).Size(200, 300)
	}, 200, 300)
	tt.Frame()
	s.ScrollToOffset(40*26 - 120)
	tt.Frame()
	tt.Frame()
	if first, _ := s.Visible(); first != 35 {
		t.Fatalf("first row shown %d, want 35 (row 40 120 below the top)", first)
	}
}
