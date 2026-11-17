package editor

import (
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// stickyText is a function of 60 lines, each a word on its own.
func stickyText() string {
	var b strings.Builder
	b.WriteString("func outer() {\n")
	for i := 1; i < 60; i++ {
		fmt.Fprintf(&b, "\tline%d()\n", i)
	}
	b.WriteString("}|")
	return b.String()
}

// inkInRow returns how many pixels of row in the text's area are not
// the background, as the last frame painted it.
func (ct *editorTest) inkInRow(row int) int {
	img := ct.tt.Image()
	s := float32(img.Bounds().Dx()) / 600
	lh := ct.st.ed.view.lineHeight
	n := 0
	for y := int(float32(row) * lh * s); y < int(float32(row+1)*lh*s); y++ {
		for x := int(ct.st.ed.gutterW * s); x < int(300*s); x++ {
			c := img.RGBAAt(x, y)
			if c.R < 128 {
				n++
			}
		}
	}
	return n
}

// A sticky line is drawn in the top row while the text scrolls past it,
// over the line that would show there.
func TestStickyLinesDrawAboveText(t *testing.T) {
	ct := newEditorTest(t, "", stickyText())
	ct.st.Theme, ct.st.NoActiveLine = &Theme{Background: ui.RGB(255, 255, 255), Text: ui.RGB(0, 0, 0)}, true
	ct.st.ScrollTo(20)
	ct.tt.Frame()
	plain := ct.inkInRow(0)
	ct.st.StickyLines = []int{0}
	ct.tt.Frame()
	sticky := ct.inkInRow(0)
	if sticky == plain || sticky == 0 {
		t.Fatalf("the top row has %d dark pixels with the sticky line, %d without", sticky, plain)
	}
	if first, _ := ct.st.VisibleLines(); first != 20 {
		t.Fatalf("the first line in view is %d, want 20", first)
	}
	// At most five rows are pinned.
	ct.st.StickyLines = []int{0, 1, 2, 3, 4, 5, 6}
	ct.tt.Frame()
	if n := ct.st.ed.stickyCount(); n != 5 {
		t.Fatalf("%d sticky rows, want 5", n)
	}
}

// A click on a sticky line scrolls it to the top and puts the caret at
// its start.
func TestStickyLineClickScrolls(t *testing.T) {
	ct := newEditorTest(t, "", stickyText())
	ct.st.StickyLines = []int{0, 3}
	ct.st.ScrollTo(20)
	ct.tt.Frame()
	lh := ct.st.ed.view.lineHeight
	ct.tt.ClickAt(200, lh+lh/2)
	ct.tt.Frame()
	if y, want := ct.st.ScrollY(), ct.st.LineTop(3); y != want {
		t.Fatalf("scrolled to %v, want %v (line 3's top)", y, want)
	}
	if sels := ct.st.Selections(); len(sels) != 1 || sels[0].Head != ct.st.ed.view.text.LineStart(3) {
		t.Fatalf("selections %v, want a caret at line 3's start", sels)
	}
}

// With no sticky lines the top row shows the line scrolled there, and a
// click there lands in it.
func TestNilStickyDrawsNone(t *testing.T) {
	ct := newEditorTest(t, "", stickyText())
	ct.st.ScrollTo(20)
	ct.tt.Frame()
	if n := ct.st.ed.stickyCount(); n != 0 {
		t.Fatalf("%d sticky rows, want 0", n)
	}
	lh := ct.st.ed.view.lineHeight
	ct.tt.ClickAt(200, lh/2)
	ct.tt.Frame()
	if y, want := ct.st.ScrollY(), ct.st.LineTop(20); y != want {
		t.Fatalf("scrolled to %v, want %v", y, want)
	}
	if line := ct.st.ed.view.text.LineOf(ct.st.Selections()[0].Head); line != 20 {
		t.Fatalf("the click put the caret on line %d, want 20", line)
	}
}
