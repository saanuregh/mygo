//go:build linux && !android && (amd64 || arm64)

package linux

import "github.com/ebitengine/purego"

// glibc's mallopt parameters (malloc.h).
const (
	mallocTrimThreshold = -1
	mallocTopPad        = -2
	mallocMmapMax       = -4
)

// keepFreedMemory has glibc keep the memory C code frees for its next
// allocations, and reports whether it took every setting. GTK 3 allocates
// and clears an image as large as what it repaints on every paint of a
// window it composites with OpenGL: 46 MB for a whole 4116x2800 window,
// above glibc's largest mmap threshold, so by default each paint maps fresh
// pages and faults on every one of them, about 17 ms. Taken from the heap
// and kept there, the image reuses pages already present: about 2 ms.
// Where mallopt is missing nothing changes.
func keepFreedMemory() bool {
	mallopt, err := purego.Dlsym(purego.RTLD_DEFAULT, "mallopt")
	if err != nil || mallopt == 0 {
		return false
	}
	settings := [...]struct{ param, value int }{
		{mallocMmapMax, 0},             // large blocks from the heap, not mappings of their own
		{mallocTrimThreshold, 1 << 30}, // the heap's free top is kept, up to 1 GiB
		{mallocTopPad, 256 << 20},      // and the heap grows 256 MiB beyond each need
	}
	took := true
	for _, s := range settings {
		if r, _, _ := purego.SyscallN(mallopt, uintptr(s.param), uintptr(s.value)); int32(r) != 1 {
			took = false
		}
	}
	return took
}
