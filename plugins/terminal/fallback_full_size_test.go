package terminal

import (
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
	"golang.org/x/image/font/gofont/goregular"
)

// TestFallbackFullSize: with Options.FallbackFullSize a glyph wider than
// its cell keeps its font's size, drawn from the cell's start, and pushes
// the row's text after it right, as a line of text would; without it the
// glyph shrinks into its cell and the next one keeps its own cell.
func TestFallbackFullSize(t *testing.T) {
	loadLib(t)
	if err := ui.RegisterFont(goregular.TTF, "MyGo Full Size Sans"); err != nil {
		t.Fatal(err)
	}
	// glyphs returns the row's glyphs of Ǳ and x, and the cells' width in
	// DIPs. Go Regular draws Ǳ wider than any of its ASCII glyphs, as a
	// fallback font draws ✘ wider than a monospaced font's cells.
	glyphs := func(full bool) (wide, next ui.Glyph, cw float32) {
		term, err := New(Options{Conn: newPipe(), Font: Font{Family: "MyGo Full Size Sans", Size: 20}, NoBlink: true, FallbackFullSize: full})
		if err != nil {
			t.Fatal(err)
		}
		defer term.Close()
		tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill() }, 600, 200)
		term.Feed([]byte("Ǳx"))
		tt.Frame()
		term.mu.Lock()
		defer term.mu.Unlock()
		v := term.v
		var got []ui.Glyph
		for _, r := range v.lines[0].runs {
			got = append(got, r.glyphs...)
		}
		if len(got) != 2 {
			t.Fatalf("full size %v: %d glyphs in the row, want 2", full, len(got))
		}
		return got[0], got[1], v.cellX / v.scale
	}
	shrunkWide, shrunkNext, cw := glyphs(false)
	wide, next, _ := glyphs(true)
	if wide.Advance <= cw || shrunkWide.Advance >= wide.Advance {
		t.Fatalf("Ǳ is %v wide at full size and %v shrunk, in cells %v wide", wide.Advance, shrunkWide.Advance, cw)
	}
	if math.Abs(float64(wide.X)) > 0.01 {
		t.Errorf("at full size Ǳ is drawn at %v, not its cell's start", wide.X)
	}
	if shrunkNext.X >= cw+cw/2 {
		t.Errorf("without FallbackFullSize, x is drawn at %v, past its cell at %v", shrunkNext.X, cw)
	}
	// x goes from its cell's start, moved by what Ǳ takes past its own:
	// right after Ǳ.
	if want := wide.X + wide.Advance; math.Abs(float64(next.X-want)) > 0.01 {
		t.Errorf("at full size x is drawn at %v, want %v (after Ǳ)", next.X, want)
	}
}
