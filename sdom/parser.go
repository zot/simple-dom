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
// ParserState is one pass over one source: the position, the live text run, the
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
	src    string
	pos    int
	origin *Origin
	out    []Node
	parser Parser

	// R224: the LIVE text run — the Text every declined byte is extending, or nil
	// when the last node emitted was a marker. There is one representation of
	// parse state, the array, and the last node in it is always current.
	text *Text
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
			st.Advance(1) // nothing recognized here, so this byte is text
		}
	}
}

// Src returns the whole source being parsed.
func (st *ParserState) Src() string { return st.src }

// Pos returns the position the walk has reached.
func (st *ParserState) Pos() int { return st.pos }

// SetPos moves the walk WITHOUT consuming: a lookahead moves forward and back with
// it and no byte becomes text. A parser that takes bytes as text uses Advance.
func (st *ParserState) SetPos(n int) { st.pos = n }

// CRC: crc-ParserState.md | Seq: seq-collaborate.md#1.4 | R224
//
// Advance consumes n bytes as text, extending the live run — creating it on the
// first declined byte after a marker — so the last node in the array is current
// after every move. The extension is a substring of the source and a length bump:
// nothing is allocated per byte, and the location stays faithful.
func (st *ParserState) Advance(n int) {
	from := st.pos
	st.pos += n
	if st.text == nil {
		st.text = NewText(st.src[from:st.pos], st.At(from, n))
		st.append(st.text)
		return
	}
	start := st.text.loc.Offset()
	st.text.text = st.src[start:st.pos]
	st.text.loc.length = st.pos - start
}

// CRC: crc-ParserState.md | R158, R225
//
// Last returns the most recently emitted node, or nil before any — the live text
// run when the walk is inside one, a marker just after one was emitted. One node,
// not the slice: a parser that wants to know what precedes the position looks here.
func (st *ParserState) Last() Node {
	if len(st.out) == 0 {
		return nil
	}
	return st.out[len(st.out)-1]
}

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
// R158: this, Emit, Advance and Last are the whole surface; the node slice is not exported. A
// parser appends and asks how many; it never needs the array, and handing out the
// live one is the aliasing shape O2 and O21 already record.
func (st *ParserState) NodeCount() int { return len(st.out) }

// CRC: crc-ParserState.md | Seq: seq-collaborate.md#1.6 | R76, R77, R156, R225
//
// Emit appends n, advances past it, and ends the live text run — the next declined
// byte starts a new one, so a text run is everything between two markers (R76) and
// the array is in document order without anyone ordering it. The width comes from
// the node's own location, so the two cannot disagree.
//
// R225, stated rather than guarded: a parser never emits a node over bytes already
// in the live run. None does — each is offered every position and never backs up —
// but the walk relies on it, since the run is not shrunk here.
func (st *ParserState) Emit(n Node) {
	st.append(n)
	st.pos += n.Location().Length()
	st.text = nil
}

func (st *ParserState) append(n Node) { st.out = append(st.out, n) }

// CRC: crc-ParserState.md | Seq: seq-parse.md#3.4.2 | R350
//
// Rewind drops the nodes past count, moves the position back to pos, and makes the
// last remaining node the live text run again when it is a Text — so the bytes re-read
// from there extend it as declined bytes do. After a rewind to a marker there is no
// live run and the next declined byte starts one. This is the one way back, and it is
// what keeps R225 true for the one parser that backs up: the bracket parser demoting
// an opener never closed.
func (st *ParserState) Rewind(count, pos int) {
	st.out = st.out[:count]
	st.pos = pos
	st.text = nil
	if count > 0 {
		st.text, _ = st.out[count-1].(*Text) // nil unless what remains ends in a run
	}
}
