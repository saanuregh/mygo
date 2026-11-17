package editor

import (
	"github.com/egoist/mygo/internal/l10n"
	"regexp"
	"regexp/syntax"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

// finder is an editor's find bar and its matches, as Zed's buffer
// search: a query, case sensitive or not, whole words, a regular
// expression, in the selections or everywhere, and its replacement.
type finder struct {
	open        bool
	replacing   bool
	query       string
	replace     string
	caseSens    bool
	wholeWord   bool
	regex       bool
	inSelection bool
	scope       []sel
	focusQuery  bool
	typed       string // the query as the matches were last chosen for
	// The matches of the last search, and the text and options it was of;
	// subs are a regular expression's submatches, as found in the text.
	matches  []diffRange
	subs     [][]int
	dirty    []lineSpan // lines edits changed, to search again
	current  int
	stale    bool
	version  uint64
	searched string
	options  [4]bool
	err      error
	lower    string // the text lowered, for searching without case
	lowerVer uint64
	scrollTo bool
	builtW   float32 // the editor's width the bar was built for
	// origin is where the selection started as the bar opened
	// (FindFromOrigin).
	origin int
	// chosen is the query, options and text the match after origin was
	// last chosen for (FindFromOrigin): a change of any chooses again.
	chosen findKey
	// focusReplace: the replacement's field takes the focus at the next
	// frame (FindFromOrigin's FindReplace over a query).
	focusReplace bool
	// seeded: show put the selection in the query; selectQuery: the
	// query is selected as it takes the focus, had it or not (FocusQuery).
	seeded, selectQuery bool
	// none: no match is the current one, as the bar opened again on the
	// matches it had (FindFromOrigin).
	none bool
	// unicode: \w, \d and \s (and their opposites) take any script's
	// characters, as zode's rure does (FindFromOrigin).
	unicode bool
}

// pattern is the query as a Go regular expression: multi-line, without
// case unless told, its classes Unicode's and whole words between word
// boundaries (unicode).
func (f *finder) pattern() string {
	pat := f.query
	if f.unicode {
		pat = unicodeClasses(pat)
	}
	if f.unicode && f.wholeWord {
		// zode's find.zig: whole words of a pattern are what it matches
		// between word boundaries, not its matches that are whole words.
		pat = `\b(?:` + pat + `)\b`
	}
	if !f.caseSens {
		pat = "(?i)" + pat
	}
	return "(?m)" + pat
}

// The Unicode classes of Rust's regex crate (rure), as Go writes them:
// \w is a letter, a mark, a decimal digit or a connector; \s Unicode's
// White_Space.
const (
	unicodeWord  = `\p{L}\p{M}\p{Nd}\p{Pc}`
	unicodeSpace = `\t-\r \x{85}\x{A0}\x{1680}\x{2000}-\x{200A}\x{2028}\x{2029}\x{202F}\x{205F}\x{3000}`
)

// unicodeClasses rewrites pat's \w, \W, \d, \D, \s and \S as Unicode's
// classes (Go's are ASCII's). Inside brackets \W and \S stay Go's: a
// class cannot hold the opposite of a union. \b stays ASCII's: Go has no
// way to say Unicode's.
func unicodeClasses(pat string) string {
	var b strings.Builder
	inClass := false
	for i := 0; i < len(pat); i++ {
		ch := pat[i]
		switch {
		case ch == '\\' && i+1 < len(pat):
			next := pat[i+1]
			i++
			switch next {
			case 'w':
				if inClass {
					b.WriteString(unicodeWord)
				} else {
					b.WriteString("[" + unicodeWord + "]")
				}
				continue
			case 'd':
				b.WriteString(`\p{Nd}`)
				continue
			case 'D':
				b.WriteString(`\P{Nd}`)
				continue
			case 's':
				if inClass {
					b.WriteString(unicodeSpace)
				} else {
					b.WriteString("[" + unicodeSpace + "]")
				}
				continue
			case 'W', 'S':
				if !inClass {
					set := unicodeWord
					if next == 'S' {
						set = unicodeSpace
					}
					b.WriteString("[^" + set + "]")
					continue
				}
			}
			b.WriteByte('\\')
			b.WriteByte(next)
		case ch == '[' && !inClass:
			inClass = true
			b.WriteByte(ch)
			// A ] first (after ^) is one of the class's characters.
			if i+1 < len(pat) && pat[i+1] == '^' {
				i++
				b.WriteByte('^')
			}
			if i+1 < len(pat) && pat[i+1] == ']' {
				i++
				b.WriteByte(']')
			}
		case ch == '[' && inClass && i+1 < len(pat) && pat[i+1] == ':':
			// A POSIX class inside, as [[:alpha:]]: as it is.
			end := strings.Index(pat[i:], ":]")
			if end < 0 {
				b.WriteByte(ch)
				continue
			}
			b.WriteString(pat[i : i+end+2])
			i += end + 1
		case ch == ']' && inClass:
			inClass = false
			b.WriteByte(ch)
		default:
			b.WriteByte(ch)
		}
	}
	return b.String()
}

// findKey is what FindFromOrigin's choice of a match follows.
type findKey struct {
	query   string
	options [4]bool
	version uint64
}

func (f *finder) key(ed *editor) findKey {
	return findKey{f.query, [4]bool{f.caseSens, f.wholeWord, f.regex, f.inSelection}, ed.view.version}
}

// maxMatches bounds the matches a search finds; zigMaxMatches those of
// FindFromOrigin's, as zode's find bar.
const (
	maxMatches    = 100000
	zigMaxMatches = 20000
)

// limit is the matches a search of ed finds at most.
func (f *finder) limit(ed *editor) int {
	if ed.o.findFromOrigin {
		return zigMaxMatches
	}
	return maxMatches
}

func (f *finder) show(ed *editor, replace bool) {
	wasOpen := f.open
	f.open = true
	f.replacing = f.replacing || replace
	if replace {
		f.replacing = true
	}
	if ed.o.findFromOrigin {
		// Find hides replacing, as zode's bar.
		f.replacing = replace
	}
	// The selection, or the word at the caret, seeds the query
	// (FindFromOrigin's: a selection under 1000 bytes only).
	p := ed.primary()
	f.origin = p.start()
	seed := !ed.o.findFromOrigin || p.end()-p.start() < 1000
	f.seeded = false
	if !p.empty() && seed && ed.view.text.LineOf(p.start()) == ed.view.text.LineOf(p.end()) {
		f.query = ed.slice(p.start(), p.end())
		f.seeded = true
	} else if p.empty() && !ed.o.findFromOrigin {
		if a, z := wordAt(ed.view.text, p.head, ed.wordChars()); z > a && f.query == "" {
			f.query = ed.slice(a, z)
		}
	}
	if f.inSelection {
		f.scope = nil
	}
	f.focusQuery = true
	ed.needsBuild = true
	if ed.o.findFromOrigin {
		if replace && f.query != "" {
			// Replacing what is found: the replacement takes the keyboard.
			f.focusQuery, f.focusReplace = false, true
		}
		// A match is chosen again as the query, options or text change
		// (update); the bar opened again after it closed has none chosen,
		// and counts the matches; opened while open, it keeps its own.
		if !wasOpen {
			f.none = true
		}
		return
	}
	f.stale = true
	f.scrollTo = true
}

// update searches again when the text, the query or the options
// changed, and selects the match at or after the caret as the query is
// typed, as Zed's search does.
func (f *finder) update(ed *editor) {
	if f.query != f.typed {
		f.typed = f.query
		f.scrollTo = true
	}
	if ed.o.findFromOrigin {
		// Any change of the query, the options or the text chooses the
		// first match from the origin again (zode's find.zig draw).
		if k := f.key(ed); k != f.chosen {
			f.chosen = k
			f.scrollTo = true
		}
	}
	f.search(ed)
	if !f.scrollTo || len(f.matches) == 0 {
		return
	}
	f.scrollTo = false
	f.none = false
	from := ed.primary().start()
	if ed.o.findFromOrigin {
		from = f.origin
	}
	f.current = f.nearest(from)
	m := f.matches[f.current]
	// In the middle of the view, and out of a fold.
	ed.setSels([]sel{ed.newSel(m.start, m.end)}, revealCenter)
	if ed.o.sharedScroll == nil {
		ed.doReveal()
	}
}

// search finds the matches of the query in the text again when either
// changed: after edits, a plain query only in the lines they changed,
// the others' matches having moved with them (moved).
func (f *finder) search(ed *editor) {
	opts := [4]bool{f.caseSens, f.wholeWord, f.regex, f.inSelection}
	if !f.stale && f.version == ed.view.version && f.searched == f.query && f.options == opts {
		if len(f.dirty) > 0 {
			f.searchLines(ed)
		}
		return
	}
	if opts[3] != f.options[3] {
		// Finding in the selection takes the selections there are now.
		f.scope = nil
	}
	f.stale, f.dirty = false, nil
	f.version, f.searched, f.options = ed.view.version, f.query, opts
	f.matches, f.subs = f.matches[:0], f.subs[:0]
	f.err = nil
	if f.query == "" {
		return
	}
	if f.inSelection && f.scope == nil {
		for _, s := range ed.sels {
			if !s.empty() {
				f.scope = append(f.scope, s)
			}
		}
	}
	text := ed.view.text.String()
	if !f.regex && !f.caseSens && isASCII(f.query) {
		if f.lowerVer != ed.view.version || f.lower == "" && text != "" {
			f.lower, f.lowerVer = asciiLower(text), ed.view.version
		}
	}
	f.scan(ed, text, f.lower, 0, func(m diffRange, sub []int) bool {
		f.matches = append(f.matches, m)
		if sub != nil {
			f.subs = append(f.subs, sub)
		}
		return len(f.matches) < f.limit(ed)
	})
	f.current = min(f.current, max(0, len(f.matches)-1))
}

// scan finds the query in text, which starts at byte base of the editor's
// text (lower its lowered ASCII, when known), and calls found with each
// match it keeps, and a regular expression's submatches, until found
// returns false.
func (f *finder) scan(ed *editor, text, lower string, base int, found func(m diffRange, sub []int) bool) {
	extra := ed.wordChars()
	keep := func(a, b int) bool {
		if f.wholeWord && !(f.regex && f.unicode) && !isWholeWord(text, a, b, extra) {
			return false
		}
		if f.inSelection && len(f.scope) > 0 {
			for _, s := range f.scope {
				if base+a >= s.start() && base+b <= s.end() {
					return true
				}
			}
			return false
		}
		return true
	}
	switch {
	case f.regex:
		f.unicode = ed.o.findFromOrigin
		pat := f.pattern()
		re, err := regexp.Compile(pat)
		if err != nil {
			f.err = err
			if ed.o.findFromOrigin {
				// zode's find.zig: a pattern that is not one is
				// searched for as it is typed, its error still said.
				f.regex = false
				f.scan(ed, text, lower, base, found)
				f.regex = true
			}
			return
		}
		var all [][]int
		if words, hasWords, onlyBoundaries := rureWords(pat); f.unicode && hasWords && !(onlyBoundaries && isASCII(text)) {
			// Word assertions as rure's (find_words.go): Go's \b is
			// ASCII's, and it has no \< or \>.
			m, err := compileWordMatcher(words)
			if err != nil {
				f.err = err
				return
			}
			all = m.findAll(text, maxMatches)
		} else {
			all = re.FindAllStringSubmatchIndex(text, maxMatches)
			if ed.o.findFromOrigin {
				all = withAbuttingEmpty(pat, text, all)
			}
		}
		for _, m := range all {
			// Empty matches count as FindFromOrigin's (zode's rure) do.
			if m[1] == m[0] && !ed.o.findFromOrigin || !keep(m[0], m[1]) {
				continue
			}
			if !found(diffRange{base + m[0], base + m[1]}, m) {
				return
			}
		}
	case f.caseSens || isASCII(f.query):
		q, in := f.query, text
		if !f.caseSens {
			if lower == "" {
				lower = asciiLower(text)
			}
			q, in = asciiLower(f.query), lower
		}
		for at := 0; ; {
			i := strings.Index(in[at:], q)
			if i < 0 {
				return
			}
			a, b := at+i, at+i+len(q)
			if keep(a, b) && !found(diffRange{base + a, base + b}, nil) {
				return
			}
			at = b
		}
	default:
		re := regexp.MustCompile("(?i)" + regexp.QuoteMeta(f.query))
		for _, m := range re.FindAllStringIndex(text, maxMatches) {
			if keep(m[0], m[1]) && !found(diffRange{base + m[0], base + m[1]}, nil) {
				return
			}
		}
	}
}

// moved follows edits of the text the matches were found in: those an
// edit touches go, the others move with the text, and the lines it
// changed are searched again (searchLines). A regular expression, whose
// matches can span lines, or a query of several lines, searches again.
func (f *finder) moved(cs []change) {
	if f.stale || f.regex || f.query == "" || strings.Contains(f.query, "\n") {
		f.stale = true
		return
	}
	out := f.matches[:0]
	for _, m := range f.matches {
		touched := false
		for _, c := range cs {
			if m.start <= c.oldEnd && m.end >= c.oldStart {
				touched = true
				break
			}
		}
		if !touched {
			out = append(out, diffRange{mapOffset(m.start, cs, biasRight), mapOffset(m.end, cs, biasLeft)})
		}
	}
	f.matches = out
	for i := range f.dirty {
		f.dirty[i] = lineSpan{mapLine(f.dirty[i].from, cs), mapLine(f.dirty[i].to, cs)}
	}
	for _, c := range cs {
		f.dirty = append(f.dirty, lineSpan{c.newLine, c.newEndLine})
	}
}

// searchLines searches the lines edits changed again, as moved left them.
func (f *finder) searchLines(ed *editor) {
	r := &ed.view.text
	n := r.LineCount()
	spans := mergeSpans(f.dirty)
	f.dirty = nil
	for _, sp := range spans {
		from, to := min(sp.from, n-1), min(sp.to, n-1)
		a, z := r.LineStart(from), r.LineEnd(to)
		f.matches = slices.DeleteFunc(f.matches, func(m diffRange) bool { return m.start >= a && m.start <= z })
		i, _ := slices.BinarySearchFunc(f.matches, a, func(m diffRange, off int) int { return m.start - off })
		var found []diffRange
		f.scan(ed, r.Slice(a, z), "", a, func(m diffRange, _ []int) bool {
			found = append(found, m)
			return true
		})
		f.matches = slices.Insert(f.matches, i, found...)
	}
	if n := f.limit(ed); len(f.matches) > n {
		f.matches = f.matches[:n]
	}
	f.current = min(f.current, max(0, len(f.matches)-1))
}

// asciiLower lowers the ASCII letters of s, keeping its length.
func asciiLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}

