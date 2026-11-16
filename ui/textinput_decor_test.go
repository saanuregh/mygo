package ui

import (
	"math"
	"testing"

	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/internal/scene"
	"github.com/egoist/mygo/internal/text"
)

// caretAt returns where the caret before rune i of a laid out text area
// el is, in the window: its x, top and height.
func caretAt(el *node, i int) (x, y, h float32) {
	ed, st := el.st.editor, el.st
	cx, cy, ch := ed.area.caretAt(ed, i, 0)
	return st.x + ed.originX + cx, st.y + ed.originY + float32(cy-ed.area.scroll), ch
}

func closeTo(a, b, by float32) bool { return float32(math.Abs(float64(a-b))) <= by }

// filled reports whether the last frame filled a box in color c.
func filled(tt *Tester, c Color) bool {
	for _, op := range tt.h.last.Ops {
		if op.Kind == scene.OpFill && op.Color == c.scene() {
			return true
		}
	}
	return false
}

// A range's font, size and slant lay its runes out in them, and its
// strikethrough and background paint over and behind them.
func TestTextRangeFontSizeAndSlant(t *testing.T) {
	s := "Big title\ncode then slant done"
	tint := RGB(30, 160, 90)
	var el *node
	tt := coreNewTester(func(c *context) {
		el = coreTextAreaBase(c, &s).Width(300).Height(120).FontSize(13).TextRanges(
			TextRange{Start: 0, End: 9, Size: 26, Weight: 700},
			TextRange{Start: 10, End: 14, Font: "monospace", Background: tint, Radius: 3},
			TextRange{Start: 20, End: 25, Italic: true},
			TextRange{Start: 26, End: 30, Strikethrough: true},
		)
	}, 320, 140)
	defer tt.rt.close()
	ed := el.st.editor
	title, body := ed.area.paraLayout(ed, 0), ed.area.paraLayout(ed, 1)
	if title.Height <= body.Height+4 {
		t.Errorf("the 26 DIP title's line is %v high, the body's %v", title.Height, body.Height)
	}
	spans := text.DecodeSpans(body.Params.Spans)
	var mono, italic bool
	for _, sp := range spans {
		mono = mono || sp.Family == "monospace"
		italic = italic || sp.Italic
	}
	if !mono || !italic {
		t.Errorf("the body's spans %+v: monospace %v, italic %v", spans, mono, italic)
	}
	_, sp := ed.rangeSpans(ed.buf.start(1), ed.buf.end(1))
	struck := false
	for _, s := range sp.spans {
		struck = struck || s.Strikethrough
	}
	if !struck {
		t.Error("no span struck through")
	}
	if !filled(tt, tint) {
		t.Error("the code's background was not painted")
	}
}

// Hidden runes take no room while the caret is away from their reveal
// range, and show as the caret comes to it; hidden for good, the caret
// steps over them and a click among them puts it at an edge.
func TestTextRangesHidden(t *testing.T) {
	s := "a **b** c"
	var el *node
	tt := coreNewTester(func(c *context) {
		el = coreTextAreaBase(c, &s).Width(300).Height(80).AutoFocus().TextRanges(
			TextRange{Start: 2, End: 4, Hidden: true, RevealStart: 2, RevealEnd: 7},
			TextRange{Start: 4, End: 5, Weight: 700},
			TextRange{Start: 5, End: 7, Hidden: true, RevealStart: 2, RevealEnd: 7},
		)
	}, 320, 100)
	defer tt.rt.close()
	tt.Key(0, KeyEnd)
	ed := el.st.editor
	plain := ed.area.paraLayout(ed, 0).Params
	plain.Text, plain.Spans = "a b c", text.EncodeSpans([]text.Span{{End: 2}, {End: 3, Weight: 700}})
	want := ed.sys.Shape(plain).Lines[0].Width
	if w := ed.area.paraLayout(ed, 0).Lines[0].Width; !closeTo(w, want, 0.5) {
		t.Errorf("hidden, the line is %v wide, %q %v", w, "a b c", want)
	}
	tt.Key(0, KeyLeft)
	tt.Key(0, KeyLeft) // at the closing marks: shown
	if w := ed.area.paraLayout(ed, 0).Lines[0].Width; w < want+5 {
		t.Errorf("shown, the line is %v wide, hidden %v", w, want)
	}
	tt.Key(0, KeyEnd)
	if w := ed.area.paraLayout(ed, 0).Lines[0].Width; !closeTo(w, want, 0.5) {
		t.Errorf("hidden again, the line is %v wide", w)
	}

	s2 := "ab**cd"
	tt2 := coreNewTester(func(c *context) {
		el = coreTextAreaBase(c, &s2).Width(300).Height(80).AutoFocus().
			TextRanges(TextRange{Start: 2, End: 4, Hidden: true})
	}, 320, 100)
	defer tt2.rt.close()
	ed = el.st.editor
	tt2.Key(0, KeyHome)
	tt2.Key(0, KeyRight)
	tt2.Key(0, KeyRight)
	tt2.Key(0, KeyRight)
	if ed.caret != 4 {
		t.Errorf("Right from b: caret %d, want 4 past the hidden marks", ed.caret)
	}
	tt2.Key(0, KeyLeft)
	if ed.caret != 2 {
		t.Errorf("Left: caret %d, want 2", ed.caret)
	}
	x, y, h := caretAt(el, 2)
	tt2.ClickAt(x, y+h/2)
	if ed.caret != 2 && ed.caret != 4 {
		t.Errorf("a click on the hidden marks: caret %d, inside them", ed.caret)
	}
}

