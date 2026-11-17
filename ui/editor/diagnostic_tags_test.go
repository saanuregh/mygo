package editor

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// A Faded diagnostic's text is lighter than the rest, and a faded hint is
// not underlined; a Struck one has a line through its middle.
func TestDiagnosticFadedAndStruck(t *testing.T) {
	ct := newEditorTest(t, "", "mmmm mmmm mmmm\n|")
	ct.st.Theme = &Theme{Background: ui.RGB(255, 255, 255), Text: ui.RGB(0, 0, 0), Hint: ui.RGB(0, 0, 255)}
	ct.st.StraightUnderlines, ct.st.NoActiveLine = true, true
	ct.st.SetDiagnostics(ct.st.Buffer.Version(), []Diagnostic{
		{Start: 0, End: 4, Severity: SeverityHint, Message: "unused", Faded: true},
		{Start: 10, End: 14, Severity: SeverityHint, Message: "deprecated", Struck: true},
	})
	ct.tt.Frame()
	ed := &ct.st.ed
	cw, lh := ed.view.charWidth, ed.view.lineHeight
	top := float32(-ed.scrollY)
	// The darkest pixel over a word's glyphs (between its top and bottom quarters).
	darkest := func(word int) int {
		low := 255
		for x := ed.textX + float32(word*5)*cw; x < ed.textX+float32(word*5+4)*cw; x++ {
			for y := top + lh*0.25; y < top+lh*0.75; y++ {
				r, _, _ := ct.pixel(x, y)
				low = min(low, int(r))
			}
		}
		return low
	}
	if faded, plain := darkest(0), darkest(1); faded < plain+60 {
		t.Fatalf("faded text not lighter: darkest %d, plain %d", faded, plain)
	}
	for x := ed.textX + 1; x < ed.textX+4*cw; x++ {
		for dy := float32(1); dy < 6; dy++ {
			if r, g, b := ct.pixel(x, top+lh-dy); b > 150 && r < 100 && g < 100 {
				t.Fatalf("a faded hint is underlined at %v", x)
			}
		}
	}
	// Struck: a row of dark pixels across the third word, none across the second's gaps.
	across := func(word int) bool {
		for y := top + lh/2 - 2; y <= top+lh/2+2; y++ {
			dark := 0
			for x := ed.textX + float32(word*5)*cw; x < ed.textX+float32(word*5+4)*cw; x++ {
				if r, _, _ := ct.pixel(x, y); r < 128 {
					dark++
				}
			}
			if float32(dark) >= 4*cw-2 {
				return true
			}
		}
		return false
	}
	if !across(2) || across(1) {
		t.Fatalf("struck: third word %v, second %v", across(2), across(1))
	}
}