// nearest returns the match at or after off, round to the first.
func (f *finder) nearest(off int) int {
	i := sort.Search(len(f.matches), func(i int) bool { return f.matches[i].start >= off })
	if i == len(f.matches) {
		return 0
	}
	return i
}

// step selects the next match (or previous) after the newest caret.
func (f *finder) step(ed *editor, dir int) bool {
	f.search(ed)
	f.chosen = f.key(ed)
	if len(f.matches) == 0 {
		return false
	}
	p := ed.primary()
	var i int
	if dir > 0 {
		i = f.nearest(p.end())
		if f.matches[i].start == p.start() && f.matches[i].end == p.end() && i+1 <= len(f.matches) {
			i = (i + 1) % len(f.matches)
		}
	} else {
		i = f.nearest(p.start()) - 1
		if i < 0 {
			i = len(f.matches) - 1
		}
	}
	f.current, f.none = i, false
	m := f.matches[i]
	ed.setSels([]sel{ed.newSel(m.start, m.end)}, revealCenter)
	return true
}

// replaceOne replaces the current match when selected, and goes to the
// next.
func (f *finder) replaceOne(ed *editor) {
	f.search(ed)
	if f.none {
		// None is the current one: the next is, as zode's find bar.
		f.step(ed, 1)
		return
	}
	p := ed.primary()
	text := f.textForReplacing(ed)
	for i, m := range f.matches {
		if m.start == p.start() && m.end == p.end() {
			with := lf(f.replacement(text, i))
			ed.apply([]Edit{{Start: m.start, End: m.end, Text: with}}, editOther, func(move func(int, bias) int) []sel {
				at := move(m.start, biasLeft) + len(with)
				return []sel{ed.newSel(at, at)}
			})
			f.search(ed)
			f.current = i
			f.step(ed, 1)
			return
		}
	}
	f.step(ed, 1)
}

