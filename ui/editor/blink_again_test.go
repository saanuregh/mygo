package editor

import (
	"testing"
	"time"
)

// Selections set between frames, long after the last, show the caret at
// the next frame, its blink started again.
func TestSelectionsSetLateShowTheCaret(t *testing.T) {
	ct := newEditorTest(t, "", "|one\ntwo\nthree")
	ct.st.Theme, ct.st.NoActiveLine = redTheme, true
	ct.tt.Frame()
	ed := &ct.st.ed
	for _, wait := range []time.Duration{0, 300 * time.Millisecond, 700 * time.Millisecond} {
		ct.tt.AdvanceTo(ed.frameNow.Add(1250*time.Millisecond + wait))
		ct.st.GoTo(Pos{Line: 1, Col: 0})
		ct.tt.Frame()
		if ed.frameNow.Sub(ed.blinkStart) != 0 {
			t.Fatalf("after %v the blink starts %v before the frame", wait, ed.frameNow.Sub(ed.blinkStart))
		}
	}
}
