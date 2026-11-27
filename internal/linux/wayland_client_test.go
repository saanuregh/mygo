//go:build linux && !android && (amd64 || arm64)

package linux

import (
	"os"
	"testing"
	"unsafe"

	"github.com/ebitengine/purego"
)

// The structs are wayland-util.h's on LP64 (amd64, arm64): libwayland reads
// the interfaces built in Go's layout, so a wrong offset is a crash.
func TestWaylandInterfaceLayout(t *testing.T) {
	var m wlMessage
	var i wlInterface
	for _, c := range []struct {
		what      string
		got, want uintptr
	}{
		{"sizeof(struct wl_message)", unsafe.Sizeof(m), 24},
		{"wl_message.name", unsafe.Offsetof(m.name), 0},
		{"wl_message.signature", unsafe.Offsetof(m.signature), 8},
		{"wl_message.types", unsafe.Offsetof(m.types), 16},
		{"sizeof(struct wl_interface)", unsafe.Sizeof(i), 40},
		{"wl_interface.name", unsafe.Offsetof(i.name), 0},
		{"wl_interface.version", unsafe.Offsetof(i.version), 8},
		{"wl_interface.method_count", unsafe.Offsetof(i.methodCount), 12},
		{"wl_interface.methods", unsafe.Offsetof(i.methods), 16},
		{"wl_interface.event_count", unsafe.Offsetof(i.eventCount), 24},
		{"wl_interface.events", unsafe.Offsetof(i.events), 32},
	} {
		if c.got != c.want {
			t.Errorf("%s = %d, want %d", c.what, c.got, c.want)
		}
	}

	c, err := waylandClientAsGDKLoadsIt()
	if err != nil {
		t.Skipf("no libwayland-client: %v", err)
	}
	// libwayland's own interface, read through the Go layout.
	type message struct {
		name, signature string
		types           []string // interface names, "" for NULL
	}
	read := func(array ptr, n int32, args []int) []message {
		var out []message
		if n == 0 {
			return out
		}
		for k, m := range unsafe.Slice((*wlMessage)(cAt(array)), n) {
			got := message{name: goStr(m.name), signature: goStr(m.signature)}
			for _, t := range unsafe.Slice((*ptr)(cAt(m.types)), args[k]) {
				if t == 0 {
					got.types = append(got.types, "")
				} else {
					got.types = append(got.types, goStr((*wlInterface)(cAt(t)).name))
				}
			}
			out = append(out, got)
		}
		return out
	}
	sub := (*wlInterface)(cAt(c.subsurfaceInterface))
	if name := goStr(sub.name); name != "wl_subsurface" || sub.methodCount != 6 || sub.eventCount != 0 {
		t.Fatalf("libwayland's wl_subsurface_interface read as %q with %d requests and %d events, want wl_subsurface, 6, 0",
			name, sub.methodCount, sub.eventCount)
	}
	if got := read(sub.methods, sub.methodCount, []int{0, 2, 1, 1, 0, 0}); got[1].name != "set_position" || got[1].signature != "ii" ||
		got[2].name != "place_above" || got[2].signature != "o" || got[2].types[0] != "wl_surface" {
		t.Errorf("wl_subsurface's requests read as %+v", got)
	}

	// The built interfaces, as viewporter.xml and fractional-scale-v1.xml
	// define them (wayland-protocols 1.50, wayland-scanner private-code).
	type want struct {
		iface            ptr
		name             string
		version          int32
		requests, events []message
		reqArgs, evArgs  []int
	}
	for _, w := range []want{
		{iface: c.viewporterInterface, name: "wp_viewporter", version: 1,
			requests: []message{{"destroy", "", nil}, {"get_viewport", "no", []string{"wp_viewport", "wl_surface"}}},
			reqArgs:  []int{0, 2}},
		{iface: c.viewportInterface, name: "wp_viewport", version: 1,
			requests: []message{{"destroy", "", nil}, {"set_source", "ffff", []string{"", "", "", ""}}, {"set_destination", "ii", []string{"", ""}}},
			reqArgs:  []int{0, 4, 2}},
		{iface: c.fractionManagerInterface, name: "wp_fractional_scale_manager_v1", version: 1,
			requests: []message{{"destroy", "", nil}, {"get_fractional_scale", "no", []string{"wp_fractional_scale_v1", "wl_surface"}}},
			reqArgs:  []int{0, 2}},
		{iface: c.fractionInterface, name: "wp_fractional_scale_v1", version: 1,
			requests: []message{{"destroy", "", nil}}, reqArgs: []int{0},
			events: []message{{"preferred_scale", "u", []string{""}}}, evArgs: []int{1}},
	} {
		i := (*wlInterface)(cAt(w.iface))
		if got := goStr(i.name); got != w.name || i.version != w.version ||
			int(i.methodCount) != len(w.requests) || int(i.eventCount) != len(w.events) {
			t.Errorf("%s: built as %q version %d with %d requests and %d events, want version %d, %d and %d",
				w.name, got, i.version, i.methodCount, i.eventCount, w.version, len(w.requests), len(w.events))
			continue
		}
		if (i.events == 0) != (len(w.events) == 0) {
			t.Errorf("%s: events at %#x with %d events", w.name, i.events, len(w.events))
		}
		for _, side := range []struct {
			what      string
			got, want []message
		}{
			{"request", read(i.methods, i.methodCount, w.reqArgs), w.requests},
			{"event", read(i.events, i.eventCount, w.evArgs), w.events},
		} {
			for k := range side.want {
				g, x := side.got[k], side.want[k]
				if g.name != x.name || g.signature != x.signature || len(g.types) != len(x.types) {
					t.Errorf("%s %s %d: %+v, want %+v", w.name, side.what, k, g, x)
					continue
				}
				for a := range x.types {
					if g.types[a] != x.types[a] {
						t.Errorf("%s.%s argument %d: interface %q, want %q", w.name, x.name, a, g.types[a], x.types[a])
					}
				}
			}
		}
	}
}

