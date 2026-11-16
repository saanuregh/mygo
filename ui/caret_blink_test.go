package ui

import (
	"bytes"
	"testing"
	"time"
)

// Theme.CaretBlink sets the caret's half period: at 500 milliseconds,
// shown again a second after it moved (nanoui's), where the usual 530 has
// it hidden.
func TestCaretBlinkSetsTheHalfPeriod(t *testing.T) {
	for _, blink := range []time.Duration{500 * time.Millisecond, 0} {
		text := "abc"
		tt := NewTester(func(c *Context) {
			th := LightTheme()
			th.CaretBlink = blink
			c.SetTheme(th)
			TextInput(c, &text).Label("Field").Width(200)
		}, 300, 100)
		if err := tt.Click("Field"); err != nil {
			t.Fatal(err)
		}
		tt.Frame()
		shown := bytes.Clone(tt.Image().Pix)
		tt.Advance(1020 * time.Millisecond)
		tt.Frame()
		same := bytes.Equal(shown, tt.Image().Pix)
		if same != (blink == 500*time.Millisecond) {
			t.Errorf("blink %v: shown again at 1.02 s %v", blink, same)
		}
		tt.rt.close()
	}
}
