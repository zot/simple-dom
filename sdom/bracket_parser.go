package sdom

import "strings"

// CRC: crc-BracketParser.md | Seq: seq-parse.md, seq-collaborate.md | R57, R72, R73, R74, R75, R76, R77, R78, R165, R166
//
// BracketParser is the table-driven parser, as a Parser.
//
// The language, the context and the stack are ITS OWN rather than the walk's: the
// walk holds no language, and each parser owns the context it fills. The source and
// the position belong to ParserState, which it is handed.
//
// The nesting lives on the CALL STACK — parseBody recurses — and is simply absent
// from the data afterwards, which is what makes the emitted array flat without
// anything having to flatten it.
//
// R87: it records NO links while it parses. Every bracket link is derivable from the
// finished array, so the context derives them once, on demand, and the parse stays a
// parse. Recording them here would be a second copy of a fact the array already
// carries — and a second copy is a thing that can disagree.
type BracketParser struct {
	lang *BracketLang
	pats *patterns
	ctx  *BracketContext
}

// CRC: crc-BracketParser.md | R296
// NewBracketParser returns a parser for lang, with the context it will fill. It
// compiles the table's patterns once, and a table that cannot be constructed panics
// here naming the group — a library invariant, not caller input.
func NewBracketParser(lang *BracketLang) *BracketParser {
	pats, err := lang.check()
	if err != nil {
		panic(err)
	}
	return &BracketParser{lang: lang, pats: pats, ctx: newBracketContext(lang, pats)}
}

// Context returns the pairing links this parser recorded. Concrete, so no caller
// asserts anything.
func (bp *BracketParser) Context() *BracketContext { return bp.ctx }

// CRC: crc-BracketParser.md | Seq: seq-collaborate.md#2 | R165
//
// Parse recognizes one thing at the head of the input. On an opener it TAKES THE
// LOOP, recursing until the matching closer — which is what makes a restricted
// group's exclusivity structural rather than a flag, and what leaves a parser
// registered only in the outermost loop with no position offered inside a group.
func (bp *BracketParser) Parse(st *ParserState) {
	bp.bind(st)
	if g, m := bp.matchOpen(st, nil); g != nil {
		bp.open(st, g, m)
		return
	}
	if m := bp.matchAnyClose(st); m != "" {
		st.Emit(NewCloser(m, st.At(st.Pos(), len(m))))
	}
}

// CRC: crc-BracketParser.md | R160
// NodeType reports the kind of the node Parse would emit here, without emitting it.
func (bp *BracketParser) NodeType(st *ParserState) (string, bool) {
	bp.bind(st)
	if g, _ := bp.matchOpen(st, nil); g != nil {
		return g.Kind, true
	}
	if m := bp.matchAnyClose(st); m != "" {
		return "", true
	}
	return "", false
}

// CRC: crc-BracketParser.md | Seq: seq-stamp.md | R157
// Done binds the context to the finished document and stamps it, so the first read
// of a freshly parsed document rebuilds nothing.
func (bp *BracketParser) Done(d *Doc) { bp.ctx.attach(d) }

// bind points the context at this pass's origin. Assigning the same pointer on
// every call is free and idempotent; a parser instance belongs to one parse, as the
// unexported walk it replaced always did.
func (bp *BracketParser) bind(st *ParserState) { bp.ctx.origin = st.origin }

// CRC: crc-BracketParser.md | Seq: seq-parse.md#1 | R64, R294
// parseBody parses until enclosing's closer is found, or to end of input. opened is
// the text that opened enclosing, carried down so a CloseIsOpen group closes on
// exactly those bytes.
func (bp *BracketParser) parseBody(st *ParserState, enclosing *BracketGroup, opened string) {
	if enclosing != nil && enclosing.Restricted() {
		bp.parseRestricted(st, enclosing, opened)
		return
	}
	bp.parseCode(st, enclosing, opened)
}

