package terminal

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestContextMenuItems: Options.ContextMenu lists the context menu's
// items, in order; Copy and Paste alone leave out Select All and Clear.
func TestContextMenuItems(t *testing.T) {
	loadLib(t)
	conn := newPipe()
	term, err := New(Options{Conn: conn, ContextMenu: []Action{ActionCopy, ActionPaste}})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	term.Feed([]byte("some text"))
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	tt.Frame()
	x, y := cellCenter(term, 1, 0)
	tt.RightClickAt(x, y)
	if got := strings.Join(tt.Menu(), ","); got != "Copy,Paste" {
		t.Errorf("context menu = %q", got)
	}
	tt.SetClipboard("pasted")
	if err := tt.ChooseMenuItem("Paste"); err != nil {
		t.Fatal(err)
	}
	if got := conn.take(term); got != "pasted" {
		t.Errorf("the menu's Paste sent %q", got)
	}
}

// TestNoLinks: with Options.NoLinks, Command+click opens no URL and
// underlines none.
func TestNoLinks(t *testing.T) {
	loadLib(t)
	opened := 0
	term, err := New(Options{Conn: newPipe(), NoLinks: true, OpenLink: func(string) bool { opened++; return true }})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	term.Feed([]byte("see https://x.org"))
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	tt.Frame()
	x, y := cellCenter(term, 8, 0)
	tt.HoldModifiers(ui.Cmd)
	tt.Move(x, y)
	tt.Frame()
	if link, _, _ := term.HoveredLink(); link != "" {
		t.Errorf("hovered link %q", link)
	}
	tt.HoldModifiers(0)
	tt.ClickAtWith(ui.Cmd, x, y)
	if opened != 0 || len(tt.OpenedURLs()) != 0 {
		t.Errorf("opened %d, system %q", opened, tt.OpenedURLs())
	}
}
