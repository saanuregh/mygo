package editor

import (
	"slices"
	"time"

	"github.com/egoist/mygo/ui"
)

// State is a code editor's state, kept by the app across
// frames: the buffer it edits, its options, and what the editor keeps of
// its own, as its selections, folds and scroll position. Set the options
// before building the editor; zero values are the defaults, those of Zed.
type State struct {
	// Buffer is the text the editor edits; one is made when nil.
	Buffer *Buffer
	// Font is the editor's font: the system's monospaced font of 13 DIPs
	// when zero.
	Font ui.Font
	// LineHeight is the height of lines as a multiple of the font's size,
	// 1.5 when 0.
	LineHeight float32
	// TabSize is how many columns a tab and a level of indentation take:
	// the language's, else 4, when 0.
	TabSize int
	// Indent says whether indenting inserts tabs or spaces: as the
	// language's files usually do (Go's tabs) by default.
	Indent IndentStyle
	// SoftWrap wraps lines at the editor's width, or at WrapColumn
	// columns when narrower.
	SoftWrap   bool
	WrapColumn int
	// WrapAtSpaces wraps as editors that count columns do: a character
	// takes a column, a wide one two; rows break only after spaces and
	// tabs (a word longer than a row before the character that does not
	// fit, never inside one) and are not indented. By default rows also
	// break between words and punctuation, and are indented as their
	// line.
	WrapAtSpaces bool
	// LineNumbers shows the numbers of lines in the gutter, absolute by
	// default.
	LineNumbers LineNumbers
	// ReadOnly makes the text selectable and copyable, not editable.
	ReadOnly bool
	// Whitespace says where spaces and tabs show as dots and arrows:
	// within selections by default.
	Whitespace Whitespace
	// WrapGuides draws guides at these columns, as 80 and 100.
	WrapGuides []int
	// NoIndentGuides hides the lines that show indentation levels.
	NoIndentGuides bool
	// NoScrollPastEnd stops scrolling at the last line rather than
	// letting it scroll up to the top.
	NoScrollPastEnd bool
	// NoAutoClose stops closing brackets and quotes as they are typed.
	NoAutoClose bool
	// NoFolding hides the fold markers of the gutter and turns folding
	// off.
	NoFolding bool
	// NoActiveLine stops highlighting the lines with a caret.
	NoActiveLine bool
	// CursorShape is the shape of the carets: bars by default, or blocks
	// over the character after them, as Vim's normal mode has, or
	// underlines, as its pending operators do.
	CursorShape CursorShape
	// FitContent makes the editor as high as its lines, to show in a
	// scroll view or a list, rather than scrolling them itself.
	FitContent bool
	// KeepScrollbars keeps the thumbs of the scroll bars, and the marks
	// of the carets on the vertical one, showing while the text has
	// anywhere to scroll, rather than only while the pointer is over the
	// editor or drags a thumb, as ui's KeepScrollbars does for a scroll
	// container; they look the same either way.
	// NoScrollbars takes the scroll bars away, with the marks of changes,
	// matches and diagnostics on them, and their room, which goes to the
	// text; the wheel, the keys and SetScrollY still scroll it.
	KeepScrollbars bool
	NoScrollbars   bool
	// Scrollbar styles the editor's scroll bars, as a theme's
	// ScrollbarStyle does: their track and thumb colors, the widths of
	// their bars and thumbs, their corners and shortest thumb. Its zero
	// fields follow the theme's; its TrackWidth, 0, is 10 DIPs unless the
	// theme's style sets one, as the editor's bar has always been.
	Scrollbar ui.ScrollbarStyle
	// Placeholder shows while the buffer is empty.
	Placeholder string
	// Label names the editor for assistive technology.
	Label string
	// Theme colors the editor; DefaultTheme of the window's theme
	// when nil.
	Theme *Theme
	// Bindings add key bindings, or replace the default ones for their
	// keys (DefaultBindings).
	Bindings []Binding
	// DiffBase, when set, is the text the buffer's changes are shown
	// against in the gutter, as the version of a file in Git: bars mark
	// lines added and changed, and a mark where lines were deleted.
	DiffBase *Buffer
	// DiffPlain compares the whole texts again at each change of either,
	// with Myers's algorithm on what their ends leave, none of the
	// changes slid (zode's Zig diff.zig), where the default compares the
	// lines near an edit, with git's histogram and its sliding, and puts
	// the line Enter opened where the edit was. Large texts are compared
	// off the main thread as the default's whole comparisons are.
	DiffPlain bool

	// OnCompletion asks the app for completions where the caret is, as
	// a word is typed or Ctrl+Space pressed: answer with ShowCompletions,
	// at once or later.
	OnCompletion func(CompletionRequest)
	// CompletionScore, when set, ranks the completions for the word typed
	// in place of the editor's own matching: a higher score first, ties by
	// their SortText; ok false leaves one out.
	CompletionScore func(query, text string) (score int, ok bool)
	// OnHover asks the app what to show for the text under the pointer
	// once it rests there: answer with ShowHover.
	OnHover func(HoverRequest)
	// HoverDelay is how long the pointer rests on the text before the
	// hover is asked for: a tooltip's, 600 ms, when zero or less.
	HoverDelay time.Duration
	// NoPointerHover shows no hover as the pointer rests, an app's nor a
	// diagnostic's, as VS Code's editor.hover.enabled off: ShowHover, as
	// Cmd+K Cmd+I, still asks for one at the caret.
	NoPointerHover bool
	// OnDefinition is called for Cmd+click (Ctrl+click elsewhere), with
	// any other modifiers held too, and F12 on a word, as to go to its
	// definition.
	OnDefinition func(offset int)
	// NoDefinitionClick leaves Cmd+click to the editor's other clicks,
	// with no link under it: F12 and the context menu still go to the
	// definition.
	NoDefinitionClick bool
	// NoDefinitionLink shows no link under the pointer with Cmd held,
	// though Cmd+click still goes to the definition.
	NoDefinitionLink bool
	// OnInlayHover is told which inlay hint the pointer is over as it
	// comes onto one, moves to another and leaves them (Index -1), as to
	// show a hint's tooltip once it rests there (ShowHover at the hint's
	// Offset keeps it while the pointer stays on the hint). While the
	// pointer is on a hint, OnHover is not asked about the text beside
	// it. SetInlays forgets the hint under the pointer: the next move
	// tells it again. Trailing inlays are not hints: OnHover is asked
	// about them as about the line's end.
	OnInlayHover func(InlayHover)
	// OnInlayClick is told of a press of the primary button on an inlay
	// hint, not a trailing inlay, as Cmd+click (Ctrl+click elsewhere) to
	// go where its label points; returning true takes the press, which
	// then neither moves the carets nor goes to a definition.
	OnInlayClick func(InlayClick) bool
	// OnGutterClick is called for a click on a line's number, as to set a
	// breakpoint.
	OnGutterClick func(line int)
	// OnChangeBarClick is told of a press on a change bar (or a deletion's
	// wedge) against DiffBase, with that bar's hunk; true takes the press
	// (OnGutterClick is not called). The bar's strip is its 3 DIPs and 3
	// more each side. The pointer is a hand over a bar while it is set.
	OnChangeBarClick func(h DiffHunk) bool
	// ContextMenu adds items to the editor's context menu, after its
	// own (or Menu's).
	ContextMenu func(m *ui.Menu)
	// Menu builds the editor's context menu in place of its own items,
	// which DefaultMenu adds, as an app with commands of its own for
	// code builds its menu; ContextMenu's items follow it.
	Menu func(m *ui.Menu)
	// GutterMenu builds the menu of a right click on a line's number in
	// place of the editor's, for that line, as a debugger's breakpoint
	// menu; the selections stay as they are.
	GutterMenu func(m *ui.Menu, line int)
	// Blocks are rows between the lines that are not text, as the lines
	// an inline diff deleted, each showing what RenderBlock builds in
	// it: the carets and the keys moving them pass over them, and a
	// fold hiding a line hides its blocks. As LineBackgrounds, they keep
	// their lines as the text is edited: compute them again when the
	// buffer's Version changes.
	Blocks []Block
	// RenderBlock builds a block in the element the editor places over
	// it, as wide as the editor but its vertical scroll bar's track and
	// as high as the block: the editor's background shows unless it is
	// given one, and its content lines up with the text from TextLeft.
	// The wheel over it scrolls the editor; presses are its own.
	RenderBlock func(c *ui.Context, box ui.Element, b Block)
	// HoverBelow places the hover as zode's hover_card.zig: from a rect 10
	// above the resting pointer and 26 high, or the caret's (the keyboard
	// asked), 4 below it while there is room below for its height or 160,
	// else 4 above it; 12 left of it and 8 inside the window, at most 600
	// wide and 320 high (or the room on its side). With HoverAppOnly, a
	// press on the word hovered keeps it.
	HoverBelow bool
	// HoverAppOnly shows in the hover only what the app gives ShowHover:
	// no diagnostics, and nothing before its answer. The hover goes as
	// soon as the pointer leaves the text hovered, but into the hover.
	// The pointer rests on a name only (nothing on punctuation), and
	// HoverDelay's wait runs on across the name, starting again on another
	// name alone; the hover is asked at the name's start.
	HoverAppOnly bool
	// GutterHoverMark is drawn on the line under the pointer while it is
	// over the gutter, where the line has no mark of its own: where a
	// click would put one (its Line is not read).
	GutterHoverMark GutterMark
	// NoContextMenu leaves the editor without a menu of its own: a right
	// click goes to the nearest element around it that has one.
	NoContextMenu bool
	// GutterMarks marks lines in the gutter, as breakpoints or
	// bookmarks: a dot of their color.
	GutterMarks []GutterMark
	// LineBackgrounds fill the rows of lines with colors, under the
	// text and the selections, as the statement a key runs, from the
	// text's column: not the gutter, nor the lines folds hide. They keep their line numbers as
	// the text is edited, unlike diagnostics and inlays, which move with
	// it: compute them again when the buffer's Version changes.
	LineBackgrounds []LineBackground
	// TextBackgrounds fill ranges of the text with colors, under the
	// text and over line backgrounds, as the words a merge changed:
	// across wrapped rows, tabs and wide characters as the editor lays
	// them out, not the lines folds hide. As LineBackgrounds, they stay
	// where they are as the text is edited: compute them again when the
	// buffer's Version changes.
	TextBackgrounds []TextBackground
	// OnInput sees each input event before the editor, as keys for a
	// mode of the app's such as Vim's: returning true takes it, and the
	// editor leaves it. Text typed at the carets comes as InputText;
	// what an input method composes, commits or replaces, as the accents
	// of press and hold, goes to the editor alone. The key after the
	// first of a chord of Bindings, as Cmd+K Cmd+0, is the chord's.
	OnInput func(ev ui.InputEvent) bool
	// OnType, when set, types the text typed at the carets (InputText,
	// and what an input method commits) in place of the editor's own
	// typing, which closes brackets and outdents a line its text closes:
	// it returns the edits, in the text as it is, and the selections
	// after them; the editor makes them as typing, one step of undo with
	// the typing around it, and asks for completions as it would. ok
	// false leaves the text to the editor.
	OnType func(text string) (edits []Edit, after []Selection, ok bool)
	// NoInputMethod turns the system's input methods off while the
	// editor has the keyboard (ui's InputMethod): keys type the
	// characters of the keyboard's layout, which come to OnInput as
	// InputText, and nothing is composed, as Vim's normal mode wants
	// with a Chinese input method on. Browsers cannot turn input methods
	// off: there they compose in the text still.
	NoInputMethod bool

	// RenderFindBar, RenderGoToLine, RenderCompletions and RenderHover
	// build the editor's find bar, go to line bar, completions and hovers
	// in place of its own (DefaultFindBar, DefaultGoToLine,
	// DefaultCompletions and DefaultHover, which they can call to wrap
	// them): each is given the element the editor places, as a toolbar
	// over its top right corner or a list below the caret, to style and
	// fill, and a handle to what the part shows and does.
	RenderFindBar     func(c *ui.Context, bar ui.Element, f *FindBar)
	RenderGoToLine    func(c *ui.Context, bar ui.Element, g *GoToLineBar)
	RenderCompletions func(c *ui.Context, panel ui.Element, m *CompletionMenu)
	RenderHover       func(c *ui.Context, panel ui.Element, h *HoverCard)
	// CaretCard, when set, builds a card the app shows at the newest
	// caret, as a language server's signature help: given the element
	// the editor places, its bottom left 4 DIPs above the caret's line
	// (4 below it while completions show, or where the window has no room
	// above), to style and fill. Its end margin moves it as AttachTo's
	// margins do: MarginEnd(40) starts it 40 DIPs left of the caret.
	CaretCard func(c *ui.Context, card ui.Element)
	// CaretCardShift places CaretCard's card as at a caret this many DIPs
	// right of the caret (zode's view.zig measured its popups' caret from
	// 6 points past the text's left edge).
	CaretCardShift float32
	// FindBarAbove places the find bar in a strip across the editor's
	// top, as wide as the editor and as high as RenderFindBar builds it,
	// above the text, which starts below it while it shows (painting,
	// the pointer and scrolling go by the text's box), rather than over
	// the text's top right corner.
	FindBarAbove bool
	// WideSidewaysRoom keeps room past the caret as it moves sideways of
	// a quarter of the text's width, at most 48 DIPs, as zode's editor
	// (view.zig applyAutoscroll), rather than two characters.
	WideSidewaysRoom bool
	// SidewaysCut, with WideSidewaysRoom, is how much narrower than the
	// text's area (short of MarginRight) the text counts as when keeping
	// the caret in view sideways.
	SidewaysCut float32
	// PaddingTop is room above the text, in the editor's background, that
	// lines never scroll into (below the find bar while it shows above).
	PaddingTop float32
	// Gutter, when set, lays the gutter out by its sizes in place of the
	// editor's own (GutterLayout).
	Gutter *GutterLayout
	// LineNotes are short texts drawn after their lines' text, as a
	// diagnostic's message or who last changed the line: in NoteFont
	// (italic), cut at the editor's edge, and no part of the line, which
	// they neither lengthen nor wrap. A line shows its first note.
	LineNotes []LineNote
	// NoteFont is the notes' font: the editor's, a DIP smaller, when
	// zero. They are drawn italic.
	NoteFont ui.Font
	// MarginRight is room at the right edge the text leaves to the app,
	// as for a minimap drawn over it: lines wrap and scroll short of it.
	// The scroll bars stay at the edge, over it (NoScrollbars hides them).
	MarginRight float32
	// FindFromOrigin has the find bar's query seeded by the selection
	// alone (not the word at the caret, nor one of 1000 bytes or more),
	// and any change of the query, its options or the text select the
	// match at or after where the selection started as the bar opened,
	// rather than after the match last selected. Find hides replacing;
	// FindReplace over a query gives the replacement the focus
	// (FindBar.FocusReplacement). Empty matches of a regular expression
	// count, its \w, \d and \s (and opposites) are Unicode's, and a
	// pattern that is not one is searched for as typed, its error still
	// said, as zode's rure did.
	FindFromOrigin bool
	// FilledBrackets marks the bracket next to the caret (after it, else
	// before it) and its partner, filled with Theme.MatchingBracket,
	// rather than outlining the brackets around the caret; with
	// FitContent, only while the editor has the keyboard.
	FilledBrackets bool
	// CaretBeforeInlays draws a caret at an inlay hint's offset before
	// the hint rather than after it.
	CaretBeforeInlays bool
	// NoOccurrences stops highlighting the other occurrences of the
	// selected text.
	NoOccurrences bool
	// RoundRanges rounds the selections' corners by 3 DIPs, and draws the
	// find bar's matches and the app's text backgrounds a DIP short of a
	// row's top and bottom, rounded alike: an empty one, as an empty
	// match, as a bar 2 DIPs wide.
	RoundRanges bool
	// CaretRowOnly highlights the caret's row (not every row of its
	// wrapped line) and only while there is one caret and no selection.
	CaretRowOnly bool
	// GuidesFromBelow gives a blank line the indent guides of the next
	// line with text (none after the last), not the deeper of the lines
	// around it, and draws a line's guides beside its first row alone,
	// where the characters' columns fall, not rounded to a whole DIP (as
	// zode's editor).
	GuidesFromBelow bool
	// CaretSolidFor keeps the caret shown this long after it moved or the
	// text changed, before it blinks (shown first).
	CaretSolidFor time.Duration
	// NoCaretBlink keeps the carets shown while the editor has the
	// keyboard.
	NoCaretBlink bool
	// StickyLines are lines pinned in the top rows of the text, outermost
	// first, while the text scrolls under them (at most five), drawn as
	// the editor draws lines, with their numbers. A click on one scrolls
	// it to the top and puts the caret at its start. Nil pins none.
	StickyLines []int
	// FontCarets draws carets as tall as the font's line and 4 DIPs (the
	// row at most), centered in the row and rounded by 1; a block at 55%
	// of Theme.Cursor; and none while the editor has not the keyboard.
	FontCarets bool
	// HintsFollowText keeps an inlay hint before text inserted at it and
	// drops one whose text is replaced around it, as zode's editor did;
	// unset, a hint moves after text inserted at it. Trailing inlays and
	// suggestions move as without it.
	HintsFollowText bool
	// StraightUnderlines underlines diagnostics with a straight rounded
	// line, 2 DIPs thick, rather than a wavy one; a diagnostic of no
	// width under the character after it, or, at a line's end, with a
	// mark 2 DIPs wide rather than a character's.
	StraightUnderlines bool
	// GuidesUnderRanges paints the indent guides under the app's text
	// ranges, the find bar's matches, the brackets and the selections,
	// and the brackets under the selections, as zode's editor did.
	GuidesUnderRanges bool
	// BandsInset, when set, has the caret line's band and LineBackgrounds
	// span from the gutter's edge to BandsInset DIPs short of the
	// editor's right edge, as zode's editor drew its rows' bands.
	BandsInset float32
	// PageMovesCarets has Page Up and Page Down move the carets a page of
	// the text below PaddingTop less two rows, scrolling only as far as
	// keeps the caret in view, rather than scrolling a page.
	PageMovesCarets bool
	// BlockCaretWhenEmpty draws CursorBlock's block only for a caret
	// with nothing selected, a bar for a selection's, as zode's Vim.
	BlockCaretWhenEmpty bool
	// PlainClicks leaves out Alt+Shift dragging a column selection,
	// Alt+click taking a caret away and a click on a fold's placeholder
	// unfolding it: Alt+Shift+click extends the selection as Shift+click,
	// Alt+click adds a caret.
	PlainClicks bool
	// PrimarySelection keeps the primary selection as Linux desktops'
	// editors do: what the focused editor selects goes there once the
	// selection settles, and a middle press puts the caret at the pointer
	// and pastes it there. The clipboard is left as it is; where the
	// system has no primary selection it does nothing.
	PrimarySelection bool

	ed editor
}

