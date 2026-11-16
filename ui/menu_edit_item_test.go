package ui

import "testing"

func TestMenuEditItem(t *testing.T) {
	value := "hello"
	view := func(c *context) {
		coreColumn(c).Padding(20).Gap(10).Children(func() {
			in := coreTextInput(c, &value).Label("Name")
			in.PopoverMenu(func(m *Menu) {
				m.EditItem("copy").Trailing("Ctrl+C")
				m.EditItem("selectAll")
			})
			coreRow(c).Padding(8).Label("Plain").Children(func() { coreText(c, "Plain row") }).
				PopoverMenu(func(m *Menu) { m.EditItem("copy") })
		})
	}
	tt := coreNewTester(view, 400, 300)
	tt.RightClick("Name")
	got := tt.PopoverMenu()
	if len(got) != 2 || got[0].Label != "Copy" || !got[0].Disabled || got[0].Trailing != "Ctrl+C" || got[1].Label != "Select All" || got[1].Disabled {
		t.Fatalf("with nothing selected: %+v, want Copy disabled with its keys, then Select All", got)
	}
	if err := tt.Click("Select All"); err != nil {
		t.Fatal(err)
	}
	tt.RightClick("Name")
	if got := tt.PopoverMenu(); len(got) != 2 || got[0].Disabled {
		t.Fatalf("with the text selected: %+v", got)
	}
	if err := tt.Click("Copy"); err != nil {
		t.Fatal(err)
	}
	if tt.Clipboard() != "hello" {
		t.Errorf("Select All, Copy: clipboard %q", tt.Clipboard())
	}
	tt.RightClick("Plain")
	if got := tt.PopoverMenu(); len(got) != 1 || got[0].Label != "Copy" || !got[0].Disabled {
		t.Errorf("on no text: %+v, want Copy disabled", got)
	}
}