// replaceAll replaces every match, as one step of undo.
func (f *finder) replaceAll(ed *editor) {
	f.search(ed)
	if len(f.matches) == 0 {
		return
	}
	edits := make([]Edit, len(f.matches))
	text := f.textForReplacing(ed)
	for i, m := range f.matches {
		edits[i] = Edit{Start: m.start, End: m.end, Text: f.replacement(text, i)}
	}
	ed.apply(edits, editOther, nil)
}

// textForReplacing returns the text a regular expression's replacements
// expand their submatches from, nothing else needing it.
func (f *finder) textForReplacing(ed *editor) string {
	if !f.regex || len(f.subs) == 0 {
		return ""
	}
	return ed.view.text.String()
}

// replacement returns the replacement of match i: the replace text, with
// $1 and ${name} expanded for a regular expression, from the submatches
// found in the whole text, as \b and ^ need the text around a match.
func (f *finder) replacement(text string, i int) string {
	if !f.regex || i >= len(f.subs) {
		return f.replace
	}
	re, err := regexp.Compile(f.pattern())
	if err != nil {
		return f.replace
	}
	return string(re.ExpandString(nil, f.replace, text, f.subs[i]))
}

// status is what the bar says of the matches, in the language of tx.
func (f *finder) status(tx l10n.Messages) string {
	switch {
	case f.err != nil:
		return tx.Invalid()
	case f.query == "":
		return ""
	case len(f.matches) == 0:
		return tx.NoResults()
	case len(f.matches) >= maxMatches:
		return tx.MatchesMore(f.current+1, len(f.matches))
	}
	return tx.Matches(f.current+1, len(f.matches))
}

