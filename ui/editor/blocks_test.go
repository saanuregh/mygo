package editor

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// Blocks take room between their lines, show what RenderBlock builds
// there, and the carets pass over them.
func TestBlocksBetweenLines(t *testing.T) {
	ct := newEditorTest(t, "", "o|ne\ntwo\nthree")
	ct.st.Blocks = []Block{{Line: 1, Height: 40, Key: "deleted"}, {Line: 1, Below: true, Height: 20}}
	ct.st.RenderBlock = func(c *ui.Context, box ui.Element, b Block) {
		box.Children(func() {
			if b.Key == "deleted" {
				ui.Text(c, "a deleted line")
			} else {
				ui.Text(c, "a note")
			}
		})
	}
	ct.tt.Frame()
	ct.tt.Frame()
	lh := float64(ct.st.ed.view.lineHeight)
	// LineTop is the top of the line's blocks above it.
	if got := ct.st.LineTop(1); got != lh {
		t.Fatalf("line 1's top %v, want %v", got, lh)
	}
	if got := ct.st.LineTop(2); got != 2*lh+60 {
		t.Fatalf("line 2's top %v, want %v", got, 2*lh+60)
	}
	r, ok := ct.tt.Find("a deleted line")
	if !ok {
		t.Fatalf("the block's text is not shown: %q", ct.tt.Texts())
	}
	if float64(r.Y) < lh || float64(r.Y) >= lh+40 {
		t.Fatalf("the block's text at %v, want between %v and %v", r.Y, lh, lh+40)
	}
	if !ct.tt.HasText("a note") {
		t.Fatal("the block below line 1 is not shown")
	}
	ct.key(0, ui.KeyDown)
	ct.want("one\nt|wo\nthree")
	ct.key(0, ui.KeyDown)
	ct.want("one\ntwo\nt|hree")
	// A press on a block is the block's: the caret stays.
	ct.tt.ClickAt(200, float32(lh+20))
	ct.tt.Frame()
	ct.want("one\ntwo\nt|hree")
}

// A fold hiding a line hides its blocks; unfolded, they show again.
func TestBlocksOfFoldedLines(t *testing.T) {
	ct := newEditorTest(t, "go", "func f() {\n\tx := 1\n\ty := 2\n}\n")
	ct.st.Blocks = []Block{{Line: 2, Height: 30, Key: 1}}
	ct.st.RenderBlock = func(c *ui.Context, box ui.Element, b Block) {
		box.Children(func() { ui.Text(c, "block text") })
	}
	ct.tt.Frame()
	ct.tt.Frame()
	if !ct.tt.HasText("block text") {
		t.Fatal("the block is not shown")
	}
	ct.st.Fold(0)
	ct.tt.Frame()
	ct.tt.Frame()
	if ct.tt.HasText("block text") {
		t.Fatal("the block of a folded line is shown")
	}
	lh := float64(ct.st.ed.view.lineHeight)
	if got := ct.st.LineTop(4); got != 2*lh {
		t.Fatalf("after the fold, line 4's top %v, want %v", got, 2*lh)
	}
}

// Blocks set again, as the text changes, place themselves anew.
func TestBlocksChange(t *testing.T) {
	ct := newEditorTest(t, "", strings.Repeat("line\n", 5))
	ct.st.Blocks = []Block{{Line: 3, Height: 25}}
	ct.tt.Frame()
	lh := float64(ct.st.ed.view.lineHeight)
	if got := ct.st.LineTop(4); got != 4*lh+25 {
		t.Fatalf("line 4's top %v, want %v", got, 4*lh+25)
	}
	ct.st.Blocks = []Block{{Line: 3, Height: 50}}
	ct.tt.Frame()
	if got := ct.st.LineTop(4); got != 4*lh+50 {
		t.Fatalf("with the block higher, line 4's top %v, want %v", got, 4*lh+50)
	}
	ct.st.Blocks = nil
	ct.tt.Frame()
	if got := ct.st.LineTop(4); got != 4*lh {
		t.Fatalf("without blocks, line 4's top %v, want %v", got, 4*lh)
	}
}
