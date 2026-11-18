package web

import (
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo/internal/platform"
)

// iframeClipPath returns the CSS clip-path showing what of a web view's
// iframe shows: its Clip less its Covers, in the page's DIPs from the
// iframe's corner. The clip path also cuts the iframe's pointer events, so
// what covers the page takes the pointer. The area is cut into rows of
// rectangles, which the nonzero rule joins however the covers overlap.
func iframeClipPath(p *platform.WebViewPlacement) string {
	clip := p.Clip
	var covers []platform.RectF
	for _, c := range p.Covers {
		if c = intersect(c, clip); c.W > 0 && c.H > 0 {
			covers = append(covers, c)
		}
	}
	if len(covers) == 0 && clip == p.Frame {
		return "none"
	}
	xs := []float64{clip.X, clip.X + clip.W}
	ys := []float64{clip.Y, clip.Y + clip.H}
	for _, c := range covers {
		xs = append(xs, c.X, c.X+c.W)
		ys = append(ys, c.Y, c.Y+c.H)
	}
	slices.Sort(xs)
	slices.Sort(ys)
	xs, ys = slices.Compact(xs), slices.Compact(ys)

	sx, sy := p.Scale()
	px := func(x float64) string { return strconv.FormatFloat((x-p.Frame.X)/sx, 'f', -1, 64) }
	py := func(y float64) string { return strconv.FormatFloat((y-p.Frame.Y)/sy, 'f', -1, 64) }
	covered := func(x, y float64) bool {
		for _, c := range covers {
			if c.Contains(x, y) {
				return true
			}
		}
		return false
	}
	var b strings.Builder
	for j := 0; j+1 < len(ys); j++ {
		top, bottom := ys[j], ys[j+1]
		cy := (top + bottom) / 2
		for i := 0; i+1 < len(xs); {
			if covered((xs[i]+xs[i+1])/2, cy) {
				i++
				continue
			}
			k := i + 1
			for k+1 < len(xs) && !covered((xs[k]+xs[k+1])/2, cy) {
				k++
			}
			if b.Len() > 0 {
				b.WriteByte(' ')
			}
			b.WriteString("M" + px(xs[i]) + " " + py(top) + "H" + px(xs[k]) + "V" + py(bottom) + "H" + px(xs[i]) + "Z")
			i = k
		}
	}
	if b.Len() == 0 {
		return "polygon(0 0,0 0,0 0)"
	}
	return `path("` + b.String() + `")`
}

func intersect(a, b platform.RectF) platform.RectF {
	x0, y0 := max(a.X, b.X), max(a.Y, b.Y)
	x1, y1 := min(a.X+a.W, b.X+b.W), min(a.Y+a.H, b.Y+b.H)
	if x1 <= x0 || y1 <= y0 {
		return platform.RectF{}
	}
	return platform.RectF{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}
