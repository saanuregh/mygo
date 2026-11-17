//go:build unix

package mygo

import (
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestSingleInstanceFolderIsPrivate checks that the socket and the lock
// are in a folder of the user's own, made for them, and that a folder
// another user could reach is refused.
func TestSingleInstanceFolderIsPrivate(t *testing.T) {
	uid := os.Getuid()
	base := t.TempDir()
	made := filepath.Join(base, "made")
	if err := privateFolder(made, uid); err != nil {
		t.Fatalf("a missing folder: %v", err)
	}
	if info, err := os.Lstat(made); err != nil || !info.IsDir() || info.Mode().Perm() != 0o700 {
		t.Fatalf("made %v, %v; want a folder of mode 0700", info, err)
	}
	if err := privateFolder(made, uid); err != nil {
		t.Fatalf("the folder made before: %v", err)
	}
	if err := privateFolder(made, uid+1); err == nil {
		t.Fatal("a folder of another user was taken")
	}
	for i, mode := range []os.FileMode{0o755, 0o770, 0o777, 0o777 | os.ModeSticky} {
		shared := filepath.Join(base, "shared"+strconv.Itoa(i))
		if err := os.Mkdir(shared, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(shared, mode); err != nil {
			t.Fatal(err)
		}
		if err := privateFolder(shared, uid); err == nil {
			t.Fatalf("a folder of mode %v was taken", mode)
		}
	}
	link := filepath.Join(base, "link")
	if err := os.Symlink(made, link); err != nil {
		t.Fatal(err)
	}
	if err := privateFolder(link, uid); err == nil {
		t.Fatal("a link to a folder was taken")
	}
	file := filepath.Join(base, "file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := privateFolder(file, uid); err == nil {
		t.Fatal("a file was taken as the folder")
	}
}

// TestSingleInstanceSocketInRuntimeFolder checks that the socket is in a
// private folder under XDG_RUNTIME_DIR, and that none is given when that
// folder can be reached by others.
func TestSingleInstanceSocketInRuntimeFolder(t *testing.T) {
	runtime := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", runtime)
	socket, err := singleInstanceSocket(singleInstanceKey("MyGoTest-" + t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	if dir := filepath.Dir(socket); dir != filepath.Join(runtime, "mygo") {
		t.Fatalf("socket %q, want it in %q", socket, filepath.Join(runtime, "mygo"))
	}
	if err := os.Chmod(filepath.Join(runtime, "mygo"), 0o777); err != nil {
		t.Fatal(err)
	}
	if socket, err := singleInstanceSocket(singleInstanceKey("MyGoTest-" + t.Name())); err == nil {
		t.Fatalf("a folder others can write was used: %q", socket)
	}
}

// TestSingleInstanceSocketWithoutRuntimeFolder checks that without
// XDG_RUNTIME_DIR the socket is in a private folder of the user's under
// the temporary folder, never in the shared folder itself.
func TestSingleInstanceSocketWithoutRuntimeFolder(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", "")
	// Short enough for a socket path, which t.TempDir may not be.
	tmp, err := os.MkdirTemp("/tmp", "mygo-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(tmp) })
	t.Setenv("TMPDIR", tmp)
	socket, err := singleInstanceSocket(singleInstanceKey("MyGoTest-" + t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Dir(socket)
	if filepath.Dir(dir) != tmp || !strings.HasPrefix(filepath.Base(dir), "mygo-") {
		t.Fatalf("socket %q, want it in a folder of %q", socket, tmp)
	}
	if info, err := os.Lstat(dir); err != nil || info.Mode().Perm() != 0o700 {
		t.Fatalf("folder %v, %v; want mode 0700", info, err)
	}
}

// TestSingleInstanceUnsafeFolderRunsAlone checks that instances that find
// the folder reachable by others run without the single instance lock
// rather than trust it: each one runs.
func TestSingleInstanceUnsafeFolderRunsAlone(t *testing.T) {
	runtime := t.TempDir()
	if err := os.Mkdir(filepath.Join(runtime, "mygo"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(runtime, "mygo"), 0o777); err != nil {
		t.Fatal(err)
	}
	name := "MyGoTest-" + t.Name()
	first := startInstance(t, name, "hold", "XDG_RUNTIME_DIR="+runtime)
	first.expect(t, "locked")
	second := startInstance(t, name, "hold", "XDG_RUNTIME_DIR="+runtime)
	second.expect(t, "locked")
	first.exit(t)
	second.exit(t)
}

// TestSingleInstancePeerIsChecked checks that a command line is taken
// only from a process of the same user.
func TestSingleInstancePeerIsChecked(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.sock")
	ln, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	client, err := net.Dial("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	conn, err := ln.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if !peerIsUser(conn, os.Getuid()) {
		t.Fatal("a process of the same user was refused")
	}
	if peerIsUser(conn, os.Getuid()+1) {
		t.Fatal("a process of another user was taken")
	}
}
