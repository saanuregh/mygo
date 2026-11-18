package editor

// DefinitionLinkRequest asks the app where Cmd+click would go from
// (State.OnDefinitionLink).
type DefinitionLinkRequest struct {
	// Offset is the place under the pointer; -1 tells that Cmd was let go,
	// so that what the app found for these questions can be forgotten.
	Offset int
	// Type: Shift is held too, for the definition of the type there.
	Type bool
}

// definitionLink is the app's answer of where Cmd+click goes from
// (State.OnDefinitionLink), of the text at version: the range start to
// end (none when end <= start), of kind typ; and the question last asked
// (asked -1: none), answered or not.
type definitionLink struct {
	version    uint64
	known      bool
	start, end int
	typ        bool
	asked      int
	askedType  bool
	answered   bool
	// active: asked since Cmd was held, so its letting go is told.
	active bool
}

// forget drops the answer when the text is no longer at version.
func (d *definitionLink) forget(version uint64) {
	if !d.known || d.version != version {
		*d = definitionLink{version: version, known: true, asked: -1, active: d.active}
	}
}

// askedLink is the link at off for a click of kind typ, as the app
// answered it, asking the app about off when it is outside the range
// answered and not asked yet. While a question waits, the last answer
// still shows, as Zed's link does, unless it is of the other kind.
func (ed *editor) askedLink(off int, typ bool) [2]int {
	d := &ed.definitionLink
	d.forget(ed.view.version)
	if d.typ != typ {
		d.start, d.end = 0, 0
	}
	if off >= d.start && off < d.end {
		return [2]int{d.start, d.end}
	}
	if off != d.asked || typ != d.askedType {
		d.asked, d.askedType, d.answered, d.active = off, typ, false, true
		ed.o.onDefinitionLink(DefinitionLinkRequest{Offset: off, Type: typ})
	}
	if !d.answered && d.end > d.start {
		return [2]int{d.start, d.end}
	}
	return [2]int{}
}

// letGoOfLink forgets the link as Cmd is let go, and tells the app once.
func (ed *editor) letGoOfLink() {
	d := &ed.definitionLink
	if !d.active {
		return
	}
	*d = definitionLink{version: d.version, known: d.known, asked: -1}
	if ed.o.onDefinitionLink != nil {
		ed.o.onDefinitionLink(DefinitionLinkRequest{Offset: -1})
	}
}
