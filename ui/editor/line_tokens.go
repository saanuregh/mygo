package editor

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// LineTokens appends the tokens of line i of the buffer to dst, from the
// buffer's highlighter (the one its editors share), and reports whether
// they are final; a plain text's are one token of text. The window of c
// is drawn again when the highlighter's work in the background is done,
// as for an editor showing the buffer: an app drawing lines of its own,
// as a diff's rows, colors them so. Main thread only.
func (b *Buffer) LineTokens(c *ui.Context, i int, dst []Token) ([]Token, bool) {
	text := b.Line(i)
	b.watch(c.Services())
	hl := b.highlighter()
	if hl == nil {
		return append(dst, Token{End: len(text), Kind: TokenText}), true
	}
	hl.Highlight(i, 4*time.Millisecond)
	return hl.Tokens(i, text, dst)
}