// GutterLayout sizes the gutter: Left DIPs, the line numbers
// right-aligned in MinDigits digits at least, then Right DIPs, the text
// starting TextGap DIPs after the gutter. The change bars against
// DiffBase are 3 DIPs wide, BarRight DIPs left of the gutter's right edge,
// and fold markers sit at the gutter's left edge: Folded and Unfolded
// drawn as a symbol FoldIconSize DIPs big (in a box 1.2 times that, its
// top where the size centred in the row would start), 4 DIPs from it,
// in the Theme's FoldedIcon and FoldIcon, when set, else chevrons
// stroked. With FoldStripClicksText a press left of Left on a line that
// does not fold is the text's, at the line's start, not OnGutterClick's,
// and a right press there has no menu.
type GutterLayout struct {
	Left, Right, TextGap, BarRight float32
	MinDigits                      int
	Folded, Unfolded               *ui.SVG
	FoldIconSize                   float32
	FoldStripClicksText            bool
}

// LineNote is a note drawn after line Line's text (State.LineNotes), in
// Color, else in Theme.Note.
type LineNote struct {
	Line  int
	Text  string
	Color ui.Color
}

// IndentStyle is what indenting inserts.
type IndentStyle uint8

const (
	IndentAuto IndentStyle = iota
	IndentSpaces
	IndentTabs
)

