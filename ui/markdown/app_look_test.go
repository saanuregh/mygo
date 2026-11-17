package markdown

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// Room beside code spans (Theme.CodeRoom) is objects in the text that
// copy as nothing; a table of equal columns (Theme.TableEqualColumns) fills the
// view, its cells as wide as each other.
func TestCodeRoomAndEqualColumns(t *testing.T) {
	src := "Call `Valid` now.\n\n| Package | Before | After |\n|---|---|---|\n| src/auth | FAIL | ok |\n"
	plain := ui.NewTester(func(c *ui.Context) { View(c, src, nil) }, 600, 400)
	o := &Options{Theme: &Theme{CodeRoom: 4, TableEqualColumns: true}}
	tt := ui.NewTester(func(c *ui.Context) { View(c, src, o) }, 600, 400)
	var para ui.Rect
	label := ""
	for _, s := range tt.Texts() {
		if strings.HasPrefix(s, "Call") {
			para, _ = tt.Find(s)
			label = s
		}
	}
	if strings.Count(label, "\uFFFC") != 2 || !plain.HasText("Call Valid now.") {
		t.Fatalf("no room beside the code: %q", label)
	}
	tt.ClickAt(para.X+2, para.Y+para.H/2)
	tt.Command("selectAll")
	tt.Command("copy")
	if got := tt.Clipboard(); !strings.HasPrefix(got, "Call Valid now.\n") {
		t.Errorf("copied %q", got)
	}
	a, _ := tt.Find("Package")
	b, _ := tt.Find("Before")
	c, _ := tt.Find("After")
	if d1, d2 := b.X-a.X, c.X-b.X; d1 < 150 || d1-d2 > 1 || d2-d1 > 1 {
		t.Errorf("columns at %v, %v, %v", a.X, b.X, c.X)
	}
}

// The sizes and gaps an app's theme sets: code blocks' code
// (CodeBlockSize), a quote's gap (QuoteGap), a heading's line height
// (HeadingStyle.LineHeight) and a table's text (TableSize).
func TestAppLookSizes(t *testing.T) {
	src := "```\nabcdefgh\n```\n\n> quoted\n\n# Title\n\n| Wide header |\n|---|\n| cell |\n"
	find := func(th *Theme, s string) ui.Rect {
		t.Helper()
		tt := ui.NewTester(func(c *ui.Context) { View(c, src, &Options{Theme: th}) }, 600, 600)
		r, ok := tt.Find(s)
		if !ok {
			t.Fatalf("no %q", s)
		}
		return r
	}
	if a, b := find(nil, "abcdefgh"), find(&Theme{CodeBlockSize: 0.5}, "abcdefgh"); b.W > a.W*0.7 {
		t.Errorf("code block's code %v wide at 0.5 em, %v by default", b.W, a.W)
	}
	if a, b := find(nil, "quoted"), find(&Theme{QuoteGap: 40}, "quoted"); b.X-a.X < 20 {
		t.Errorf("quote's text at %v with a gap of 40, %v by default", b.X, a.X)
	}
	tall := &Theme{}
	tall.Headings[0].LineHeight = 3
	if a, b := find(nil, "Title"), find(tall, "Title"); b.H < a.H*1.5 {
		t.Errorf("heading %v high at line height 3, %v by default", b.H, a.H)
	}
	if a, b := find(&Theme{TableEqualColumns: true}, "Wide header"), find(&Theme{TableEqualColumns: true, TableSize: 0.5}, "Wide header"); b.H > a.H*0.7 {
		t.Errorf("table text %v high at 0.5 em, %v at 1", b.H, a.H)
	}
}

// CodeKeepsColor keeps a code span in a quote in the Code color, which
// otherwise takes the quote's.
func TestCodeKeepsColor(t *testing.T) {
	src := "> see `code` here\n"
	red := ui.RGB(255, 0, 0)
	reddish := func(th *Theme) bool {
		tt := ui.NewTester(func(c *ui.Context) { View(c, src, &Options{Theme: th}) }, 400, 200)
		var r ui.Rect
		for _, s := range tt.Texts() {
			if strings.Contains(s, "code") {
				r, _ = tt.Find(s)
			}
		}
		img := tt.Image()
		for y := int(r.Y); y < int(r.Y+r.H); y++ {
			for x := int(r.X); x < int(r.X+r.W); x++ {
				c := img.RGBAAt(x, y)
				if c.R > 180 && c.G < 80 && c.B < 80 {
					return true
				}
			}
		}
		return false
	}
	if reddish(&Theme{Code: red, QuoteText: ui.RGB(0, 0, 255)}) {
		t.Error("code in a quote kept the Code color without CodeKeepsColor")
	}
	if !reddish(&Theme{Code: red, QuoteText: ui.RGB(0, 0, 255), CodeKeepsColor: true}) {
		t.Error("code in a quote lost the Code color with CodeKeepsColor")
	}
}
