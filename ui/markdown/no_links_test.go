package markdown

import (
	"slices"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// TestViewNoLinks checks that NoLinks shows links as text: a click
// opening nothing and telling no OnLink, no hand cursor, no tooltip; the
// same text with links has the hand over them.
func TestViewNoLinks(t *testing.T) {
	var clicked []string
	o := &Options{NoLinks: true, OnLink: func(url string) { clicked = append(clicked, url) }}
	src := "Some **bold** and a [link](https://go.dev \"Go's site\"), a note[^1].\n\n[^1]: The note.\n"
	tt := ui.NewTester(func(c *ui.Context) { View(c, src, o) }, 600, 400)
	r, ok := tt.Find("Some bold and a link, a note[1].")
	if !ok {
		t.Fatalf("no paragraph in %q", tt.Texts())
	}
	// Over each point of the paragraph's line: what the cursor is, and a
	// click.
	sweep := func(click bool) (hand bool) {
		for x := r.X + 1; x < r.X+r.W; x += 4 {
			tt.Move(x, r.Y+r.H/2)
			tt.Frame()
			hand = hand || tt.Cursor() == ui.CursorPointer
			if click {
				tt.ClickAt(x, r.Y+r.H/2)
				tt.Frame()
			}
		}
		return hand
	}
	if sweep(true) {
		t.Error("a hand cursor over a link")
	}
	tt.Advance(2 * time.Second)
	tt.Frame()
	if tt.HasText("Go's site") {
		t.Error("a link's tooltip")
	}
	if len(clicked) > 0 || len(tt.OpenedURLs()) > 0 {
		t.Errorf("OnLink got %q, opened %q", clicked, tt.OpenedURLs())
	}
	o.NoLinks = false
	tt.Frame()
	if !sweep(false) {
		t.Error("no hand cursor over a followed link")
	}
}

// TestNoLinksLeavesCodeLinkToTheApp: NoLinks makes a plain link text, with
// no icon (LinkIcon) and no menu of its own (LinkMenu), while code CodeLink
// links stays a link the app follows, with its menu and icon.
func TestNoLinksLeavesCodeLinkToTheApp(t *testing.T) {
	var opened, iconFor, menuFor []string
	o := &Options{
		NoLinks:  true,
		CodeLink: func(code string) (string, bool) { return "file:" + code, code == "src/app.go:12" },
		OnLink:   func(url string) { opened = append(opened, url) },
		LinkIcon: func(dest string, size float32) (InlineObject, bool) {
			iconFor = append(iconFor, dest)
			return InlineObject{}, false
		},
		LinkMenu: func(m *ui.Menu, url string) {
			menuFor = append(menuFor, url)
			m.Item("Open")
		},
	}
	src := "See [site](https://go.dev) and `src/app.go:12`.\n"
	tt := ui.NewTester(func(c *ui.Context) { View(c, src, o) }, 600, 400)
	if err := tt.Click("site"); err == nil {
		t.Error("the plain link is a link")
	}
	if err := tt.Click("src/app.go:12"); err != nil {
		t.Fatalf("%v in %q", err, tt.Texts())
	}
	if !slices.Equal(opened, []string{"file:src/app.go:12"}) {
		t.Errorf("opened %q", opened)
	}
	if slices.Contains(iconFor, "https://go.dev") {
		t.Errorf("LinkIcon asked for the plain link: %q", iconFor)
	}
	if err := tt.RightClick("site"); err == nil && slices.Contains(menuFor, "https://go.dev") {
		t.Errorf("the plain link has its menu: %q", menuFor)
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	menuFor = nil
	if err := tt.RightClick("src/app.go:12"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(menuFor, []string{"file:src/app.go:12"}) {
		t.Errorf("the code link's menu was built for %q", menuFor)
	}
}
