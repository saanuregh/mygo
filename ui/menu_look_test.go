package ui

import (
	"bytes"
	"image/color"
	"slices"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// lookApp is a row with a menu drawn in the window, with an icon, a Danger
// item and a NextMenu item, in the style given.
type lookApp struct {
	style    MenuStyle
	context  bool // the row's menu is a ContextMenu, not a PopoverMenu
	drawn    bool // the window draws its menus (SetMenuPresentation)
	chosen   []string
	disabled bool
	icon     MenuIcon
}

func (a *lookApp) view(c *context) {
	th := LightTheme()
	th.Menu = a.style
	c.SetTheme(th)
	if a.drawn {
		c.SetMenuPresentation(PopoverMenus)
	}
	coreColumn(c).Fill().Padding(20).Children(func() {
		row := coreRow(c).Padding(8).Label("Report").Children(func() { coreText(c, "Report.pdf") })
		build := func(m *Menu) {
			if m.Item("Rename…").Icon(a.icon).Trailing("F2 or Enter").Chosen() {
				a.chosen = append(a.chosen, "Rename")
			}
			m.NextMenu("Snooze…", func(m *Menu) {
				for _, when := range []string{"In an Hour", "Tomorrow"} {
					if m.Item(when).Shortcut(0, KeyF2).Chosen() {
						a.chosen = append(a.chosen, when)
					}
				}
			}).Disabled(a.disabled)
			m.Separator()
			if m.Item("Delete…").Icon(a.icon).Danger().Chosen() {
				a.chosen = append(a.chosen, "Delete")
			}
		}
		if a.context {
			row.ContextMenu(build)
		} else {
			row.PopoverMenu(build)
		}
	})
}

// testIcon is a template icon for the items of a test's menu.
func testIcon(t *testing.T) MenuIcon {
	return testMenuIcon{&platform.MenuIcon{PNG: menuIconPNG(t, color.NRGBA{A: 255}), Template: true}}
}

// menuPanel returns the box of the outermost menu drawn in the window.
func menuPanel(t *testing.T, tt *Tester) Rect {
	t.Helper()
	var found []Rect
	for _, s := range tt.rt.states {
		if s.role == RoleMenu && s.seen == tt.rt.frame {
			found = append(found, Rect{X: s.vx, Y: s.vy, W: s.vw, H: s.vh})
		}
	}
	if len(found) != 1 {
		t.Fatalf("%d menus drawn, want 1", len(found))
	}
	return found[0]
}

// menuRows returns the boxes of the items of the menus drawn in the window.
func menuRows(tt *Tester) []Rect {
	var rows []Rect
	for _, s := range tt.rt.states {
		if s.role == RoleMenuItem && s.seen == tt.rt.frame {
			rows = append(rows, Rect{X: s.vx, Y: s.vy, W: s.vw, H: s.vh})
		}
	}
	return rows
}

func TestPopoverMenuTellsItsItems(t *testing.T) {
	a := &lookApp{icon: testIcon(t)}
	tt := coreNewTester(a.view, 600, 400)
	if tt.PopoverMenu() != nil {
		t.Fatal("PopoverMenu tells of a menu before one opened")
	}
	r, _ := tt.Find("Report.pdf")
	tt.RightClickAt(r.X+10, r.Y+r.H/2)
	got := tt.PopoverMenu()
	want := []TestMenuItem{
		{Label: "Rename…", Icon: a.icon, Trailing: "F2 or Enter"},
		{Label: "Snooze…"},
		{Separator: true},
		{Label: "Delete…", Icon: a.icon, Danger: true},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("PopoverMenu() = %+v, want %+v", got, want)
	}
	tt.Key(0, KeyEscape)
	if tt.PopoverMenu() != nil {
		t.Error("PopoverMenu tells of a menu after Escape closed it")
	}
}

func TestMenuStyleSizes(t *testing.T) {
	hover := RGB(230, 230, 230)
	a := &lookApp{style: MenuStyle{Width: 220, Padding: 5, Gap: 2, Offset: 6, ItemHeight: 30, ItemPadding: 8, FontSize: 13, Hover: hover}}
	tt := coreNewTester(a.view, 600, 400)
	r, _ := tt.Find("Report.pdf")
	x, y := r.X+10, r.Y+r.H/2
	tt.RightClickAt(x, y)
	p := menuPanel(t, tt)
	if p.W != 220 {
		t.Errorf("the menu is %v wide, want the style's 220", p.W)
	}
	if p.X != x || p.Y != y+6 {
		t.Errorf("the menu at (%v, %v), want 6 below the click at (%v, %v)", p.X, p.Y, x, y)
	}
	rows := menuRows(tt)
	if len(rows) != 3 {
		t.Fatalf("%d items, want 3", len(rows))
	}
	for _, row := range rows {
		if row.H != 30 || row.W != 210 || row.X != p.X+5 {
			t.Errorf("an item at %+v, want 30 high and 210 wide, 5 inside the panel at %+v", row, p)
		}
	}
	slices.SortFunc(rows, func(a, b Rect) int { return int(a.Y - b.Y) })
	if gap := rows[1].Y - (rows[0].Y + rows[0].H); gap != 2 {
		t.Errorf("the items %v apart, want the style's Gap 2", gap)
	}
	// The item highlighted is filled with Hover, its label in the text's
	// color.
	tt.Key(0, KeyDown)
	img := tt.Image()
	if got := img.RGBAAt(int(rows[0].X+2), int(rows[0].Y+rows[0].H/2)); !nearColor(got, hover) {
		t.Errorf("the item highlighted is %v, want the style's Hover %v", got, hover)
	}
	// The next menu takes the style's width too.
	tt.Click("Snooze…")
	if w := menuPanel(t, tt).W; w != 220 {
		t.Errorf("the next menu is %v wide, want the style's 220", w)
	}
}

// TestMenuStyleZeroIsTheDefaults checks each field of MenuStyle left zero
// draws as the value its doc names: the zero style is MyGo's own look.
func TestMenuStyleZeroIsTheDefaults(t *testing.T) {
	th := LightTheme()
	explicit := MenuStyle{
		Background: th.Background, Shadow: RGBA(0, 0, 0, 0.18), ShadowBlur: 20, ShadowY: 6,
		Radius: th.Radius + 2, Padding: th.Space(1) + 1,
		ItemPadding: th.Space(2), ItemGap: th.Space(2), ItemRadius: th.Radius,
		FontSize: th.FontSize, ShortcutSize: th.FontSize, IconSize: 16, MarkSize: th.Space(4),
	}
	draw := func(style MenuStyle) []byte {
		a := &lookApp{style: style, icon: testIcon(t)}
		tt := coreNewTester(a.view, 600, 400)
		r, _ := tt.Find("Report.pdf")
		tt.RightClickAt(r.X+10, r.Y+r.H/2)
		tt.Key(0, KeyDown)
		return tt.Image().Pix
	}
	if !bytes.Equal(draw(MenuStyle{}), draw(explicit)) {
		t.Error("the zero MenuStyle draws otherwise than its documented defaults")
	}
}

// TestThemeMenuStyleAppliesToContextMenus checks a ContextMenu, in a
// window that draws its menus, takes the theme's MenuStyle: its rows'
// height and the fill of the item highlighted.
func TestThemeMenuStyleAppliesToContextMenus(t *testing.T) {
	hover := RGB(200, 220, 240)
	a := &lookApp{context: true, drawn: true, style: MenuStyle{ItemHeight: 30, Hover: hover}}
	tt := coreNewTester(a.view, 600, 400)
	r, _ := tt.Find("Report.pdf")
	tt.RightClickAt(r.X+10, r.Y+r.H/2)
	if tt.Menu() == nil || tt.PopoverMenu() == nil {
		t.Fatal("the ContextMenu is not drawn in the window")
	}
	rows := menuRows(tt)
	if len(rows) != 3 {
		t.Fatalf("%d items, want 3", len(rows))
	}
	for _, row := range rows {
		if row.H != 30 {
			t.Errorf("an item %v high, want the style's 30", row.H)
		}
	}
	tt.Key(0, KeyDown)
	b, _ := tt.Find("Rename…")
	img := tt.Image()
	if got := img.RGBAAt(int(rows[0].X+2), int(b.Y+b.H/2)); !nearColor(got, hover) {
		t.Errorf("the item highlighted is %v, want the style's Hover %v", got, hover)
	}
	tt.Key(0, KeyEnter)
	if !slices.Equal(a.chosen, []string{"Rename"}) {
		t.Errorf("chose %q, want Rename", a.chosen)
	}
}

func TestNextMenuOpensInPlace(t *testing.T) {
	a := &lookApp{}
	tt := coreNewTester(a.view, 600, 400)
	r, _ := tt.Find("Report.pdf")
	tt.RightClickAt(r.X+10, r.Y+r.H/2)
	first := menuPanel(t, tt)
	tt.Click("Snooze…")
	got := tt.PopoverMenu()
	if len(got) != 2 || got[0].Label != "In an Hour" || got[1].Label != "Tomorrow" || got[0].Shortcut == "" {
		t.Fatalf("after Snooze…, PopoverMenu() = %+v, want the two times with their keys", got)
	}
	if p := menuPanel(t, tt); p.X != first.X || p.Y != first.Y {
		t.Errorf("the next menu at (%v, %v), not where the first was (%v, %v)", p.X, p.Y, first.X, first.Y)
	}
	if len(a.chosen) != 0 {
		t.Errorf("opening the next menu chose %q", a.chosen)
	}
	tt.Frame()
	if tt.PopoverMenu() == nil {
		t.Fatal("the next menu closed a frame after it opened")
	}
	tt.Click("Tomorrow")
	if !slices.Equal(a.chosen, []string{"Tomorrow"}) {
		t.Errorf("chose %q, want Tomorrow", a.chosen)
	}
	if tt.PopoverMenu() != nil {
		t.Error("the next menu shows after its item was chosen")
	}

	// From the keys: the next menu starts on its first item.
	tt.RightClickAt(r.X+10, r.Y+r.H/2)
	tt.Key(0, KeyDown)
	tt.Key(0, KeyDown)
	tt.Key(0, KeyEnter)
	tt.Key(0, KeyEnter)
	if !slices.Equal(a.chosen, []string{"Tomorrow", "In an Hour"}) {
		t.Errorf("chose %q by the keys, want In an Hour after Tomorrow", a.chosen)
	}
}

func TestNextMenuDisabled(t *testing.T) {
	a := &lookApp{disabled: true}
	tt := coreNewTester(a.view, 600, 400)
	r, _ := tt.Find("Report.pdf")
	tt.RightClickAt(r.X+10, r.Y+r.H/2)
	tt.Click("Snooze…")
	if got := tt.PopoverMenu(); len(got) == 0 || got[0].Label != "Rename…" {
		t.Errorf("a disabled NextMenu item opened %+v", got)
	}
}

// TestNextMenuInTheSystemsMenu checks a menu the system shows has the
// next menu's items as a submenu.
func TestNextMenuInTheSystemsMenu(t *testing.T) {
	a := &lookApp{context: true}
	tt := coreNewTester(a.view, 600, 400)
	r, _ := tt.Find("Report.pdf")
	tt.RightClickAt(r.X+10, r.Y+r.H/2)
	if !slices.Equal(tt.Menu(), []string{"Rename…", "Snooze…", "-", "Delete…"}) {
		t.Fatalf("the system's menu is %q", tt.Menu())
	}
	if tt.PopoverMenu() != nil {
		t.Error("a ContextMenu is drawn in a window that does not draw its menus")
	}
	if err := tt.ChooseMenuItem("Snooze…", "Tomorrow"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(a.chosen, []string{"Tomorrow"}) {
		t.Errorf("chose %q, want Tomorrow", a.chosen)
	}
}

// TestMenuStyleOffsetFromTheButton checks a Menu button's menu, in a
// window that draws its menus, goes the style's Offset below the button.
func TestMenuStyleOffsetFromTheButton(t *testing.T) {
	top := func(offset float32) (float32, string) {
		chosen := ""
		view := func(c *context) {
			th := LightTheme()
			th.Menu = MenuStyle{Offset: offset}
			c.SetTheme(th)
			c.SetMenuPresentation(PopoverMenus)
			coreColumn(c).Fill().Padding(20).Children(func() {
				coreButton(c, "More").Menu(func(m *Menu) {
					if m.Item("Duplicate").Chosen() {
						chosen = "Duplicate"
					}
				})
			})
		}
		tt := coreNewTester(view, 400, 300)
		tt.Click("More")
		if len(tt.PopoverMenu()) != 1 {
			t.Fatalf("drawn %+v: want Duplicate drawn", tt.PopoverMenu())
		}
		y := menuPanel(t, tt).Y
		tt.Click("Duplicate")
		return y, chosen
	}
	by4, chosen := top(0)
	if chosen != "Duplicate" {
		t.Errorf("chose %q", chosen)
	}
	if by6, _ := top(6); by6-by4 != 2 {
		t.Errorf("the menu's top %v with Offset 6, %v with the default 4 apart", by6, by4)
	}
}

// TestOpenPopoverMenu checks a button inside a row, built only while the
// row is hovered, opens a menu on the row at a point, which stays as the
// pointer leaves the row and the button goes.
func TestOpenPopoverMenu(t *testing.T) {
	chosen, shows := "", false
	view := func(c *context) {
		coreColumn(c).Fill().Padding(20).Gap(10).Children(func() {
			row := coreRow(c).Padding(8).Gap(8).Label("Report")
			row.PopoverMenu(func(m *Menu) {
				if m.Item("Open").Chosen() {
					chosen = "Open"
				}
			})
			shows = row.MenuShows()
			hovered := row.Hovered()
			row.Children(func() {
				coreText(c, "Report.pdf")
				if hovered || row.MenuShows() {
					b := coreButton(c, "Later")
					if b.Clicked() {
						r := b.Bounds()
						row.OpenPopoverMenu(r.X, r.Y+r.H, func(m *Menu) {
							if m.Item("In an Hour").Chosen() {
								chosen = "In an Hour"
							}
						})
					}
				}
			})
			coreText(c, "Below")
		})
	}
	tt := coreNewTester(view, 400, 300)
	r, _ := tt.Find("Report.pdf")
	tt.Move(r.X+5, r.Y+5)
	b, ok := tt.Find("Later")
	if !ok {
		t.Fatal("no Later button while the row is hovered")
	}
	tt.Click("Later")
	got := tt.PopoverMenu()
	if len(got) != 1 || got[0].Label != "In an Hour" || !shows {
		t.Fatalf("PopoverMenu() = %+v, MenuShows %v after Later", got, shows)
	}
	if item, _ := tt.Find("In an Hour"); item.Y < b.Y+b.H || item.X < b.X-20 || item.X > b.X+20 {
		t.Errorf("the menu's item at %+v, not below the button at %+v", item, b)
	}
	tt.Click("In an Hour")
	if chosen != "In an Hour" || tt.PopoverMenu() != nil || shows {
		t.Errorf("chose %q, menu %+v, MenuShows %v: want In an Hour chosen and the menu gone", chosen, tt.PopoverMenu(), shows)
	}
	// The row's own menu still opens on a right-click.
	tt.RightClickAt(r.X+5, r.Y+5)
	if got := tt.PopoverMenu(); len(got) != 1 || got[0].Label != "Open" {
		t.Errorf("the row's menu is %+v after OpenPopoverMenu's closed", got)
	}
}

// TestMenuStyleShiftIntoWindow checks a menu opened near the window's end
// moves back into it with ShiftIntoWindow, and opens before the point
// without.
func TestMenuStyleShiftIntoWindow(t *testing.T) {
	for _, shift := range []bool{false, true} {
		a := &lookApp{style: MenuStyle{Width: 240, ShiftIntoWindow: shift}}
		tt := coreNewTester(a.view, 400, 400)
		r, _ := tt.Find("Report.pdf")
		x := float32(300)
		tt.RightClickAt(x, r.Y+r.H/2)
		p := menuPanel(t, tt)
		switch {
		case shift && (p.X != 400-windowMargin-240 || p.X+p.W != 400-windowMargin):
			t.Errorf("shifted: the menu at %v..%v, want it against the window's end", p.X, p.X+p.W)
		case !shift && p.X+p.W != x:
			t.Errorf("not shifted: the menu at %v..%v, want it ending at the point %v", p.X, p.X+p.W, x)
		}
	}
}
