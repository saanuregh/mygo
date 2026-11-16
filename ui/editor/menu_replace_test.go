package editor

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// Menu builds the context menu in place of the editor's items, which
// DefaultMenu adds back; ContextMenu's follow either.
func TestMenuReplacesTheEditorsItems(t *testing.T) {
	ct := newEditorTest(t, "", "hello|")
	ran := false
	ct.st.Menu = func(m *ui.Menu) {
		if m.Item("Ask").Chosen() {
			ran = true
		}
	}
	ct.st.ContextMenu = func(m *ui.Menu) { m.Item("Extra") }
	ct.tt.Frame()
	ct.tt.RightClickAt(200, 10)
	if got := strings.Join(ct.tt.Menu(), ","); got != "Ask,-,Extra" {
		t.Fatalf("menu %q, want Ask, a separator and Extra", got)
	}
	if err := ct.tt.ChooseMenuItem("Ask"); err != nil {
		t.Fatal(err)
	}
	if !ran {
		t.Fatal("the app's item did not run")
	}
	ct.st.Menu = func(m *ui.Menu) {
		m.Item("Mine")
		m.Separator()
		ct.st.DefaultMenu(m)
	}
	ct.st.ContextMenu = nil
	ct.tt.Frame()
	ct.tt.RightClickAt(200, 10)
	items := ct.tt.Menu()
	if len(items) < 4 || items[0] != "Mine" || items[1] != "-" || items[2] != "Cut" {
		t.Fatalf("menu %q, want Mine then the editor's own", items)
	}
	ct.tt.CloseMenu()
}
