package ui

import (
	"runtime"
	"time"
)

// Theme holds the colors and metrics widgets use. Change a copy of
// LightTheme or DarkTheme and set it with Context.SetTheme.
type Theme struct {
	Dark bool
	// DisableAutoDimming keeps disabled elements at full opacity. Applications
	// opting in must provide their own disabled colors; input, focus and
	// accessibility still use the element's disabled state.
	DisableAutoDimming bool
	// Background fills the window.
	Background Color
	// Surface is the face of buttons, inputs and other controls;
	// SurfaceHover and SurfacePressed while hovered or pressed.
	Surface        Color
	SurfaceHover   Color
	SurfacePressed Color
	// Border outlines controls.
	Border Color
	// Text and TextMuted color text; TextMuted is for secondary text and
	// placeholders.
	Text      Color
	TextMuted Color
	// Accent colors primary buttons, checked controls and focus rings;
	// AccentText is text on it.
	Accent        Color
	AccentHover   Color
	AccentPressed Color
	AccentText    Color
	// InactiveAccent is what the accent fades to in a window that is not
	// drawn as active (Context.WindowActive), as AppKit fades an inactive
	// window's: the fill of selected rows, of checked check boxes, radio
	// buttons and switches, of sliders and progress bars, the line of the
	// selected tab and the icons of a sidebar, with AccentText on it as on
	// the accent; primary buttons show as the others there. Left zero, it
	// is a gray AccentText stands out on; set it to Accent to keep the
	// accent in every window.
	InactiveAccent Color
	Danger         Color
	// Warning and Success color what calls for attention and what went
	// well, as a Meter's levels.
	Warning Color
	Success Color
	// Selection highlights selected text.
	Selection Color
	// InputFocus is how a text input with the keyboard focus shows it
	// (InputFocusStyle); its zero value is MyGo's accent border.
	InputFocus InputFocusStyle
	// InputPadding is the room inside a text input's border, above and
	// below its text and on its sides, in DIPs: Space(1.5) and
	// Space(2.5) where zero.
	InputPadding [2]float32
	// InputNoHover keeps a text input's border as it is under the
	// pointer, as nanoui's text field, rather than darkening it.
	InputNoHover bool
	// InputSelection is how a one-line text input shows its selection,
	// as nanoui's text field: its rectangles' corners rounded by Radius,
	// and, with NoCaret, no caret while text is selected.
	InputSelection InputSelectionStyle
	// CaretBlink is how long a text input's caret shows, then hides, as
	// it blinks; 0 is 530 milliseconds, the system's usual.
	CaretBlink time.Duration
	// SteadyTextAreaCaret has a text area's caret show steadily rather
	// than blink, and none while text is selected, as a multiline field of
	// nanoui's draws it; a one-line input's caret still blinks. Off, an
	// area's caret blinks as an input's does.
	SteadyTextAreaCaret bool
	// Focus is the ring around the control with the keyboard focus,
	// FocusRingWidth DIPs wide (2 by default; 0 is 2) and FocusRingOffset
	// DIPs outside its edge (1 in LightTheme and DarkTheme): 0 puts the
	// ring on the edge, and a negative offset inside it.
	// Element.FocusRingOffset and Element.FocusRingColor change them for
	// one element.
	Focus           Color
	FocusRingWidth  float32
	FocusRingOffset float32
	// BordersTakeNoRoom makes Border draw over an element's edges instead
	// of taking room inside them: the padding counts from the edge and the
	// children may reach it. BorderWidth's borders, lines on chosen sides,
	// still take room.
	BordersTakeNoRoom bool
	// OverlaysKeepAlignment keeps an overlay that passes the window's
	// end along its target (a menu below its button, wider than the room
	// right of it) aligned as asked and moves it into the window, as
	// nanoui's kit.place does, rather than aligning it the other way.
	OverlaysKeepAlignment bool
	// Inverse fills tooltips and toasts, the theme turned over so that
	// they stand out from what they are over: dark in a light theme, light
	// in a dark one. InverseText is text on it. Left zero, they are Text
	// and Background.
	Inverse     Color
	InverseText Color
	// Scrollbar colors scroll bar thumbs, ScrollbarWidth DIPs wide (6 by
	// default; 0 is 6): the defaults of ScrollbarStyle, which overrides
	// them where it sets a field. It styles their tracks, hover and drag
	// colors, the width of their bars and thumbs, their corners and
	// shortest thumb; a theme that sets neither draws as it always has.
	Scrollbar      Color
	ScrollbarWidth float32
	ScrollbarStyle ScrollbarStyle
	// Radius rounds the corners of controls.
	Radius float32
	// Spacing is the unit of the room widgets leave inside and between
	// their parts: their paddings and gaps, and the sizes of check boxes,
	// switches, sliders and the rows of tables and trees, are multiples
	// of it. 4 by default: 3 makes widgets compact, 5 roomy; 0 is 4.
	Spacing float32
	// FontSize is the size of text, Font its family ("" is the system's).
	FontSize float32
	Font     string
	// TooltipDelay is how long the pointer rests on an element before its
	// tooltip shows (Tooltip, TooltipBase): 0 is 0.6 seconds, and a
	// negative delay shows it at once. Within two thirds of it after
	// another tip goes, as the pointer moves along a toolbar, the next one
	// shows at once: 0.4 seconds by default.
	TooltipDelay time.Duration
	// TooltipAfterClick: a press hides a tooltip only while the pointer is
	// down, and it shows again as the pointer comes up and stays, rather
	// than staying hidden until the pointer leaves (Base UI's closeOnClick,
	// the default). Escape still hides it.
	TooltipAfterClick bool
	// TooltipsHiddenByPopups: no tip shows while a popover or a menu
	// drawn in the window is open (often the one its element opened), but
	// those of elements inside one.
	TooltipsHiddenByPopups bool
	// Tooltip is how Tooltip's tips look (TooltipStyle); its zero value is
	// MyGo's look.
	Tooltip TooltipStyle
	// Menu is the look of the menus MyGo draws in the window: those of
	// PopoverMenu and PopoverMenuButton, and of ContextMenu and Menu in a
	// window that draws its menus (SetMenuPresentation). Its zero value
	// is MyGo's own look.
	Menu MenuStyle

	// followed is what WithPreferences changed, nil where it has not.
	followed *followedTheme
}

