package terminal

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestKeysAsCharacters: a key held with Control or Alt is encoded by the
// character it types, so Ctrl+Shift+A sends what Ctrl+A does and Ctrl+=
// reaches the program (no keybinds taking them); arrows are encoded by
// their key. Without the option Ctrl+Shift+A and Ctrl+= send nothing.
func TestKeysAsCharacters(t *testing.T) {
	loadLib(t)
	for _, as := range []bool{false, true} {
		conn := newPipe()
		term, err := New(Options{Conn: conn, KeysAsCharacters: as, NoDefaultKeybinds: true})
		if err != nil {
			t.Fatal(err)
		}
		tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
		tt.Frame()
		key := func(mods ui.Modifiers, k ui.Key) string {
			tt.Key(mods, k)
			return conn.take(term)
		}
		ctrlShiftA, ctrlEqual := key(ui.Ctrl|ui.Shift, ui.KeyA), key(ui.Ctrl, ui.KeyEqual)
		if as {
			for _, c := range []struct {
				mods      ui.Modifiers
				key       ui.Key
				got, want string
			}{
				{ui.Ctrl | ui.Shift, ui.KeyA, ctrlShiftA, "\x01"},
				{ui.Ctrl, ui.KeyEqual, ctrlEqual, "\x1b[61;5u"},
				{ui.Ctrl, ui.KeyC, key(ui.Ctrl, ui.KeyC), "\x03"},
				{ui.Ctrl, ui.KeyUp, key(ui.Ctrl, ui.KeyUp), "\x1b[1;5A"},
				// Upstream's C0 controls (#1523), not fixterms' CSI u.
				{ui.Ctrl, ui.KeyI, key(ui.Ctrl, ui.KeyI), "\t"},
				{ui.Ctrl, ui.KeyM, key(ui.Ctrl, ui.KeyM), "\r"},
				{ui.Ctrl, ui.KeyBracketLeft, key(ui.Ctrl, ui.KeyBracketLeft), "\x1b"},
			} {
				if c.got != c.want {
					t.Errorf("%v %v sent %q, want %q", c.mods, c.key, c.got, c.want)
				}
			}
			if got := key(ui.Alt, ui.KeyB); goos != "darwin" && got != "\x1bb" {
				t.Errorf("Alt+B sent %q", got)
			}
			// No text typed after the key is lost.
			tt.Type("x")
			if got := conn.take(term); got != "x" {
				t.Errorf("typed %q after the keys", got)
			}
		} else if ctrlShiftA != "" || ctrlEqual != "" {
			t.Errorf("without the option Ctrl+Shift+A sent %q and Ctrl+= %q", ctrlShiftA, ctrlEqual)
		}
		term.Close()
	}
}

func TestKeyCharacter(t *testing.T) {
	for _, c := range []struct {
		key       ui.Key
		shift     bool
		mac       bool
		r         rune
		character bool
	}{
		{ui.KeyA, false, false, 'a', true},
		{ui.KeyA, true, false, 'a', true},
		{ui.Key1, true, false, '!', true},
		{ui.KeyEqual, false, false, '=', true},
		{ui.KeyEqual, true, false, '+', true},
		{ui.KeyTab, true, false, '\t', true},
		{ui.KeySpace, false, false, ' ', true},
		{ui.KeyEnter, false, false, '\r', true},
		{ui.KeyBackspace, false, false, 0x08, true},
		{ui.KeyBackspace, false, true, 0x7f, true},
		{ui.KeyDelete, false, false, 0x7f, true},
		{ui.KeyDelete, false, true, 0, false},
		{ui.KeyUp, false, false, 0, false},
		{ui.KeyF5, false, false, 0, false},
	} {
		r, _, ok := keyCharacter(c.key, c.shift, c.mac)
		if ok != c.character || ok && r != c.r {
			t.Errorf("%v shift %v mac %v: %q %v, want %q %v", c.key, c.shift, c.mac, r, ok, c.r, c.character)
		}
	}
}

// scrolled is a terminal with 100 lines of output, in a view.
func scrolled(t *testing.T, opts Options) (*Terminal, *ui.Tester, func() uint64) {
	t.Helper()
	term, err := New(opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { term.Close() })
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	var b strings.Builder
	for i := range 100 {
		b.WriteString("line " + itoa(i) + "\r\n")
	}
	term.Feed([]byte(b.String()))
	tt.Frame()
	return term, tt, func() uint64 {
		term.mu.Lock()
		defer term.mu.Unlock()
		return term.term.Scrollbar().Offset
	}
}

