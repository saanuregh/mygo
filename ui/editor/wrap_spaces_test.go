package editor

import (
	"reflect"
	"testing"
)

// WrapAtSpaces breaks rows only after spaces, a word longer than a row
// anywhere, and does not indent the rows after the first (zode's wrap.zig
// tests, on the ASCII cases).
func TestWrapAtSpacesBreaksOnlyAfterSpaces(t *testing.T) {
	for _, c := range []struct {
		text string
		cols int
		rows []string
	}{
		{"the quick brown fox jumps", 10, []string{"the quick ", "brown fox ", "jumps"}},
		{"abcdefghijklmnopqrstuvwxyz", 10, []string{"abcdefghij", "klmnopqrst", "uvwxyz"}},
		// Wide characters take two columns; a character is never split.
		{"中文字符测试", 8, []string{"中文字符", "测试"}},
		{"abcdef👩‍👩‍👧ghijklmn", 8, []string{"abcdef👩‍👩‍👧", "ghijklmn"}},
		{"x = fmt.Println(strings.Repeat)", 10, []string{"x = ", "fmt.Printl", "n(strings.", "Repeat)"}},
		// As wrap.zig: the space after the leading ones is the last that fits.
		{"    indented words here", 12, []string{"    ", "indented ", "words here"}},
	} {
		ct := newEditorTest(t, "", c.text)
		ct.st.SoftWrap, ct.st.WrapColumn, ct.st.WrapAtSpaces = true, c.cols, true
		ct.tt.Frame()
		v := &ct.st.ed.view
		l := v.line(0)
		var rows []string
		for _, r := range l.rows {
			rows = append(rows, l.disp[r.from:r.to])
			if r.indent != 0 {
				t.Errorf("%q: a row indented %v", c.text, r.indent)
			}
		}
		if !reflect.DeepEqual(rows, c.rows) {
			t.Errorf("%q at %d: rows %q, want %q", c.text, c.cols, rows, c.rows)
		}
	}
}
