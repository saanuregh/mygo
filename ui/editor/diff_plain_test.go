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
// compares the whole texts again: the hunks shown stay, moved with the
// edit, until that comparison is done.
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
	if got := shiftedHunks(want, 1); !slices.Equal(g.d.hunks, got) {
		t.Fatalf("%d hunks while compared again, want the %d before moved down a line", len(g.d.hunks), len(got))
	}
	go (*jobs)[1]()
	wait.Until(t, "the window to draw again after the comparison", tt.FramePending)
	tt.Frame()
	want = diff.Plain(wholeLines(rope.New(old)), wholeLines(rope.New("\n"+nw)))
	if g.d.comparing() || !slices.Equal(g.d.hunks, want) {
		t.Fatalf("comparing %v, %d hunks, want %d from diff.Plain", g.d.comparing(), len(g.d.hunks), len(want))
	}
}

// Typing while a large text under DiffPlain is compared starts no other
// comparison: the hunks shown move with the edits, and the texts are
// compared again once the one under way is done.
func TestDiffPlainLargeEditsWhileComparing(t *testing.T) {
	jobs := holdDiffs(t)
	old, nw := largeDiffTexts(diffSyncLines / 2)
	st := &State{Buffer: NewBuffer(nw), DiffBase: NewBuffer(old), DiffPlain: true}
	tt := ui.NewTester(func(c *ui.Context) { View(c, st).Fill() }, 800, 600)
	tt.Frame()
	g := st.ed.gitDiff
	if g == nil || len(*jobs) != 1 {
		t.Fatalf("%d comparisons started, want 1", len(*jobs))
	}
	go (*jobs)[0]()
	wait.Until(t, "the window to draw again after the comparison", tt.FramePending)
	tt.Frame()
	first := slices.Clone(g.d.hunks)
	if len(first) == 0 || g.d.comparing() {
		t.Fatalf("comparing %v with %d hunks, want it done with some", g.d.comparing(), len(first))
	}
	st.Select(0, 0)
	for i := range 5 {
		st.Run(Newline)
		tt.Frame()
		if len(*jobs) != 2 {
			t.Fatalf("%d comparisons after %d keystrokes, want 2: one at a time", len(*jobs), i+1)
		}
		if got := shiftedHunks(first, i+1); !slices.Equal(g.d.hunks, got) {
			t.Fatalf("after %d keystrokes %d hunks, want the %d before moved down", i+1, len(g.d.hunks), len(got))
		}
	}
	// The comparison of the first keystroke's text is done: the texts
	// have changed since, so they are compared again, its hunks shown,
	// the line it added first and the others moved down by the rest.
	go (*jobs)[1]()
	wait.Until(t, "the window to draw again after the comparison", tt.FramePending)
	tt.Frame()
	if len(*jobs) != 3 || !g.d.comparing() {
		t.Fatalf("%d comparisons, comparing %v, want a third under way for the keystrokes since", len(*jobs), g.d.comparing())
	}
	taken := diff.Plain(wholeLines(rope.New(old)), wholeLines(rope.New("\n"+nw)))
	if got := g.d.hunks; len(got) != len(taken) || got[0].B0 != 0 || !slices.Equal(got[1:], shiftedHunks(taken[1:], 4)) {
		t.Fatalf("%d hunks while compared again, want the %d of the first keystroke's text moved down", len(got), len(taken))
	}
	go (*jobs)[2]()
	wait.Until(t, "the window to draw again after the comparison", tt.FramePending)
	tt.Frame()
	want := diff.Plain(wholeLines(rope.New(old)), wholeLines(rope.New(st.Buffer.String())))
	if len(*jobs) != 3 || g.d.comparing() || !slices.Equal(g.d.hunks, want) {
		t.Fatalf("%d comparisons, comparing %v, %d hunks, want 3 done with %d", len(*jobs), g.d.comparing(), len(g.d.hunks), len(want))
	}
}

// shiftedHunks returns hs with their new lines n lower.
func shiftedHunks(hs []diff.Hunk, n int) []diff.Hunk {
	out := slices.Clone(hs)
	for i := range out {
		out[i].B0 += n
		out[i].B1 += n
	}
	return out
}