// A widget is a box of its size over its runes, painted by its Paint: the
// caret steps over it at once, Backspace takes it out whole, and a click on
// it is its OnClick's, the caret staying, with a hand over it.
func TestTextWidgets(t *testing.T) {
	s := "see @src/panel.go now"
	var el *node
	var painted []Rect
	clicks := 0
	clickable := true
	tt := coreNewTester(func(c *context) {
		w := TextWidget{Start: 4, End: 17, Width: 60, Height: 18, Descent: 4,
			Paint: func(p *Painter, r Rect) { painted = append(painted, r) }}
		if clickable {
			w.OnClick = func() { clicks++ }
		}
		el = coreTextAreaBase(c, &s).Width(300).Height(80).AutoFocus().TextWidgets(w)
	}, 320, 100)
	defer tt.rt.close()
	ed := el.st.editor
	x4, y, h := caretAt(el, 4)
	x17, _, _ := caretAt(el, 17)
	if !closeTo(x17-x4, 60, 0.5) {
		t.Errorf("the widget takes %v, want 60", x17-x4)
	}
	if len(painted) == 0 || !closeTo(painted[len(painted)-1].X, x4, 0.5) || painted[len(painted)-1].W != 60 || painted[len(painted)-1].H != 18 {
		t.Fatalf("painted at %v, want at x %v, 60×18", painted, x4)
	}
	tt.Key(0, KeyEnd)
	for range 4 {
		tt.Key(0, KeyLeft)
	}
	if ed.caret != 17 {
		t.Fatalf("caret %d, want 17 after the widget", ed.caret)
	}
	// The keys come as the next frame is built, before the view gives its
	// widgets again: they move through what the last frame showed.
	tt.Key(0, KeyLeft)
	if ed.caret != 4 {
		t.Errorf("Left over the widget: caret %d, want 4", ed.caret)
	}
	tt.Key(0, KeyRight)
	if ed.caret != 17 {
		t.Errorf("Right over the widget: caret %d, want 17", ed.caret)
	}
	tt.Key(Shift, KeyLeft)
	if a, c := ed.anchor, ed.caret; a != 17 || c != 4 {
		t.Errorf("Shift+Left selects %d to %d, want 17 to 4", a, c)
	}
	tt.Key(0, KeyRight)

	tt.Move(x4+30, y+h/2)
	if c := tt.Cursor(); c != CursorPointer {
		t.Errorf("the cursor over a clickable widget is %v", c)
	}
	tt.ClickAt(x4+30, y+h/2)
	if clicks != 1 || ed.caret != 17 {
		t.Errorf("a click on the widget: %d clicks, caret %d, want 1 and 17", clicks, ed.caret)
	}
	clickable = false
	tt.Frame()
	tt.Move(x4+30, y+h/2)
	if c := tt.Cursor(); c == CursorPointer {
		t.Error("a hand over a widget without OnClick")
	}
	tt.ClickAt(x4+10, y+h/2)
	if ed.caret != 4 {
		t.Errorf("a click near the widget's start: caret %d, want 4", ed.caret)
	}
	tt.ClickAt(x4+50, y+h/2)
	if ed.caret != 17 {
		t.Errorf("a click near the widget's end: caret %d, want 17", ed.caret)
	}
	// The release ends the press as a drag from the widget's end to it,
	// which selects nothing, not the runes laid out there.
	if lo, hi := ed.selection(); lo != hi {
		t.Errorf("a click at the widget's end selects %d to %d", lo, hi)
	}
	tt.Key(0, KeyBackspace)
	if s != "see  now" {
		t.Errorf("Backspace after the widget leaves %q", s)
	}
}

