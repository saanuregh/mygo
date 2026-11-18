//go:build linux && (amd64 || arm64)

package e2e

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

const worldPage = `<!doctype html><title>worlds</title><h1>Hello</h1>
<button onclick="window.clicked = true">go</button>
<script>window.secret = 42;</script>`

// worldScript is the world script of the tests' pages: it runs in world
// zode as each document starts.
var worldScript = mygo.WorldScript{World: "zode", Source: `globalThis.layer = "loaded"`}

// evalIn is EvalInWorld that fails the test on an error.
func evalIn(t *testing.T, p *mygo.Page, world, code string) any {
	t.Helper()
	v, err := p.EvalInWorld(context.Background(), world, code)
	if err != nil {
		t.Fatalf("EvalInWorld(%q, %q): %v", world, code, err)
	}
	return v
}

// TestEvalInWorld runs scripts in an isolated world of a real page, a
// hidden window's and a never placed untrusted web view's: they share the
// page's DOM, not its globals, keep their state between calls, and Eval
// stays in the main world.
func TestEvalInWorld(t *testing.T) {
	opts := mygo.PageOptions{WorldScripts: []mygo.WorldScript{worldScript}}
	t.Run("window", func(t *testing.T) {
		w := newWindow(t, mygo.WindowOptions{Hidden: true, Page: opts})
		checkWorlds(t, w.Page())
	})
	t.Run("web view", func(t *testing.T) {
		var frames atomic.Int32
		w := newWindow(t, mygo.WindowOptions{Title: "Worlds", Width: 400, Height: 300, Content: ui.View(func(c *ui.Context) {
			frames.Add(1)
			ui.Text(c, "no web view placed")
		})})
		eventually(t, "a frame", func() bool { return frames.Load() > 0 })
		untrusted := opts
		untrusted.Untrusted = true
		v, err := w.NewWebView(mygo.WebViewOptions{Page: untrusted})
		if err != nil {
			t.Fatal(err)
		}
		checkWorlds(t, v.Page())
	})
}

func checkWorlds(t *testing.T, p *mygo.Page) {
	t.Helper()
	p.LoadHTML(worldPage, "http://worlds.test/")
	waitForPage(t, p, "window.secret === 42")

	if v := evalIn(t, p, "zode", "typeof secret"); v != "undefined" {
		t.Errorf("the page's global in world zode: typeof secret = %v", v)
	}
	if v := evalIn(t, p, "zode", "document.querySelector('h1').textContent"); v != "Hello" {
		t.Errorf("the DOM from world zode: %v", v)
	}
	evalIn(t, p, "zode", "document.body.dataset.fromWorld = 'yes'; document.querySelector('button').click()")
	if v, err := p.Eval("[document.body.dataset.fromWorld, window.clicked]"); err != nil || len(v.([]any)) != 2 || v.([]any)[0] != "yes" || v.([]any)[1] != true {
		t.Errorf("the main world after world zode changed the DOM and clicked: %v, %v", v, err)
	}
	if v := evalIn(t, p, "zode", "new Promise(r => setTimeout(() => r({n: 42}), 10))"); v == nil || v.(map[string]any)["n"] != float64(42) {
		t.Errorf("an awaited promise in world zode = %v", v)
	}

	// State: the world script's, and one call's for the next.
	if v := evalIn(t, p, "zode", "layer"); v != "loaded" {
		t.Errorf("the world script's global in world zode = %v", v)
	}
	evalIn(t, p, "zode", "globalThis.kept = 7")
	if v := evalIn(t, p, "zode", "typeof kept === 'undefined' ? 'lost' : kept"); v != float64(7) {
		t.Errorf("state set in world zode, seen by the next EvalInWorld: %v", v)
	}

	// The main world sees neither, and Eval is unchanged.
	if v, err := p.Eval("[typeof layer, typeof kept, secret]"); err != nil || len(v.([]any)) != 3 ||
		v.([]any)[0] != "undefined" || v.([]any)[1] != "undefined" || v.([]any)[2] != float64(42) {
		t.Errorf("the main world: [typeof layer, typeof kept, secret] = %v, %v", v, err)
	}
	if v, err := p.EvalInWorld(context.Background(), "", "secret"); err != nil || v != float64(42) {
		t.Errorf(`EvalInWorld("") = %v, %v, want the main world's 42`, v, err)
	}

	// Errors as Eval's: a syntax error, and a throw.
	for _, world := range []string{"", "zode"} {
		_, err := p.EvalInWorld(context.Background(), world, "return )")
		if err == nil || !strings.Contains(err.Error(), "SyntaxError") {
			t.Errorf("world %q: a syntax error = %v, want SyntaxError", world, err)
		}
		var evalErr *mygo.EvalError
		if _, err := p.EvalInWorld(context.Background(), world, "throw new Error('boom')"); !errors.As(err, &evalErr) || evalErr.Message != "boom" {
			t.Errorf("world %q: a throw = %v", world, err)
		}
	}

	// A new document runs the world script again, with fresh state.
	p.Reload()
	waitForPage(t, p, "window.secret === 42 && !document.body.dataset.fromWorld")
	if v := evalIn(t, p, "zode", "[layer, typeof kept]"); len(v.([]any)) != 2 || v.([]any)[0] != "loaded" || v.([]any)[1] != "undefined" {
		t.Errorf("world zode after a reload: [layer, typeof kept] = %v", v)
	}
}
