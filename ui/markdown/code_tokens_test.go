package markdown

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// codeSpans returns the spans of the first code block of the view el, in
// the frame that built it.
func codeSpans(el ui.Element) []ui.Span {
	st := *ui.Local(el, stateKey{}, newState)
	for _, cs := range st.codes {
		var out []ui.Span
		for _, ch := range cs.chunks {
			out = append(out, ch.spans...)
		}
		return out
	}
	return nil
}

func spanOf(spans []ui.Span, text string) (ui.Span, bool) {
	for _, s := range spans {
		if s.Text == text {
			return s, true
		}
	}
	return ui.Span{}, false
}

// Options.CodeTokens colors code blocks, asked again only when a block's
// text changes; a block it leaves goes to the lexers.
func TestCodeTokensColorCodeBlocks(t *testing.T) {
	var asked []string
	o := Options{CodeTokens: func(lang, code string) ([][]editor.Token, bool) {
		asked = append(asked, lang+":"+code)
		if lang != "zig" {
			return nil, false
		}
		var out [][]editor.Token
		for _, line := range strings.Split(strings.TrimSuffix(code, "\n"), "\n") {
			// The first word a keyword, the rest a string.
			w := strings.IndexByte(line, ' ')
			if w < 0 {
				w = len(line)
			}
			out = append(out, []editor.Token{{End: w, Kind: editor.TokenKeyword}, {End: len(line), Kind: editor.TokenString}})
		}
		return out, true
	}}
	src := "```zig\nconst x\nvar y\n```\n"
	var spans []ui.Span
	tt := ui.NewTester(func(c *ui.Context) { spans = codeSpans(View(c, src, &o)) }, 400, 300)
	th := DefaultTheme(ui.LightTheme())
	kw, ok := spanOf(spans, "var")
	if !ok || kw.Color != th.CodeColors.TokenColor(editor.TokenKeyword) {
		t.Fatalf("var is %+v in %+v", kw, spans)
	}
	if s, ok := spanOf(spans, " y"); !ok || s.Color != th.CodeColors.TokenColor(editor.TokenString) {
		t.Fatalf("\" y\" is %+v", s)
	}
	if len(asked) != 1 || asked[0] != "zig:const x\nvar y\n" {
		t.Fatalf("asked %q", asked)
	}
	tt.Frame()
	if len(asked) != 1 {
		t.Fatalf("asked again with the text unchanged: %q", asked)
	}
	src = "```zig\nconst x\nvar y\nvar z\n```\n"
	tt.Frame()
	if len(asked) != 2 {
		t.Fatalf("not asked again for the new text: %q", asked)
	}
	if _, ok := spanOf(spans, " z"); !ok {
		t.Fatalf("the new line is not among %+v", spans)
	}

	// Go, which the hook leaves: the lexer colors func as a keyword.
	src = "```go\nfunc f() {}\n```\n"
	tt.Frame()
	if s, ok := spanOf(spans, "func"); !ok || s.Color != th.CodeColors.TokenColor(editor.TokenKeyword) {
		t.Fatalf("func is %+v in %+v", s, spans)
	}
}
