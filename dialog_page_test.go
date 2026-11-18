//go:build !js

package mygo

import (
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/internal/wait"
)

// answers records the answers a backend's dialog got, and on which thread.
type answers struct {
	mu     sync.Mutex
	got    []string
	onMain []bool
}

func (a *answers) dialog(kind platform.ScriptDialogKind, msg, def string) *platform.ScriptDialog {
	return &platform.ScriptDialog{Kind: kind, Message: msg, DefaultText: def, Answer: func(accept bool, text string) {
		a.mu.Lock()
		defer a.mu.Unlock()
		if accept {
			a.got = append(a.got, "accept:"+text)
		} else {
			a.got = append(a.got, "dismiss")
		}
		a.onMain = append(a.onMain, isMainThread())
	}}
}

func (a *answers) list() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.got...)
}

func TestDialogHandler(t *testing.T) {
	w, fw := testWindow(t, WindowOptions{})
	_, otherFW := testWindow(t, WindowOptions{})
	ask := func(h platform.WindowHandler, d *platform.ScriptDialog) bool {
		return onMainValue(func() bool { return h.ScriptDialog(d) })
	}
	var a answers
	// Without a handler the web view shows its own.
	if ask(fw.H, a.dialog(platform.ScriptDialogAlert, "hi", "")) {
		t.Error("a dialog was taken without a handler")
	}

	got := make(chan *ScriptDialog, 4)
	w.Page().SetDialogHandler(func(d *ScriptDialog) bool {
		if !isMainThread() {
			t.Error("the dialog handler ran off the main thread")
		}
		got <- d
		return true
	})
	// The handler is the page's: another window's page keeps the default.
	if ask(otherFW.H, a.dialog(platform.ScriptDialogAlert, "elsewhere", "")) {
		t.Error("another page's dialog went to this page's handler")
	}

	if !ask(fw.H, a.dialog(platform.ScriptDialogPrompt, "Name?", "Ada")) {
		t.Fatal("the handler's true did not keep the dialog")
	}
	d := wait.For(t, "the prompt", got)
	if d.Kind != DialogPrompt || d.Message != "Name?" || d.DefaultText != "Ada" {
		t.Errorf("dialog = %+v", d)
	}
	if len(a.list()) != 0 {
		t.Fatal("answered before Accept")
	}
	// Answered later, from another goroutine, once: a second answer and a
	// Dismiss after it do nothing.
	done := make(chan struct{})
	go func() {
		time.Sleep(50 * time.Millisecond)
		d.Accept("Grace")
		d.Accept("again")
		d.Dismiss()
		close(done)
	}()
	<-done
	if got := a.list(); len(got) != 1 || got[0] != "accept:Grace" {
		t.Errorf("answers = %q, want one accept:Grace", got)
	}
	if !a.onMain[0] {
		t.Error("the backend's answer ran off the main thread")
	}

	ask(fw.H, a.dialog(platform.ScriptDialogConfirm, "Sure?", ""))
	c := wait.For(t, "the confirm", got)
	if c.Kind != DialogConfirm {
		t.Errorf("kind = %v", c.Kind)
	}
	c.Dismiss()
	c.Accept("")
	ask(fw.H, a.dialog(platform.ScriptDialogBeforeUnload, "", ""))
	if b := wait.For(t, "the beforeunload", got); b.Kind != DialogBeforeUnload {
		t.Errorf("kind = %v", b.Kind)
	}
	if got := a.list(); len(got) != 2 || got[1] != "dismiss" {
		t.Errorf("answers = %q", got)
	}

	// A handler returning false leaves the dialog to the web view: its
	// answers do nothing.
	w.Page().SetDialogHandler(func(d *ScriptDialog) bool { got <- d; return false })
	if ask(fw.H, a.dialog(platform.ScriptDialogAlert, "left", "")) {
		t.Error("false kept the dialog")
	}
	wait.For(t, "the alert", got).Accept("")
	if n := len(a.list()); n != 2 {
		t.Errorf("a dialog left to the web view was answered (%d answers)", n)
	}

	w.Page().SetDialogHandler(nil)
	if ask(fw.H, a.dialog(platform.ScriptDialogAlert, "hi", "")) {
		t.Error("SetDialogHandler(nil) did not restore the default")
	}
}
