package editor

import (
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// OnInput sees keys and typed text before the editor: what it takes the
// editor leaves, the rest goes on as without it.
func TestOnInputTakesKeys(t *testing.T) {
	ct := newEditorTest(t, "", "ab|")
	var texts, keys int
	ct.st.OnInput = func(ev ui.InputEvent) bool {
		switch ev.Kind {
		case ui.InputText:
			texts++
			return ev.Text == "x"
		case ui.InputKeyDown:
			keys++
			return ev.Key == ui.KeyBackspace
		}
		return false
	}
	ct.tt.Frame()
	ct.typ("x")
	ct.key(0, ui.KeyBackspace)
	ct.want("ab|")
	ct.typ("y")
	ct.key(0, ui.KeyLeft)
	ct.want("ab|y")
	if texts != 2 || keys != 2 {
		t.Fatalf("OnInput saw %d texts and %d keys, want 2 and 2", texts, keys)
	}
}

// VisibleLines are the lines the last frame showed, scrolled or not.
func TestVisibleLines(t *testing.T) {
	ct := newEditorTest(t, "", strings.Repeat("line\n", 200))
	lh := float64(ct.st.ed.view.lineHeight)
	first, last := ct.st.VisibleLines()
	if want := int(400 / lh); first != 0 || last != want {
		t.Fatalf("visible lines %d to %d, want 0 to %d", first, last, want)
	}
	ct.st.ScrollTo(50)
	ct.tt.Frame()
	first, last = ct.st.VisibleLines()
	if want := 50 + int(400/lh); first != 50 || last != want {
		t.Fatalf("scrolled, visible lines %d to %d, want 50 to %d", first, last, want)
	}
}

// pixel returns the color of the last frame at x, y in DIPs of an
// editor test's window, 600 wide.
func (ct *editorTest) pixel(x, y float32) (r, g, b uint8) {
	img := ct.tt.Image()
	s := float32(img.Bounds().Dx()) / 600
	c := img.RGBAAt(int(x*s), int(y*s))
	return c.R, c.G, c.B
}

// redTheme paints carets red on white, for tests to find them.
var redTheme = &Theme{Background: ui.RGB(255, 255, 255), Cursor: ui.RGB(255, 0, 0)}

// A block caret covers the character after it, or a space's width at the
// end of a line, in the color of carets made translucent.
func TestBlockCaret(t *testing.T) {
	ct := newEditorTest(t, "", "|abc\nab")
	ct.st.Theme, ct.st.NoActiveLine = redTheme, true
	ct.tt.Frame()
	ed := &ct.st.ed
	cw, lh := ed.view.charWidth, ed.view.lineHeight
	inBlock := func(line, col int) bool {
		r, g, _ := ct.pixel(ed.textX+(float32(col)+0.6)*cw, float32(line)*lh+2)
		return r > 200 && g < 230
	}
	if inBlock(0, 0) {
		t.Fatal("a bar caret painted a block")
	}
	ct.st.BlockCaret = true
	ct.tt.Frame()
	if !inBlock(0, 0) || inBlock(0, 1) {
		t.Fatal("the block caret did not cover the first character alone")
	}
	if r, g, b := ct.pixel(ed.textX+0.6*cw, 2); r < 250 || g < 60 || g != b {
		t.Fatalf("the block caret is %d,%d,%d, not translucent red", r, g, b)
	}
	// At the end of a line, a space's width.
	ct.st.Select(7, 7)
	ct.tt.Frame()
	if !inBlock(1, 2) || inBlock(1, 3) || inBlock(0, 0) {
		t.Fatal("the block caret at the end of a line is not a space wide")
	}
}

// Line backgrounds fill the rows of their lines across the text, not the
// gutter: those a fold hides take no rows, and a wrapped line's every row.
func TestLineBackgrounds(t *testing.T) {
	ct := newEditorTest(t, "go", "func a() {\n\tx()\n\ty()\n}\nfunc b() {}\nz|")
	blue := ui.RGB(0, 0, 255)
	ct.st.Theme, ct.st.NoActiveLine = redTheme, true
	ct.st.LineBackgrounds = []LineBackground{{First: 0, Last: 4, Color: blue}}
	ct.tt.Frame()
	ed := &ct.st.ed
	lh := ed.view.lineHeight
	filled := func(x float32, row int) bool {
		r, _, b := ct.pixel(x, float32(row)*lh+2)
		return b > 200 && r < 60
	}
	for row, want := range []bool{true, true, true, true, true, false} {
		if filled(500, row) != want {
			t.Fatalf("row %d filled %v, want %v", row, !want, want)
		}
	}
	if filled(2, 0) {
		t.Fatal("the line background filled the gutter")
	}
	// Folded, line 0 hides lines 1 and 2.
	ct.st.Fold(0)
	ct.tt.Frame()
	for row, want := range []bool{true, true, true, false} {
		if filled(500, row) != want {
			t.Fatalf("folded, row %d filled %v, want %v", row, !want, want)
		}
	}
	// Wrapped, the line fills its rows.
	ct = newEditorTest(t, "", strings.Repeat("x", 150)+"\nz|")
	ct.st.Theme, ct.st.NoActiveLine, ct.st.SoftWrap = redTheme, true, true
	ct.st.LineBackgrounds = []LineBackground{{First: 0, Last: 0, Color: blue}}
	ct.tt.Frame()
	ct.tt.Frame()
	ed = &ct.st.ed
	if rows := len(ed.view.line(0).rows); rows != 3 {
		t.Fatalf("the line wrapped into %d rows, want 3", rows)
	}
	for row, want := range []bool{true, true, true, false} {
		if filled(300, row) != want {
			t.Fatalf("wrapped, row %d filled %v, want %v", row, !want, want)
		}
	}
}

// Gutter buttons show on the lines in view that no fold hides, built by
// the app, and take their clicks from the gutter; the gutter keeps room
// for them while the list is set, empty or not.
func TestGutterButtons(t *testing.T) {
	ct := newEditorTest(t, "go", "|func a() {\n\tx()\n\ty()\n}\n"+strings.Repeat("z\n", 100))
	ed := &ct.st.ed
	plain := ed.gutterW
	ct.st.GutterButtons = []int{}
	ct.tt.Frame()
	empty := ed.gutterW
	if empty <= plain {
		t.Fatalf("gutter %v wide with an empty list of buttons, %v without", empty, plain)
	}
	var built, clicked []int
	ct.st.RenderGutterButton = func(c *ui.Context, button ui.Element, line int) {
		built = append(built, line)
		button.Label(fmt.Sprint("Run ", line))
		if button.Clicked() {
			clicked = append(clicked, line)
		}
	}
	ct.st.GutterButtons = []int{0, 1, 4, 90}
	ct.st.Fold(0)
	ct.tt.Frame()
	if ed.gutterW != empty {
		t.Fatalf("gutter %v wide with buttons, %v with none", ed.gutterW, empty)
	}
	built = nil
	ct.tt.Frame()
	if fmt.Sprint(built) != "[0 4]" {
		t.Fatalf("built buttons of lines %v, want [0 4]", built)
	}
	// Line 4 shows third, below line 0 and the fold's closing line.
	lh := ed.view.lineHeight
	r, ok := ct.tt.Find("Run 4")
	if !ok || r.X != 0 || r.Y != 2*lh || r.H != lh || r.W <= 0 || r.W >= empty {
		t.Fatalf("button of line 4 at %+v (found %v), rows %v high", r, ok, lh)
	}
	if err := ct.tt.Click("Run 4"); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(clicked) != "[4]" {
		t.Fatalf("clicked %v, want [4]", clicked)
	}
	ct.want("|func a() {\n\tx()\n\ty()\n}\n" + strings.Repeat("z\n", 100))
}
