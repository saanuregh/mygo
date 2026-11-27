//go:build linux && !android && (amd64 || arm64)

package linux

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

// The Wayland subsurface: mygo's frames drawn on a wl_subsurface of the
// window's own, with EGL, in place of the GtkGLArea, which GTK 3 composites
// through an image as large as what it repaints. Off unless
// MYGO_WAYLAND_SUBSURFACE=1, and only on Wayland. This file has the
// libwayland-client calls it needs and the globals it binds.

// waylandSubsurfaceAsked reports MYGO_WAYLAND_SUBSURFACE=1, read once.
var waylandSubsurfaceAsked = sync.OnceValue(func() bool {
	return os.Getenv("MYGO_WAYLAND_SUBSURFACE") == "1"
})

// wlMessage is struct wl_message of wayland-util.h.
type wlMessage struct {
	name, signature ptr // const char *
	types           ptr // const struct wl_interface **, one per argument
}

// wlInterface is struct wl_interface of wayland-util.h: Go pads it as C
// does, 4 bytes after eventCount.
type wlInterface struct {
	name        ptr // const char *
	version     int32
	methodCount int32
	methods     ptr // const struct wl_message *
	eventCount  int32
	events      ptr // const struct wl_message *
}

// Requests' opcodes, their order in the protocols' XML.
const (
	wlDisplayGetRegistry = 1
	wlRegistryBind       = 0
)

// waylandClient is libwayland-client: the functions mygo calls, the
// interfaces it exports, and those of the protocols it does not have,
// built here as wayland-scanner would.
type waylandClient struct {
	// marshal sends a request: args is an array of union wl_argument, 8
	// bytes each, one per argument of the request's signature. For a new_id
	// it returns the new proxy, of iface at version.
	marshal                  func(proxy ptr, opcode uint32, iface ptr, version, flags uint32, args *uint64) ptr
	addListener              func(proxy ptr, implementation ptr, data ptr) int32
	createWrapper            func(proxy ptr) ptr
	wrapperDestroy           func(wrapper ptr)
	setQueue                 func(proxy, queue ptr)
	createQueue              func(display ptr) ptr
	roundtripQueue           func(display, queue ptr) int32
	queueDestroy             func(queue ptr)
	proxyDestroy             func(proxy ptr)
	proxyGetVersion          func(proxy ptr) uint32
	displayGetError          func(display ptr) int32
	registryInterface        ptr // &wl_registry_interface
	compositorInterface      ptr
	subcompositorInterface   ptr
	subsurfaceInterface      ptr
	surfaceInterface         ptr
	regionInterface          ptr
	callbackInterface        ptr
	viewporterInterface      ptr // built here: wp_viewporter, viewporter.xml
	viewportInterface        ptr
	fractionManagerInterface ptr // fractional-scale-v1.xml
	fractionInterface        ptr
	registryListener         ptr // struct wl_registry_listener
}

// loadWaylandClient loads libwayland-client, which GDK loaded already on
// Wayland, once.
var loadWaylandClient = sync.OnceValues(func() (*waylandClient, error) {
	lib, err := open("libwayland-client.so.0")
	if err != nil {
		return nil, err
	}
	c := &waylandClient{}
	funcs := []struct {
		fn   any
		name string
	}{
		{&c.marshal, "wl_proxy_marshal_array_flags"},
		{&c.addListener, "wl_proxy_add_listener"},
		{&c.createWrapper, "wl_proxy_create_wrapper"},
		{&c.wrapperDestroy, "wl_proxy_wrapper_destroy"},
		{&c.setQueue, "wl_proxy_set_queue"},
		{&c.createQueue, "wl_display_create_queue"},
		{&c.roundtripQueue, "wl_display_roundtrip_queue"},
		{&c.queueDestroy, "wl_event_queue_destroy"},
		{&c.proxyDestroy, "wl_proxy_destroy"},
		{&c.proxyGetVersion, "wl_proxy_get_version"},
		{&c.displayGetError, "wl_display_get_error"},
	}
	for _, f := range funcs {
		if !bind(lib, f.fn, f.name) {
			return nil, fmt.Errorf("libwayland-client has no %s", f.name)
		}
	}
	vars := []struct {
		addr *ptr
		name string
	}{
		{&c.registryInterface, "wl_registry_interface"},
		{&c.compositorInterface, "wl_compositor_interface"},
		{&c.subcompositorInterface, "wl_subcompositor_interface"},
		{&c.subsurfaceInterface, "wl_subsurface_interface"},
		{&c.surfaceInterface, "wl_surface_interface"},
		{&c.regionInterface, "wl_region_interface"},
		{&c.callbackInterface, "wl_callback_interface"},
	}
	for _, v := range vars {
		if *v.addr, err = purego.Dlsym(lib, v.name); err != nil || *v.addr == 0 {
			return nil, fmt.Errorf("libwayland-client has no %s", v.name)
		}
	}
	if err := c.buildProtocols(); err != nil {
		return nil, err
	}
	return c, nil
})