// A block indents its paragraphs, which wrap at the width less the indent,
// and paints its box behind them; clicks and the caret follow the indent.
func TestTextBlocks(t *testing.T) {
	s := "plain\nan indented paragraph that wraps in the narrow area\nend"
	bg := RGB(40, 40, 70)
	var el *node
	var box Rect
	tt := coreNewTester(func(c *context) {
		el = coreTextAreaBase(c, &s).Width(220).Lines(1, 20).AutoFocus().TextBlocks(TextBlock{
			Start: 6, End: 56, Indent: 40, Background: bg, Radius: 6,
			Paint: func(p *Painter, r Rect) { box = r },
		})
	}, 260, 300)
	defer tt.rt.close()
	ed := el.st.editor
	x0, _, _ := caretAt(el, 0)
	x1, y1, h1 := caretAt(el, 6)
	if !closeTo(x1-x0, 40, 0.5) {
		t.Errorf("the block's text starts %v right of the plain text's, want 40", x1-x0)
	}
	l := ed.area.paraLayout(ed, 1)
	if !closeTo(l.Params.Width, ed.contentW-40, 0.5) || len(l.Lines) < 2 {
		t.Errorf("the block is laid out %v wide in %d lines, the area %v", l.Params.Width, len(l.Lines), ed.contentW)
	}
	if !filled(tt, bg) {
		t.Error("the block's background was not painted")
	}
	top := el.st.y + ed.originY + float32(ed.area.hs.top(1))
	bottom := el.st.y + ed.originY + float32(ed.area.hs.top(2))
	if !closeTo(box.Y, top, 0.5) || !closeTo(box.Y+box.H, bottom, 0.5) || !closeTo(box.W, ed.contentW, 0.5) {
		t.Errorf("the block's box %v, want from %v to %v, %v wide", box, top, bottom, ed.contentW)
	}
	// As high as its lines, the indented ones wrapped narrower.
	var lines float32
	for p := range ed.buf.paras {
		lines += ed.area.paraLayout(ed, p).Height
	}
	if got := el.st.h - ed.originY*2; !closeTo(got, lines, 1) {
		t.Errorf("the area is %v high, its paragraphs %v", got, lines)
	}
	tt.ClickAt(x1+1, y1+h1/2)
	if ed.caret != 6 {
		t.Errorf("a click at the indented text's start: caret %d, want 6", ed.caret)
	}
	tt.ClickAt(x0+2, y1+h1/2)
	if ed.caret != 6 {
		t.Errorf("a click in the indent: caret %d, want 6", ed.caret)
	}
}

// An input method composes beside widgets and hidden runes as in plain
// text: the composition sits at the caret, the widget's box after it
// moves past it, and the runes hidden stay hidden.
func TestTextDecorationsWhileComposing(t *testing.T) {
	s := "see @x.go **b**"
	var el *node
	var at float32
	tt := coreNewTester(func(c *context) {
		el = coreTextAreaBase(c, &s).Width(300).Height(80).AutoFocus().
			TextWidgets(TextWidget{Start: 4, End: 9, Width: 50, Height: 16, Paint: func(p *Painter, r Rect) { at = r.X }}).
			TextRanges(TextRange{Start: 10, End: 12, Hidden: true, RevealStart: 10, RevealEnd: 15},
				TextRange{Start: 13, End: 15, Hidden: true, RevealStart: 10, RevealEnd: 15})
	}, 320, 100)
	defer tt.rt.close()
	ed := el.st.editor
	tt.Key(0, KeyHome)
	for range 4 {
		tt.Key(0, KeyRight)
	}
	before := at
	tt.send(platform.SurfaceEvent{Kind: platform.TextComposition, Text: "かな", Caret: 2})
	if ed.compose != "かな" || ed.caret != 4 {
		t.Fatalf("composing %q at %d", ed.compose, ed.caret)
	}
	if at <= before+5 {
		t.Errorf("the widget at %v composing before it, %v before", at, before)
	}
	l := ed.area.paraLayout(ed, 0)
	if l.Runes[4] != 'か' {
		t.Errorf("the layout holds %q", string(l.Runes))
	}
	// The marks are hidden: "**b**" is about as wide as "b".
	x15, _, _ := l.CaretAt(text.CaretPosition{Index: 17})
	xb, _, _ := l.CaretAt(text.CaretPosition{Index: 15})
	if x15-xb > 0.1 {
		t.Errorf("the closing marks take %v while composing", x15-xb)
	}
	tt.send(platform.SurfaceEvent{Kind: platform.TextInput, Text: "かな"})
	if s != "see かな@x.go **b**" {
		t.Errorf("composed %q", s)
	}
}