// selectAll selects every match, as alt-enter in Zed's search bar.
func (f *finder) selectAll(ed *editor) {
	f.search(ed)
	if len(f.matches) == 0 {
		return
	}
	sels := make([]sel, 0, len(f.matches))
	for _, m := range f.matches {
		sels = append(sels, ed.newSel(m.start, m.end))
	}
	ed.setSels(sels, revealFit)
	f.open = false
	ed.refocus = true
}

// goToLine is the bar going to a line: line[:column], or +n and -n
// lines from the caret, as Zed's go to line, which shows the line as it
// is typed and goes back where it was on Escape.
type goToLine struct {
	open    bool
	text    string
	focus   bool
	savedY  float64
	savedX  float32
	preview int
}

func (g *goToLine) show(ed *editor) {
	g.open, g.text, g.focus = true, "", true
	g.savedY, g.savedX = ed.scrollY, ed.scrollX
	g.preview = -1
	ed.needsBuild = true
}

func (g *goToLine) close(ed *editor) {
	g.open = false
	ed.setScroll(g.savedY)
	ed.scrollX = g.savedX
	g.preview = -1
	ed.refocus = true
}

// target parses the text into a line and column, from 0.
func (g *goToLine) target(ed *editor) (int, int, bool) {
	s := strings.TrimSpace(g.text)
	if s == "" {
		return 0, 0, false
	}
	cur := ed.view.text.LineOf(ed.primary().head)
	rel := 0
	if s[0] == '+' || s[0] == '-' {
		rel = 1
		if s[0] == '-' {
			rel = -1
		}
		s = s[1:]
	}
	parts := strings.SplitN(s, ":", 2)
	line, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, false
	}
	col := 0
	if len(parts) == 2 {
		if c, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil && c > 0 {
			col = c - 1
		}
	}
	if rel != 0 {
		line = cur + rel*line
	} else {
		line--
	}
	line = max(0, min(line, ed.view.text.LineCount()-1))
	// The column counts characters, the position bytes.
	return line, byteOfRuneIn(ed.view.text.Line(line), col), true
}

