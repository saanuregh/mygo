package editor

import (
	"testing"
	"time"
)

// kindHighlighter colours every line as kind; gen is its tokens'
// generation.
type kindHighlighter struct {
	kind TokenKind
	gen  uint64
}

func (h *kindHighlighter) Edit(old, new Snapshot, changes []Change)      {}
func (h *kindHighlighter) Highlight(last int, budget time.Duration) bool { return true }
func (h *kindHighlighter) Tokens(i int, text string, dst []Token) ([]Token, bool) {
	return append(dst, Token{End: len(text), Kind: h.kind}), true
}
func (h *kindHighlighter) TokensGeneration() uint64 { return h.gen }

// A highlighter whose colours change with no edit has its lines laid out
// again once its generation changes.
func TestViewTakesNewTokensWithoutAnEdit(t *testing.T) {
	ct := newEditorTest(t, "", "one\ntwo\n")
	h := &kindHighlighter{kind: TokenKeyword}
	ct.st.Buffer.SetLanguage(&Language{Name: "Own", Highlighter: func(Snapshot, func()) Highlighter { return h }})
	ct.tt.Frame()
	if toks := ct.st.ed.view.line(0).toks; len(toks) == 0 || toks[0].Kind != TokenKeyword {
		t.Fatalf("first tokens %+v", toks)
	}
	h.kind = TokenString
	ct.tt.Frame()
	if toks := ct.st.ed.view.line(0).toks; toks[0].Kind != TokenKeyword {
		t.Fatalf("the tokens changed with the same generation: %+v", toks)
	}
	h.gen++
	ct.tt.Frame()
	if toks := ct.st.ed.view.line(0).toks; len(toks) == 0 || toks[0].Kind != TokenString {
		t.Fatalf("after a new generation, tokens %+v", toks)
	}
}
