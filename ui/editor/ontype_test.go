package editor

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestCodeEditorOnType checks that an app's OnType types in place of the
// editor: its edits and selections, as typing undo takes back with the
// typing around it, and the editor's own typing where it declines.
func TestCodeEditorOnType(t *testing.T) {
	ct := newEditorTest(t, "go", "a|b")
	ct.st.OnType = func(text string) ([]Edit, []Selection, bool) {
		if text != "(" {
			return nil, nil, false
		}
		var edits []Edit
		var after []Selection
		for i, s := range ct.st.Selections() {
			edits = append(edits, Edit{Start: s.Start(), End: s.End(), Text: "<>"})
			at := s.Start() + i*(2-(s.End()-s.Start())) + 1
			after = append(after, Selection{Anchor: at, Head: at})
		}
		return edits, after, true
	}
	ct.tt.Frame()
	ct.typ("x(")
	ct.want("ax<|>b")
	// Declined: the editor's own typing, closing brackets.
	ct.typ("[")
	ct.want("ax<[|]>b")
	ct.key(ui.Cmd, ui.KeyZ)
	ct.want("a|b")
	// Selections only move.
	ct = newEditorTest(t, "", "|ab")
	ct.st.OnType = func(string) ([]Edit, []Selection, bool) {
		return nil, []Selection{{Anchor: 1, Head: 1}}, true
	}
	ct.tt.Frame()
	ct.typ(")")
	ct.want("a|b")
	if strings.Contains(ct.st.Buffer.String(), ")") {
		t.Fatal("typed text went in")
	}
}
