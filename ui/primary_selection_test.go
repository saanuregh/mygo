package ui

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/internal/surface"
)

// clipboardOnly is a system clipboard with no primary selection, as
// macOS's, Windows' and a browser's.
type clipboardOnly struct {
	platform.Clipboard
	text *string
}

func (c clipboardOnly) ReadText() string   { return *c.text }
func (c clipboardOnly) WriteText(s string) { *c.text = s }

// withPrimary is a Linux desktop's clipboard, which has one.
type withPrimary struct {
	clipboardOnly
	primary *string
}

func (c withPrimary) ReadPrimaryText() string   { return *c.primary }
func (c withPrimary) WritePrimaryText(s string) { *c.primary = s }

// Off Linux the primary selection reads empty and takes nothing, and the
// clipboard is left as it was; on Linux it is the system's.
func TestPrimaryIsANoOpOffLinux(t *testing.T) {
	text := "clip"
	h := &windowHost{conn: &surface.Conn{Clipboard: clipboardOnly{text: &text}}}
	h.writePrimary("selected")
	if got := h.readPrimary(); got != "" {
		t.Errorf("no primary selection read %q", got)
	}
	if text != "clip" {
		t.Errorf("writing primary changed the clipboard to %q", text)
	}

	primary := ""
	h = &windowHost{conn: &surface.Conn{Clipboard: withPrimary{clipboardOnly{text: &text}, &primary}}}
	h.writePrimary("selected")
	if got := h.readPrimary(); got != "selected" || primary != "selected" {
		t.Errorf("primary read %q, holds %q", got, primary)
	}
	if text != "clip" {
		t.Errorf("writing primary changed the clipboard to %q", text)
	}

	h = &windowHost{conn: &surface.Conn{}}
	h.writePrimary("selected")
	if got := h.readPrimary(); got != "" {
		t.Errorf("no clipboard read %q", got)
	}
}