// CRC: crc-BracketParser.md | Seq: seq-parse.md#1.3 | R72, R73, R74, R75
//
// parseCode parses in code mode: openers of any group allowed here, then the open
// group's closers, then its separators, then the any-close fallback, then text.
func (bp *BracketParser) parseCode(st *ParserState, enclosing *BracketGroup, opened string) {
	for st.Pos() < len(st.Src()) {
		// R291: a group closed by its own text checks its closer before any opener,
		// or the marker would reopen rather than close; and a run of its pattern
		// that is not the opener's text is content, whole, or a rejected closer (R309).
		if enclosing != nil && enclosing.CloseIsOpen {
			if bp.closeGroup(st, enclosing, opened) {
				return
			}
			handled, closed := bp.otherRun(st, enclosing, opened)
			if closed {
				return
			}
			if handled {
				continue
			}
		}
		if g, m := bp.matchOpen(st, enclosing); g != nil {
			bp.open(st, g, m)
			continue
		}
		if enclosing != nil {
			if bp.closeGroup(st, enclosing, opened) {
				return
			}
			if m := matchAny(st.Src(), st.Pos(), enclosing.Separators); m != "" {
				st.Emit(NewSeparator(m, st.At(st.Pos(), len(m))))
				continue
			}
		}
		// The any-close fallback: a stray closer lands as a bracket rather than
		// derailing the parse.
		if m := bp.matchAnyClose(st); m != "" {
			st.Emit(NewCloser(m, st.At(st.Pos(), len(m))))
			continue
		}
		// Nothing matched here, so this byte is text. Advancing unconditionally is
		// what guarantees the parse always consumes at least one byte.
		st.Advance(1)
	}
	// A group left open at end of input closes there; the live run already holds
	// every byte, so nothing drops.
}

// CRC: crc-BracketParser.md | Seq: seq-parse.md#2 | R294
//
// parseRestricted parses inside a string or a comment: only this group's closer, its
// Escape, and the openers of the groups AllowedInner names are recognized. Every
// other byte is literal — comments inside strings are not comments, and brackets
// inside comments are not brackets.
func (bp *BracketParser) parseRestricted(st *ParserState, g *BracketGroup, opened string) {
	for st.Pos() < len(st.Src()) {
		if bp.closeGroup(st, g, opened) {
			return
		}
		if g.Escape != "" && strings.HasPrefix(st.Src()[st.Pos():], g.Escape) {
			st.Advance(len(g.Escape))
			if st.Pos() < len(st.Src()) {
				st.Advance(1) // the escaped byte is literal, whatever it is
			}
			continue
		}
		// The hatches before the run rule, so an inner run of another length can open
		// a nested group — emphasis inside emphasis — rather than be taken as content.
		if inner, m := bp.matchInner(st, g); inner != nil {
			bp.open(st, inner, m)
			continue
		}
		handled, closed := bp.otherRun(st, g, opened)
		if closed {
			return
		}
		if handled {
			continue
		}
		st.Advance(1)
	}
}

// CRC: crc-BracketParser.md | Seq: seq-pair.md#1.1 | R81
// open emits an opener for g and parses its body.
func (bp *BracketParser) open(st *ParserState, g *BracketGroup, marker string) {
	st.Emit(NewOpener(marker, st.At(st.Pos(), len(marker))))
	bp.parseBody(st, g, marker)
}

// CRC: crc-BracketParser.md | Seq: seq-parse.md#1.4 | R82, R83, R291
//
// closeGroup ends the open group when its closer is at pos, and reports whether it
// did. Code mode and restricted mode differ in what they recognize but end a group
// identically, so both call this. opened is the text that opened the group, which is
// the closer itself when CloseIsOpen.
func (bp *BracketParser) closeGroup(st *ParserState, g *BracketGroup, opened string) bool {
	m := bp.matchCloser(st.Src(), st.Pos(), bp.index(g), opened)
	if m == "" {
		return false
	}
	st.Emit(NewCloser(m, st.At(st.Pos(), len(m))))
	return true
}

// index is a group's slot in the table, and so in the parallel pattern slices.
func (bp *BracketParser) index(g *BracketGroup) int {
	for i := range bp.lang.Brackets {
		if &bp.lang.Brackets[i] == g {
			return i
		}
	}
	return -1
}

// CRC: crc-BracketParser.md | R292, R309
//
// matchGroupOpen reports the opener of group i at pos, or "": a pattern match or a
// literal opener, honouring the group's AfterOpen.
func (bp *BracketParser) matchGroupOpen(src string, pos, i int) string {
	var m string
	if bp.pats.open[i] != nil {
		m = bp.matchPattern(src, pos, i)
	} else {
		m = matchAny(src, pos, bp.lang.Brackets[i].Open)
	}
	if m == "" || !afterOK(src, pos+len(m), bp.pats.after[i]) {
		return ""
	}
	return m
}