// LineNumbers is how the gutter numbers lines.
type LineNumbers uint8

const (
	LineNumbersOn LineNumbers = iota
	LineNumbersRelative
	LineNumbersOff
)

// Whitespace is where spaces and tabs show.
type Whitespace uint8

const (
	WhitespaceSelection Whitespace = iota
	WhitespaceNone
	WhitespaceAll
	WhitespaceTrailing
	// WhitespaceBoundary shows whitespace but single spaces between
	// words.
	WhitespaceBoundary
)

// CursorShape is the shape of an editor's carets, as Zed's CursorShape
// and the terminal's CursorStyle.
type CursorShape uint8

const (
	// CursorBar is a bar before the character at the caret.
	CursorBar CursorShape = iota
	// CursorBlock covers the character after the caret, a space's width
	// at the end of a line, translucent so that the character shows. An
	// editor without the keyboard focus outlines it, as Vim's and Zed's
	// do.
	CursorBlock
	// CursorUnderline underlines the character after the caret.
	CursorUnderline
)

// GutterMark marks a line in the gutter.
type GutterMark struct {
	Line  int
	Color ui.Color
	// Shape is a dot, a square or an arrow; Size its width and height
	// (0.45 of a line's height when 0). Beside puts it 4 left of the
	// line's number rather than at the gutter's left edge.
	Shape  GutterShape
	Size   float32
	Beside bool
}

