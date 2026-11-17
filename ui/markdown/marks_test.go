package markdown

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// markedSpans returns the backgrounds of the spans of the view's
// paragraphs, by their text.
func markedSpans(st *viewState) map[string]ui.Color {
	out := map[string]ui.Color{}
	for i := range st.paras {
		for _, pc := range st.paras[i].p.pieces {
			for _, s := range pc.spans {
				out[s.Text] = s.Background
			}
		}
	}
	return out
}

// Options.Marks puts Theme.Mark under the source bytes it names, cutting
// the text's spans at its ends, and follows when the marks change.
func TestMarksPaintSourceRange(t *testing.T) {
	mark := ui.RGB(250, 200, 0)
	src := "Say hello there."
	o := &Options{Theme: &Theme{Mark: mark}, Marks: []Mark{{Start: 4, End: 9}}}
	var st *viewState
	tr := ui.NewTester(func(c *ui.Context) {
		st = *ui.Local(View(c, src, o), stateKey{}, newState)
	}, 400, 100)
	got := markedSpans(st)
	if got["hello"] != mark || got["Say "] == mark || got[" there."] == mark {
		t.Fatalf("spans: %v", got)
	}
	o.Marks = []Mark{{Start: 10, End: 15}}
	tr.Frame()
	got = markedSpans(st)
	if got["there"] != mark || got["hello"] == mark {
		t.Fatalf("after the marks moved: %v", got)
	}
	o.Marks = nil
	tr.Frame()
	got = markedSpans(st)
	if len(got) != 1 || got[src] != (ui.Color{}) {
		t.Fatalf("without marks: %v", got)
	}
}

// A mark inside a code span marks the code's own bytes, keeping the
// code's look around them.
func TestMarkInsideCodeSpan(t *testing.T) {
	mark := ui.RGB(250, 200, 0)
	src := "Run `go test` now."
	th := &Theme{Mark: mark}
	o := &Options{Theme: th, Marks: []Mark{{Start: 8, End: 12}}}
	var st *viewState
	ui.NewTester(func(c *ui.Context) {
		st = *ui.Local(View(c, src, o), stateKey{}, newState)
	}, 400, 100)
	got := markedSpans(st)
	if got["test"] != mark {
		t.Fatalf("spans: %v", got)
	}
	if bg, ok := got["go "]; !ok || bg == mark || bg.A == 0 {
		t.Fatalf("the code before the mark: %v", got)
	}
}

// The current mark takes Theme.MarkCurrent, the others Theme.Mark.
func TestCurrentMarkColour(t *testing.T) {
	mark, current := ui.RGB(250, 200, 0), ui.RGB(0, 100, 250)
	o := &Options{Theme: &Theme{Mark: mark, MarkCurrent: current}, Marks: []Mark{{Start: 0, End: 3}, {Start: 8, End: 11, Current: true}}}
	var st *viewState
	ui.NewTester(func(c *ui.Context) {
		st = *ui.Local(View(c, "one two one", o), stateKey{}, newState)
	}, 400, 100)
	var bgs []ui.Color
	for i := range st.paras {
		for _, pc := range st.paras[i].p.pieces {
			for _, s := range pc.spans {
				if s.Text == "one" {
					bgs = append(bgs, s.Background)
				}
			}
		}
	}
	if len(bgs) != 2 || bgs[0] != mark || bgs[1] != current {
		t.Fatalf("backgrounds of the marks: %v ", bgs)
	}
	if DefaultTheme(ui.LightTheme()).MarkCurrent.A == 0 {
		t.Fatal("no default MarkCurrent")
	}
}