// cAlloc returns n zeroed bytes of C memory, never freed. The protocols'
// interfaces live in it because libwayland keeps pointers to them in every
// proxy for as long as the process runs: Go's collector neither moves nor
// frees C memory, and C memory holding pointers to C memory breaks none of
// the rules for Go pointers in C.
func cAlloc(n uintptr) (ptr, error) {
	calloc, err := purego.Dlsym(purego.RTLD_DEFAULT, "calloc")
	if err != nil || calloc == 0 {
		return 0, errors.New("no calloc")
	}
	p, _, _ := purego.SyscallN(calloc, 1, n)
	if p == 0 {
		return 0, errors.New("calloc failed")
	}
	return p, nil
}

// cAt is the C memory at p as a Go pointer.
func cAt(p ptr) unsafe.Pointer { return *(*unsafe.Pointer)(unsafe.Pointer(&p)) }

// allocCString copies s, NUL terminated, into C memory.
func allocCString(s string) (ptr, error) {
	p, err := cAlloc(uintptr(len(s) + 1))
	if err != nil {
		return 0, err
	}
	copy(unsafe.Slice((*byte)(cAt(p)), len(s)), s)
	return p, nil
}

// protocolMessage is a request or event of a protocol as its XML has it:
// its signature, and the interface of each argument that is an object or a
// new_id ("" for the others).
type protocolMessage struct {
	name, signature string
	types           []string
}

// protocolInterface is an interface of a protocol as its XML has it.
type protocolInterface struct {
	name     string
	version  int32
	requests []protocolMessage
	events   []protocolMessage
	built    *ptr
}

// protocols are the interfaces of viewporter.xml (stable) and
// fractional-scale-v1.xml (staging), as wayland-scanner's private-code
// writes them.
func (c *waylandClient) protocols() []protocolInterface {
	return []protocolInterface{
		{name: "wp_viewporter", version: 1, built: &c.viewporterInterface, requests: []protocolMessage{
			{"destroy", "", nil},
			{"get_viewport", "no", []string{"wp_viewport", "wl_surface"}},
		}},
		{name: "wp_viewport", version: 1, built: &c.viewportInterface, requests: []protocolMessage{
			{"destroy", "", nil},
			{"set_source", "ffff", []string{"", "", "", ""}},
			{"set_destination", "ii", []string{"", ""}},
		}},
		{name: "wp_fractional_scale_manager_v1", version: 1, built: &c.fractionManagerInterface, requests: []protocolMessage{
			{"destroy", "", nil},
			{"get_fractional_scale", "no", []string{"wp_fractional_scale_v1", "wl_surface"}},
		}},
		{name: "wp_fractional_scale_v1", version: 1, built: &c.fractionInterface,
			requests: []protocolMessage{{"destroy", "", nil}},
			events:   []protocolMessage{{"preferred_scale", "u", []string{""}}},
		},
	}
}