// InputSelectionStyle is how a one-line text input shows its selection
// (Theme.InputSelection).
type InputSelectionStyle struct {
	Radius  float32
	NoCaret bool
}

// InputFocusStyle is how a focused text input's box shows the focus,
// beyond its border taking the accent.
type InputFocusStyle struct {
	// Width is the accent's width at the box's edge, its border included,
	// drawn within the box so that its text stays where it was: the
	// border's own width when zero.
	Width float32
	// Glow is the blur of an accent glow around the box, Opacity the
	// accent's opacity in it; none when either is zero.
	Glow, Opacity float32
}

// TooltipStyle is how tooltips look; a zero field keeps MyGo's.
type TooltipStyle struct {
	// FontSize is the text's: the theme's FontSize less 1 by default.
	FontSize float32
	// PaddingX and PaddingY are the room around the text: 2 and 1.25
	// units of Spacing by default.
	PaddingX, PaddingY float32
	// Radius rounds the corners: 1.25 units of Spacing by default.
	Radius float32
	// Gap is the room between the tip and its element: 1.5 units.
	Gap float32
	// Background and Text are its colors: the theme's Inverse and
	// InverseText by default.
	Background, Text Color
	// Border outlines it, a DIP wide, when set.
	Border Color
	// Shadow is its shadow's color: black at 0.2 by default.
	Shadow Color
	// OneLine keeps the text on one line, however long, where it wraps
	// at 80 units of Spacing by default.
	OneLine bool
}

// TooltipLook returns the theme's Tooltip with MyGo's values in its zero
// fields, for a tip of one's own (TooltipBase) that looks as Tooltip's.
func (t *Theme) TooltipLook() TooltipStyle {
	s := t.Tooltip
	or := func(v *float32, d float32) {
		if *v <= 0 {
			*v = d
		}
	}
	or(&s.FontSize, t.FontSize-1)
	or(&s.PaddingX, t.Space(2))
	or(&s.PaddingY, t.Space(1.25))
	or(&s.Radius, t.Space(1.25))
	or(&s.Gap, t.Space(1.5))
	fill, text := t.inverse()
	if s.Background == Transparent {
		s.Background = fill
	}
	if s.Text == Transparent {
		s.Text = text
	}
	if s.Shadow == Transparent {
		s.Shadow = RGBA(0, 0, 0, 0.2)
	}
	return s
}

// followedTheme is a theme before and after following the desktop's
// preferences.
type followedTheme struct {
	_        [0]func() // used by pointer, so there is no value equality to link
	from, to Theme
}

// Space returns n units of the theme's Spacing, to size elements of your
// own in step with the widgets: ui.Column(c).Gap(t.Space(2)).
func (t *Theme) Space(n float32) float32 {
	s := t.Spacing
	if s <= 0 {
		s = 4
	}
	return n * s
}

