//go:build unix

package mygo

import (
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
)

// singleInstanceFolder returns the folder of the single instance socket
// and lock, one only the user can reach: "mygo" under XDG_RUNTIME_DIR,
// else "mygo-<uid>" in the temporary folder, made if missing. A shared
// folder such as /tmp would let another user listen on the socket first
// and take the app's command lines, or hold the lock. file is the
// socket's name: Unix socket paths are limited to about 100 bytes.
func singleInstanceFolder(file string) (string, error) {
	if !peerChecked {
		return "", errors.New("this system cannot tell which user connects to a socket")
	}
	uid := os.Getuid()
	own := "mygo-" + strconv.Itoa(uid)
	var dirs []string
	if runtime := os.Getenv("XDG_RUNTIME_DIR"); runtime != "" {
		dirs = append(dirs, filepath.Join(runtime, "mygo"))
	}
	dirs = append(dirs, filepath.Join(os.TempDir(), own), filepath.Join("/tmp", own))
	dir := dirs[len(dirs)-1]
	for _, d := range dirs {
		if len(filepath.Join(d, file)) < 100 {
			dir = d
			break
		}
	}
	if err := privateFolder(dir, uid); err != nil {
		return "", err
	}
	return dir, nil
}

// privateFolder makes the folder at path with mode 0700 if it is missing,
// and checks that it is a folder, not a link, that the user uid owns and
// no other user can reach.
func privateFolder(path string, uid int) error {
	if err := os.Mkdir(path, 0o700); err != nil && !errors.Is(err, fs.ErrExist) {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a folder", path)
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("%s: its owner is unknown", path)
	}
	if int(st.Uid) != uid {
		return fmt.Errorf("%s belongs to user %d, not %d", path, st.Uid, uid)
	}
	if perm := info.Mode().Perm(); perm&0o077 != 0 {
		return fmt.Errorf("%s has mode %#o: others can reach it", path, perm)
	}
	return nil
}

// peerAllowed reports whether the process at the other end of conn may
// hand its command line over: one of the same user.
func peerAllowed(conn net.Conn) bool { return peerIsUser(conn, os.Getuid()) }

// peerIsUser reports whether the process at the other end of the Unix
// socket conn runs as the user uid.
func peerIsUser(conn net.Conn, uid int) bool {
	sc, ok := conn.(syscall.Conn)
	if !ok {
		return false
	}
	raw, err := sc.SyscallConn()
	if err != nil {
		return false
	}
	peer, peerErr := -1, error(nil)
	if err := raw.Control(func(fd uintptr) { peer, peerErr = peerUID(fd) }); err != nil || peerErr != nil {
		return false
	}
	return peer == uid
}
