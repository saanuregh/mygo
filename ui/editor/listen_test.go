package editor

import (
	"fmt"
	"strings"
	"testing"
)

// listenAll has b's listener check each change it is told of and return
// the texts told, in order.
func listenAll(t *testing.T, b *Buffer) (told *[]string, stop func()) {
	t.Helper()
	var out []string
	stop = b.Listen(func(old, new Snapshot, cs []Change) {
		if new.Version() != old.Version()+1 {
			t.Errorf("told version %d after %d", new.Version(), old.Version())
		}
		// The changes take old to new.
		text := old.String()
		for i := len(cs) - 1; i >= 0; i-- {
			c := cs[i]
			text = text[:c.OldStart] + new.Slice(c.NewStart, c.NewEnd) + text[c.OldEnd:]
		}
		if text != new.String() {
			t.Errorf("changes %v make %q of %q, not %q", cs, text, old.String(), new.String())
		}
		out = append(out, new.String())
	})
	return &out, stop
}

// Listen tells each edit as it is made, in order, whatever made it:
// Edit, an editor, undo and redo; stop ends it.
func TestListenTellsEachEdit(t *testing.T) {
	ct := newEditorTest(t, "", "ab|")
	b := ct.st.Buffer
	told, stop := listenAll(t, b)
	b.Edit(Edit{Start: 0, End: 0, Text: ">"})
	if len(*told) != 1 || (*told)[0] != ">ab" {
		t.Fatalf("after Edit, told %q", *told)
	}
	ct.typ("c")
	b.EndGroup()
	ct.run(Undo)
	ct.run(Redo)
	if got := strings.Join(*told, ","); got != ">ab,>abc,>ab,>abc" {
		t.Fatalf("told %q", got)
	}
	stop()
	b.Edit(Edit{Start: 0, End: 1})
	if len(*told) != 4 {
		t.Fatalf("told after stop: %q", *told)
	}
}

// EditRemote, Set, Reset and the buffer's own Undo and Redo tell the
// listeners too: a language server kept in step by them would otherwise
// miss the text an edit from elsewhere made.
func TestEditRemoteTellsListeners(t *testing.T) {
	b := NewBuffer("hello")
	told, _ := listenAll(t, b)
	b.Edit(Edit{Start: 5, End: 5, Text: "!"})
	b.EditRemote(Edit{Start: 0, End: 0, Text: ">"})
	b.Undo()
	b.Redo()
	b.EditRemote(Edit{Start: 0, End: 1}, Edit{Start: 6, End: 6, Text: "?"})
	b.Set("set")
	b.Reset("reset")
	want := "hello!,>hello!,>hello,>hello!,hello?!,set,reset"
	if got := strings.Join(*told, ","); got != want {
		t.Fatalf("told %q, want %q", got, want)
	}
	// An edit that changes nothing is no version and tells nothing.
	b.EditRemote(Edit{Start: 0, End: 0})
	if len(*told) != 7 {
		t.Fatalf("an empty remote edit was told: %q", *told)
	}
}

// A listener editing the buffer is told of its edit after it returns,
// in order.
func TestListenerEditing(t *testing.T) {
	b := NewBuffer("x")
	var told []string
	b.Listen(func(old, new Snapshot, cs []Change) {
		told = append(told, new.String())
		if new.String() == "xy" {
			b.Edit(Edit{Start: 2, End: 2, Text: "z"})
		}
	})
	b.Edit(Edit{Start: 1, End: 1, Text: "y"})
	if got := fmt.Sprint(told); got != "[xy xyz]" {
		t.Fatalf("told %s", got)
	}
}

// A listener too far behind the log is told one change of the whole
// text.
func TestListenerBehindTheLog(t *testing.T) {
	b := NewBuffer("")
	var told [][]Change
	var olds, news []string
	b.Listen(func(old, new Snapshot, cs []Change) {
		told = append(told, cs)
		olds, news = append(olds, old.String()), append(news, new.String())
		if new.String() == "a" {
			for range logMax + 10 {
				b.Edit(Edit{Start: 1, End: 1, Text: "b"})
			}
		}
	})
	b.Edit(Edit{Start: 0, End: 0, Text: "a"})
	if len(told) != 2 {
		t.Fatalf("told %d times, want 2", len(told))
	}
	whole := "a" + strings.Repeat("b", logMax+10)
	if olds[1] != "a" || news[1] != whole || fmt.Sprint(told[1]) != fmt.Sprint([]Change{{0, 1, 0, len(whole)}}) {
		t.Fatalf("second told %q to %q with %v", olds[1], news[1], told[1])
	}
}

// EndGroup ends a group of typing: what follows at once is a step of its
// own.
func TestEndGroup(t *testing.T) {
	ct := newEditorTest(t, "", "|")
	ct.typ("a")
	ct.typ("b")
	ct.st.Buffer.EndGroup()
	ct.typ("c")
	ct.run(Undo)
	ct.want("ab|")
	ct.run(Undo)
	ct.want("|")
}

// MarkSavedAt marks the text a save wrote saved after edits made
// meanwhile, which undoing makes the buffer unmodified again.
func TestMarkSavedAt(t *testing.T) {
	ct := newEditorTest(t, "", "a|")
	b := ct.st.Buffer
	ct.typ("b")
	saving := b.Snapshot()
	b.EndGroup()
	ct.typ("c")
	b.MarkSavedAt(saving)
	if !b.Modified() {
		t.Fatal("unmodified with an edit after the text saved")
	}
	ct.run(Undo)
	ct.want("ab|")
	if b.Modified() {
		t.Fatal("modified at the text saved")
	}
}
