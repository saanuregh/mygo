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

// Inlay hints take room in the rows, go with the character after them and
// one at the line's end takes the last word down (zode's inlays.zig tests).
func TestWrapAtSpacesCountsInlays(t *testing.T) {
	for _, c := range []struct {
		text  string
		hints []Inlay
		cols  int
		rows  []string
	}{
		{"aaaa bbbb cccc", nil, 10, []string{"aaaa bbbb ", "cccc"}},
		{"aaaa bbbb cccc", []Inlay{{Offset: 4, Text: ": long"}}, 10, []string{"aaaa", ": long ", "bbbb cccc"}},
		{"aaaa bbbb", []Inlay{{Offset: 9, Text: " // 2"}}, 10, []string{"aaaa ", "bbbb // 2"}},
		// A tab's columns count after a hint before it.
		{"\t\tf(\"some long text here to wrap\")", []Inlay{{Offset: 4, Text: "format: "}}, 30, []string{"        f(format: \"some long ", "text here to wrap\")"}},
	} {
		ct := newEditorTest(t, "", c.text)
		ct.st.SoftWrap, ct.st.WrapColumn, ct.st.WrapAtSpaces, ct.st.TabSize = true, c.cols, true, 4
		ct.st.SetInlays(c.hints)
		ct.tt.Frame()
		v := &ct.st.ed.view
		l := v.line(0)
		var rows []string
		for _, r := range l.rows {
			rows = append(rows, l.disp[r.from:r.to])
		}
		if !reflect.DeepEqual(rows, c.rows) {
			t.Errorf("%q at %d: rows %q, want %q", c.text, c.cols, rows, c.rows)
		}
	}
}
