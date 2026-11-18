//go:build !js

package mygo

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

func TestWillLoadResponse(t *testing.T) {
	w, fw := testWindow(t, WindowOptions{})
	ask := func(r platform.Response) bool {
		return onMainValue(func() bool { return fw.H.WillLoadResponse(r) })
	}
	pdf := platform.Response{URL: "http://x.test/doc", MIMEType: "application/pdf", IsMainFrame: true}
	// Without a listener every response loads.
	if !ask(pdf) {
		t.Fatal("a response was stopped without a listener")
	}
	var seen []ResponseEvent
	off := w.Page().OnWillLoadResponse(func(e *ResponseEvent) {
		if !isMainThread() {
			t.Error("the listener ran off the main thread")
		}
		seen = append(seen, ResponseEvent{URL: e.URL, MIMEType: e.MIMEType})
		if e.MIMEType == "application/pdf" {
			e.PreventDefault()
		}
	})
	if ask(pdf) {
		t.Error("a prevented response loaded")
	}
	if !ask(platform.Response{URL: "http://x.test/", MIMEType: "text/html", IsMainFrame: true}) {
		t.Error("an allowed response was stopped")
	}
	// A subframe's response is the page's own business.
	if !ask(platform.Response{URL: "http://x.test/f.pdf", MIMEType: "application/pdf"}) {
		t.Error("a subframe's response reached the listener")
	}
	if len(seen) != 2 || seen[0].URL != pdf.URL || seen[0].MIMEType != "application/pdf" || seen[1].MIMEType != "text/html" {
		t.Errorf("seen %+v", seen)
	}
	off()
	if !ask(pdf) {
		t.Error("a response was stopped after the listener went")
	}
}
