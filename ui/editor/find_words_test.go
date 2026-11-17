package editor

import (
	"fmt"
	"regexp"
	"slices"
	"testing"
)

// foundMatch is a match of the find bar with its regular expression's
// groups, -1 for a group that matched nothing.
type foundMatch struct {
	Start, End int
	Groups     []Range
}

// findWith runs the find bar of zode (FindFromOrigin) over text with a
// regular expression, whole words or not, and returns its matches.
func findWith(t *testing.T, text, query string, wholeWord bool) []foundMatch {
	t.Helper()
	ct := newEditorTest(t, "", text)
	ct.st.FindFromOrigin = true
	ct.tt.Frame()
	ct.st.ed.find.regex, ct.st.ed.find.wholeWord = true, wholeWord
	ct.st.Find(query)
	ct.tt.Frame()
	f := &ct.st.ed.find
	if f.err != nil {
		t.Fatalf("%q: %v", query, f.err)
	}
	f.search(&ct.st.ed)
	out := make([]foundMatch, len(f.matches))
	for i, m := range f.matches {
		out[i] = foundMatch{Start: m.start, End: m.end}
		if i < len(f.subs) && len(f.subs[i]) > 2 {
			sub := f.subs[i]
			for g := 2; g+1 < len(sub); g += 2 {
				out[i].Groups = append(out[i].Groups, Range{sub[g], sub[g+1]})
			}
		}
	}
	return out
}

func spans(ms []foundMatch) [][2]int {
	out := [][2]int{}
	for _, m := range ms {
		out = append(out, [2]int{m.Start, m.End})
	}
	return out
}

// E7-4: \b and \B are Unicode's words' edges, \< and \> (and
// \b{start}, \b{end} and their halves) a word's start and end, as
// rure's.
func TestFindWordAssertionsAreRures(t *testing.T) {
	for _, c := range []struct {
		pat, text string
		want      [][2]int
	}{
		// "café" is 5 bytes; "cafés" goes on in a word.
		{`\bcafé\b`, "café cafés", [][2]int{{0, 5}}},
		{`\Bé`, "café é", [][2]int{{3, 5}}},
		{`\<\w+`, "über alles", [][2]int{{0, 5}, {6, 11}}},
		{`\w+\>`, "über alles", [][2]int{{0, 5}, {6, 11}}},
		{`a\>`, "a ab ba", [][2]int{{0, 1}, {6, 7}}},
		{`\<|\>`, "ab cd", [][2]int{{0, 0}, {2, 2}, {3, 3}, {5, 5}}},
		{`\b{start}x`, "xx x", [][2]int{{0, 1}, {3, 4}}},
		{`x\b{end}`, "xx x", [][2]int{{1, 2}, {3, 4}}},
		{`\b{start-half}x`, "ax x", [][2]int{{3, 4}}},
		{`x\b{end-half}`, "xa x", [][2]int{{3, 4}}},
		// In a class, \< is left to Go: the character <.
		{`[\<]`, "a<b", [][2]int{{1, 2}}},
		// Escaped, the backslash is the text's.
		{`\\<`, `a\<`, [][2]int{{1, 3}}},
	} {
		if got := spans(findWith(t, c.text, c.pat, false)); !slices.Equal(got, c.want) {
			t.Errorf("%q in %q: %v, want %v", c.pat, c.text, got, c.want)
		}
	}
}

// E7-4: a pattern's groups found by the word matcher.
func TestFindWordAssertionsKeepGroups(t *testing.T) {
	ms := findWith(t, "über x", `\<(\w)(\w*)`, false)
	if len(ms) != 2 || !slices.Equal(ms[0].Groups, []Range{{0, 2}, {2, 5}}) {
		t.Fatalf("matches %+v", ms)
	}
}

// E7-3: zode's find.zig matches a pattern between word boundaries for
// Whole Word, rather than keeping its matches that are whole words: the
// longer alternative is tried where the shorter is not a word.
func TestFindWholeWordRegexIsBetweenBoundaries(t *testing.T) {
	for _, c := range []struct {
		pat, text string
		want      [][2]int
	}{
		{`foo|foobar`, "foobar foo", [][2]int{{0, 6}, {7, 10}}},
		{`ü`, "ü üb", [][2]int{{0, 2}}},
		{`a.`, "ab a- xab", [][2]int{{0, 2}}},
	} {
		if got := spans(findWith(t, c.text, c.pat, true)); !slices.Equal(got, c.want) {
			t.Errorf("%q in %q: %v, want %v", c.pat, c.text, got, c.want)
		}
	}
}

// The word matcher finds, in ASCII text, what Go's regexp finds with
// rure's empty matches (withAbuttingEmpty), groups and all.
func TestWordMatcherAgreesWithRegexpInASCII(t *testing.T) {
	texts := []string{"", "foo bar foobar", "ab abc abcd\nxx yy", "aaa b a_b 12 x1", "  \n\n a"}
	pats := []string{`\bfo+\b`, `(a|ab)(c|bcd)(d*)`, `\b`, `\B`, `x*`, `(?i)\bFOO\w*`, `(?m)^\w+$`, `a+?`,
		`(a+)(b)?`, `\B.`, `(?m)$|\b\w`, `(?:a|)\b`, `(?s).\b.`, `[^a]\b`, `(?P<w>\w+)\s+(\w)`}
	for _, p := range pats {
		re := regexp.MustCompile(p)
		m, err := compileWordMatcher(p)
		if err != nil {
			t.Fatalf("%q: %v", p, err)
		}
		for _, text := range texts {
			want := withAbuttingEmpty(p, text, re.FindAllStringSubmatchIndex(text, -1))
			got := m.findAll(text, maxMatches)
			if fmt.Sprint(got) != fmt.Sprint(want) {
				t.Errorf("%q in %q: %v, want %v", p, text, got, want)
			}
		}
	}
}

// rureWords stands its runes for the assertions Go's syntax lacks, and
// says which the pattern holds.
func TestRureWords(t *testing.T) {
	for _, c := range []struct {
		pat, want     string
		words, bounds bool
	}{
		{`abc`, `abc`, false, false},
		{`\bx\B`, `\bx\B`, true, true},
		{`\<x\>`, `\x{10fff0}x\x{10fff1}`, true, false},
		{`\b{start-half}\b{end}`, `\x{10fff2}\x{10fff1}`, true, false},
		{`[\b\<]\\<`, `[\b\<]\\<`, false, false},
	} {
		out, words, bounds := rureWords(c.pat)
		if out != c.want || words != c.words || bounds != c.bounds {
			t.Errorf("%q: %q %v %v, want %q %v %v", c.pat, out, words, bounds, c.want, c.words, c.bounds)
		}
	}
}
