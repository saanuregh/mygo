package editor

import "testing"

// PreserveCase gives the replacement the case pattern of the match, as VS
// Code's preserve case does.
func TestPreserveCaseRules(t *testing.T) {
	for _, c := range []struct{ matched, replacement, want string }{
		{"FOO", "bar", "BAR"},                       // all upper
		{"foo", "Bar", "bar"},                       // all lower
		{"Foo", "bar", "Bar"},                       // capitalized
		{"Foo", "barBaz", "BarBaz"},                 // capitalized keeps the rest as typed
		{"fooBar", "BazQux", "BazQux"},              // a mixed pattern: as typed
		{"FOO-bar", "baz-qux", "BAZ-qux"},           // parts by '-'
		{"Foo_Bar", "baz_qux", "Baz_Qux"},           // parts by '_'
		{"foo-bar", "baz-qux-quux", "baz-qux-quux"}, // other part counts: whole, lower
		{"Foo-Bar", "baz", "Baz"},                   // replacement without parts
		{"ÉTÉ", "hiver", "HIVER"},                   // not only ASCII
		{"123", "abc", "abc"},                       // no letters: as typed
		{"FOO", "", ""},
		{"", "bar", "bar"},
	} {
		if got := PreserveCase(c.matched, c.replacement); got != c.want {
			t.Errorf("PreserveCase(%q, %q) = %q, want %q", c.matched, c.replacement, got, c.want)
		}
	}
}

// With preserve case on, Replace All gives each match's replacement the
// match's case; off, the replacement is as typed.
func TestPreserveCaseReplaceAll(t *testing.T) {
	for _, on := range []bool{true, false} {
		ct := newEditorTest(t, "", "|foo Foo FOO")
		f := &ct.st.ed.find
		f.open, f.query, f.replace, f.stale = true, "foo", "bar", true
		*(&FindBar{ed: &ct.st.ed}).PreserveCase() = on
		f.replaceAll(&ct.st.ed)
		want := "bar bar bar"
		if on {
			want = "bar Bar BAR"
		}
		if got := ct.st.Buffer.String(); got != want {
			t.Fatalf("preserve case %v: replaced into %q, want %q", on, got, want)
		}
	}
}
