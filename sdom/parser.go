package sdom

import "strings"

// CRC: crc-ParserState.md | Seq: seq-collaborate.md#1 | R155, R161
//
// Parse walks src once with parser and returns the document it produced.
//
// It returns the document ALONE. Each parser owns the context it fills, so a
// caller reads that from the parser it constructed — a single return value could
// only be `any`, and a caller asserting it straight back to the concrete type
// would be buying nothing.
//
// base is the document's own position within an outer document; node offsets are
// relative to src.
func Parse(src string, base int, parser Parser) *Doc {
	st := &ParserState{src: src, origin: &Origin{}, parser: parser}
	st.run()
	d := New(src, base, st.out...)
	parser.Done(d)
	return d
}

// CRC: crc-Parser.md | R159, R160, R165
//
// Parser recognizes what it knows at the head of the input.
//
// R165: it has two ways to consume. It may recognize one thing and return, leaving
// the walk to offer the next position; or it may TAKE THE LOOP, recursing until its
// own terminator. Taking the loop is what makes suppression structural — a parser
// registered only in the outermost loop is offered no position inside a group.
type Parser interface {
	// Parse recognizes something at the head of the input and emits it, or does
	// nothing at all.
	Parse(st *ParserState)

	// R160: NodeType reports the kind of node Parse WOULD emit here without
	// emitting it, and reports separately that it would emit nothing. One string
	// would conflate "no node here" with "a node whose group carries no kind", and
	// those are different claims — the same reason a location's offset is stored
	// biased so that absence is not offset 0.
	NodeType(st *ParserState) (kind string, ok bool)

	// R192: Done is called once, after the document exists. A parser holding a
	// derived index binds it to that document there, and nothing earlier can — the
	// nodes are emitted before the document is built.
	Done(d *Doc)
}

// CRC: crc-ParserState.md | R156, R157, R162
//
// ParserState is one pass over one source: the position, the pending text, the
// nodes emitted so far, and the Origin every node of this pass carries.
//
// R157: the Origin lives HERE rather than on a context. It identifies one PARSE,
// and with several parsers collaborating there is still only one. A per-context
// origin would mint two for a single document, and merging a location from one with
// a location from the other is defined to panic.
//
// R162: it holds ONE parser, which may delegate. Which language rule wins is that
// parser's business; a walk arbitrating between parsers would be a second place to
// encode precedence.
type ParserState struct {
	src       string
	pos       int
	textStart int
	origin    *Origin
	out       []Node
	parser    Parser
}

// CRC: crc-ParserState.md | Seq: seq-collaborate.md#1.4 | R163, R164, R167
//
// run turns the loop. Each position is offered exactly once.
//
// R164: nothing happened means the position is unchanged AND the node count is
// unchanged. Neither alone is sound. Position alone misses a ZERO-LENGTH node, and
// the byte after one can be a bracket opener rather than the line's content. Node
// count alone misses a parser that ADVANCES WITHOUT EMITTING, as an escape inside a
// restricted group does. The failure either shortcut permits is a skipped marker
// rather than lost bytes: the source still round-trips and only a recognition count
// sees it.
//
// R167: progress is a byte or a node, so this terminates. A parser that emits at one
// position without ever advancing would spin — a parser bug, stated rather than
// guarded, since the check would cost every position of every parse.
func (st *ParserState) run() {
	for st.pos < len(st.src) {
		pos, n := st.pos, len(st.out)
		st.parser.Parse(st)
		if st.pos == pos && len(st.out) == n {
			st.pos++ // nothing recognized here, so this byte is text
		}
	}
	st.FlushText()
}

// Src returns the whole source being parsed.
func (st *ParserState) Src() string { return st.src }

// Pos returns the position the walk has reached.
func (st *ParserState) Pos() int { return st.pos }

// SetPos moves the walk. A parser that recognizes something advances past it.
func (st *ParserState) SetPos(n int) { st.pos = n }

// Origin returns the token identifying this parse.
func (st *ParserState) Origin() *Origin { return st.origin }

// CRC: crc-ParserState.md | R156
// At builds a location for a span of this parse, attributed to its origin.
func (st *ParserState) At(pos, length int) Loc {
	return Source(pos, length).In(st.origin)
}

// CRC: crc-ParserState.md | R158
//
// NodeCount reports how many nodes have been emitted.
//
// R158: this and Emit are the whole surface, and the node slice is not exported. A
// parser appends and asks how many; it never needs the array, and handing out the
// live one is the aliasing shape O2 and O21 already record.
func (st *ParserState) NodeCount() int { return len(st.out) }

// CRC: crc-ParserState.md | Seq: seq-collaborate.md#1.6 | R156, R158
//
// Emit flushes the pending text, appends n, and advances past it. The width comes
// from the node's own location, so the two cannot disagree.
func (st *ParserState) Emit(n Node) {
	st.FlushText()
	st.append(n)
	st.pos += n.Location().Length()
	st.textStart = st.pos
}

// CRC: crc-ParserState.md | Seq: seq-parse.md#1.5 | R76, R77
//
// FlushText emits the bytes accumulated since the last node. Whitespace is not a
// node of its own, so a text run is everything between two recognized markers.
func (st *ParserState) FlushText() {
	if st.pos > st.textStart {
		st.append(NewText(st.src[st.textStart:st.pos], st.At(st.textStart, st.pos-st.textStart)))
	}
	st.textStart = st.pos
}

func (st *ParserState) append(n Node) { st.out = append(st.out, n) }

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