// LineBackground fills lines First to Last, from 0, with Color. The lines
// are those of the text when it is set: an edit adding or removing lines
// above them doesn't move them.
type LineBackground struct {
	First, Last int
	Color       ui.Color
}

// TextBackground fills bytes Start to End of the text with Color. Ranges
// past the end of the text, as an edit leaves them until they are
// computed again, are cut to it; empty ones fill nothing. A range across
// lines fills the text of each, not past its end.
type TextBackground struct {
	Start, End int
	Color      ui.Color
	// Underline draws it as a bar 2 DIPs high under the text instead, as
	// a diagnostic's straight underline.
	Underline bool
}

func (st *State) options() options {
	return options{
		readOnly: st.ReadOnly, font: st.Font, lineHeight: st.LineHeight, tabSize: st.TabSize, indent: st.Indent,
		wrap: st.SoftWrap, wrapColumn: st.WrapColumn, wrapAtSpaces: st.WrapAtSpaces, lineNumbers: st.LineNumbers, whitespace: st.Whitespace,
		wrapGuides: st.WrapGuides, indentGuides: !st.NoIndentGuides, scrollPastEnd: !st.NoScrollPastEnd && !st.FitContent,
		keepScrollbars: st.KeepScrollbars, hideScrollbars: st.NoScrollbars, scrollbar: st.Scrollbar,
		autoClose: !st.NoAutoClose, folding: !st.NoFolding, activeLine: !st.NoActiveLine, cursorShape: st.CursorShape, fitContent: st.FitContent,
		placeholder: st.Placeholder, label: st.Label, theme: st.Theme, bindings: st.Bindings, gutterMarks: st.GutterMarks,
		backgrounds: st.LineBackgrounds, textBgs: st.TextBackgrounds, diffBase: st.DiffBase, diffPlain: st.DiffPlain, onCompletion: st.OnCompletion, completionScore: st.CompletionScore, onHover: st.OnHover, onDefinition: st.OnDefinition,
		hoverDelay: st.HoverDelay, noPointerHover: st.NoPointerHover,
		noDefinitionClick: st.NoDefinitionClick, noDefinitionLink: st.NoDefinitionLink,
		onInlayHover: st.OnInlayHover, onInlayClick: st.OnInlayClick, onGutterClick: st.OnGutterClick, onChangeBarClick: st.OnChangeBarClick, contextMenu: st.ContextMenu, menu: st.Menu, blocks: st.Blocks, renderBlock: st.RenderBlock, gutterMenu: st.GutterMenu, noContextMenu: st.NoContextMenu, hoverBelow: st.HoverBelow, hoverAppOnly: st.HoverAppOnly, gutterHoverMark: st.GutterHoverMark, onInput: st.OnInput, onType: st.OnType, noInputMethod: st.NoInputMethod, minDigits: 3,
		renderFindBar: st.RenderFindBar, renderGoToLine: st.RenderGoToLine,
		renderCompletions: st.RenderCompletions, renderHover: st.RenderHover, caretCard: st.CaretCard, caretCardShift: st.CaretCardShift, findBarAbove: st.FindBarAbove, paddingTop: st.PaddingTop, wideSidewaysRoom: st.WideSidewaysRoom, sidewaysCut: st.SidewaysCut,
		gutter: st.Gutter, lineNotes: st.LineNotes, noteFont: st.NoteFont, straightUnderlines: st.StraightUnderlines, marginRight: st.MarginRight, findFromOrigin: st.FindFromOrigin, noOccurrences: st.NoOccurrences, filledBrackets: st.FilledBrackets, caretBeforeInlays: st.CaretBeforeInlays,
		roundRanges: st.RoundRanges, caretRowOnly: st.CaretRowOnly, guidesFromBelow: st.GuidesFromBelow, caretSolidFor: st.CaretSolidFor, noCaretBlink: st.NoCaretBlink, stickyLines: st.StickyLines, fontCarets: st.FontCarets, hintsFollowText: st.HintsFollowText,
		guidesUnderRanges: st.GuidesUnderRanges, bandsInset: st.BandsInset, pageMovesCarets: st.PageMovesCarets, blockCaretWhenEmpty: st.BlockCaretWhenEmpty, plainClicks: st.PlainClicks,
		primarySelection: st.PrimarySelection,
	}
}

