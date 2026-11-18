package editor

import (
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/text"
)

// An edit lays its view's lines out anew, which shaped every line in view
// again: a key typed in a tall editor cost each frame milliseconds and
// megabytes. The lines it did not change keep their shapes.
func TestAnEditShapesOnlyTheLinesItChanges(t *testing.T) {
	var shaped []string
	useShaper(t, func(sys *text.System, p text.Params) *text.Layout {
		shaped = append(shaped, p.Text)
		return sys.Shape(p)
	})
	var lines []string
	for i := range 60 {
		lines = append(lines, fmt.Sprintf("line %d of the text", i))
	}
	ct := newEditorTest(t, "", "|"+strings.Join(lines, "\n"))
	ct.tt.SetSize(600, 1400)
	ct.tt.Frame()
	ct.tt.Frame()
	shaped = nil
	ct.tt.Type("x")
	ct.tt.Frame()
	for _, s := range shaped {
		if s != "xline 0 of the text" {
			t.Errorf("shaped %q again, which the edit left as it was (%d lines shaped)", s, len(shaped))
			break
		}
	}
	if len(shaped) == 0 {
		t.Error("the edited line was not shaped")
	}
}
