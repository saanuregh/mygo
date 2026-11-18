//go:build js

package web

import (
	"net/url"
	"strings"
	"syscall/js"

	"github.com/egoist/mygo/internal/platform"
)

// The sandboxes of iframes. A page of another origin, as a dev server
// on another port, keeps its origin, and with it its storage and cookies,
// but cannot navigate the app away (no allow-top-navigation). Anything
// else, HTML given to LoadHTML, a page of the app's own origin, data:,
// blob: and about: URLs, gets an opaque origin: scripts with the app's
// origin could reach into the app.
const (
	crossOriginSandbox = "allow-scripts allow-same-origin allow-forms allow-popups allow-popups-to-escape-sandbox allow-modals allow-downloads"
	opaqueSandbox      = "allow-scripts"
)

// webView is a web view in a browser: an iframe over the window's canvas,
// which PlaceWebViews places. Its page is usually of another origin, which
// the app cannot reach into: it navigates, reloads and hears loads, and
// the rest (Eval, capture, history, zoom) is unsupported.
type webView struct {
	parent, frame js.Value
	// origin is the app's, which no page of the iframe may have.
	origin string
	h      platform.WindowHandler
	run    func(func())
	onLoad js.Func
	// onClose is called once Close removed the iframe.
	onClose func()

	url     string
	html    *string // LoadHTML's document, shown in srcdoc
	loading bool
	closed  bool
}

// newWebView adds the iframe of a web view to parent, hidden until placed.
// run calls the handler on the main goroutine.
func newWebView(parent js.Value, origin string, h platform.WindowHandler, run func(func())) *webView {
	v := &webView{parent: parent, origin: origin, h: h, run: run}
	v.frame = js.Global().Get("document").Call("createElement", "iframe")
	st := v.frame.Get("style")
	for k, val := range map[string]string{
		"position": "absolute", "left": "0", "top": "0", "border": "0", "margin": "0", "padding": "0",
		"transformOrigin": "0 0", "display": "none", "colorScheme": "normal",
	} {
		st.Set(k, val)
	}
	v.frame.Call("setAttribute", "sandbox", opaqueSandbox)
	v.onLoad = js.FuncOf(func(js.Value, []js.Value) any {
		v.run(v.loaded)
		return nil
	})
	v.frame.Call("addEventListener", "load", v.onLoad)
	parent.Call("appendChild", v.frame)
	return v
}

// loaded tells the handler the iframe's page loaded: a navigation inside
// it too, whose URL an app cannot read across origins, and which reports
// the URL the app loaded.
func (v *webView) loaded() {
	if v.closed || (v.url == "" && v.html == nil) {
		return // the iframe's first, blank document
	}
	v.loading = false
	u := v.url
	if v.html != nil {
		u = "about:blank"
	}
	v.h.NavigationCommitted(u)
	v.h.LoadFinished()
}

// place shows the iframe where p says: at the page's own size, scaled to
// Frame, cut to Clip less Covers.
func (v *webView) place(p *platform.WebViewPlacement) {
	w, h := p.Size()
	sx, sy := p.Scale()
	st := v.frame.Get("style")
	st.Set("display", "")
	st.Set("left", ftoa(p.Frame.X)+"px")
	st.Set("top", ftoa(p.Frame.Y)+"px")
	st.Set("width", ftoa(w)+"px")
	st.Set("height", ftoa(h)+"px")
	st.Set("transform", "scale("+ftoa(sx)+","+ftoa(sy)+")")
	r := p.Radii
	st.Set("borderRadius", ftoa(r[0])+"px "+ftoa(r[1])+"px "+ftoa(r[2])+"px "+ftoa(r[3])+"px")
	st.Set("clipPath", iframeClipPath(p))
	if p.Opacity > 0 && p.Opacity < 1 {
		st.Set("opacity", ftoa(p.Opacity))
	} else {
		st.Set("opacity", "")
	}
}

// hide hides the iframe, which keeps its page.
func (v *webView) hide() { v.frame.Get("style").Set("display", "none") }

// sandboxFor returns the sandbox of a page at raw, and false for a URL an
// iframe must not load: javascript: runs in the app's origin.
func (v *webView) sandboxFor(raw string) (string, bool) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", false
	}
	switch strings.ToLower(u.Scheme) {
	case "javascript", "":
		return "", false
	case "http", "https":
		if pageOrigin := strings.ToLower(u.Scheme + "://" + u.Host); pageOrigin != strings.ToLower(v.origin) {
			return crossOriginSandbox, true
		}
	}
	return opaqueSandbox, true
}

func (v *webView) LoadURL(raw string) {
	if v.closed {
		return
	}
	sandbox, ok := v.sandboxFor(raw)
	if !ok {
		v.h.LoadFailed(raw, 0, "mygo: browsers do not load this URL in a web view")
		return
	}
	v.url, v.html, v.loading = raw, nil, true
	// The sandbox applies from the next navigation: set it first.
	v.frame.Call("setAttribute", "sandbox", sandbox)
	v.frame.Call("removeAttribute", "srcdoc")
	v.frame.Set("srcdoc", "")
	v.frame.Call("setAttribute", "src", raw)
}

