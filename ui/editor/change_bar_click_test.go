package editor

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// A press on a change bar is OnChangeBarClick's, with the bar's hunk; a
// press on the line's number stays OnGutterClick's.
func TestChangeBarClickTakesTheBarsPress(t *testing.T) {
	var hunks []DiffHunk
	gutterClicks := 0
	st := &State{Buffer: NewBuffer("one\nTWO\nthree\nfour"), DiffBase: NewBuffer("one\ntwo\nthree\nfour"), DiffPlain: true,
		Font:          ui.Font{Family: "monospace", Size: 14},
		Gutter:        &GutterLayout{Left: 18, Right: 14, BarRight: 8, MinDigits: 3},
		OnGutterClick: func(int) { gutterClicks++ },
		OnChangeBarClick: func(h DiffHunk) bool {
			hunks = append(hunks, h)
			return true
		}}
	tt := ui.NewTester(func(c *ui.Context) { View(c, st).Fill() }, 400, 200)
	tt.Frame()
	lh := st.ed.view.lineHeight
	barX := st.ed.gutterW - 8 + 1.5
	tt.ClickAt(barX, lh*1.5)
	tt.Frame()
	if want := (DiffHunk{1, 2, 1, 2}); len(hunks) != 1 || hunks[0] != want || gutterClicks != 0 {
		t.Fatalf("bar press: hunks %+v, gutter clicks %d", hunks, gutterClicks)
	}
	// A line without a change is the gutter's.
	tt.ClickAt(barX, lh*2.5)
	tt.Frame()
	// The number is the gutter's.
	tt.ClickAt(30, lh*1.5)
	tt.Frame()
	if len(hunks) != 1 || gutterClicks != 2 {
		t.Errorf("away from a bar: hunks %+v, gutter clicks %d", hunks, gutterClicks)
	}
	// Declined, the press goes on to OnGutterClick.
	st.OnChangeBarClick = func(DiffHunk) bool { return false }
	tt.Frame()
	tt.ClickAt(barX, lh*1.5)
	tt.Frame()
	if gutterClicks != 3 {
		t.Errorf("declined: gutter clicks %d", gutterClicks)
	}
}

// ChangeHunks are the bars' hunks, as DiffPlain compares them.
func TestChangeHunksAreTheBars(t *testing.T) {
	st := &State{Buffer: NewBuffer("a\nb\nX\nd\ne\nf"), DiffPlain: true}
	if h := st.ChangeHunks(); h != nil {
		t.Fatalf("without a DiffBase: %+v", h)
	}
	st.DiffBase = NewBuffer("a\nb\nc\nd\nf")
	tt := ui.NewTester(func(c *ui.Context) { View(c, st).Fill() }, 400, 200)
	tt.Frame()
	want := []DiffHunk{{2, 3, 2, 3}, {4, 4, 4, 5}}
	got := st.ChangeHunks()
	if len(got) != len(want) {
		t.Fatalf("hunks %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("hunks %+v, want %+v", got, want)
		}
		for l := got[i].NewStart; l < got[i].NewEnd; l++ {
			if st.ed.gitDiff.status(l) == 0 {
				t.Errorf("line %d of %+v has no bar", l, got[i])
			}
		}
	}
	st.Insert("z\n")
	tt.Frame()
	if got := st.ChangeHunks(); len(got) != 3 || got[0] != (DiffHunk{0, 0, 0, 1}) {
		t.Errorf("after an edit: %+v", got)
	}
}