// In a single-line input too, a widget is one box the caret steps over,
// and a double click on it selects it whole, with the word it is in.
func TestTextWidgetsInALine(t *testing.T) {
	s := "to @src/panel.go now"
	var el *node
	var painted int
	tt := coreNewTester(func(c *context) {
		el = coreTextInputBase(c, &s).Width(300).AutoFocus().TextWidgets(TextWidget{Start: 3, End: 16, Width: 48, Height: 16,
			Paint: func(p *Painter, r Rect) { painted++ }})
	}, 320, 60)
	defer tt.rt.close()
	ed := el.st.editor
	if painted == 0 {
		t.Fatal("the widget was not painted")
	}
	x3, _, _ := ed.layout.CaretAt(text.CaretPosition{Index: 3})
	x16, _, _ := ed.layout.CaretAt(text.CaretPosition{Index: 16})
	if !closeTo(x16-x3, 48, 0.5) {
		t.Errorf("the widget takes %v, want 48", x16-x3)
	}
	tt.Key(0, KeyHome)
	for range 4 {
		tt.Key(0, KeyRight)
	}
	if ed.caret != 16 {
		t.Errorf("caret %d after four steps, want 16", ed.caret)
	}
	y := el.st.y + el.st.h/2
	x := el.st.x + ed.originX + x3 + 24
	tt.ClickAt(x, y)
	tt.ClickAt(x, y)
	if lo, hi := ed.selection(); lo > 3 || hi < 16 {
		t.Errorf("a double click on the widget selects %d to %d", lo, hi)
	}
}

// A block's marker is drawn in the indent before its first line, and a
// click there is its OnMarkerClick's, the caret staying.
func TestTextBlockMarkers(t *testing.T) {
	s := "- [ ] a task whose words wrap onto a second line here\nnext"
	var el *node
	var mark Rect
	clicks := 0
	tt := coreNewTester(func(c *context) {
		el = coreTextAreaBase(c, &s).Width(200).Lines(1, 10).AutoFocus().
			TextRanges(TextRange{Start: 0, End: 6, Hidden: true}).
			TextBlocks(TextBlock{Start: 0, End: 53, Indent: 24,
				Marker:        func(p *Painter, r Rect) { mark = r },
				OnMarkerClick: func() { clicks++ }})
	}, 240, 200)
	defer tt.rt.close()
	ed := el.st.editor
	x6, y6, h6 := caretAt(el, 6)
	if !closeTo(mark.X+mark.W, x6, 0.5) || mark.W != 24 || !closeTo(mark.Y, y6, 0.5) || !closeTo(mark.H, h6, 0.5) {
		t.Errorf("the marker at %v, the text from %v, %v", mark, x6, y6)
	}
	// The second line hangs at the indent too.
	l := ed.area.paraLayout(ed, 0)
	if len(l.Lines) < 2 {
		t.Fatal("the task does not wrap")
	}
	x2, _, _ := caretAt(el, l.Lines[1].Start)
	if !closeTo(x2, x6, 0.5) {
		t.Errorf("the second line starts at %v, the first's words at %v", x2, x6)
	}
	caret := ed.caret
	tt.Move(mark.X+mark.W/2, mark.Y+mark.H/2)
	if c := tt.Cursor(); c != CursorPointer {
		t.Errorf("the cursor over a clickable marker is %v", c)
	}
	tt.ClickAt(mark.X+mark.W/2, mark.Y+mark.H/2)
	if clicks != 1 || ed.caret != caret {
		t.Errorf("a click on the marker: %d clicks, caret %d, was %d", clicks, ed.caret, caret)
	}
}
