package text

import "strings"

// DigitCells, among a Style's or Span's Features, sets each digit 0-9 in a
// cell as wide as the font's widest digit, centred in it, as nanoui lays
// out tabular text: unlike the "tnum" feature, the digits keep the font's
// own glyphs, and fonts without tabular figures line up too. It is no
// OpenType tag, so the engines leave it out of the features they set.
const DigitCells = "digit-cells"

// hasDigitCells reports whether a Features list holds DigitCells.
func hasDigitCells(list string) bool {
	if !strings.Contains(list, DigitCells) {
		return false
	}
	for _, item := range strings.Split(list, ",") {
		if strings.TrimSpace(item) == DigitCells {
			return true
		}
	}
	return false
}

// shape is the engine's shape, with the digits of runs styled DigitCells
// put in their cells: each glyph after one moves by what the cell adds.
// The engine broke the lines at the digits' own widths.
func (s *System) shape(text []rune, style Style, spans []Span, width float32, rtl, wholeWords bool) []shapedLine {
	e := s.engine()
	lines := e.shape(text, style, spans, width, rtl, wholeWords)
	if !hasDigitCells(style.Features) && !spansHaveDigitCells(spans) {
		return lines
	}
	for i := range lines {
		l := &lines[i]
		shift := float32(0)
		for j := range l.runs {
			r := &l.runs[j]
			st := styleAt(style, spans, r.start)
			cells := hasDigitCells(st.Features)
			cell := float32(0)
			if cells {
				cell = s.digitCell(st)
			}
			for k := range r.glyphs {
				g := &r.glyphs[k]
				g.X += shift
				if !cells || g.Cluster < 0 || g.Cluster >= len(text) || text[g.Cluster] < '0' || text[g.Cluster] > '9' || g.Advance <= 0 {
					continue
				}
				// Drawn centred in the cell, which the pen moves by.
				if r.centres == nil {
					r.centres = make([]float32, len(r.glyphs))
				}
				r.centres[k] = (cell - g.Advance) / 2
				g.X += r.centres[k]
				shift += cell - g.Advance
				g.Advance = cell
			}
		}
	}
	return lines
}

func spansHaveDigitCells(spans []Span) bool {
	for _, sp := range spans {
		if hasDigitCells(sp.Features) {
			return true
		}
	}
	return false
}

// styleAt is the style of rune i of a text of style base and spans.
func styleAt(base Style, spans []Span, i int) Style {
	for _, sp := range spans {
		if i < sp.End {
			return sp.style(base)
		}
	}
	return base
}

// digitCell is the advance of the widest digit of the font of style, its
// features aside, as the widest of the digits shaped alone.
func (s *System) digitCell(style Style) float32 {
	key := style
	key.Features = ""
	if w, ok := s.digitCells[key]; ok {
		return w
	}
	var cell float32
	for _, l := range s.engine().shape([]rune("0123456789"), key, nil, 0, false, false) {
		for _, r := range l.runs {
			for _, g := range r.glyphs {
				cell = max(cell, g.Advance)
			}
		}
	}
	if s.digitCells == nil || len(s.digitCells) >= 256 {
		s.digitCells = map[Style]float32{}
	}
	s.digitCells[key] = cell
	return cell
}