// Selections returns the editor's selections, in order.
func (st *State) Selections() []Selection {
	st.ensure()
	out := make([]Selection, len(st.ed.sels))
	for i, s := range st.ed.sels {
		out[i] = Selection{Anchor: s.anchor, Head: s.head}
	}
	return out
}

// Selection returns the newest selection, the one input methods and
// scrolling follow.
func (st *State) Selection() Selection {
	st.ensure()
	p := st.ed.primary()
	return Selection{Anchor: p.anchor, Head: p.head}
}

// SetSelections sets the selections, the last the newest, and scrolls
// it into view.
func (st *State) SetSelections(sels ...Selection) {
	st.ensure()
	if len(sels) == 0 {
		return
	}
	n := st.ed.view.text.Len()
	out := make([]sel, len(sels))
	for i, s := range sels {
		out[i] = st.ed.newSel(max(0, min(s.Anchor, n)), max(0, min(s.Head, n)))
	}
	st.ed.setSels(out, revealFit)
}

// Select selects the text from start to end, a caret alone at end when
// they are equal.
func (st *State) Select(start, end int) { st.SetSelections(Selection{Anchor: start, Head: end}) }

// Cursor returns the offset of the newest caret.
func (st *State) Cursor() int { return st.Selection().Head }

// CursorPos returns the line and column of the newest caret.
func (st *State) CursorPos() Pos {
	st.ensure()
	return posOf(st.ed.view.text, st.ed.primary().head)
}

