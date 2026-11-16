package editor

import (
	"reflect"
	"testing"
)

// With HintsFollowText, hints move with edits as zode's inlays.zig had
// them: one stays before text inserted at it, one inside replaced text
// goes and one at its end follows it (that file's test).
func TestHintsFollowText(t *testing.T) {
	ct := newEditorTest(t, "", "0123456789abcdefghijklmnopqrstuvwxyz")
	ct.st.HintsFollowText = true
	ct.st.SetInlays([]Inlay{{Offset: 10, Text: ": i32"}, {Offset: 2, Text: "a: "}, {Offset: 20, Text: "b: "}})
	ct.tt.Frame()
	at := func() []int {
		var out []int
		for _, in := range ct.st.ed.view.inlays {
			out = append(out, in.at)
		}
		return out
	}
	step := func(start, end int, text string, want []int) {
		t.Helper()
		ct.st.Buffer.Edit(Edit{Start: start, End: end, Text: text})
		ct.tt.Frame()
		if got := at(); !reflect.DeepEqual(got, want) {
			t.Errorf("after [%d,%d) = %q: hints at %v, want %v", start, end, text, got, want)
		}
	}
	// An insertion before them all.
	step(0, 0, "xyz", []int{5, 13, 23})
	// An insertion at a hint: it stays before the new text.
	step(13, 13, "w", []int{5, 13, 24})
	// A deletion around one: it goes; one at the deletion's end follows it.
	step(12, 24, "", []int{5, 12})
	if in := ct.st.ed.view.inlays[1]; in.text != "b: " {
		t.Errorf("the hint left is %q", in.text)
	}
}

// With HintsFollowText, trailing inlays move as upstream moves them:
// after text inserted at them, and kept when text around them is
// replaced; the inlays stay in order.
func TestHintsFollowTextLeavesTrailingInlays(t *testing.T) {
	ct := newEditorTest(t, "", "abc\ndef")
	ct.st.HintsFollowText = true
	ct.st.SetInlays([]Inlay{{Offset: 3, Text: "  blame", Trailing: true}, {Offset: 3, Text: ": T"}, {Offset: 5, Text: "x: "}})
	ct.tt.Frame()
	type mark struct {
		at   int
		text string
	}
	got := func() []mark {
		var out []mark
		for _, in := range ct.st.ed.view.inlays {
			out = append(out, mark{in.at, in.text})
		}
		return out
	}
	// Typed at the line's end: the hint stays before the text, the
	// trailing inlay goes after it.
	ct.st.Buffer.Edit(Edit{Start: 3, End: 3, Text: "!"})
	ct.tt.Frame()
	if want := []mark{{3, ": T"}, {4, "  blame"}, {6, "x: "}}; !reflect.DeepEqual(got(), want) {
		t.Fatalf("after typing at the end: %v, want %v", got(), want)
	}
	// Replaced around them: the hint inside goes, the trailing inlay stays.
	ct.st.Buffer.Edit(Edit{Start: 2, End: 7, Text: ""})
	ct.tt.Frame()
	if want := []mark{{2, "  blame"}}; !reflect.DeepEqual(got(), want) {
		t.Fatalf("after replacing around them: %v, want %v", got(), want)
	}
}

// RowStarts tells where soft wrap breaks a line, as it was laid out.
func TestRowStarts(t *testing.T) {
	ct := newEditorTest(t, "", "")
	ct.st.SoftWrap, ct.st.WrapColumn = true, 10
	ct.st.Buffer.Edit(Edit{Text: "aaaa bbbb cccc dddd eeee\nshort"})
	ct.tt.Frame()
	starts := ct.st.RowStarts(0, nil)
	if len(starts) == 0 {
		t.Fatal("a line wider than its 10 columns has no rows after its first")
	}
	l := ct.st.ed.view.line(0)
	for i, s := range starts {
		if want := l.fromDisp(l.rows[i+1].from); s != want {
			t.Fatalf("row %d starts at %d, want %d", i+1, s, want)
		}
	}
	if got := ct.st.RowStarts(1, nil); len(got) != 0 {
		t.Fatalf("a short line's row starts %v", got)
	}
}