// LoadHTML shows html in srcdoc, with an opaque origin; a base URL becomes
// the document's <base>.
func (v *webView) LoadHTML(html, baseURL string) {
	if v.closed {
		return
	}
	if baseURL != "" {
		html = `<base href="` + attrEscaper.Replace(baseURL) + `">` + html
	}
	v.url, v.html, v.loading = "", &html, true
	v.frame.Call("setAttribute", "sandbox", opaqueSandbox)
	v.frame.Call("removeAttribute", "src")
	v.frame.Set("srcdoc", html)
}

var attrEscaper = strings.NewReplacer(`&`, "&amp;", `"`, "&quot;", `<`, "&lt;", `>`, "&gt;")

func (v *webView) LoadFile(path, _ string) {
	v.h.LoadFailed("file://"+path, 0, errUnsupported.Error())
}

// Reload loads the page again: the URL the app loaded, as the iframe's own
// navigations cannot be read.
func (v *webView) Reload(bool) {
	switch {
	case v.closed:
	case v.html != nil:
		v.loading = true
		v.frame.Set("srcdoc", *v.html)
	case v.url != "":
		v.loading = true
		v.frame.Call("setAttribute", "src", v.url)
	}
}

func (v *webView) StopLoading() {
	if !v.closed && v.loading {
		v.loading = false
		v.frame.Call("setAttribute", "src", "about:blank")
	}
}

func (v *webView) Focus() {
	if !v.closed {
		v.frame.Call("focus")
	}
}

func (v *webView) TabInto(bool) { v.Focus() }

func (v *webView) SetBackgroundColor(c platform.Color) {
	v.frame.Get("style").Set("background", "rgba("+itoa(int(c.R))+","+itoa(int(c.G))+","+itoa(int(c.B))+","+ftoa(float64(c.A)/255)+")")
}

func (v *webView) Close() {
	if v.closed {
		return
	}
	v.closed = true
	v.frame.Call("removeEventListener", "load", v.onLoad)
	v.frame.Call("remove")
	// A load already queued may still call it once: it finds the view closed.
	load := v.onLoad
	var release js.Func
	release = js.FuncOf(func(js.Value, []js.Value) any {
		load.Release()
		release.Release()
		return nil
	})
	js.Global().Call("setTimeout", release, 1000)
	if v.onClose != nil {
		v.onClose()
	}
}

func (v *webView) URL() string {
	if v.html != nil {
		return "about:blank"
	}
	return v.url
}

func (v *webView) WebViewHandle() uintptr               { return 0 }
func (v *webView) GoBack()                              {}
func (v *webView) GoForward()                           {}
func (v *webView) CanGoBack() bool                      { return false }
func (v *webView) CanGoForward() bool                   { return false }
func (v *webView) IsLoading() bool                      { return v.loading }
func (v *webView) Eval(string)                          {}
func (v *webView) SetUserScripts([]platform.UserScript) {}
func (v *webView) SetContentRules([]any)                {}
func (v *webView) SetZoom(float64)                      {}
func (v *webView) Zoom() float64                        { return 1 }
func (v *webView) SetUserAgent(string)                  {}
func (v *webView) UserAgent() string                    { return "" }
func (v *webView) ProcessID() int                       { return 0 }
func (v *webView) GPUProcessID() int                    { return 0 }
func (v *webView) IsAudible() bool                      { return false }
func (v *webView) SetAudioMuted(bool)                   {}
func (v *webView) IsAudioMuted() bool                   { return false }
func (v *webView) TogglePictureInPicture()              {}
func (v *webView) SessionState() []byte                 { return nil }
func (v *webView) RestoreSessionState([]byte) bool      { return false }
func (v *webView) OpenDevTools()                        {}
func (v *webView) CloseDevTools()                       {}
func (v *webView) IsDevToolsOpened() bool               { return false }
func (v *webView) DroppedFiles() []string               { return nil }
func (v *webView) Print()                               {}
func (v *webView) PictureInPictureActive(cb func(bool)) { cb(false) }
func (v *webView) CaptureState(platform.CaptureKind) platform.CaptureState {
	return platform.CaptureNone
}
func (v *webView) SetCaptureState(platform.CaptureKind, platform.CaptureState) {}

// later calls fn on the main goroutine after the caller returns, as
// backends answer asynchronous calls.
func (v *webView) later(fn func()) { go v.run(fn) }

func (v *webView) CallAsyncFunction(_, _ string, cb func(string, error)) {
	v.later(func() { cb("", errUnsupported) })
}

func (v *webView) CapturePage(cb func([]byte, error)) {
	v.later(func() { cb(nil, errUnsupported) })
}

func (v *webView) PrintToPDF(_ platform.PDFOptions, cb func([]byte, error)) {
	v.later(func() { cb(nil, errUnsupported) })
}