// SelectedText returns the text of the selections, joined by newlines.
func (st *State) SelectedText() string {
	st.ensure()
	return st.ed.selectedText()
}

// Insert replaces each selection with text, as typing or pasting it.
func (st *State) Insert(text string) {
	st.ensure()
	st.ed.retry(func() bool { st.ed.paste(text); return true })
}

// Apply makes edits, in order and not overlapping, in the text as it is,
// as one step of undo, and puts the selections after them, the last the
// newest: undo takes the step back and puts the selections back where
// they were before it, and redo where Apply put them, as the editor's
// own Enter does. It reports whether it edited; a read-only editor
// does not.
func (st *State) Apply(edits []Edit, after []Selection) bool {
	st.ensure()
	ed := &st.ed
	return ed.retry(func() bool { return ed.applyWith(edits, after) })
}

// applyWith makes edits as one step of undo, placing the selections
// after, as given in the text the edits make.
func (ed *editor) applyWith(edits []Edit, after []Selection) bool {
	if len(edits) == 0 || len(after) == 0 {
		return false
	}
	n := ed.view.text.Len()
	for _, e := range edits {
		n += len(e.Text) - (e.End - e.Start)
	}
	return ed.apply(edits, editOther, func(func(int, bias) int) []sel {
		sels := make([]sel, len(after))
		for i, s := range after {
			sels[i] = ed.newSel(max(0, min(s.Anchor, n)), max(0, min(s.Head, n)))
		}
		return sels
	})
}

// Run runs a command as its key would, and reports whether it did
// something.
func (st *State) Run(cmd Command) bool {
	st.ensure()
	return st.ed.retry(func() bool { return st.ed.run(cmd) })
}

// GoTo puts the caret at a line and column, from 0, and scrolls it to
// the middle of the editor.
func (st *State) GoTo(p Pos) {
	st.ensure()
	off := offsetOf(st.ed.view.text, p)
	st.ed.setSels([]sel{st.ed.newSel(off, off)}, revealCenter)
}

