//go:build linux && (amd64 || arm64)

package e2e

import (
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/linux"
	"github.com/egoist/mygo/internal/wait"
	"github.com/egoist/mygo/ui"
)

const dialogPage = `<!doctype html><title>dialogs</title><body>dialogs</body>`

// dialogWindow is a window showing dialogPage.
func dialogWindow(t *testing.T) *mygo.Window {
	t.Helper()
	w := newWindow(t, mygo.WindowOptions{Title: "Dialogs", Width: 400, Height: 300})
	w.Page().LoadHTML(dialogPage, "http://dialogs.test/")
	waitFor(t, w, "document.readyState === 'complete'")
	return w
}

// takeDialogs keeps every dialog of the page for the test to answer.
func takeDialogs(p *mygo.Page) <-chan *mygo.ScriptDialog {
	got := make(chan *mygo.ScriptDialog, 16)
	p.SetDialogHandler(func(d *mygo.ScriptDialog) bool {
		got <- d
		return true
	})
	return got
}

func evalOK(t *testing.T, p *mygo.Page, code string) {
	t.Helper()
	if _, err := p.Eval(code); err != nil {
		t.Fatalf("%s: %v", code, err)
	}
}

// TestDialogHandler answers a page's alert, confirm, prompt and
// beforeunload later, from another goroutine, while the page waits.
func TestDialogHandler(t *testing.T) {
	w := dialogWindow(t)
	got := takeDialogs(w.Page())
	evalOK(t, w.Page(), `setTimeout(() => {
		const ms = [], r = {};
		let s = performance.now();
		const lap = () => { ms.push(performance.now() - s); s = performance.now(); };
		alert("hello"); lap();
		r.yes = confirm("yes?"); lap();
		r.no = confirm("no?"); lap();
		r.typed = prompt("name?", "Ada"); lap();
		r.empty = prompt("empty?"); lap();
		r.canceled = prompt("cancel?", "x"); lap();
		window.result = {r, ms};
	}, 0)`)
	type want struct {
		kind         mygo.DialogKind
		message, def string
		accept       bool
		text         string
	}
	const delay = 300 * time.Millisecond
	for _, a := range []want{
		{mygo.DialogAlert, "hello", "", true, ""},
		{mygo.DialogConfirm, "yes?", "", true, ""},
		{mygo.DialogConfirm, "no?", "", false, ""},
		{mygo.DialogPrompt, "name?", "Ada", true, "Grace"},
		{mygo.DialogPrompt, "empty?", "", true, ""},
		{mygo.DialogPrompt, "cancel?", "x", false, ""},
	} {
		d := wait.For(t, "a dialog", got)
		if d.Kind != a.kind || d.Message != a.message || d.DefaultText != a.def {
			t.Errorf("dialog = %v %q %q, want %v %q %q", d.Kind, d.Message, d.DefaultText, a.kind, a.message, a.def)
		}
		go func() {
			time.Sleep(delay)
			if a.accept {
				d.Accept(a.text)
			} else {
				d.Dismiss()
			}
			// Only the first answer counts.
			d.Accept("late")
			d.Dismiss()
		}()
	}
	waitFor(t, w, "window.result")
	res, err := mygo.EvalAs[map[string]any](w.Page(), "window.result")
	if err != nil {
		t.Fatal(err)
	}
	r := res["r"].(map[string]any)
	if r["yes"] != true || r["no"] != false || r["typed"] != "Grace" || r["empty"] != "" || r["canceled"] != nil {
		t.Errorf("the page got %v", r)
	}
	for i, ms := range res["ms"].([]any) {
		if time.Duration(ms.(float64))*time.Millisecond < delay-50*time.Millisecond {
			t.Errorf("dialog %d returned after %vms, before its answer", i, ms)
		}
	}

	// beforeunload: Dismiss stays on the page, Accept leaves it.
	evalOK(t, w.Page(), `window.marker = 1; window.onbeforeunload = e => { e.preventDefault(); e.returnValue = "stay"; }`)
	evalOK(t, w.Page(), `setTimeout(() => { location.href = "about:blank#left"; setTimeout(() => { window.stayed = true; }, 0); }, 0)`)
	d := wait.For(t, "a dialog", got)
	if d.Kind != mygo.DialogBeforeUnload {
		t.Fatalf("kind = %v, want beforeunload", d.Kind)
	}
	go d.Dismiss()
	waitFor(t, w, "window.stayed && window.marker === 1")
	evalOK(t, w.Page(), `setTimeout(() => { location.href = "about:blank#left"; }, 0)`)
	d = wait.For(t, "a dialog", got)
	go d.Accept("")
	waitFor(t, w, `location.href === "about:blank#left" && !window.marker`)
}

