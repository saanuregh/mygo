package editor

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// A text loaded with Reset after the editor's first frame leaves the
// caret where it was (the start), not past the loaded text as typing
// would.
func TestResetKeepsTheCaretAtTheStart(t *testing.T) {
	st := &State{Buffer: NewBuffer("")}
	tt := ui.NewTester(func(c *ui.Context) { View(c, st).Size(400, 200) }, 400, 200)
	tt.Frame()
	st.Buffer.Reset("first line\nsecond line\n")
	tt.Frame()
	if got := st.Selections(); len(got) != 1 || got[0].Start() != 0 || got[0].End() != 0 {
		t.Fatalf("selections after Reset %+v, want the caret at 0", got)
	}
}
