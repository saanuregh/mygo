package text

import (
	"testing"

	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gobolditalic"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/goregular"
)

// Go Bold's OS/2 weight class is 600.
func TestFaceCSSReadsWeightAndStyleFromTheFile(t *testing.T) {
	for _, c := range []struct {
		name          string
		data          []byte
		weight, style string
	}{
		{"regular", goregular.TTF, "400", "normal"},
		{"bold", gobold.TTF, "600", "normal"},
		{"italic", goitalic.TTF, "400", "italic"},
		{"bold italic", gobolditalic.TTF, "600", "italic"},
		{"not a font", []byte("nope"), "400", "normal"},
	} {
		w, s := faceCSS(c.data)
		if w != c.weight || s != c.style {
			t.Errorf("%s: got %s %s, want %s %s", c.name, w, s, c.weight, c.style)
		}
	}
}
