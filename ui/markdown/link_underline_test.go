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

// Theme.LinkCodeColor keeps a link's inline code in Code, underlined.
func TestThemeLinkCodeColor(t *testing.T) {
	code := ui.RGB(200, 100, 0)
	o := &Options{Theme: &Theme{LinkUnderline: true, LinkCodeColor: true, Code: code, Link: ui.RGB(0, 0, 255)}}
	var st *viewState
	ui.NewTester(func(c *ui.Context) {
		st = *ui.Local(View(c, "See [`Foo`](https://example.com).", o), stateKey{}, newState)
	}, 400, 100)
	found := false
	for i := range st.paras {
		for _, pc := range st.paras[i].p.pieces {
			for _, s := range pc.spans {
				if s.Text == "Foo" {
					found = true
					if s.Color != code || !s.Underline {
						t.Fatalf("code in a link: color %v, underline %v", s.Color, s.Underline)
					}
				}
			}
		}
	}
	if !found {
		t.Fatal("no Foo span")
	}
}
