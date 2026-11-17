package editor

import "slices"

// Listen has fn told of each edit of the buffer's text, as soon as it is
// made, in the order of the versions: the text before and after it, and
// the ranges it replaced, as a Highlighter's Edit is, whichever editor or
// method made it, undo, redo and EditRemote included. It returns a
// function that stops it.
//
// fn runs on the goroutine that made the edit, after it, with the buffer
// unlocked: it may read the buffer, and edit it, which it is told of once
// it returns. An edit another goroutine makes while fn runs is told on
// this one, after it.
func (b *Buffer) Listen(fn func(old, new Snapshot, changes []Change)) (stop func()) {
	b.mu.Lock()
	defer b.mu.Unlock()
	l := &listener{fn: fn, text: b.snapshotLocked()}
	b.listeners = append(b.listeners, l)
	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		b.listeners = slices.DeleteFunc(b.listeners, func(x *listener) bool { return x == l })
	}
}

// tell tells the listeners of the versions since they were last told;
// b.mu is not held.
func (b *Buffer) tell() {
	b.mu.Lock()
	if b.telling || len(b.listeners) == 0 {
		b.mu.Unlock()
		return
	}
	b.telling = true
	defer func() {
		b.mu.Lock()
		b.telling = false
		b.mu.Unlock()
	}()
	for {
		var l *listener
		for _, x := range b.listeners {
			if x.text.version != b.version {
				l = x
				break
			}
		}
		if l == nil {
			b.mu.Unlock()
			return
		}
		old := l.text
		next := b.snapshotLocked()
		var changes []Change
		if e, ok := b.entryAfterLocked(old.version); ok {
			next = Snapshot{t: e.text, version: e.version, ends: e.ends, restarts: e.restarts}
			changes = publicChanges(e.changes)
		} else {
			// Too far behind the log, as after a listener's own hundreds
			// of edits: the whole text replaced.
			changes = []Change{{OldStart: 0, OldEnd: old.Len(), NewStart: 0, NewEnd: next.Len()}}
		}
		l.text = next
		b.mu.Unlock()
		l.fn(old, next, changes)
		b.mu.Lock()
	}
}

// entryAfterLocked returns the log's entry of the version after since,
// false when the log no longer holds it; b.mu is held.
func (b *Buffer) entryAfterLocked(since uint64) (logEntry, bool) {
	if len(b.log) == 0 || since < b.log[0].version-1 || since >= b.version {
		return logEntry{}, false
	}
	i := int(since + 1 - b.log[0].version)
	if i >= len(b.log) || b.log[i].version != since+1 {
		return logEntry{}, false
	}
	return b.log[i], true
}
