package mygo

import (
	"errors"
	"sync"

	"github.com/ebitengine/purego"
)

const peerChecked = true

// getpeereid is libSystem's getpeereid(3), which the syscall package
// leaves out.
var getpeereid = sync.OnceValues(func() (func(fd int32, uid, gid *uint32) int32, error) {
	lib, err := purego.Dlopen("/usr/lib/libSystem.B.dylib", purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		return nil, err
	}
	var fn func(fd int32, uid, gid *uint32) int32
	purego.RegisterLibFunc(&fn, lib, "getpeereid")
	return fn, nil
})

// peerUID returns the user of the process at the other end of the Unix
// socket fd.
func peerUID(fd uintptr) (int, error) {
	fn, err := getpeereid()
	if err != nil {
		return -1, err
	}
	var uid, gid uint32
	if fn(int32(fd), &uid, &gid) != 0 {
		return -1, errors.New("getpeereid failed")
	}
	return int(uid), nil
}