// TestNoWheelZoom: with Control (Command on macOS) the wheel scrolls the
// scrollback, and OnZoom sees nothing.
func TestNoWheelZoom(t *testing.T) {
	loadLib(t)
	zooms := 0
	term, _, offset := scrolled(t, Options{Conn: newPipe(), NoWheelZoom: true, OnZoom: func(Zoom) bool { zooms++; return true }})
	bottom := offset()
	term.v.scroll(ui.InputEvent{Kind: ui.InputScroll, DY: -hostNotch(), Mods: ui.Cmd})
	if got := offset(); got != bottom-3 || zooms != 0 {
		t.Errorf("Control+wheel: offset %d (want %d), %d zooms", got, bottom-3, zooms)
	}
}

// TestNoScrollbar: no bar shows, and a press on its strip selects rather
// than scrolls.
func TestNoScrollbar(t *testing.T) {
	loadLib(t)
	_, tt, offset := scrolled(t, Options{Conn: newPipe(), NoScrollbar: true})
	tt.Move(395, 100)
	tt.Frame()
	img := tt.Image()
	s := img.Bounds().Dx() / 400
	if img.RGBAAt(395*s, 188*s) != img.RGBAAt(380*s, 188*s) {
		t.Error("a bar shows")
	}
	bottom := offset()
	tt.Press(395, 188)
	tt.Move(395, 20)
	tt.Release(395, 20)
	if got := offset(); got != bottom {
		t.Errorf("a drag on the strip scrolled to %d, from %d", got, bottom)
	}
}

// TestNoAlternateScroll: the wheel over a full-screen program that does
// not take the mouse sends nothing.
func TestNoAlternateScroll(t *testing.T) {
	loadLib(t)
	conn := newPipe()
	term, tt, _ := scrolled(t, Options{Conn: conn, NoAlternateScroll: true})
	term.Feed([]byte("\x1b[?1049h"))
	tt.Frame()
	tt.Wheel(100, 100, 0, -hostNotch())
	if got := conn.take(term); got != "" {
		t.Errorf("the wheel sent %q", got)
	}
}

// TestNoAltClick: Alt+click on a prompt's input moves nothing, and selects
// as a click does.
func TestNoAltClick(t *testing.T) {
	loadLib(t)
	conn := newPipe()
	term, err := New(Options{Conn: conn, NoAltClick: true})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	tt.Frame()
	term.Feed([]byte("\x1b]133;A\a$ \x1b]133;B\aecho hello\x1b[3D"))
	tt.Frame()
	x, y := cellCenter(term, 3, 0)
	tt.ClickAtWith(ui.Alt, x, y)
	if got := conn.take(term); got != "" {
		t.Errorf("Alt+click sent %q", got)
	}
}

// TestNoFocusReports: a program asking for focus reports gets none.
func TestNoFocusReports(t *testing.T) {
	loadLib(t)
	conn := newPipe()
	term, err := New(Options{Conn: conn, NoFocusReports: true})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	focus := false
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			ui.Button(c, "Away")
			e := View(c, term).Fill()
			if focus {
				e.Focus()
				focus = false
			}
		})
	}, 400, 200)
	term.Feed([]byte("\x1b[?1004h"))
	tt.Frame()
	focus = true
	tt.Frame()
	tt.Frame()
	if err := tt.Click("Away"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if got := conn.take(term); got != "" {
		t.Errorf("focus changes sent %q", got)
	}
}

// TestCursorLook: a see-through cursor keeps its alpha and goes over the
// text, which shows through; FixedCursor keeps the block a program asks
// to be a bar; unfocused, the hollow block takes UnfocusedCursor.
func TestCursorLook(t *testing.T) {
	loadLib(t)
	bg, fg := ui.RGB(0, 0, 0), ui.RGB(255, 255, 255)
	cursor := ui.RGB(0, 0, 255).Alpha(0.75)
	faint := ui.RGB(255, 0, 0)
	theme := &Theme{Foreground: fg, Background: bg, Cursor: cursor, UnfocusedCursor: faint}
	term, err := New(Options{Conn: newPipe(), Theme: theme, FixedCursor: true, NoBlink: true})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	focused := true
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			ui.Button(c, "Away")
			e := View(c, term).Fill()
			if focused {
				e.Focus()
			}
		})
	}, 400, 200)
	// A bar asked for (DECSCUSR 6), the cursor over an M.
	term.Feed([]byte("\x1b[6 qM\x1b[D"))
	tt.Frame()
	tt.Frame()
	v := term.v
	cw, ch := float32(v.cellW)/v.scale, float32(v.cellH)/v.scale
	cell, _, _, _ := v.cursorRect(0, 0, cw, ch)
	if cell.W < cw*0.9 {
		t.Errorf("the cursor is %v wide, want a block of %v", cell.W, cw)
	}
	_, c, _, block := v.cursorRect(0, 0, cw, ch)
	if c != cursor || !block {
		t.Errorf("the focused cursor is %v (block %v), want %v", c, block, cursor)
	}
	focused = false
	if err := tt.Click("Away"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	tt.Frame()
	if _, c, _, _ := v.cursorRect(0, 0, cw, ch); c != faint {
		t.Errorf("the unfocused cursor is %v, want %v", c, faint)
	}
}
