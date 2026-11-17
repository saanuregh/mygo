package editor

import (
	"regexp/syntax"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// zode's find (FindFromOrigin) searches as Rust's regex crate (rure)
// does, whose word assertions Go's regexp lacks or reads as ASCII: \b and
// \B of Unicode's words, \< and \b{start} (a word starts), \> and
// \b{end} (a word ends), \b{start-half} (no word before) and
// \b{end-half} (no word after). A pattern holding one is matched here,
// by a Pike machine over Go's compiled program.

// Private-use runes standing in the pattern for the assertions Go's
// syntax has no operator for; the matcher reads them as assertions.
const (
	wordStartRune rune = 0x10FFF0 + iota
	wordEndRune
	wordStartHalfRune
	wordEndHalfRune
)

// rureWordAssertions are the word assertions of rure that Go's syntax
// lacks, longest first, and the rune each stands as.
var rureWordAssertions = []struct {
	syntax string
	stand  rune
}{
	{`\b{start-half}`, wordStartHalfRune},
	{`\b{end-half}`, wordEndHalfRune},
	{`\b{start}`, wordStartRune},
	{`\b{end}`, wordEndRune},
	{`\<`, wordStartRune},
	{`\>`, wordEndRune},
}

// rureWords rewrites pat's word assertions that Go's syntax lacks as the
// runes standing for them, outside character classes. It reports whether
// pat holds a word assertion (\b and \B among them), and whether only \b
// and \B: those Go's regexp reads alike in ASCII text.
func rureWords(pat string) (out string, words, onlyBoundaries bool) {
	var b strings.Builder
	inClass := false
	onlyBoundaries = true
	for i := 0; i < len(pat); i++ {
		ch := pat[i]
		switch {
		case ch == '\\' && i+1 < len(pat):
			if !inClass {
				replaced := false
				for _, a := range rureWordAssertions {
					if strings.HasPrefix(pat[i:], a.syntax) {
						b.WriteString(`\x{` + strconv.FormatInt(int64(a.stand), 16) + `}`)
						i += len(a.syntax) - 1
						words, onlyBoundaries, replaced = true, false, true
						break
					}
				}
				if replaced {
					continue
				}
				if pat[i+1] == 'b' || pat[i+1] == 'B' {
					words = true
				}
			}
			b.WriteString(pat[i : i+2])
			i++
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
	return b.String(), words, words && onlyBoundaries
}

// isRureWord reports whether r is a character of a word as rure's \b
// reads it (Unicode's \w: Alphabetic, marks, decimal digits, connector
// punctuation and the joiners).
func isRureWord(r rune) bool {
	if r < utf8.RuneSelf {
		return r == '_' || '0' <= r && r <= '9' || 'a' <= r && r <= 'z' || 'A' <= r && r <= 'Z'
	}
	return unicode.IsLetter(r) || unicode.IsMark(r) || unicode.Is(unicode.Nd, r) || unicode.Is(unicode.Pc, r) ||
		unicode.Is(unicode.Nl, r) || unicode.Is(unicode.Other_Alphabetic, r) || unicode.Is(unicode.Join_Control, r)
}

// wordContext is the assertions true between before and after (-1 at
// the text's ends), its word boundaries Unicode's.
func wordContext(before, after rune) syntax.EmptyOp {
	op := syntax.EmptyOpContext(before, after) &^ (syntax.EmptyWordBoundary | syntax.EmptyNoWordBoundary)
	if (before >= 0 && isRureWord(before)) != (after >= 0 && isRureWord(after)) {
		return op | syntax.EmptyWordBoundary
	}
	return op | syntax.EmptyNoWordBoundary
}

// wordAssertion reports whether the assertion a standing rune is holds
// between before and after.
func wordAssertion(r, before, after rune) bool {
	wb, wa := before >= 0 && isRureWord(before), after >= 0 && isRureWord(after)
	switch r {
	case wordStartRune:
		return !wb && wa
	case wordEndRune:
		return wb && !wa
	case wordStartHalfRune:
		return !wb
	case wordEndHalfRune:
		return !wa
	}
	return false
}

// standing is the assertions a rune instruction stands for: its ranges
// within the standing runes (Go's parser makes a class of alternatives
// of one rune, as \<|\>), and whether it matches nothing else.
type standing struct {
	runes []rune
	only  bool
}

func standingOf(in *syntax.Inst) standing {
	var s standing
	if in.Op != syntax.InstRune && in.Op != syntax.InstRune1 {
		return s
	}
	rs := in.Rune
	if len(rs) == 1 {
		rs = []rune{rs[0], rs[0]}
	}
	s.only = true
	for i := 0; i+1 < len(rs); i += 2 {
		if rs[i] >= wordStartRune && rs[i+1] <= wordEndHalfRune {
			for r := rs[i]; r <= rs[i+1]; r++ {
				s.runes = append(s.runes, r)
			}
		} else {
			s.only = false
		}
	}
	if len(s.runes) == 0 {
		s.only = false
	}
	return s
}

// wordMatcher matches a pattern rureWords rewrote, leftmost first as
// rure and Go's regexp do.
type wordMatcher struct {
	prog       *syntax.Prog
	ncap       int
	stands     []standing // per instruction
	run, next  wordQueue
	pool       []*wordThread
	matched    []int
	scratchCap []int
}

type wordThread struct{ cap []int }

type wordEntry struct {
	pc uint32
	t  *wordThread
}

// wordQueue is a sparse set of the program's instructions, in the order
// of their priority.
type wordQueue struct {
	sparse []uint32
	dense  []wordEntry
}

func newWordQueue(n int) wordQueue {
	return wordQueue{sparse: make([]uint32, n), dense: make([]wordEntry, 0, n)}
}

func compileWordMatcher(pat string) (*wordMatcher, error) {
	re, err := syntax.Parse(pat, syntax.Perl)
	if err != nil {
		return nil, err
	}
	ncap := 2 * (re.MaxCap() + 1)
	prog, err := syntax.Compile(re.Simplify())
	if err != nil {
		return nil, err
	}
	n := len(prog.Inst)
	stands := make([]standing, n)
	for pc := range prog.Inst {
		stands[pc] = standingOf(&prog.Inst[pc])
	}
	return &wordMatcher{prog: prog, ncap: ncap, stands: stands, run: newWordQueue(n), next: newWordQueue(n), scratchCap: make([]int, ncap)}, nil
}

// findAll returns up to limit matches of text, each with its
// submatches as regexp's FindAllStringSubmatchIndex, but as rure's
// search from each match's end finds them: an empty match right after a
// match counts.
func (m *wordMatcher) findAll(text string, limit int) [][]int {
	var out [][]int
	for pos := 0; pos <= len(text) && len(out) < limit; {
		found := m.match(text, pos)
		if found == nil {
			break
		}
		out = append(out, found)
		switch {
		case found[1] > found[0]:
			pos = found[1]
		case found[1] < len(text):
			_, w := utf8.DecodeRuneInString(text[found[1]:])
			pos = found[1] + w
		default:
			pos = len(text) + 1
		}
	}
	return out
}

// match returns the leftmost-first match starting at or after from, and
// its submatches, or nil.
func (m *wordMatcher) match(text string, from int) []int {
	m.matched = nil
	at := func(pos int) (rune, int) {
		if pos >= len(text) {
			return -1, 0
		}
		return utf8.DecodeRuneInString(text[pos:])
	}
	before := rune(-1)
	if from > 0 {
		before, _ = utf8.DecodeLastRuneInString(text[:from])
	}
	pos := from
	r, w := at(pos)
	r1, w1 := rune(-1), 0
	if w > 0 {
		r1, w1 = at(pos + w)
	}
	run, next := &m.run, &m.next
	for {
		if len(run.dense) == 0 && m.matched != nil {
			break
		}
		if m.matched == nil {
			for i := range m.scratchCap {
				m.scratchCap[i] = -1
			}
			if len(m.scratchCap) > 0 {
				m.scratchCap[0] = pos
			}
			m.add(run, uint32(m.prog.Start), pos, m.scratchCap, before, r, nil)
		}
		m.step(run, next, pos, pos+w, r, r1)
		if w == 0 {
			break
		}
		before = r
		pos += w
		r, w = r1, w1
		if w > 0 {
			r1, w1 = at(pos + w)
		} else {
			r1, w1 = -1, 0
		}
		run, next = next, run
	}
	m.clear(run)
	m.clear(next)
	return m.matched
}

func (m *wordMatcher) clear(q *wordQueue) {
	for _, d := range q.dense {
		if d.t != nil {
			m.pool = append(m.pool, d.t)
		}
	}
	q.dense = q.dense[:0]
}

func (m *wordMatcher) alloc() *wordThread {
	if n := len(m.pool); n > 0 {
		t := m.pool[n-1]
		m.pool = m.pool[:n-1]
		return t
	}
	return &wordThread{cap: make([]int, m.ncap)}
}

// add puts pc's thread on q at pos, following the instructions that take
// no character (the assertions between before and after among them).
func (m *wordMatcher) add(q *wordQueue, pc uint32, pos int, cap []int, before, after rune, t *wordThread) *wordThread {
	if pc == 0 {
		return t
	}
	if j := q.sparse[pc]; int(j) < len(q.dense) && q.dense[j].pc == pc {
		return t
	}
	j := len(q.dense)
	q.dense = q.dense[:j+1]
	q.dense[j] = wordEntry{pc: pc}
	q.sparse[pc] = uint32(j)
	in := &m.prog.Inst[pc]
	switch in.Op {
	case syntax.InstFail:
	case syntax.InstAlt, syntax.InstAltMatch:
		t = m.add(q, in.Out, pos, cap, before, after, t)
		t = m.add(q, in.Arg, pos, cap, before, after, t)
	case syntax.InstEmptyWidth:
		if syntax.EmptyOp(in.Arg)&^wordContext(before, after) == 0 {
			t = m.add(q, in.Out, pos, cap, before, after, t)
		}
	case syntax.InstNop:
		t = m.add(q, in.Out, pos, cap, before, after, t)
	case syntax.InstCapture:
		if int(in.Arg) < len(cap) {
			old := cap[in.Arg]
			cap[in.Arg] = pos
			m.add(q, in.Out, pos, cap, before, after, nil)
			cap[in.Arg] = old
		} else {
			t = m.add(q, in.Out, pos, cap, before, after, t)
		}
	default:
		if st := m.stands[pc]; len(st.runes) > 0 {
			for _, r := range st.runes {
				if wordAssertion(r, before, after) {
					t = m.add(q, in.Out, pos, cap, before, after, t)
					break
				}
			}
			if st.only {
				return t
			}
		}
		if t == nil {
			t = m.alloc()
		}
		copy(t.cap, cap)
		q.dense[j].t = t
		t = nil
	}
	return t
}

// step runs the threads of run over r, the character from pos to
// nextPos (after it r1), into next; a match cuts off the threads of
// lower priority.
func (m *wordMatcher) step(run, next *wordQueue, pos, nextPos int, r, r1 rune) {
	for j := 0; j < len(run.dense); j++ {
		d := &run.dense[j]
		t := d.t
		if t == nil {
			continue
		}
		in := &m.prog.Inst[d.pc]
		take := false
		switch in.Op {
		case syntax.InstMatch:
			if len(t.cap) > 0 {
				t.cap[1] = pos
			}
			m.matched = append(m.matched[:0], t.cap...)
			for _, rest := range run.dense[j+1:] {
				if rest.t != nil {
					m.pool = append(m.pool, rest.t)
				}
			}
			run.dense = run.dense[:0]
		case syntax.InstRune:
			take = r >= 0 && in.MatchRune(r)
		case syntax.InstRune1:
			take = r >= 0 && r == in.Rune[0]
		case syntax.InstRuneAny:
			take = r >= 0
		case syntax.InstRuneAnyNotNL:
			take = r >= 0 && r != '\n'
		}
		if take {
			t = m.add(next, in.Out, nextPos, t.cap, r, r1, t)
		}
		if t != nil {
			m.pool = append(m.pool, t)
		}
	}
	run.dense = run.dense[:0]
}
