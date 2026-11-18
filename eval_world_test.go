//go:build !js

package mygo

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// TestEvalInWorld: EvalInWorld runs in the world it names, the check of
// which form the code is included, and Eval stays in the main world.
func TestEvalInWorld(t *testing.T) {
	w, fw := testWindow(t, WindowOptions{})
	fw.AsyncFunction = func(body string) (string, error) {
		if strings.HasPrefix(body, "const F=") {
			if strings.Contains(body, "let x") {
				return "body", nil
			}
			return "expr", nil
		}
		if strings.Contains(body, "await(\nlet x = 1; return x\n)") {
			return "", errors.New("SyntaxError: Unexpected token")
		}
		return `{"ok":true,"v":1}`, nil
	}
	if v, err := w.Page().EvalInWorld(context.Background(), "zode", "1"); err != nil || v != float64(1) {
		t.Errorf("EvalInWorld = %v, %v", v, err)
	}
	if _, err := w.Page().EvalInWorld(context.Background(), "zode", "let x = 1; return x"); err != nil {
		t.Errorf("EvalInWorld statements = %v", err)
	}
	if _, err := w.Page().Eval("1"); err != nil {
		t.Errorf("Eval = %v", err)
	}
	if want := []string{"zode", "zode", "zode", "zode", "", ""}; !slices.Equal(fw.AsyncWorlds, want) {
		t.Errorf("worlds = %q, want %q", fw.AsyncWorlds, want)
	}
}

// TestWorldScripts: PageOptions.WorldScripts reach the backend as user
// scripts in their worlds, after the page's own.
func TestWorldScripts(t *testing.T) {
	_, fw := testWindow(t, WindowOptions{Page: PageOptions{
		PreloadScript: "window.main = 1",
		WorldScripts:  []WorldScript{{World: "zode", Source: "globalThis.layer = 1"}},
	}})
	scripts := onMainValue(func() []platform.UserScript { return slices.Clone(fw.Opts.UserScripts) })
	last := scripts[len(scripts)-1]
	if last.World != "zode" || last.Source != "globalThis.layer = 1" || last.AllFrames || last.AtDocumentEnd {
		t.Errorf("the last script = %+v", last)
	}
	for _, s := range scripts[:len(scripts)-1] {
		if s.World != "" {
			t.Errorf("%q runs in world %q, want the main world", s.Source, s.World)
		}
	}
}
