//go:build linux && (amd64 || arm64)

package linux

import (
	"slices"

	"github.com/ebitengine/purego"

	"github.com/egoist/mygo/internal/platform"
)

// JavaScript dialogs of pages: the web view's script-dialog signal, which
// the app answers later (WindowHandler.ScriptDialog).

var (
	webkitScriptDialogGetDialogType        func(d ptr) int32
	webkitScriptDialogGetMessage           func(d ptr) ptr
	webkitScriptDialogConfirmSetConfirmed  func(d ptr, confirmed bool)
	webkitScriptDialogPromptGetDefaultText func(d ptr) ptr
	webkitScriptDialogPromptSetText        func(d ptr, text *byte)
	webkitScriptDialogRef                  func(d ptr) ptr
	webkitScriptDialogUnref                func(d ptr)
	webkitScriptDialogClose                func(d ptr)

	cbScriptDialog, cbScriptDialogAnswer ptr
	// answering holds, by id, the dialogs answered while their signal ran,
	// until an idle callback answers them after it (main thread only).
	answering    = map[uintptr]*scriptDialog{}
	nextAnswerID uintptr
)

func bindScriptDialogs() {
	w := libWebKit
	mustBind(w, &webkitScriptDialogGetDialogType, "webkit_script_dialog_get_dialog_type")
	mustBind(w, &webkitScriptDialogGetMessage, "webkit_script_dialog_get_message")
	mustBind(w, &webkitScriptDialogConfirmSetConfirmed, "webkit_script_dialog_confirm_set_confirmed")
	mustBind(w, &webkitScriptDialogPromptGetDefaultText, "webkit_script_dialog_prompt_get_default_text")
	mustBind(w, &webkitScriptDialogPromptSetText, "webkit_script_dialog_prompt_set_text")
	mustBind(w, &webkitScriptDialogRef, "webkit_script_dialog_ref")
	mustBind(w, &webkitScriptDialogUnref, "webkit_script_dialog_unref")
	mustBind(w, &webkitScriptDialogClose, "webkit_script_dialog_close")
}

// scriptDialog is a dialog the app took, which holds a reference to
// WebKit's until it is answered or its page closes. Everything about it
// runs on the main thread.
//
// WebKit drops its own reference right after the signal, through the web
// view's current dialog, which closing the dialog clears: a dialog must
// not close during its signal, so an answer then waits for an idle
// callback. Closing is WebKit's to guard (the page's completion handler
// runs once, and the web view's dispose closes a dialog still open), the
// reference ours: released exactly once, by the answer or the page's
// close, after which an answer does nothing.
type scriptDialog struct {
	w        *window
	ref      ptr
	kind     platform.ScriptDialogKind
	signal   bool // in its script-dialog signal still
	released bool
	// pending is an answer given during the signal.
	pending *dialogAnswer
}

type dialogAnswer struct {
	accept bool
	text   string
}

func initScriptDialogCallbacks() {
	// Returning false leaves the dialog to WebKit: its own, in the view.
	cbScriptDialog = purego.NewCallback(func(web, dialog, data ptr) bool {
		w := theBackend.window(data)
		if w == nil {
			return false
		}
		kind := platform.ScriptDialogKind(webkitScriptDialogGetDialogType(dialog))
		pd := &platform.ScriptDialog{Kind: kind, Message: goStr(webkitScriptDialogGetMessage(dialog))}
		if kind == platform.ScriptDialogPrompt {
			pd.DefaultText = goStr(webkitScriptDialogPromptGetDefaultText(dialog))
		}
		d := &scriptDialog{w: w, ref: webkitScriptDialogRef(dialog), kind: kind, signal: true}
		w.dialogs = append(w.dialogs, d)
		pd.Answer = d.answer
		taken := w.h.ScriptDialog(pd)
		d.signal = false
		if !taken || d.released {
			// The handler left it to WebKit, or closed the page.
			d.release()
			return taken
		}
		if d.pending != nil {
			nextAnswerID++
			answering[nextAnswerID] = d
			gIdleAddFull(0, cbScriptDialogAnswer, nextAnswerID, 0)
		}
		return true
	})
	cbScriptDialogAnswer = purego.NewCallback(func(id ptr) int32 {
		if d := answering[id]; d != nil {
			delete(answering, id)
			d.answer(d.pending.accept, d.pending.text)
		}
		return 0 // G_SOURCE_REMOVE
	})
}

// answer sets the dialog's answer and closes it.
func (d *scriptDialog) answer(accept bool, text string) {
	if d.released {
		return
	}
	if d.signal {
		if d.pending == nil {
			d.pending = &dialogAnswer{accept, text}
		}
		return
	}
	switch d.kind {
	case platform.ScriptDialogConfirm, platform.ScriptDialogBeforeUnload:
		webkitScriptDialogConfirmSetConfirmed(d.ref, accept)
	case platform.ScriptDialogPrompt:
		if accept {
			// Without a text, the prompt returns null.
			webkitScriptDialogPromptSetText(d.ref, cs(text))
		}
	}
	webkitScriptDialogClose(d.ref)
	d.release()
}

// release drops the reference to the dialog. One never closed closes then,
// as WebKit's own default answer, unless the web view already did.
func (d *scriptDialog) release() {
	if d.released {
		return
	}
	d.released = true
	d.w.dialogs = slices.DeleteFunc(d.w.dialogs, func(x *scriptDialog) bool { return x == d })
	webkitScriptDialogUnref(d.ref)
}

// releaseDialogs releases the dialogs of a page that closes, which no
// answer reaches after.
func (w *window) releaseDialogs() {
	for len(w.dialogs) > 0 {
		w.dialogs[0].release()
	}
}

// TestScriptDialogShown reports whether the page of a window shows
// WebKit's own JavaScript dialog, for the GUI tests in internal/e2e. It
// must run on the main thread.
func TestScriptDialogShown(handle uintptr) bool {
	w := windowByHandle(handle)
	if w == nil || w.web == 0 {
		return false
	}
	var fromName func(name *byte) uintptr
	mustBind(libGObject, &fromName, "g_type_from_name")
	typ := fromName(cs("WebKitScriptDialogImpl"))
	if typ == 0 {
		return false // never made one
	}
	if gtkContainerForall == nil {
		mustBind(libGTK, &gtkContainerForall, "gtk_container_forall")
	}
	shown = false
	shownType = typ
	if cbShown == 0 {
		cbShown = purego.NewCallback(func(widget, data ptr) {
			if gTypeCheckInstanceIsA(widget, shownType) {
				shown = true
			}
		})
	}
	gtkContainerForall(w.web, cbShown, 0)
	return shown
}

var (
	cbShown   ptr
	shown     bool
	shownType uintptr
)
