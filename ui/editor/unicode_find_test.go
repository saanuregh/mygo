package editor

import (
	"regexp"
	"testing"
)

// With FindFromOrigin (zode's rure), \w, \d and \s take any script's
// characters.
func TestFindClassesAreUnicodes(t *testing.T) {
	for _, c := range []struct{ pat, text, want string }{
		{`\w+`, "héllo wörld", "héllo"},
		{`[\w]+`, "ñandú!", "ñandú"},
		{`\d+`, "x ٣٤ y", "٣٤"},
		{`a\sb`, "a b", "a b"},
		{`\W`, "é!", "!"},
		{`[[:alpha:]\d]+`, "ab٣-", "ab٣"},
		{`[^\w]`, "é?", "?"},
	} {
		re, err := regexp.Compile(unicodeClasses(c.pat))
		if err != nil {
			t.Fatalf("%q: %v", c.pat, err)
		}
		if got := re.FindString(c.text); got != c.want {
			t.Errorf("%q in %q: %q, want %q", c.pat, c.text, got, c.want)
		}
	}
}

// With FindFromOrigin, a regular expression that is not one is searched
// for as it is typed, its error still said; without it, it finds nothing.
func TestFindAnInvalidPatternAsTyped(t *testing.T) {
	for _, origin := range []bool{false, true} {
		ct := newEditorTest(t, "", "f(x) and f(y|")
		ct.st.FindFromOrigin = origin
		ct.tt.Frame()
		ct.st.ed.find.regex = true
		ct.st.Find("f(")
		ct.tt.Frame()
		n, _ := ct.st.FindMatches()
		if want := map[bool]int{false: 0, true: 2}[origin]; n != want || ct.st.ed.find.err == nil {
			t.Errorf("FindFromOrigin %v: %d matches, error %v; want %d and an error", origin, n, ct.st.ed.find.err, want)
		}
	}
}
