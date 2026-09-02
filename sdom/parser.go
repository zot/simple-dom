package sdom

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