// TestDialogHandlerDefault: without a handler, or with one that returns
// false, the web view shows its own dialog.
func TestDialogHandlerDefault(t *testing.T) {
	for _, name := range []string{"nil", "false"} {
		t.Run(name, func(t *testing.T) {
			w := dialogWindow(t)
			asked := make(chan *mygo.ScriptDialog, 1)
			if name == "false" {
				w.Page().SetDialogHandler(func(d *mygo.ScriptDialog) bool {
					asked <- d
					return false
				})
			}
			evalOK(t, w.Page(), `setTimeout(() => alert("own"), 0)`)
			eventually(t, "WebKit's own dialog", func() (shown bool) {
				mygo.RunOnMain(func() { shown = linux.TestScriptDialogShown(w.NativeHandle()) })
				return shown
			})
			if name == "false" {
				// It was asked, and its answer is not the page's.
				wait.For(t, "the handler", asked).Accept("")
				time.Sleep(100 * time.Millisecond)
				var shown bool
				mygo.RunOnMain(func() { shown = linux.TestScriptDialogShown(w.NativeHandle()) })
				if !shown {
					t.Error("an answer closed a dialog left to WebKit")
				}
			}
			// Destroyed with WebKit's dialog open (newWindow's cleanup).
		})
	}
}

// TestDialogHandlerPageClosed closes pages, a window's and a web view's,
// while a dialog waits: nothing crashes, and a late answer does nothing.
func TestDialogHandlerPageClosed(t *testing.T) {
	w := dialogWindow(t)
	got := takeDialogs(w.Page())
	evalOK(t, w.Page(), `setTimeout(() => confirm("pending"), 0)`)
	d := wait.For(t, "a dialog", got)
	w.Destroy()
	d.Accept("")
	d.Dismiss()

	var view *mygo.WebView
	host := newWindow(t, mygo.WindowOptions{Title: "Dialog host", Width: 400, Height: 300, Content: ui.View(func(c *ui.Context) {
		if view != nil {
			ui.WebView(c, view).Grow(1)
		}
	})})
	v, err := host.NewWebView(mygo.WebViewOptions{})
	if err != nil {
		t.Fatal(err)
	}
	host.Update(func() { view = v })
	v.Page().LoadHTML(dialogPage, "http://dialogs.test/")
	waitForPage(t, v.Page(), "document.readyState === 'complete'")
	got = takeDialogs(v.Page())
	if _, err := v.Page().Eval(`setTimeout(() => prompt("pending", "x"), 0)`); err != nil {
		t.Fatal(err)
	}
	d = wait.For(t, "a dialog", got)
	host.Update(func() { view = nil })
	v.Destroy()
	d.Accept("late")

	// The app goes on: a new page answers.
	w2 := dialogWindow(t)
	if title, err := mygo.EvalAs[string](w2.Page(), "document.title"); err != nil || title != "dialogs" {
		t.Errorf("after the closed pages: title %q, %v", title, err)
	}
}

// TestDialogHandlerAnswersAtOnce: a handler may answer before it returns,
// which closes the dialog only once WebKit's signal ended.
func TestDialogHandlerAnswersAtOnce(t *testing.T) {
	w := dialogWindow(t)
	w.Page().SetDialogHandler(func(d *mygo.ScriptDialog) bool {
		d.Accept("now")
		return true
	})
	evalOK(t, w.Page(), `setTimeout(() => { window.answers = [confirm("now?"), prompt("now?")]; }, 0)`)
	waitFor(t, w, `window.answers && window.answers[0] === true && window.answers[1] === "now"`)
}