// withAbuttingEmpty adds to all, pat's matches in text, the empty match
// right after a match that Go's FindAll leaves out and zode's rure keeps
// (it searches again from a match's end): where the next match does not
// start there and the pattern matches empty there, in the text's context.
func withAbuttingEmpty(pat, text string, all [][]int) [][]int {
	parsed, err := syntax.Parse(pat, syntax.Perl)
	if err != nil {
		return all
	}
	prog, err := syntax.Compile(parsed.Simplify())
	if err != nil {
		return all
	}
	out := make([][]int, 0, len(all))
	for k, m := range all {
		out = append(out, m)
		e := m[1]
		if m[1] == m[0] || k+1 < len(all) && all[k+1][0] == e {
			continue
		}
		if emptyMatchAt(prog, text, e) {
			empty := make([]int, len(m))
			for i := range empty {
				empty[i] = -1
			}
			empty[0], empty[1] = e, e
			out = append(out, empty)
		}
	}
	return out
}

// emptyMatchAt reports whether prog matches the empty text at byte pos of
// text: a path to its match taking no character, its assertions (^, $, \b)
// true there.
func emptyMatchAt(prog *syntax.Prog, text string, pos int) bool {
	r1, r2 := rune(-1), rune(-1)
	if pos > 0 {
		r1, _ = utf8.DecodeLastRuneInString(text[:pos])
	}
	if pos < len(text) {
		r2, _ = utf8.DecodeRuneInString(text[pos:])
	}
	ctx := syntax.EmptyOpContext(r1, r2)
	seen := make([]bool, len(prog.Inst))
	stack := []uint32{uint32(prog.Start)}
	for len(stack) > 0 {
		pc := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if seen[pc] {
			continue
		}
		seen[pc] = true
		in := &prog.Inst[pc]
		switch in.Op {
		case syntax.InstMatch:
			return true
		case syntax.InstAlt, syntax.InstAltMatch:
			stack = append(stack, in.Out, in.Arg)
		case syntax.InstNop, syntax.InstCapture:
			stack = append(stack, in.Out)
		case syntax.InstEmptyWidth:
			if syntax.EmptyOp(in.Arg)&^ctx == 0 {
				stack = append(stack, in.Out)
			}
		}
	}
	return false
}
