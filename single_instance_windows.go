package mygo

import (
	"net"
	"os"
)

// singleInstanceFolder returns the folder of the single instance socket
// and lock: the user's own temporary folder, which other users cannot
// reach.
func singleInstanceFolder(string) (string, error) { return os.TempDir(), nil }

// peerAllowed reports whether the process at the other end of conn may
// hand its command line over: only the user reaches the socket's folder.
func peerAllowed(net.Conn) bool { return true }
