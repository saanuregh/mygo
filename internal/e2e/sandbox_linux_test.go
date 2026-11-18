//go:build linux && (amd64 || arm64)

package e2e

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/linux"
)

// TestWebProcessSandbox: the pages of the default session and of other
// sessions run in WebKit's sandbox (bubblewrap), and still answer Eval
// in both worlds.
func TestWebProcessSandbox(t *testing.T) {
	if os.Getenv("WEBKIT_DISABLE_SANDBOX_THIS_IS_DANGEROUS") != "" {
		t.Skip("WEBKIT_DISABLE_SANDBOX_THIS_IS_DANGEROUS turns WebKit's sandbox off")
	}
	for _, s := range []struct {
		name    string
		session *mygo.Session
	}{
		{"default", nil},
		{"persistent", mygo.NewSession("sandbox-test")},
		{"ephemeral", mygo.NewEphemeralSession()},
	} {
		t.Run(s.name, func(t *testing.T) {
			w := newWindow(t, mygo.WindowOptions{Hidden: true, Page: mygo.PageOptions{Session: s.session}})
			w.Page().LoadHTML(worldPage, "http://sandbox.test/")
			waitFor(t, w, "window.secret === 42")
			var on bool
			mygo.RunOnMain(func() { on = linux.TestSandboxEnabled(w.NativeHandle()) })
			if !on {
				t.Error("webkit_web_context_get_sandbox_enabled is false for the page's context")
			}
			if v := evalIn(t, w.Page(), "zode", "typeof secret"); v != "undefined" {
				t.Errorf("typeof secret in world zode = %v", v)
			}
			if n := sandboxedWebProcesses(t); n == 0 {
				t.Error("no WebKitWebProcess of this test runs under bwrap")
			}
		})
	}
}

// sandboxedWebProcesses counts this process's descendants that are bwrap
// running a WebKitWebProcess.
func sandboxedWebProcesses(t *testing.T) int {
	t.Helper()
	entries, err := os.ReadDir("/proc")
	if err != nil {
		t.Fatal(err)
	}
	parent := map[int]int{}
	cmdline := map[int]string{}
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		stat, err := os.ReadFile(filepath.Join("/proc", e.Name(), "stat"))
		if err != nil {
			continue
		}
		// pid (comm) state ppid ...: comm may hold spaces.
		fields := strings.Fields(string(stat[strings.LastIndexByte(string(stat), ')')+1:]))
		if len(fields) < 2 {
			continue
		}
		parent[pid], _ = strconv.Atoi(fields[1])
		b, _ := os.ReadFile(filepath.Join("/proc", e.Name(), "cmdline"))
		cmdline[pid] = strings.ReplaceAll(string(b), "\x00", " ")
	}
	self, n := os.Getpid(), 0
	for pid, cmd := range cmdline {
		if !strings.Contains(filepath.Base(strings.Fields(cmd + " x")[0]), "bwrap") || !strings.Contains(cmd, "WebKitWebProcess") {
			continue
		}
		for p := parent[pid]; p > 1; p = parent[p] {
			if p == self {
				n++
				break
			}
		}
	}
	return n
}
