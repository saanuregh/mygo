package terminal

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// A right click's menu drawn in the window leaves the terminal as focused
// (no focus report goes out while it shows), as the menu gives the focus
// back as it closes.
func TestMenuKeepsTheTerminalFocused(t *testing.T) {
	loadLib(t)
	conn := newPipe()
	term, err := New(Options{Conn: conn, ContextMenu: []Action{ActionCopy, ActionPaste}})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	term.Feed([]byte("\x1b[?1004hsome text"))
	tt := ui.NewTester(func(c *ui.Context) {
		c.SetMenuPresentation(ui.PopoverMenus)
		View(c, term).Fill().AutoFocus()
	}, 400, 200)
	tt.Frame()
	tt.Frame()
	conn.take(term)
	x, y := cellCenter(term, 1, 0)
	tt.RightClickAt(x, y)
	tt.Frame()
	if tt.PopoverMenu() == nil {
		t.Fatal("no menu")
	}
	if got := conn.take(term); got != "" {
		t.Errorf("sent %q while the menu shows", got)
	}
}
