package editor

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// Outside text with as many lines as there are carets pastes a line at
// each caret, a trailing newline aside, as VS Code's spread paste; other
// counts, and one caret, paste the whole text at each.
func TestOutsideLinesSpreadAcrossCarets(t *testing.T) {
	for _, c := range []struct{ start, clip, want string }{
		{"a|\nb|\nc|", "spread1\nspread2\nspread3", "aspread1|\nbspread2|\ncspread3|"},
		{"a|\nb|", "spread1\nspread2\n", "aspread1|\nbspread2|"},
		{"[x]\n[y]", "spreadA\nspreadB", "spreadA|\nspreadB|"},
		{"a|\nb|", "spread1\nspread2\nspread3", "aspread1\nspread2\nspread3|\nbspread1\nspread2\nspread3|"},
		{"a|", "spreadX\n", "aspreadX\n|"},
	} {
		ct := newEditorTest(t, "", c.start)
		ct.tt.SetClipboard(c.clip)
		ct.key(ui.Cmd, ui.KeyV)
		ct.want(c.want)
	}
}
