package ui

import "testing"

// A field's caret stays while a drawn menu another element opened shows,
// the field focused as it opened (a picker's query over its rows' menus).
func TestAFieldKeepsItsCaretUnderARowsMenu(t *testing.T) {
	query := ""
	var field uint64
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Padding(20).Gap(10).Children(func() {
			in := coreTextInput(c, &query).Label("Query")
			field = in.id
			row := coreRow(c).Padding(8).Label("Row").Children(func() { coreText(c, "Row one") })
			row.PopoverMenu(func(m *Menu) { m.Item("Choose").Chosen() })
		})
	}, 400, 300)
	tt.Click("Query")
	tt.Frame()
	if tt.rt.caretUnderMenu(field) {
		t.Fatal("kept with no menu")
	}
	if err := tt.RightClick("Row one"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText("Choose") {
		t.Fatal("no menu")
	}
	if !tt.rt.caretUnderMenu(field) {
		t.Fatal("the field's caret went with the menu")
	}
}
