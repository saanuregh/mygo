package editor

import (
	"regexp"
	"slices"
	"testing"
)

// withAbuttingEmpty: the empty matches right after a match that rure's
// search from each match's end finds and Go's FindAll leaves out, their
// assertions read in the text's context.
func TestWithAbuttingEmpty(t *testing.T) {
	for _, tc := range []struct {
		pat, text string
		want      [][2]int
	}{
		{"x*", "axxbxc", [][2]int{{0, 0}, {1, 3}, {3, 3}, {4, 5}, {5, 5}, {6, 6}}},
		{"x*", "ab", [][2]int{{0, 0}, {1, 1}, {2, 2}}},
		// After "ab" at 2 a word ends: \b holds; inside a word it does not.
		{`ab|\b`, "ab cd", [][2]int{{0, 2}, {2, 2}, {3, 3}, {5, 5}}},
		{`(?m)a|^`, "a\nb", [][2]int{{0, 1}, {2, 2}}},
	} {
		re := regexp.MustCompile(tc.pat)
		var got [][2]int
		for _, m := range withAbuttingEmpty(tc.pat, tc.text, re.FindAllStringSubmatchIndex(tc.text, -1)) {
			got = append(got, [2]int{m[0], m[1]})
		}
		if !slices.Equal(got, tc.want) {
			t.Errorf("%q in %q: %v, want %v", tc.pat, tc.text, got, tc.want)
		}
	}
}