// waylandClientAsGDKLoadsIt loads libwayland-client as an app finds it:
// already loaded by libgdk-3, whose run path names it where the library
// path may not.
func waylandClientAsGDKLoadsIt() (*waylandClient, error) {
	if _, err := open("libgdk-3.so.0"); err != nil {
		return nil, err
	}
	return loadWaylandClient()
}

// The globals a subsurface needs, bound on a compositor the test connects
// to itself: MYGO_TEST_WAYLAND_DISPLAY names its socket in XDG_RUNTIME_DIR
// (a headless sway, say), never the desktop's own display by default.
func TestWaylandGlobalsUnderSway(t *testing.T) {
	name := os.Getenv("MYGO_TEST_WAYLAND_DISPLAY")
	if name == "" {
		t.Skip("MYGO_TEST_WAYLAND_DISPLAY is not set")
	}
	c, err := waylandClientAsGDKLoadsIt()
	if err != nil {
		t.Fatal(err)
	}
	lib, err := open("libwayland-client.so.0")
	if err != nil {
		t.Fatal(err)
	}
	var connect func(name *byte) ptr
	var disconnect func(display ptr)
	mustBind(lib, &connect, "wl_display_connect")
	mustBind(lib, &disconnect, "wl_display_disconnect")
	display := connect(cs(name))
	if display == 0 {
		t.Fatalf("cannot connect to %s in %s", name, os.Getenv("XDG_RUNTIME_DIR"))
	}
	defer disconnect(display)

	g, err := c.findWaylandGlobals(display, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("found: %s", g)
	defer func() {
		for _, p := range []ptr{g.compositor, g.subcompositor, g.viewporter, g.fractionManager} {
			if p != 0 {
				c.proxyDestroy(p)
			}
		}
	}()
	if !g.complete() {
		t.Errorf("globals: %s, want all four", g)
	}
	// Moved to the default queue, as GDK's source dispatches it.
	var getQueue func(proxy ptr) ptr
	mustBind(lib, &getQueue, "wl_proxy_get_queue")
	var defaultQueue ptr
	if q, err := purego.Dlsym(lib, "wl_display_get_default_queue"); err == nil && q != 0 {
		r, _, _ := purego.SyscallN(q, display)
		defaultQueue = r
	}
	for _, p := range []ptr{g.compositor, g.subcompositor, g.viewporter, g.fractionManager} {
		if p != 0 && defaultQueue != 0 && getQueue(p) != defaultQueue {
			t.Errorf("a global left on another queue than the default")
		}
	}
	// The compositor took the binds: a roundtrip on the default queue
	// reports no protocol error.
	var roundtrip func(display ptr) int32
	mustBind(lib, &roundtrip, "wl_display_roundtrip")
	if roundtrip(display) < 0 {
		t.Errorf("roundtrip after binding: error %d", c.displayGetError(display))
	}
}
