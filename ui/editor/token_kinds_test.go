package editor

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// The kinds parsers tell apart take the colors of the nearest kinds of
// the default theme.
func TestDefaultThemeColorsTheParsersKinds(t *testing.T) {
	for _, dark := range []bool{false, true} {
		tt := ui.LightTheme()
		if dark {
			tt = ui.DarkTheme()
		}
		ct := DefaultTheme(tt)
		if ct.TokenColor(TokenBuiltin) != ct.TokenColor(TokenConstant) || ct.TokenColor(TokenLabel) != ct.TokenColor(TokenMeta) ||
			ct.TokenColor(TokenNamespace) != ct.TokenColor(TokenType) || ct.TokenColor(TokenParameter) != ct.TokenColor(TokenVariable) {
			t.Fatalf("dark %v: the parsers' kinds take other colors", dark)
		}
	}
}

// Bracket pair colors take VS Code's defaults, three apart from the text.
func TestBracketTokenKindsHaveStyles(t *testing.T) {
	want := map[bool][3]string{
		true:  {"#FFD700", "#DA70D6", "#179FFF"},
		false: {"#0431FA", "#319331", "#7B3814"},
	}
	for _, dark := range []bool{false, true} {
		tt := ui.LightTheme()
		if dark {
			tt = ui.DarkTheme()
		}
		ct := DefaultTheme(tt)
		for i, k := range []TokenKind{TokenBracket1, TokenBracket2, TokenBracket3} {
			if got := ct.TokenColor(k); got != ui.Hex(want[dark][i]) {
				t.Fatalf("dark %v: bracket %d is %v, want %s", dark, i+1, got, want[dark][i])
			}
		}
	}
}
