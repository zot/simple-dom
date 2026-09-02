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
	ctx  *BracketContext
}

// NewBracketParser returns a parser for lang, with the context it will fill.
func NewBracketParser(lang *BracketLang) *BracketParser {
	return &BracketParser{lang: lang, ctx: newBracketContext(lang)}
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

// CRC: crc-BracketParser.md | Seq: seq-parse.md#1 | R64, R65
// parseBody parses until enclosing's closer is found, or to end of input.
func (bp *BracketParser) parseBody(st *ParserState, enclosing *BracketGroup) {
	if enclosing != nil && enclosing.Restricted() {
		bp.parseRestricted(st, enclosing)
		return
	}
	bp.parseCode(st, enclosing)
}

// CRC: crc-BracketParser.md | Seq: seq-parse.md#1.3 | R72, R73, R74, R75
//
// parseCode parses in code mode: openers of any group allowed here, then the open
// group's closers, then its separators, then the any-close fallback, then text.
func (bp *BracketParser) parseCode(st *ParserState, enclosing *BracketGroup) {
	for st.Pos() < len(st.Src()) {
		if g, m := bp.matchOpen(st, enclosing); g != nil {
			bp.open(st, g, m)
			continue
		}
		if enclosing != nil {
			if bp.closeGroup(st, enclosing) {
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
		st.SetPos(st.Pos() + 1)
	}
	st.FlushText() // a group left open at end of input closes there; no bytes drop
}

// CRC: crc-BracketParser.md | Seq: seq-parse.md#2 | R65
//
// parseRestricted parses inside a string or a comment: only this group's Close, its
// Escape, and the openers named in AllowedInner are recognized. Every other byte is
// literal — comments inside strings are not comments, and brackets inside comments
// are not brackets.
func (bp *BracketParser) parseRestricted(st *ParserState, g *BracketGroup) {
	for st.Pos() < len(st.Src()) {
		if bp.closeGroup(st, g) {
			return
		}
		if g.Escape != "" && strings.HasPrefix(st.Src()[st.Pos():], g.Escape) {
			st.SetPos(st.Pos() + len(g.Escape))
			if st.Pos() < len(st.Src()) {
				st.SetPos(st.Pos() + 1) // the escaped byte is literal, whatever it is
			}
			continue
		}
		if inner, m := bp.matchInner(st, g); inner != nil {
			bp.open(st, inner, m)
			continue
		}
		st.SetPos(st.Pos() + 1)
	}
	st.FlushText()
}

// CRC: crc-BracketParser.md | Seq: seq-pair.md#1.1 | R81
// open emits an opener for g and parses its body.
func (bp *BracketParser) open(st *ParserState, g *BracketGroup, marker string) {
	st.Emit(NewOpener(marker, st.At(st.Pos(), len(marker))))
	bp.parseBody(st, g)
}

// CRC: crc-BracketParser.md | Seq: seq-parse.md#1.4 | R82, R83
//
// closeGroup ends the open group when one of its closers is at pos, and reports
// whether it did. Code mode and restricted mode differ in what they recognize but
// end a group identically, so both call this.
func (bp *BracketParser) closeGroup(st *ParserState, g *BracketGroup) bool {
	m := matchAny(st.Src(), st.Pos(), g.Close)
	if m == "" {
		return false
	}
	st.Emit(NewCloser(m, st.At(st.Pos(), len(m))))
	return true
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
		if m := matchAny(st.Src(), st.Pos(), g.Open); m != "" {
			return g, m
		}
	}
	return nil, ""
}

// CRC: crc-BracketParser.md | R65
// matchInner finds an opener named in g.AllowedInner, and the group owning it.
func (bp *BracketParser) matchInner(st *ParserState, g *BracketGroup) (*BracketGroup, string) {
	for _, op := range g.AllowedInner {
		if !matchAt(st.Src(), st.Pos(), op) {
			continue
		}
		if owner := bp.lang.groupFor(op); owner != nil {
			return owner, op
		}
	}
	return nil, ""
}

// CRC: crc-BracketParser.md | Seq: seq-parse.md#3.1 | R73
// matchAnyClose recognizes any code-mode group's closer, so depth stays consistent
// even when the document is unbalanced.
func (bp *BracketParser) matchAnyClose(st *ParserState) string {
	for i := range bp.lang.Brackets {
		g := &bp.lang.Brackets[i]
		if g.Restricted() {
			continue
		}
		if m := matchAny(st.Src(), st.Pos(), g.Close); m != "" {
			return m
		}
	}
	return ""
}
