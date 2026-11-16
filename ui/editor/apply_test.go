package editor

import "testing"

// TestStateApply checks that edits applied with the selections after them
// are one step of undo that puts the selections back where they were,
// and redo where Apply put them.
func TestStateApply(t *testing.T) {
	ct := newEditorTest(t, "", "ab|c\nde|f")
	st := ct.st
	before := st.Selections()
	ok := st.Apply([]Edit{{Start: 2, End: 2, Text: "\n"}, {Start: 6, End: 6, Text: "\n"}},
		[]Selection{{Anchor: 3, Head: 3}, {Anchor: 8, Head: 8}})
	if !ok || st.Buffer.String() != "ab\nc\nde\nf" {
		t.Fatalf("applied %v: %q", ok, st.Buffer.String())
	}
	after := st.Selections()
	if len(after) != 2 || after[0].Head != 3 || after[1].Head != 8 || st.Cursor() != 8 {
		t.Fatalf("selections after %+v, newest at %d", after, st.Cursor())
	}
	st.Run(Undo)
	if got := st.Selections(); st.Buffer.String() != "abc\ndef" || len(got) != 2 || got[0] != before[0] || got[1] != before[1] {
		t.Fatalf("after undo %q, selections %+v, want %+v", st.Buffer.String(), got, before)
	}
	st.Run(Redo)
	if got := st.Selections(); st.Buffer.String() != "ab\nc\nde\nf" || len(got) != 2 || got[0] != after[0] || got[1] != after[1] {
		t.Fatalf("after redo %q, selections %+v, want %+v", st.Buffer.String(), got, after)
	}
	st.ReadOnly = true
	ct.tt.Frame()
	if st.Apply([]Edit{{Start: 0, End: 0, Text: "x"}}, []Selection{{}}) {
		t.Fatal("a read-only editor applied edits")
	}
}
