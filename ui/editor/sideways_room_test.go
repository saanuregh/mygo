package editor

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// WideSidewaysRoom: End on a long line scrolls it so a quarter of the
// text's width, at most 48, shows past the caret (zode's view.zig).
func TestWideSidewaysRoomPastTheCaret(t *testing.T) {
	ct := newEditorTest(t, "", strings.Repeat("x", 300)+"\n|")
	ct.st.WideSidewaysRoom, ct.st.NoScrollbars = true, true
	ct.tt.Frame()
	ct.st.SetSelections(Selection{Anchor: 0, Head: 0})
	ct.tt.Frame()
	ct.key(0, ui.KeyEnd)
	ct.tt.Frame()
	ed := &ct.st.ed
	cr, _ := ed.caretRect(300)
	w := ed.r.W - ed.textX - ed.o.marginRight
	if want := cr.X - w + min(48, w/4); abs(ed.scrollX-want) > 0.5 {
		t.Fatalf("scrolled %v sideways, want %v", ed.scrollX, want)
	}
}
