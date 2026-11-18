package mygo

import (
	"sync/atomic"

	"github.com/egoist/mygo/internal/platform"
)

// DialogKind is the kind of a page's JavaScript dialog (ScriptDialog).
type DialogKind int

const (
	// DialogAlert is alert(message).
	DialogAlert DialogKind = iota
	// DialogConfirm is confirm(message): Accept makes it true.
	DialogConfirm
	// DialogPrompt is prompt(message, defaultText): Accept(text) makes it
	// text, Dismiss null.
	DialogPrompt
	// DialogBeforeUnload is a beforeunload handler asking to stay on the
	// page: Accept leaves it, Dismiss stays.
	DialogBeforeUnload
)

// ScriptDialog is a JavaScript dialog the page waits on: an alert, a
// confirm, a prompt, or a beforeunload asking to stay (SetDialogHandler).
type ScriptDialog struct {
	Kind DialogKind
	// Message is the text the page shows. WebKit gives a beforeunload
	// dialog its own, not the page's.
	Message string
	// DefaultText is a prompt's suggested answer.
	DefaultText string

	answer   func(accept bool, text string)
	answered atomic.Bool
}

// Accept answers OK: an alert closes, a confirm returns true, a prompt
// text, and a beforeunload leaves the page. It can be called from any
// goroutine. Only a dialog's first answer counts; one after the page
// closed, or after the handler returned false, does nothing.
func (d *ScriptDialog) Accept(text string) { d.close(true, text) }

// Dismiss answers Cancel: an alert closes, a confirm returns false, a
// prompt null, and a beforeunload stays on the page. As Accept, it can be
// called from any goroutine and only the first answer counts.
func (d *ScriptDialog) Dismiss() { d.close(false, "") }

func (d *ScriptDialog) close(accept bool, text string) {
	if d.answer == nil || !d.answered.CompareAndSwap(false, true) {
		return
	}
	onMain(func() { d.answer(accept, text) })
}

// SetDialogHandler answers the page's JavaScript dialogs, its alert,
// confirm and prompt calls and the beforeunload handlers that ask to stay,
// in place of the web view's own dialog. fn runs on the main thread; it
// returns true to answer the dialog later with Accept or Dismiss, from any
// goroutine, while the page waits as it does on a dialog the user sees, or
// false to let the web view show its own. A dialog never answered keeps
// the page waiting until it closes. nil restores the default.
//
//	page.SetDialogHandler(func(d *mygo.ScriptDialog) bool {
//		go func() { d.Accept(askTheUser(d.Message)) }()
//		return true
//	})
//
// Linux only (WebKitWebView::script-dialog); on macOS, Windows and in
// browsers fn is never called and the web view shows its own dialogs.
func (p *Page) SetDialogHandler(fn func(d *ScriptDialog) bool) {
	p.w.mu.Lock()
	p.w.dialogHandler = fn
	p.w.mu.Unlock()
}

func (h *windowHandler) ScriptDialog(pd *platform.ScriptDialog) bool {
	h.w.mu.Lock()
	fn := h.w.dialogHandler
	h.w.mu.Unlock()
	if fn == nil {
		return false
	}
	d := &ScriptDialog{Kind: DialogKind(pd.Kind), Message: pd.Message, DefaultText: pd.DefaultText, answer: pd.Answer}
	if !fn(d) {
		// The web view answers it: Accept and Dismiss do nothing.
		d.answered.Store(true)
		return false
	}
	return true
}
