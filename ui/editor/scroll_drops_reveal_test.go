package editor

import (
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// Selections set and then an explicit scroll, before the next frame: the
// scroll stands, as a Vim's 40zt puts line 40 at the top; the selection's
// pending reveal is dropped.
func TestAnExplicitScrollDropsAPendingReveal(t *testing.T) {
	var b strings.Builder
	for i := range 80 {
		fmt.Fprintf(&b, "line %d\n", i)
	}
	st := &State{Buffer: NewBuffer(b.String()), Font: ui.Font{Family: "monospace", Size: 14}}
	tt := ui.NewTester(func(c *ui.Context) { View(c, st).Fill() }, 300, 200)
	tt.Frame()
	at := st.Buffer.Snapshot().LineStart(40)
	st.SetSelections(Selection{Anchor: at, Head: at})
	st.ScrollTo(40)
	tt.Frame()
	if st.ScrollY() != st.LineTop(40) {
		t.Errorf("ScrollTo: scrolled %v, line 40 at %v", st.ScrollY(), st.LineTop(40))
	}
	at = st.Buffer.Snapshot().LineStart(70)
	st.SetSelections(Selection{Anchor: at, Head: at})
	want := st.LineTop(71) - st.ViewportHeight()
	st.SetScrollY(want)
	tt.Frame()
	if st.ScrollY() != want {
		t.Errorf("SetScrollY: scrolled %v, want %v", st.ScrollY(), want)
	}
	// A selection alone is still revealed.
	at = st.Buffer.Snapshot().LineStart(5)
	st.SetSelections(Selection{Anchor: at, Head: at})
	tt.Frame()
	if top := st.LineTop(5); top < st.ScrollY() || top > st.ScrollY()+st.ViewportHeight() {
		t.Errorf("line 5 at %v not revealed: scrolled %v", top, st.ScrollY())
	}
}
