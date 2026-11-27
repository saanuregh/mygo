//go:build linux && !android && (amd64 || arm64)

package waylandsubsurface

import "github.com/egoist/mygo/internal/linux"

func init() { linux.LinkWaylandSubsurface() }