// ScrollTo scrolls a line to the top of the editor.
func (st *State) ScrollTo(line int) {
	st.ensure()
	st.ed.setScroll(st.ed.view.heights.top(line))
}

// ScrollY returns how far the editor's lines are scrolled up, in DIPs, 0
// for an editor as high as its lines (FitContent); SetScrollY scrolls
// them, as far as they scroll, as editors side by side scroll together.
func (st *State) ScrollY() float64 { return st.ed.scrollY }

func (st *State) SetScrollY(y float64) {
	st.ensure()
	st.ed.setScroll(y)
}

// ViewportHeight returns the height of the part of the text the editor
// shows, in DIPs, as the last frame laid it out: from ScrollY to ScrollY
// plus ViewportHeight, against LineTop, is what is in view, as a minimap
// beside the editor shows it. An editor as high as its lines
// (FitContent) is that high.
func (st *State) ViewportHeight() float64 { return float64(st.ed.viewH) }

// LineTop returns the top of a line, from 0, in DIPs below the top of the
// first, as the last frame laid the lines out: less ScrollY, it is where
// the line shows in the editor. The lines a fold hides take no room, and
// LineTop of the count of lines is the height of them all. With SoftWrap,
// the lines of a long text far from the view may take the rows they are
// estimated to for a few frames, until they are measured.
func (st *State) LineTop(line int) float64 {
	st.ensure()
	return st.ed.view.heights.top(line)
}

// RowStarts appends to into the bytes of line where its rows after the
// first start, as soft wrap breaks it (none when it does not wrap), as
// the last frame laid the editor out.
func (st *State) RowStarts(line int, into []int) []int {
	st.ensure()
	v := &st.ed.view
	if !v.wrap || line < 0 || line >= v.text.LineCount() {
		return into
	}
	l := v.line(line)
	for _, r := range l.rows[min(1, len(l.rows)):] {
		into = append(into, l.fromDisp(r.from))
	}
	return into
}

// Find opens the find bar with query, and selects its first match after
// the caret.
func (st *State) Find(query string) {
	st.ensure()
	f := &st.ed.find
	f.open, f.query, f.stale = true, query, true
	f.origin = st.ed.primary().start()
	f.step(&st.ed, 1)
}

// FindMatches returns how many matches the find bar has, and which is
// chosen, from 0.
func (st *State) FindMatches() (count, current int) {
	st.ensure()
	f := &st.ed.find
	if !f.open {
		return 0, 0
	}
	f.search(&st.ed)
	return len(f.matches), f.current
}

// Fold folds the lines indented under line, up to its closing bracket,
// or the run of imports it starts, as the gutter's marker does; Unfold
// unfolds them.
func (st *State) Fold(line int) {
	st.ensure()
	if !st.ed.isFolded(line) {
		st.ed.toggleFoldLine(line)
	}
}

func (st *State) Unfold(line int) {
	st.ensure()
	if st.ed.isFolded(line) {
		st.ed.toggleFoldLine(line)
	}
}

// FoldImports folds the first run of imports of the text, or the first
// import alone by its brackets, as editors can on opening a file
// (Language's Imports).
func (st *State) FoldImports() {
	st.ensure()
	ed := &st.ed
	l := ed.language()
	if l == nil || l.Imports == nil {
		return
	}
	r := &ed.view.text
	for line := range r.LineCount() {
		if !ed.isImport(l, line) {
			continue
		}
		if !ed.isFolded(line) {
			ed.toggleFoldLine(line)
		}
		return
	}
}

// Folded returns the lines folds start at.
func (st *State) Folded() []int {
	var out []int
	for _, f := range st.ed.folds {
		out = append(out, f.start)
	}
	slices.Sort(out)
	return out
}

// SetDiagnostics shows diagnostics of the text at version, the buffer's
// Version when they were made: those of an older version move with the
// edits since, as a language server's that arrive late.
func (st *State) SetDiagnostics(version uint64, ds []Diagnostic) {
	st.ensure()
	ed := &st.ed
	ed.sync()
	ds = slices.Clone(ds)
	for i := range ds {
		// Of an older version, the line in the text now: near enough.
		ds[i].Line = ed.view.text.LineOf(min(max(ds[i].Start, 0), ed.view.text.Len()))
	}
	if version != ed.view.version {
		// Moved through the edits since, up to the text the editor shows.
		entries, _, ok := st.Buffer.entriesSince(version)
		if !ok {
			return
		}
		for _, e := range entries {
			if e.version > ed.view.version {
				break
			}
			for i := range ds {
				d := &ds[i]
				d.Start, d.End = mapOffset(d.Start, e.changes, biasRight), mapOffset(d.End, e.changes, biasLeft)
				d.End = max(d.End, d.Start)
			}
		}
	}
	slices.SortStableFunc(ds, func(a, b Diagnostic) int { return a.Start - b.Start })
	ed.diags = ds
}

