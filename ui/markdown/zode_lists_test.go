package markdown

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

// Theme.MarkerGap: markers as text, "•" at every depth, "☐" and "☑" for
// tasks, the item's text MarkerGap past its marker, nested lists
// NestIndent further in per level from the outermost list's edge.
func TestThemeMarkerGapTextMarkers(t *testing.T) {
	o := &Options{Theme: &Theme{MarkerGap: 8, NestIndent: 16, FontSize: 14}}
	src := "- top\n  - inner\n- [ ] todo\n- [x] done\n\n1. first\n"
	tt := ui.NewTester(func(c *ui.Context) { View(c, src, o) }, 500, 400)
	for _, s := range []string{"•", "☐", "☑", "1.", "top", "inner"} {
		if !tt.HasText(s) {
			t.Errorf("no %q in %q", s, tt.Texts())
		}
	}
	if tt.HasText("◦") {
		t.Error("a nested bullet is not •")
	}
	top, _ := tt.Find("top")
	inner, _ := tt.Find("inner")
	bullet := top.X - 8
	var w float32
	for _, g := range ui.Shape("•", ui.Font{Size: 14}) {
		w += g.Advance
	}
	bullet -= w
	if d := inner.X - (bullet + 16 + w + 8); d < -1 || d > 1 {
		t.Errorf("inner at %v, top at %v: nested %v off", inner.X, top.X, d)
	}
}

// Theme.TableAlignsLeft and Theme.ImageAltAsLink.
func TestThemeTableAlignsLeftAndImageAltAsLink(t *testing.T) {
	link := ui.RGB(0, 0, 255)
	o := &Options{NoLinks: true, Theme: &Theme{TableAlignsLeft: true, TableEqualColumns: true, ImageAltAsLink: true, LinkUnderline: true, Link: link}}
	src := "| left | right |\n|---|---:|\n| a | b |\n\nSee ![a cat](cat.png) here.\n"
	var st *viewState
	tt := ui.NewTester(func(c *ui.Context) {
		st = *ui.Local(View(c, src, o), stateKey{}, newState)
	}, 500, 400)
	a, _ := tt.Find("a")
	b, _ := tt.Find("b")
	l, _ := tt.Find("left")
	r, _ := tt.Find("right")
	if a.X-l.X > 1 || b.X-r.X > 1 {
		t.Errorf("cells not at their start: a %v left %v, b %v right %v", a.X, l.X, b.X, r.X)
	}
	found := false
	for i := range st.paras {
		for _, pc := range st.paras[i].p.pieces {
			for _, s := range pc.spans {
				if s.Text == "[a cat]" {
					t.Error("the description in brackets")
				}
				if s.Text == "a cat" {
					found = s.Underline && s.Color == link
				}
			}
		}
	}
	if !found {
		t.Error("the description is not in the link's look")
	}
}

// Theme.MarkerGap keeps a task's text marker the check box of upstream's
// tasks: a control of OnTask's, named by its task, that reports its line
// and the other state, and read-only without OnTask.
func TestThemeMarkerGapTasksStayCheckBoxes(t *testing.T) {
	var got []taskCall
	o := &Options{Theme: &Theme{MarkerGap: 8, NestIndent: 16}, OnTask: func(line int, checked bool) { got = append(got, taskCall{line, checked}) }}
	src := "- [ ] milk\n- [x] eggs\n  - [ ] free range\n- bread\n"
	tt := ui.NewTester(func(c *ui.Context) { View(c, src, o) }, 500, 400)
	boxes := checkBoxes(tt)
	names := []string{"milk", "eggs", "free range"}
	want := []taskCall{{1, true}, {2, false}, {3, true}}
	if len(boxes) != len(want) {
		t.Fatalf("%d check boxes, want %d", len(boxes), len(want))
	}
	for k, a := range boxes {
		if a.Name != names[k] || a.Checked == want[k].checked || !a.Focusable || a.ReadOnly {
			t.Errorf("box %d is %+v, want %q, checked %v, focusable and not read-only", k, a, names[k], !want[k].checked)
		}
		clickBox(tt, a)
	}
	if !slices.Equal(got, want) {
		t.Errorf("OnTask got %v, want %v", got, want)
	}
	o.OnTask = nil
	tt.Frame()
	for k, a := range checkBoxes(tt) {
		if !a.ReadOnly || a.Focusable {
			t.Errorf("box %d without OnTask is %+v", k, a)
		}
	}
}
