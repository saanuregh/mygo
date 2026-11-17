package terminal

import (
	"math"
	"runtime"
	"strings"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"

	"github.com/egoist/mygo/ui"
)

// TestFractionalCells: with Options.FractionalCells and fractional text a
// cell is the font's advance, fractions of a pixel included, a row the
// font's line rounded up to a DIP times the line height rounded up, a
// double click far along a row selects the word drawn there, and
// Font.CellWidth scales the fractional advance.
func TestFractionalCells(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("the hinted advances are Pango's")
	}
	loadLib(t)
	if err := ui.RegisterFont(gomono.TTF, "MyGo Test Fractional Mono"); err != nil {
		t.Fatal(err)
	}
	ui.SetFractionalText(true)
	t.Cleanup(func() { ui.SetFractionalText(false) })
	ttf, err := sfnt.Parse(gomono.TTF)
	if err != nil {
		t.Fatal(err)
	}
	const size, scale = 13, 2
	var b sfnt.Buffer
	g, _ := ttf.GlyphIndex(&b, 'M')
	adv, _ := ttf.GlyphAdvance(&b, g, fixed.Int26_6(size*64), font.HintingNone)
	advance := float64(adv) / 64
	m, _ := ttf.Metrics(&b, fixed.Int26_6(size*64), font.HintingNone)
	row := math.Ceil(math.Ceil(float64(m.Height)/64) * 1.2)

	term, err := New(Options{Conn: newPipe(), Font: Font{Family: "MyGo Test Fractional Mono", Size: size, LineHeight: 1.2}, FractionalCells: true})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	term.Feed([]byte(strings.Repeat("x", 44) + " word here"))
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 600, 200)
	tt.SetScale(scale)
	tt.Frame()
	v := term.v
	if math.Abs(float64(v.cellX)-advance*scale) > 0.02 {
		t.Errorf("a cell is %v pixels wide, want the advance's %v", v.cellX, advance*scale)
	}
	if v.cellH != int(row*scale) {
		t.Errorf("a row is %d pixels high, want %v", v.cellH, row*scale)
	}
	if want := int(float64(600*scale-2*v.ox) / (advance * scale)); v.cols != want {
		t.Errorf("%d columns, want %d", v.cols, want)
	}
	x := (float64(v.ox) + 46.5*advance*scale) / scale
	y := (float32(v.oy) + float32(v.cellH)/2) / scale
	for range 2 {
		tt.Press(float32(x), y)
		tt.Release(float32(x), y)
	}
	copyMods := ui.Ctrl | ui.Shift
	tt.Key(copyMods, ui.KeyC)
	if got := tt.Clipboard(); got != "word" {
		t.Errorf("a double click on the word copied %q", got)
	}
	// Font.CellWidth scales the advance first, and the cell keeps the
	// product's fraction.
	term.SetFont(Font{Family: "MyGo Test Fractional Mono", Size: size, LineHeight: 1.2, CellWidth: 1.25})
	tt.Frame()
	if want := advance * 1.25 * scale; math.Abs(float64(term.v.cellX)-want) > 0.02 {
		t.Errorf("a cell at CellWidth 1.25 is %v pixels wide, want %v", term.v.cellX, want)
	}
}