// buildProtocols builds the interfaces of protocols in C memory, and the
// registry's listener, once.
func (c *waylandClient) buildProtocols() error {
	list := c.protocols()
	// Every interface first, so that messages can point at those after them.
	for _, pi := range list {
		p, err := cAlloc(unsafe.Sizeof(wlInterface{}))
		if err != nil {
			return err
		}
		*pi.built = p
	}
	byName := map[string]ptr{"wl_surface": c.surfaceInterface}
	for _, pi := range list {
		byName[pi.name] = *pi.built
	}
	messages := func(ms []protocolMessage) (ptr, error) {
		if len(ms) == 0 {
			return 0, nil
		}
		array, err := cAlloc(uintptr(len(ms)) * unsafe.Sizeof(wlMessage{}))
		if err != nil {
			return 0, err
		}
		out := unsafe.Slice((*wlMessage)(cAt(array)), len(ms))
		for i, m := range ms {
			if out[i].name, err = allocCString(m.name); err != nil {
				return 0, err
			}
			if out[i].signature, err = allocCString(m.signature); err != nil {
				return 0, err
			}
			// One entry per argument; libwayland reads none for a request
			// without arguments, but scanner's code points at NULLs there too.
			n := max(len(m.types), 1)
			if out[i].types, err = cAlloc(uintptr(n) * unsafe.Sizeof(ptr(0))); err != nil {
				return 0, err
			}
			types := unsafe.Slice((*ptr)(cAt(out[i].types)), n)
			for j, t := range m.types {
				if t == "" {
					continue
				}
				if types[j] = byName[t]; types[j] == 0 {
					return 0, fmt.Errorf("no interface %s", t)
				}
			}
		}
		return array, nil
	}
	for _, pi := range list {
		iface := (*wlInterface)(cAt(*pi.built))
		var err error
		if iface.name, err = allocCString(pi.name); err != nil {
			return err
		}
		iface.version = pi.version
		iface.methodCount = int32(len(pi.requests))
		if iface.methods, err = messages(pi.requests); err != nil {
			return err
		}
		iface.eventCount = int32(len(pi.events))
		if iface.events, err = messages(pi.events); err != nil {
			return err
		}
	}
	listener, err := cAlloc(2 * unsafe.Sizeof(ptr(0)))
	if err != nil {
		return err
	}
	initWaylandCallbacks()
	fns := unsafe.Slice((*ptr)(cAt(listener)), 2)
	fns[0], fns[1] = cbRegistryGlobal, cbRegistryGlobalRemove
	c.registryListener = listener
	return nil
}

// The registry's listener, made once: data is the id of the scan in
// registryScans.
var (
	waylandCallbacksOnce                     sync.Once
	cbRegistryGlobal, cbRegistryGlobalRemove ptr
)

// registryScans are the registries being read, by id, on the thread that
// reads each.
var registryScans struct {
	sync.Mutex
	next  ptr
	scans map[ptr]*registryScan
}

// registryScan is what a registry announced: each global's name and
// version, by interface.
type registryScan struct {
	names    map[string]uint32
	versions map[string]uint32
}

func initWaylandCallbacks() {
	waylandCallbacksOnce.Do(func() {
		cbRegistryGlobal = purego.NewCallback(func(data, registry ptr, name uint32, iface ptr, version uint32) {
			registryScans.Lock()
			s := registryScans.scans[data]
			registryScans.Unlock()
			if s == nil {
				return
			}
			i := goStr(iface)
			s.names[i] = name
			s.versions[i] = version
		})
		cbRegistryGlobalRemove = purego.NewCallback(func(data, registry ptr, name uint32) {})
	})
}

// waylandGlobals are the globals a subsurface needs, bound on a display's
// default queue: 0 where the compositor has none.
type waylandGlobals struct {
	// display is GDK's wl_display, which they are bound on.
	display                                                ptr
	compositor, subcompositor, viewporter, fractionManager ptr
	// Their versions as bound.
	compositorVersion, subcompositorVersion, viewporterVersion, fractionManagerVersion uint32
}

// complete reports every global a subsurface needs.
func (g *waylandGlobals) complete() bool {
	return g.compositor != 0 && g.subcompositor != 0 && g.viewporter != 0 && g.fractionManager != 0
}

// String names the globals found with their versions, and those missing.
func (g *waylandGlobals) String() string {
	var found, missing []string
	for _, x := range []struct {
		what    string
		proxy   ptr
		version uint32
	}{
		{"compositor", g.compositor, g.compositorVersion},
		{"subcompositor", g.subcompositor, g.subcompositorVersion},
		{"viewporter", g.viewporter, g.viewporterVersion},
		{"fractional scale", g.fractionManager, g.fractionManagerVersion},
	} {
		if x.proxy == 0 {
			missing = append(missing, x.what)
		} else {
			found = append(found, fmt.Sprintf("%s %d", x.what, x.version))
		}
	}
	s := strings.Join(found, ", ")
	if len(missing) > 0 {
		if s != "" {
			s += "; "
		}
		s += "missing " + strings.Join(missing, ", ")
	}
	return s
}