// inactiveAccent returns what the accent fades to in a window that is not
// drawn as active (InactiveAccent).
func (t *Theme) inactiveAccent() Color {
	if t.InactiveAccent != Transparent {
		return t.InactiveAccent
	}
	// As far from AccentText as text on it needs, 4.5 times its contrast.
	far := Color{A: 255}
	if t.AccentText.gray().R <= 127 {
		far = Color{R: 255, G: 255, B: 255, A: 255}
	}
	return t.AccentText.Mix(far, 0.55)
}

// inverse returns the fill of tooltips and toasts, and the color of text
// on it.
func (t *Theme) inverse() (fill, text Color) {
	fill, text = t.Inverse, t.InverseText
	if fill == Transparent {
		fill = t.Text
	}
	if text == Transparent {
		text = t.Background
	}
	return fill, text
}

// focusRingWidth returns how wide the focus ring is (FocusRingWidth).
func (t *Theme) focusRingWidth() float32 {
	if t.FocusRingWidth <= 0 {
		return 2
	}
	return t.FocusRingWidth
}

func (t *Theme) scrollbarWidth() float32 {
	if t.ScrollbarWidth <= 0 {
		return 6
	}
	return t.ScrollbarWidth
}

// tooltipDelay returns how long the pointer rests on an element before its
// tip shows (TooltipDelay), and how soon after a tip goes the next one
// shows at once.
func (t *Theme) tooltipDelay() (delay, warm time.Duration) {
	switch d := t.TooltipDelay; {
	case d == 0:
		return tooltipDelay, tooltipWarm
	case d < 0:
		return 0, 0
	default:
		return d, d * 2 / 3
	}
}

// Rem returns n times the theme's FontSize, as CSS's rem, to size elements
// with the text.
func (t *Theme) Rem(n float32) float32 {
	if t.FontSize <= 0 {
		return n * defaultFontSize()
	}
	return n * t.FontSize
}

func defaultFontSize() float32 {
	if runtime.GOOS == "darwin" {
		return 13
	}
	return 14
}

// LightTheme returns the theme for a light appearance.
func LightTheme() *Theme {
	return &Theme{
		Background:      Hex("#ffffff"),
		Surface:         Hex("#f4f4f5"),
		SurfaceHover:    Hex("#e9e9ec"),
		SurfacePressed:  Hex("#dddde1"),
		Border:          Hex("#d9d9de"),
		Text:            Hex("#18181b"),
		TextMuted:       Hex("#71717a"),
		Accent:          Hex("#2563eb"),
		AccentHover:     Hex("#1d4ed8"),
		AccentPressed:   Hex("#1e40af"),
		AccentText:      Hex("#ffffff"),
		Danger:          Hex("#dc2626"),
		Warning:         Hex("#d97706"),
		Success:         Hex("#16a34a"),
		Selection:       RGBA(37, 99, 235, 0.25),
		Focus:           RGBA(37, 99, 235, 0.55),
		FocusRingWidth:  2,
		FocusRingOffset: 1,
		Scrollbar:       RGBA(0, 0, 0, 0.32),
		Radius:          6,
		Spacing:         4,
		FontSize:        defaultFontSize(),
	}
}

// DarkTheme returns the theme for a dark appearance.
func DarkTheme() *Theme {
	return &Theme{
		Dark:            true,
		Background:      Hex("#18181b"),
		Surface:         Hex("#27272a"),
		SurfaceHover:    Hex("#323236"),
		SurfacePressed:  Hex("#3c3c41"),
		Border:          Hex("#3f3f46"),
		Text:            Hex("#f4f4f5"),
		TextMuted:       Hex("#a1a1aa"),
		Accent:          Hex("#3b82f6"),
		AccentHover:     Hex("#60a5fa"),
		AccentPressed:   Hex("#2563eb"),
		AccentText:      Hex("#ffffff"),
		Danger:          Hex("#ef4444"),
		Warning:         Hex("#f59e0b"),
		Success:         Hex("#22c55e"),
		Selection:       RGBA(59, 130, 246, 0.4),
		Focus:           RGBA(96, 165, 250, 0.6),
		FocusRingWidth:  2,
		FocusRingOffset: 1,
		Scrollbar:       RGBA(255, 255, 255, 0.35),
		Radius:          6,
		Spacing:         4,
		FontSize:        defaultFontSize(),
	}
}

// inputPadding is a text input's room inside its border, down and across
// (Theme.InputPadding).
func (t *Theme) inputPadding() (y, x float32) {
	y, x = t.InputPadding[0], t.InputPadding[1]
	if y == 0 {
		y = t.Space(1.5)
	}
	if x == 0 {
		x = t.Space(2.5)
	}
	return y, x
}
