package editor

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/internal/diff"
	"github.com/egoist/mygo/internal/rope"
	"github.com/egoist/mygo/internal/wait"
	"github.com/egoist/mygo/ui"
)

// E8-7: with DiffPlain a line added among equal blank lines is marked as
// diff.zig's Myers finds it, not where the edit was.
func TestDiffPlainMarksWhatTheWholeCompareFinds(t *testing.T) {
	for _, plain := range []bool{false, true} {
		ct := newEditorTest(t, "", "a\n\nb\n")
		ct.st.DiffBase = NewBuffer("a\n\nb\n")
		ct.st.DiffPlain = plain
		ct.tt.Frame()
		// Enter at the end of a: the line it opens is
		// the one added, as the editor shows it.
		ct.st.Select(1, 1)
		ct.st.Run(Newline)
		ct.tt.Frame()
		g := ct.st.ed.gitDiff
		if g == nil {
			t.Fatal("no diff")
		}
		marked := []int{g.status(1), g.status(2)}
		want := []int{1, 0}
		if plain {
			want = []int{0, 1}
		}
		if marked[0] != want[0] || marked[1] != want[1] {
			t.Errorf("plain %v: lines 1 and 2 marked %v, want %v", plain, marked, want)
		}
	}
}

// A large text under DiffPlain is compared off the main thread as the
// default's whole comparisons are, with diff.Plain, and each edit
// compares the whole texts again.
func TestDiffPlainLargeComparesOffMainThread(t *testing.T) {
	jobs := holdDiffs(t)
	old, nw := largeDiffTexts(diffSyncLines)
	st := &State{Buffer: NewBuffer(nw), DiffBase: NewBuffer(old), DiffPlain: true}
	tt := ui.NewTester(func(c *ui.Context) { View(c, st).Fill() }, 800, 600)
	tt.Frame()
	g := st.ed.gitDiff
	if g == nil || len(*jobs) != 1 || !g.d.comparing() {
		t.Fatalf("%d comparisons started off the main thread, want 1 under way", len(*jobs))
	}
	go (*jobs)[0]()
	wait.Until(t, "the window to draw again after the comparison", tt.FramePending)
	tt.Frame()
	want := diff.Plain(wholeLines(rope.New(old)), wholeLines(rope.New(nw)))
	if g.d.comparing() || !slices.Equal(g.d.hunks, want) || len(want) == 0 {
		t.Fatalf("comparing %v, %d hunks, want %d from diff.Plain", g.d.comparing(), len(g.d.hunks), len(want))
	}
	st.Select(0, 0)
	st.Run(Newline)
	tt.Frame()
	if len(*jobs) != 2 {
		t.Fatalf("%d comparisons after an edit, want the whole texts compared again", len(*jobs))
	}
}