// Diagnostics returns the diagnostics shown, where the edits since moved
// them.
func (st *State) Diagnostics() []Diagnostic { return slices.Clone(st.ed.diags) }

// ShowCompletions shows completions of the word before the caret, which
// starts at offset start (the request's Offset less its Prefix), or -1
// for the word there now. Typing filters them; an empty list closes them.
func (st *State) ShowCompletions(start int, items []Completion) {
	st.ensure()
	st.ed.sync()
	st.ed.showCompletions(start, items)
}

// CompletionsShown reports whether the completions show, and
// HoverShown whether a hover does: the popups Escape closes first, one
// at a time (Cancel), before the find bar or the selections.
func (st *State) CompletionsShown() bool { st.ensure(); return st.ed.complete.open() }
func (st *State) HoverShown() bool       { st.ensure(); return st.ed.hover.open() }

// ShowHover shows what the text from start to end is: text in the
// editor's font, and what build adds, as a hover of a language server.
func (st *State) ShowHover(start, end int, text string, build func(c *ui.Context)) {
	st.ensure()
	h := &st.ed.hover
	h.shown, h.start, h.end, h.text, h.build = true, start, end, text, build
	h.fresh = true
	if h.diags == nil {
		h.diags = st.ed.diagnosticsAt(start)
	}
}

// SetInlays shows inlay hints in the lines, as types and parameter
// names: they move with the edits until set again.
func (st *State) SetInlays(inlays []Inlay) {
	st.ensure()
	st.ed.setInlays(inlays)
}

// SetReplacements shows ranges of the text as something else, as text,
// as an element, or as nothing, as a Markdown editor hides the marks of
// emphasis and draws a check box in place of "[ ]": they move with the
// edits until set again, as inlays do. Set them again as the text or the
// selections change, as such an editor shows the marks of the lines with
// a caret. Empty ranges, and ranges overlapping one before them, are left
// out.
//
// A range may cover lines: the lines it covers from their start through
// their newline show no rows, as folded ones, but the one it starts on
// when it shows something there; the line it ends in shows what follows
// it.
func (st *State) SetReplacements(rs []Replacement) {
	st.ensure()
	st.ed.setReplacements(rs)
}

// Replacements returns the replacements shown, where the edits since
// moved them.
func (st *State) Replacements() []Replacement { return slices.Clone(st.ed.view.repls) }

// SetLineStyles draws lines in other fonts, as a Markdown editor's
// headings in larger type: each line's rows are as high as its style
// makes them, and only the lines in view are laid out still. They move
// with the edits until set again, a style going with its line's text.
// Set them again as the text changes: setting the same styles costs
// nothing, and a change lays out again only the lines whose style
// changed. Styles of lines that don't exist are left out, and the last
// given for a line wins. A diff's sides take none.
//
// The gutter numbers styled lines in the editor's font, on the baseline
// of their first row, and a replacement's element without a Width is as
// wide as the line's rows are high.
func (st *State) SetLineStyles(styles []LineStyle) {
	st.ensure()
	st.ed.setLineStyles(styles)
}

// LineStyles returns the styles shown, where the edits since moved them.
func (st *State) LineStyles() []LineStyle {
	st.ensure()
	out := make([]LineStyle, len(st.ed.view.lineStyles))
	for i, s := range st.ed.view.lineStyles {
		out[i] = s.LineStyle
	}
	return out
}

// SetSuggestion shows text at offset, muted, as an edit prediction:
// Tab inserts it, and an edit or a move of the caret drops it. An empty
// text takes it away.
func (st *State) SetSuggestion(offset int, text string) {
	st.ensure()
	st.ed.setSuggestion(offset, text)
}

// Focused reports whether the editor had the keyboard focus in the last
// frame.
func (st *State) Focused() bool { return st.ed.focused }

// SoftWrapped reports whether lines wrap: SoftWrap, unless
// ToggleSoftWrap flipped it.
func (st *State) SoftWrapped() bool { return st.SoftWrap != st.ed.toggledWrap }

// VisibleLines returns the first and the last line, from 0, that the
// last frame showed, those hidden by folds among them.
func (st *State) VisibleLines() (first, last int) { return st.ed.shown[0], st.ed.shown[1] }

// ensure readies the state for methods called before the editor is
// built.
func (st *State) ensure() {
	if st.Buffer == nil {
		st.Buffer = NewBuffer("")
	}
	ed := &st.ed
	if ed.view.buf != st.Buffer || ed.sels == nil {
		ed.init(st.Buffer, st.options(), ui.LightTheme())
	}
	ed.sync()
}

// ChangeHunks returns the hunks the gutter's bars show (nil without a
// DiffBase), as the editor last compared them.
func (st *State) ChangeHunks() []DiffHunk {
	g := st.ed.gitDiff
	if g == nil {
		return nil
	}
	out := make([]DiffHunk, len(g.d.hunks))
	for i, h := range g.d.hunks {
		out[i] = DiffHunk{h.A0, h.A1, h.B0, h.B1}
	}
	return out
}
