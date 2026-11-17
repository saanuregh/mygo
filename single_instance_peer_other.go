//go:build unix && !linux && !darwin

package mygo

import "errors"

// peerChecked is false where peerUID cannot tell who connects: the app
// runs without the single instance lock there (singleInstanceFolder).
const peerChecked = false

func peerUID(uintptr) (int, error) {
	return -1, errors.New("the user of a socket's peer is unknown on this system")
}
