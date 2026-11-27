// Package waylandsubsurface draws a native UI window's frames on a Wayland
// subsurface of its own, with EGL, rather than in GTK 3's GtkGLArea, which
// composites each frame through an image as large as what it repaints.
// It is opt-in, as apps that do not import it link none of its code; an
// app opts in by importing the package:
//
//	import _ "github.com/egoist/mygo/plugins/waylandsubsurface"
//
// Then the environment variable MYGO_WAYLAND_SUBSURFACE=1 turns it on, on
// Linux under Wayland, where OpenGL draws on a GPU; elsewhere, or where
// the compositor lacks what it needs, windows draw as without it, which
// the app's log tells. On other systems the package does nothing.
package waylandsubsurface
