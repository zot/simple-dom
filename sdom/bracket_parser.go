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

	// R356: the groups open around the parse, innermost last, pushed and popped by
	// open. Scratch rather than a record — empty whenever a parse is not inside one.
	stack []frame
}

// frame is one open group and the text that opened it, which a CloseIsOpen closer
// must equal.
type frame struct {
	g      *BracketGroup
	opened string
}

// CRC: crc-BracketParser.md | R296
// NewBracketParser returns a parser for lang, with the context it will fill. It
// compiles the table's patterns once, and a table that cannot be constructed panics
// here naming the group. A table that is caller input is checked with Check first.
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

// CRC: crc-BracketParser.md | Seq: seq-parse.md#1 | R64, R294, R346
// parseBody parses until enclosing's closer is found, or to end of input. opened is
// the text that opened enclosing, carried down so a CloseIsOpen group closes on
// exactly those bytes. It reports whether the group ENDED — by its closer or a
// rejected run — rather than running out of input or meeting its bound.
func (bp *BracketParser) parseBody(st *ParserState, enclosing *BracketGroup, opened string, bound bool) bool {
	if enclosing != nil && enclosing.Restricted() {
		return bp.parseRestricted(st, enclosing, opened, bound)
	}
	return bp.parseCode(st, enclosing, opened, bound)
}

// CRC: crc-BracketParser.md | Seq: seq-parse.md#3.4.1 | R347, R353
// bounded decides once, at the opener, whether this instance of g ends at a blank
// line: a BlankLineBound group, unless LineHeadUnbound and the opener has only spaces
// or tabs before it on its line — a fence.
func bounded(src string, pos int, g *BracketGroup) bool {
	if !g.BlankLineBound {
		return false
	}
	if !g.LineHeadUnbound {
		return true
	}
	beforeOpener := src[strings.LastIndexByte(src[:pos], '\n')+1 : pos]
	return strings.TrimLeft(beforeOpener, " \t") != ""
}

// CRC: crc-BracketParser.md | Seq: seq-parse.md#3.4.1 | R347
// atBound reports whether a bounded group ends unclosed here: at a blank line — a
// newline followed by a line of only spaces or tabs.
func atBound(st *ParserState, bound bool) bool {
	return bound && blankLineAt(st.Src(), st.Pos())
}

// blankLineAt reports whether pos is at a newline followed by a blank line.
func blankLineAt(src string, pos int) bool {
	if pos >= len(src) || src[pos] != '\n' {
		return false
	}
	i := pos + 1
	for i < len(src) && (src[i] == ' ' || src[i] == '\t') {
		i++
	}
	return i >= len(src) || src[i] == '\n'
}

// CRC: crc-BracketParser.md | Seq: seq-parse.md#1.3 | R72, R73, R74, R75
//
// parseCode parses in code mode: openers of any group allowed here, then the open
// group's closers, then its separators, then the any-close fallback, then text.
func (bp *BracketParser) parseCode(st *ParserState, enclosing *BracketGroup, opened string, bound bool) bool {
	for st.Pos() < len(st.Src()) && !atBound(st, bound) {
		// R291: a group closed by its own text checks its closer before any opener,
		// or the marker would reopen rather than close; and a run of its pattern
		// that is not the opener's text is content, whole, or a rejected closer (R309).
		if enclosing != nil && enclosing.CloseIsOpen {
			if bp.closeGroup(st, enclosing, opened) {
				return true
			}
			handled, closed := bp.otherRun(st, enclosing, opened)
			if closed {
				return true
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
				return true
			}
			if m := matchAny(st.Src(), st.Pos(), enclosing.Separators); m != "" {
				st.Emit(NewSeparator(m, st.At(st.Pos(), len(m))))
				continue
			}
			// R356: an enclosing group's closer ends this one, unclosed, without
			// consuming — the frame it belongs to meets the same position next.
			if bp.enclosingCloses(st) {
				return false
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
	// A group left open at end of input closes there (R75) unless it demotes, which
	// is open's decision; the live run already holds every byte, so nothing drops.
	return false
}

// CRC: crc-BracketParser.md | Seq: seq-parse.md#2 | R294
//
// parseRestricted parses inside a string or a comment: only this group's closer, its
// Escape, and the openers of the groups AllowedInner names are recognized. Every
// other byte is literal — comments inside strings are not comments, and brackets
// inside comments are not brackets.
func (bp *BracketParser) parseRestricted(st *ParserState, g *BracketGroup, opened string, bound bool) bool {
	for st.Pos() < len(st.Src()) && !atBound(st, bound) {
		if bp.closeGroup(st, g, opened) {
			return true
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
			return true
		}
		if handled {
			continue
		}
		st.Advance(1)
	}
	return false
}

// CRC: crc-BracketParser.md | Seq: seq-pair.md#1.1, seq-parse.md#3.4.2 | R81, R346, R349
//
// open emits an opener for g and parses its body. When the body ends without the
// group ending and the group demotes, the opener was text: the state rewinds to
// before the opener, the marker's bytes are taken as text, the demotion is recorded
// — after dropping any recorded inside the group, which the outer re-parse records
// again — and the enclosing loop re-reads everything the group had enclosed in its
// own mode. The indent parser and the markdown wrapper were never offered a position
// inside the group, so nothing above this parser has state to rewind.
func (bp *BracketParser) open(st *ParserState, g *BracketGroup, marker string) {
	count, pos, recorded := st.NodeCount(), st.Pos(), len(bp.ctx.demoted)
	bound := bounded(st.Src(), pos, g)
	st.Emit(NewOpener(marker, st.At(pos, len(marker))))
	bp.stack = append(bp.stack, frame{g, marker})
	ended := bp.parseBody(st, g, marker, bound)
	bp.stack = bp.stack[:len(bp.stack)-1]
	if ended || !g.DemoteUnclosed {
		return
	}
	st.Rewind(count, pos)
	st.Advance(len(marker))
	run := st.Last().(*Text)
	demotion := DemotedOpener{Marker: marker, Run: run, Offset: pos - run.Location().Offset()}
	bp.ctx.demoted = append(bp.ctx.demoted[:recorded], demotion)
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

// CRC: crc-BracketParser.md | Seq: seq-parse.md#1.6 | R356
// enclosingCloses reports whether a group open below the current one closes here, the
// nearest first. The search stops at a parse-restricted group: its own closer can end
// it, but it reads any other closer as text, so nothing below it can be reached.
func (bp *BracketParser) enclosingCloses(st *ParserState) bool {
	for j := len(bp.stack) - 2; j >= 0; j-- {
		f := bp.stack[j]
		if bp.matchCloser(st.Src(), st.Pos(), bp.index(f.g), f.opened) != "" {
			return true
		}
		if f.g.Restricted() {
			return false
		}
	}
	return false
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
