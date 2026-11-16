package markdown

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// Theme.LinkUnderline underlines link text, and only link text.
func TestThemeLinkUnderline(t *testing.T) {
	o := &Options{Theme: &Theme{LinkUnderline: true}}
	var st *viewState
	ui.NewTester(func(c *ui.Context) {
		st = *ui.Local(View(c, "Read [the docs](https://example.com) here.", o), stateKey{}, newState)
	}, 400, 100)
	seen := map[string]bool{}
	for i := range st.paras {
		for _, pc := range st.paras[i].p.pieces {
			for _, s := range pc.spans {
				seen[s.Text] = s.Underline
			}
		}
	}
	if !seen["the docs"] || seen["Read "] || seen[" here."] {
		t.Fatalf("underlined: %v", seen)
	}
}
