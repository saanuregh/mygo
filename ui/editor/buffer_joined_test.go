package editor

import "testing"

func TestEditJoinedIsUndoneWithTheStepBefore(t *testing.T) {
	b := NewBuffer("<a></a>")
	b.Edit(Edit{Start: 0, End: 0, Text: "x"})
	b.Edit(Edit{Start: 3, End: 3, Text: "b"})
	b.EditJoined(Edit{Start: 8, End: 8, Text: "b"})
	if got := b.String(); got != "x<ab></ab>" {
		t.Fatalf("String = %q", got)
	}
	b.Undo()
	if got := b.String(); got != "x<a></a>" {
		t.Fatalf("after one undo %q, want both edits of the step taken back", got)
	}
	b.Redo()
	if got := b.String(); got != "x<ab></ab>" {
		t.Fatalf("after redo %q", got)
	}
	b.Undo()
	b.Undo()
	if got := b.String(); got != "<a></a>" || b.Modified() {
		t.Fatalf("after undoing all %q, modified %v", got, b.Modified())
	}
}

func TestEditJoinedWithoutAStepMakesOne(t *testing.T) {
	b := NewBuffer("a")
	b.EditJoined(Edit{Start: 1, End: 1, Text: "b"})
	if !b.CanUndo() {
		t.Fatal("no step to undo")
	}
	b.Undo()
	if got := b.String(); got != "a" {
		t.Fatalf("after undo %q", got)
	}
}
