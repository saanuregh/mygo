package text

import (
	"bytes"
	"strconv"

	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype"
)

// faceCSS returns the CSS weight and style of a font file's face, read
// from its tables (OS/2's weight class and italic bit, or the style in its
// names): a browser told none takes every face as regular, and the last
// added of a family wins. A file that does not parse is regular.
func faceCSS(data []byte) (weight, style string) {
	weight, style = "400", "normal"
	ld, err := opentype.NewLoader(bytes.NewReader(data))
	if err != nil {
		return weight, style
	}
	a, _ := font.Describe(ld, nil)
	a.Aspect.SetDefaults()
	weight = strconv.Itoa(int(a.Aspect.Weight))
	if a.Aspect.Style == font.StyleItalic {
		style = "italic"
	}
	return weight, style
}