// findWaylandGlobals binds the globals a subsurface needs on display.
// compositor is a wl_compositor already bound (GDK's), used in place of a
// second one; 0 binds one. The registry is read on a queue of its own, from
// a wrapper of the display, so that GDK's handlers see none of its events
// and no other queue is dispatched; the globals then move to the default
// queue, which GDK dispatches on the main thread.
func (c *waylandClient) findWaylandGlobals(display, compositor ptr) (*waylandGlobals, error) {
	queue := c.createQueue(display)
	if queue == 0 {
		return nil, errors.New("wl_display_create_queue failed")
	}
	defer c.queueDestroy(queue)
	wrapper := c.createWrapper(display)
	if wrapper == 0 {
		return nil, errors.New("wl_proxy_create_wrapper failed")
	}
	defer c.wrapperDestroy(wrapper)
	c.setQueue(wrapper, queue)
	var args [4]uint64 // a new_id is NULL in its wl_argument
	registry := c.marshal(wrapper, wlDisplayGetRegistry, c.registryInterface, c.proxyGetVersion(wrapper), 0, &args[0])
	if registry == 0 {
		return nil, errors.New("wl_display.get_registry failed")
	}
	defer c.proxyDestroy(registry)

	scan := &registryScan{names: map[string]uint32{}, versions: map[string]uint32{}}
	registryScans.Lock()
	if registryScans.scans == nil {
		registryScans.scans = map[ptr]*registryScan{}
	}
	registryScans.next++
	id := registryScans.next
	registryScans.scans[id] = scan
	registryScans.Unlock()
	defer func() {
		registryScans.Lock()
		delete(registryScans.scans, id)
		registryScans.Unlock()
	}()
	if c.addListener(registry, c.registryListener, id) != 0 {
		return nil, errors.New("wl_proxy_add_listener failed")
	}
	if c.roundtripQueue(display, queue) < 0 {
		return nil, fmt.Errorf("reading the registry: error %d", c.displayGetError(display))
	}

	g := &waylandGlobals{display: display}
	bindGlobal := func(iface ptr, out *ptr, version *uint32) {
		name := goStr((*wlInterface)(cAt(iface)).name)
		advertised, ok := scan.versions[name]
		if !ok {
			return
		}
		v := min(advertised, uint32((*wlInterface)(cAt(iface)).version))
		// wl_registry.bind's signature "usun": the global's name, then the
		// untyped new_id as the interface's name, its version and the id.
		args := [4]uint64{uint64(scan.names[name]), uint64((*wlInterface)(cAt(iface)).name), uint64(v), 0}
		if *out = c.marshal(registry, wlRegistryBind, iface, v, 0, &args[0]); *out != 0 {
			*version = v
		}
	}
	if compositor != 0 {
		g.compositor, g.compositorVersion = compositor, c.proxyGetVersion(compositor)
	} else {
		bindGlobal(c.compositorInterface, &g.compositor, &g.compositorVersion)
	}
	bindGlobal(c.subcompositorInterface, &g.subcompositor, &g.subcompositorVersion)
	bindGlobal(c.viewporterInterface, &g.viewporter, &g.viewporterVersion)
	bindGlobal(c.fractionManagerInterface, &g.fractionManager, &g.fractionManagerVersion)
	// The compositor answers the binds before the roundtrip's sync: an
	// error in one is seen here, not later on GDK's queue.
	failed := c.roundtripQueue(display, queue) < 0
	for _, p := range []*ptr{&g.subcompositor, &g.viewporter, &g.fractionManager, &g.compositor} {
		if *p == 0 || *p == compositor {
			continue
		}
		if failed {
			c.proxyDestroy(*p)
			*p = 0
		} else {
			c.setQueue(*p, 0)
		}
	}
	if failed {
		return nil, fmt.Errorf("binding the globals: error %d", c.displayGetError(display))
	}
	return g, nil
}
