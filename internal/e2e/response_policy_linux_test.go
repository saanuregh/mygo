//go:build linux && (amd64 || arm64)

package e2e

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/egoist/mygo"
)

// TestWillLoadResponse follows a link in a real page to a PDF whose
// address does not say so: OnWillLoadResponse sees its MIME type and drops
// it, so the page stays; an HTML document's response passes.
func TestWillLoadResponse(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!doctype html><title>start</title><a id=doc href="/served">doc</a><a id=next href="/next">next</a>`)
	})
	mux.HandleFunc("/served", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		w.Write([]byte("%PDF-1.4\n%%EOF\n"))
	})
	mux.HandleFunc("/next", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!doctype html><title>next</title>`)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	w := newWindow(t, mygo.WindowOptions{Hidden: true})
	p := w.Page()
	var mu sync.Mutex
	var seen []string
	p.OnWillLoadResponse(func(e *mygo.ResponseEvent) {
		mu.Lock()
		seen = append(seen, e.MIMEType+" "+e.URL)
		mu.Unlock()
		if e.MIMEType == "application/pdf" {
			e.PreventDefault()
		}
	})
	got := func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), seen...)
	}
	if err := p.LoadURL(srv.URL + "/"); err != nil {
		t.Fatal(err)
	}
	waitForPage(t, p, "document.title === 'start'")
	if _, err := p.Eval("document.getElementById('doc').click()"); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the PDF's response", func() bool {
		for _, s := range got() {
			if s == "application/pdf "+srv.URL+"/served" {
				return true
			}
		}
		return false
	})
	waitForPage(t, p, "document.title === 'start'")
	if u := p.URL(); u != srv.URL+"/" {
		t.Errorf("the page after the PDF was dropped: %q", u)
	}
	if _, err := p.Eval("document.getElementById('next').click()"); err != nil {
		t.Fatal(err)
	}
	waitForPage(t, p, "document.title === 'next'")
	want := []string{"text/html " + srv.URL + "/", "application/pdf " + srv.URL + "/served", "text/html " + srv.URL + "/next"}
	if s := got(); fmt.Sprint(s) != fmt.Sprint(want) {
		t.Errorf("responses seen %q, want %q", s, want)
	}
}