// CRC: crc-BracketParser.md | R292
//
// matchPattern reports what group i's OpenRegex matches at pos, or "", without the
// lookahead. It is gated by the pattern's literal prefix so the regexp runs only where
// it could match, and the bytes it matched honour the word-boundary rule like a
// literal marker.
func (bp *BracketParser) matchPattern(src string, pos, i int) string {
	rest := src[pos:]
	if !strings.HasPrefix(rest, bp.pats.prefix[i]) {
		return ""
	}
	loc := bp.pats.open[i].FindStringIndex(rest)
	if loc == nil || loc[1] == 0 || !boundaryOK(src, pos, pos+loc[1]) {
		return ""
	}
	return rest[:loc[1]]
}

// CRC: crc-BracketParser.md | Seq: seq-parse.md#1.4 | R309
//
// otherRun handles a match of a close-is-open group's pattern that is not the text
// that opened it. A shorter run is content, taken whole so the parse never stands one
// byte into a run and reads its tail as a marker. A longer run is content too, unless
// the group rejects longer closes: then it is an unbalanced closer — emitted, paired
// with nothing — and the group ends there. handled says the position moved; closed
// says the group ended.
func (bp *BracketParser) otherRun(st *ParserState, g *BracketGroup, opened string) (handled, closed bool) {
	if !g.CloseIsOpen {
		return false, false
	}
	i := bp.index(g)
	if bp.pats.open[i] == nil {
		return false, false
	}
	m := bp.matchPattern(st.Src(), st.Pos(), i)
	if m == "" || m == opened {
		return false, false
	}
	if g.RejectLongerCloses && len(m) > len(opened) {
		st.Emit(NewCloser(m, st.At(st.Pos(), len(m))))
		return true, true
	}
	st.Advance(len(m))
	return true, false
}

// CRC: crc-BracketParser.md | Seq: seq-parse.md#1.4 | R291, R309
// matchCloser reports group i's closer at pos, or "": its literal Close, or with
// CloseIsOpen the very bytes that opened it — for a pattern group, the pattern's match
// here equal to them — under the group's BeforeClose. opened is unread for a group that
// is not CloseIsOpen, which is what lets the any-close fallback ask this question with
// no group open.
func (bp *BracketParser) matchCloser(src string, pos, i int, opened string) string {
	g := &bp.lang.Brackets[i]
	want := g.Close
	if g.CloseIsOpen {
		want = opened
		// A pattern group closes on the whole run here: matching the opened bytes is
		// not enough, or a longer run would close on its prefix.
		if bp.pats.open[i] != nil && bp.matchPattern(src, pos, i) != opened {
			return ""
		}
	}
	if !matchAt(src, pos, want) || !beforeOK(src, pos, bp.pats.before[i]) {
		return ""
	}
	return want
}

// CRC: crc-BracketParser.md | R66
// matchOpen finds the first group whose opener matches here and whose AllowedParent
// permits the enclosing group.
func (bp *BracketParser) matchOpen(st *ParserState, enclosing *BracketGroup) (*BracketGroup, string) {
	for i := range bp.lang.Brackets {
		g := &bp.lang.Brackets[i]
		if !g.parentAllowed(enclosing) {
			continue
		}
		if m := bp.matchGroupOpen(st.Src(), st.Pos(), i); m != "" {
			return g, m
		}
	}
	return nil, ""
}

// CRC: crc-BracketParser.md | Seq: seq-parse.md#2.2.3 | R294, R295
// matchInner finds an opener of a group named in g.AllowedInner — by a literal opener
// or by its pattern — and matches that group's own opener rather than the name.
func (bp *BracketParser) matchInner(st *ParserState, g *BracketGroup) (*BracketGroup, string) {
	for _, name := range g.AllowedInner {
		i := bp.lang.indexFor(name)
		if i < 0 {
			continue
		}
		if m := bp.matchGroupOpen(st.Src(), st.Pos(), i); m != "" {
			return &bp.lang.Brackets[i], m
		}
	}
	return nil, ""
}

// CRC: crc-BracketParser.md | Seq: seq-parse.md#3.1 | R73, R297
// matchAnyClose recognizes any code-mode group's literal closer, so depth stays
// consistent even when the document is unbalanced. A close-is-open marker outside
// its group is an opener and has already matched as one.
func (bp *BracketParser) matchAnyClose(st *ParserState) string {
	for i := range bp.lang.Brackets {
		g := &bp.lang.Brackets[i]
		if g.Restricted() || g.CloseIsOpen {
			continue
		}
		if m := bp.matchCloser(st.Src(), st.Pos(), i, ""); m != "" {
			return m
		}
	}
	return ""
}
