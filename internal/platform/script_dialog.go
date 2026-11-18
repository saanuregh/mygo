package platform

// ScriptDialogKind is the kind of a page's JavaScript dialog, numbered as
// WebKitGTK's WebKitScriptDialogType.
type ScriptDialogKind int

const (
	ScriptDialogAlert ScriptDialogKind = iota
	ScriptDialogConfirm
	ScriptDialogPrompt
	ScriptDialogBeforeUnload
)

// ScriptDialog is a page's alert, confirm, prompt or beforeunload dialog
// (WindowHandler.ScriptDialog).
type ScriptDialog struct {
	Kind    ScriptDialogKind
	Message string
	// DefaultText is a prompt's.
	DefaultText string
	// Answer closes the dialog: accept is OK (a confirm's true, a
	// beforeunload's leave), text a prompt's answer. It runs on the main
	// thread; the backend answers once, and not at all after the page
	// closed, and may answer after the call returns.
	Answer func(accept bool, text string)
}
