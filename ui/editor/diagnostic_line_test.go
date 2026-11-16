package editor

import "testing"

// A diagnostic's Line stays the line SetDiagnostics put it on while an
// edit above moves its Start down.
func TestDiagnosticLineStaysAsGiven(t *testing.T) {
	ct := newEditorTest(t, "", "|a\nb\nc")
	ct.tt.Frame()
	ct.st.SetDiagnostics(ct.st.Buffer.Version(), []Diagnostic{{Start: 2, End: 3, Message: "b"}})
	ct.typ("x\n")
	ct.tt.Frame()
	d := ct.st.Diagnostics()
	if len(d) != 1 || d[0].Line != 1 || d[0].Start != 4 {
		t.Fatalf("diagnostics %+v, want Line 1 at Start 4", d)
	}
}
