//go:build js

package web

import (
	"syscall/js"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// domStub is the little of the DOM a web view uses, for Node, which has
// none: elements with attributes, a style, listeners and children.
const domStub = `(() => {
	const el = (tag) => {
		const e = {tagName: tag.toUpperCase(), style: {}, attrs: {}, listeners: {}, children: [], parent: null, focused: false};
		e.setAttribute = (k, v) => { e.attrs[k] = String(v); };
		e.getAttribute = (k) => (k in e.attrs ? e.attrs[k] : null);
		e.hasAttribute = (k) => k in e.attrs;
		e.removeAttribute = (k) => { delete e.attrs[k]; };
		e.addEventListener = (t, f) => { (e.listeners[t] ||= []).push(f); };
		e.removeEventListener = (t, f) => { e.listeners[t] = (e.listeners[t] || []).filter((g) => g !== f); };
		e.appendChild = (c) => { c.parent = e; e.children.push(c); return c; };
		e.remove = () => { if (e.parent) { e.parent.children = e.parent.children.filter((c) => c !== e); e.parent = null; } };
		e.focus = () => { e.focused = true; };
		e.fire = (t) => { for (const f of e.listeners[t] || []) f({type: t}); };
		return e;
	};
	return {createElement: el, root: el("div")};
})()`

// navHandler records what a web view tells its page.
type navHandler struct {
	platform.WindowHandler
	events []string
}

func (h *navHandler) NavigationCommitted(url string) { h.events = append(h.events, "commit "+url) }
func (h *navHandler) LoadFinished()                  { h.events = append(h.events, "finish") }
func (h *navHandler) LoadFailed(url string, code int, desc string) {
	h.events = append(h.events, "fail "+url)
}

func (h *navHandler) take() []string { e := h.events; h.events = nil; return e }

func TestWebViewIframe(t *testing.T) {
	stub := js.Global().Call("eval", domStub)
	js.Global().Set("document", stub)
	defer js.Global().Delete("document")
	root := stub.Get("root")
	h := &navHandler{}
	v := newWebView(root, "http://zode.test:8080", h, func(fn func()) { fn() })

	f := v.frame
	if root.Get("children").Length() != 1 || !root.Get("children").Index(0).Equal(f) || f.Get("tagName").String() != "IFRAME" {
		t.Fatal("the web view's iframe is not in the window")
	}
	attr := func(k string) string {
		if a := f.Call("getAttribute", k); !a.IsNull() {
			return a.String()
		}
		return "<none>"
	}
	style := func(k string) string {
		if s := f.Get("style").Get(k); !s.IsUndefined() {
			return s.String()
		}
		return ""
	}
	want := func(what, got, want string) {
		t.Helper()
		if got != want {
			t.Errorf("%s = %q, want %q", what, got, want)
		}
	}
	events := func(want ...string) {
		t.Helper()
		got := h.take()
		if len(got) != len(want) {
			t.Errorf("events %q, want %q", got, want)
			return
		}
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("events %q, want %q", got, want)
				return
			}
		}
	}
	want("display before it is placed", style("display"), "none")

	// Placed at half its size, a menu over it.
	v.place(&platform.WebViewPlacement{
		Frame: platform.RectF{X: 10, Y: 20, W: 100, H: 50}, Clip: platform.RectF{X: 10, Y: 20, W: 100, H: 50},
		Width: 200, Height: 100, Radii: [4]float64{8, 8, 0, 0}, Opacity: 1,
		Covers: []platform.RectF{{X: 10, Y: 20, W: 100, H: 25}},
	})
	want("display", style("display"), "")
	want("left", style("left"), "10px")
	want("top", style("top"), "20px")
	want("width", style("width"), "200px")
	want("height", style("height"), "100px")
	want("transform", style("transform"), "scale(0.5,0.5)")
	want("border radius", style("borderRadius"), "8px 8px 0px 0px")
	want("clip path", style("clipPath"), `path("M0 50H200V100H0Z")`)
	v.hide()
	want("display hidden", style("display"), "none")

	// A load event before anything loads is the iframe's first blank page.
	f.Call("fire", "load")
	events()

	// The preview port is another origin: it keeps its own, and cannot
	// navigate the app away.
	v.LoadURL("http://zode.test:4000/")
	want("src", attr("src"), "http://zode.test:4000/")
	want("sandbox of another origin", attr("sandbox"), crossOriginSandbox)
	// The page's address (a ticket, a token) is no frame's referrer.
	want("referrer policy", attr("referrerpolicy"), "no-referrer")
	if !v.IsLoading() || v.URL() != "http://zode.test:4000/" {
		t.Errorf("loading %v at %q", v.IsLoading(), v.URL())
	}
	f.Call("fire", "load")
	events("commit http://zode.test:4000/", "finish")
	if v.IsLoading() {
		t.Error("still loading after the load event")
	}

	// A page of the app's own origin gets an opaque one: never scripts
	// with the app's origin.
	v.LoadURL("http://zode.test:8080/other")
	want("sandbox of the app's origin", attr("sandbox"), "allow-scripts")
	for _, u := range []string{"data:text/html,<p>x", "about:blank", "blob:http://zode.test:8080/1"} {
		v.LoadURL(u)
		want("sandbox of "+u, attr("sandbox"), "allow-scripts")
	}
	v.LoadURL("javascript:alert(1)")
	events("fail javascript:alert(1)")
	want("src after javascript:", attr("src"), "blob:http://zode.test:8080/1")

	// HTML goes in srcdoc, sandboxed to scripts alone.
	v.LoadHTML("<p>hi</p>", "")
	want("srcdoc", f.Get("srcdoc").String(), "<p>hi</p>")
	want("src with srcdoc", attr("src"), "<none>")
	want("sandbox of HTML", attr("sandbox"), "allow-scripts")
	f.Call("fire", "load")
	events("commit about:blank", "finish")

	// Reload loads the HTML again.
	f.Set("srcdoc", "")
	v.Reload(false)
	want("srcdoc after reload", f.Get("srcdoc").String(), "<p>hi</p>")

	v.LoadURL("http://zode.test:4000/a")
	if f.Get("srcdoc").Truthy() {
		t.Error("srcdoc kept after LoadURL")
	}
	v.Focus()
	if !f.Get("focused").Bool() {
		t.Error("Focus did not focus the iframe")
	}
	if v.CanGoBack() || v.CanGoForward() {
		t.Error("an iframe has no history to go through")
	}
	done := make(chan error, 1)
	v.CapturePage(func(_ []byte, err error) { done <- err })
	if err := <-done; err != errUnsupported {
		t.Errorf("CapturePage: %v, want errUnsupported", err)
	}

	closed := false
	v.onClose = func() { closed = true }
	v.Close()
	if root.Get("children").Length() != 0 || !closed {
		t.Error("Close left the iframe in the window")
	}
	f.Call("fire", "load")
	events()
}
