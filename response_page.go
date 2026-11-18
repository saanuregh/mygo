package mygo

import "github.com/egoist/mygo/internal/platform"

// ResponseEvent is passed to Page.OnWillLoadResponse listeners: the
// response of a document the page's main frame is about to show (or
// download), with its MIME type. Preventing it drops the response: the
// page stays as it was, as after a navigation that was not taken.
type ResponseEvent struct {
	Preventable
	URL      string
	MIMEType string
}

// OnWillLoadResponse is called on the main thread when the response of a
// document the page navigates to arrives, before the web view shows or
// downloads it: an app can send a type it does not want shown (a PDF)
// elsewhere, which OnWillNavigate cannot see from the URL alone.
//
//	page.OnWillLoadResponse(func(e *mygo.ResponseEvent) {
//		if e.MIMEType == "application/pdf" {
//			e.PreventDefault()
//			openWithSystem(e.URL)
//		}
//	})
//
// Linux only (WebKitGTK 2.40+, the response policy decision); macOS,
// Windows and browsers never call it.
func (p *Page) OnWillLoadResponse(fn func(e *ResponseEvent)) (off func()) {
	return p.w.onWillLoadResponse.add(fn, false)
}

func (h *windowHandler) WillLoadResponse(r platform.Response) bool {
	if !r.IsMainFrame || h.w.onWillLoadResponse.len() == 0 {
		return true
	}
	e := &ResponseEvent{URL: r.URL, MIMEType: r.MIMEType}
	fire1(&h.w.onWillLoadResponse, e)
	return !e.prevented
}
