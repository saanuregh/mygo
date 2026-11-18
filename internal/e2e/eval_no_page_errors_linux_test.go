//go:build linux && (amd64 || arm64)

package e2e

import (
	"context"
	"testing"

	"github.com/egoist/mygo"
)

const errorsPage = `<!doctype html><title>errors</title>
<script>window.seen = [];
window.onerror = (m) => { window.seen.push(String(m)); };
window.addEventListener("error", (e) => window.seen.push("event: " + e.message), true);</script>`

// TestEvalNoPageErrors runs a statement and an expression with Eval and
// EvalInWorld: deciding which one the code is reaches none of the page's
// error handlers.
func TestEvalNoPageErrors(t *testing.T) {
	w := newWindow(t, mygo.WindowOptions{Hidden: true})
	p := w.Page()
	p.LoadHTML(errorsPage, "http://errors.test/")
	waitForPage(t, p, "Array.isArray(window.seen)")

	if _, err := p.Eval("let a = 1; a += 1"); err != nil {
		t.Fatalf("Eval of a statement: %v", err)
	}
	if v, err := p.Eval("1 + 1"); err != nil || v != float64(2) {
		t.Fatalf("Eval of an expression = %v, %v", v, err)
	}
	if v, err := p.Eval("const b = 3; return b"); err != nil || v != float64(3) {
		t.Fatalf("Eval of a body that returns = %v, %v", v, err)
	}
	for _, world := range []string{"", "zode"} {
		if _, err := p.EvalInWorld(context.Background(), world, "var c = 1; c++"); err != nil {
			t.Fatalf("EvalInWorld(%q) of a statement: %v", world, err)
		}
	}
	// A timer lets an error event queued by the calls above arrive first.
	if v, err := p.Eval("new Promise(r => setTimeout(() => r(window.seen), 50))"); err != nil || len(v.([]any)) != 0 {
		t.Errorf("the page's error handlers saw %v (%v), want nothing", v, err)
	}
}
