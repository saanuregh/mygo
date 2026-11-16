package editor

import "testing"

// F7-2: against an empty base (an added file), the gutter marks the
// file's lines, not the empty one after its final newline; a base with
// text keeps mygo's marks.
func TestAddedFileBarsLeaveTheLineAfterTheLastNewline(t *testing.T) {
	ct := newEditorTest(t, "", "a\nb\n|")
	ct.st.DiffBase = NewBuffer("")
	ct.tt.Frame()
	g := ct.st.ed.gitDiff
	if g == nil {
		t.Fatal("no diff")
	}
	if got := []int{g.status(0), g.status(1), g.status(2)}; got[0] != 1 || got[1] != 1 || got[2] != 0 {
		t.Fatalf("statuses %v, want [1 1 0]", got)
	}
	ct = newEditorTest(t, "", "a\nb|")
	ct.st.DiffBase = NewBuffer("")
	ct.tt.Frame()
	if g := ct.st.ed.gitDiff; g.status(1) != 1 {
		t.Fatalf("without a final newline, the last line's status %d", g.status(1))
	}
}
