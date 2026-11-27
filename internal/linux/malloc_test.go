//go:build linux && !android && (amd64 || arm64)

package linux

import (
	"os"
	"runtime"
	"syscall"
	"testing"
	"unsafe"

	"github.com/ebitengine/purego"
)

// The main goroutine stays on the main thread, whose C allocations come
// from glibc's main heap, as GTK's paints on the UI thread do.
func init() { runtime.LockOSThread() }

// imageBytes is GTK's image for a whole 4116x2800 window.
const imageBytes = 4116 * 2800 * 4

// paintFaults holds the page faults of an image allocated, cleared and freed
// after another, as two paints do, measured on the main thread by TestMain.
var paintFaults struct {
	glibc, took                     bool
	plain, kept, afterIdle, trimmed int64
}

func TestMain(m *testing.M) {
	malloc, _ := purego.Dlsym(purego.RTLD_DEFAULT, "malloc")
	free, _ := purego.Dlsym(purego.RTLD_DEFAULT, "free")
	_, err := purego.Dlsym(purego.RTLD_DEFAULT, "gnu_get_libc_version")
	if paintFaults.glibc = err == nil && malloc != 0 && free != 0 && mallocTrim != 0; paintFaults.glibc {
		paint := func() int64 {
			start := threadFaults()
			p, _, _ := purego.SyscallN(malloc, imageBytes)
			if p == 0 {
				panic("malloc failed")
			}
			clear(unsafe.Slice((*byte)(*(*unsafe.Pointer)(unsafe.Pointer(&p))), imageBytes)) // C memory, which Go never moves
			purego.SyscallN(free, p)
			return threadFaults() - start
		}
		paint()
		paintFaults.plain = paint()
		b := &Backend{keepsFreedMemory: keepFreedMemory()}
		paintFaults.took = b.keepsFreedMemory
		paint()
		paintFaults.kept = paint()
		(&surface{w: &window{b: b}}).Idle()
		paintFaults.afterIdle = paint()
		purego.SyscallN(mallocTrim, 0)
		paintFaults.trimmed = paint()
	}
	os.Exit(m.Run())
}

func threadFaults() int64 {
	var usage syscall.Rusage
	if err := syscall.Getrusage(1, &usage); err != nil { // RUSAGE_THREAD
		panic(err)
	}
	return usage.Minflt
}

// A paint's image reuses the pages of the last once glibc keeps freed
// memory, and the surface's Idle leaves them: only a trim gives them back.
func TestFreedMemoryKeptForTheNextPaint(t *testing.T) {
	if !paintFaults.glibc {
		t.Skip("not glibc")
	}
	if !paintFaults.took {
		t.Fatal("glibc's mallopt refused a setting")
	}
	pages := int64(imageBytes / os.Getpagesize())
	t.Logf("page faults of a paint's image (of %d pages): %d by default, %d kept, %d after Idle, %d after malloc_trim",
		pages, paintFaults.plain, paintFaults.kept, paintFaults.afterIdle, paintFaults.trimmed)
	if paintFaults.plain < pages/2 {
		t.Errorf("by default %d faults, want most pages: the test no longer shows the cost", paintFaults.plain)
	}
	if paintFaults.kept > pages/100 {
		t.Errorf("kept: %d faults, want almost none", paintFaults.kept)
	}
	if paintFaults.afterIdle > pages/100 {
		t.Errorf("after Idle: %d faults, want almost none", paintFaults.afterIdle)
	}
	if paintFaults.trimmed < pages/2 {
		t.Errorf("after malloc_trim %d faults, want most pages: Idle's check proves nothing", paintFaults.trimmed)
	}
}
