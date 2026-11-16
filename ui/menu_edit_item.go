package ui

import "github.com/egoist/mygo/internal/l10n"

// EditItem adds the item of one editing command of the text the menu
// opened on, as EditItems adds it: name is "undo", "cut", "copy", "paste",
// "delete" or "selectAll". It is for a menu that orders or dresses its
// editing items itself, as one with Copy alone, with an icon; choosing it
// does the command on the text, without reaching the menu's function. On a
// menu that opened on no text, the item shows disabled.
//
//	m.EditItem("copy").Icon(copyIcon)
func (m *Menu) EditItem(name string) *MenuItem {
	texts := l10n.In(m.rt.lang)
	labels := map[string]string{"undo": texts.Undo(), "cut": texts.Cut(), "copy": texts.Copy(),
		"paste": texts.Paste(), "delete": texts.Delete(), "selectAll": texts.SelectAll()}
	on := m.text != nil && (m.text.editor != nil || m.text.textClient != nil) && m.rt.editCommandOn(m.text, name)
	it := m.Item(labels[name]).Disabled(!on)
	if on && m.run.collect {
		m.run.commands[it.seq-1] = name
	}
	return it
}
