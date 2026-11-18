package editor

// definitionLink is the app's answer of where Cmd+click goes from
// (State.OnDefinitionLink), of the text at version: the range start to
// end (none when end <= start), and the offset last asked about (-1 none).
type definitionLink struct {
	version    uint64
	known      bool
	start, end int
	asked      int
}

// forget drops the answer when the text is no longer at version.
func (d *definitionLink) forget(version uint64) {
	if !d.known || d.version != version {
		*d = definitionLink{version: version, known: true, asked: -1}
	}
}

// askedLink is the link at off as the app answered it, asking the app
// about off when it is outside the range answered and not asked yet.
func (ed *editor) askedLink(off int) [2]int {
	d := &ed.definitionLink
	d.forget(ed.view.version)
	if off >= d.start && off < d.end {
		return [2]int{d.start, d.end}
	}
	if off != d.asked {
		d.asked = off
		ed.o.onDefinitionLink(off)
	}
	return [2]int{}
}
