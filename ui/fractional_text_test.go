package ui

import (
	"testing"

	"golang.org/x/image/font/gofont/gomono"
)

// SetFractionalText reaches the text system of each Tester, made before
// the call or after it, as it does the windows': a run of characters of
// a monospaced font at 13 is as wide as its advance, fraction included,
// rather than hinted to whole DIPs, and turning it off hints it again.
func TestFractionalTextReachesATestersTextSystem(t *testing.T) {
	const family = "MyGo Test Fractional Tester Mono"
	if err := RegisterFont(gomono.TTF, family); err != nil {
		t.Fatal(err)
	}
	view := func(c *context) {
		coreRow(c).Children(func() {
			coreText(c, "0123456789").Font(family).FontSize(13).SingleLine().Label("digits")
		})
	}
	width := func(tt *Tester) float32 {
		tt.Frame()
		r, ok := tt.Find("digits")
		if !ok {
			t.Fatal("no text")
		}
		return r.W
	}
	before := coreNewTester(view, 400, 100)
	hinted := width(before)
	SetFractionalText(true)
	defer SetFractionalText(false)
	after := coreNewTester(view, 400, 100)
	for name, tt := range map[string]*Tester{"made before": before, "made after": after} {
		if w := width(tt); w == hinted || w == float32(int(w)) {
			t.Errorf("%s: ten digits %v wide with fractional text, %v without", name, w, hinted)
		}
	}
	SetFractionalText(false)
	if w := width(coreNewTester(view, 400, 100)); w != hinted {
		t.Errorf("off again: ten digits %v wide, want %v", w, hinted)
	}
}
