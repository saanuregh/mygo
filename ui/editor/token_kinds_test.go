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
