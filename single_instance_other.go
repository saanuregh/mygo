//go:build !unix && !windows && !js

package mygo

import (
	"errors"
	"net"
)

func singleInstanceFolder(string) (string, error) {
	return "", errors.New("no folder only the user can reach is known on this system")
}

func peerAllowed(net.Conn) bool { return false }
