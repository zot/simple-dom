# The parser protocol

One source, **one pass**, several parsers. A parser recognizes what it knows at the
head of the input and does nothing anywhere else; an outer loop accumulates whatever
nobody recognized into text.

This is not a new mechanism. The bracket parser already ran exactly this loop with a
fixed matcher list; the protocol opens that list, so a second layer — indent scope —
can contribute nodes to the same pass instead of splitting and re-carving the array
afterwards.

## The walk

```go
// ParserState is one pass over one source: the position, the nodes emitted so
// far — the last of which is always current — and the Origin every node produced
// by this pass carries.
type ParserState struct { /* … */ }

func (st *ParserState) Src() string          // the whole source
func (st *ParserState) Pos() int             // where the walk is
func (st *ParserState) SetPos(n int)         // move it WITHOUT consuming — a lookahead
func (st *ParserState) Advance(n int)        // consume n bytes as text, extending the live run
func (st *ParserState) At(pos, length int) Loc  // a location in this pass
func (st *ParserState) Emit(n Node)          // append n, advance past it, end the live run
func (st *ParserState) Rewind(count, pos int) // drop the nodes past count, move back to pos, and make the last node live again
func (st *ParserState) NodeCount() int       // how many nodes have been emitted
func (st *ParserState) Last() Node           // the most recent node, nil before any
```

**The array is the only parse state, and its last node is live.** There is no pending
text held apart from the array: the first byte nobody recognizes **creates** a `Text`,
and every further byte consumed as text **extends** it — a substring of the source and
a length bump, allocating nothing and keeping the location faithful. `Emit` ends the
run; the next declined byte starts a new one. So at any moment the last node in the
array is a faithful picture of where the parse stands, and a parser that wants to know
what precedes the position — an indent parser asking whether the previous line ended
in a continuation marker, a markdown parser asking whether it is at a line head — reads
`Last` rather than any private state.

**Two ways to move, and they mean different things.** `Advance` consumes: the bytes
become text. `SetPos` only moves: a lookahead goes forward and back with it and no byte
changes kind. A parser that takes bytes as literal — an escape, the interior of a
restricted group — advances; a parser peeking at what a later position would parse as
sets the position and restores it.

**One rule a parser honours, stated rather than guarded:** it never emits a node over
bytes already in the live run. The walk relies on it, since `Emit` does not shrink the
run. The one parser that backs up — the bracket parser demoting an opener never closed —
does so through **`Rewind`**, which truncates the array to a node count, moves the position
back, and points the live run at the last remaining node when that node is a `Text`, so the
bytes re-read from there extend it as declined bytes do; a rewind to a marker leaves no
live run, and the next declined byte starts one. The rule holds because the only way back
is the verb that keeps the run true.

**`Emit`, `NodeCount` and `Last` rather than an exported node slice.** A parser needs
to append, to know how many nodes exist, and to see the one before it; it never needs
the array. Handing out the live slice is the shape two gaps already record, where a
caller writing through a returned value corrupts state in place.

**The `Origin` belongs to the walk, not to a context.** It identifies one *parse*,
and with several parsers collaborating there is still only one. A per-context origin
would mint two for a single document, and merging a location from one with a location
from the other is defined to panic.

## The parser

```go
// Parser recognizes what it knows at the head of the input.
type Parser interface {
    // Parse either recognizes something here and emits it, or does nothing at all.
    Parse(st *ParserState)

    // NodeType reports the kind of the node Parse would emit here, without
    // emitting it. ok is false when Parse would emit nothing.
    NodeType(st *ParserState) (kind string, ok bool)

    // Done is called once, after the document exists.
    Done(d *Doc)
}

// Parse walks src once with parser and returns the document it produced.
func Parse(src string, base int, parser Parser) *Doc
```

**A parser owns its context**, so `Parse` returns only the document and the caller
reads the context off the parser it constructed. Each schema's context is its own
concrete type; a single return value could only be `any`, and a caller asserting it
straight back to the concrete type buys nothing.

**`NodeType` returns two values on purpose.** A single string would conflate *no node
here* with *a node whose group carries no kind*, and those are different claims — the
same reason a location's offset is stored biased, so that absence is not offset 0.

**`Done` fires once, after the document is built**, which is where a parser holding
a derived index binds it to that document. Nothing earlier can — the nodes are
emitted before the document exists — so the parser is told rather than the caller
having to know a context needs attaching.

**A parser records nothing while it parses**, and that is a rule rather than a
habit. Whatever it could record is already implied by the array it is building, so
recording it would be a second copy of the same fact — and a second copy is a thing
that can disagree. The context derives what it owes on demand, once.

**`ParserState` holds one parser, which may delegate.** Composition is the parser's
own business, not the walk's: an indent parser holds a bracket parser and hands off
when it does not match. **Precedence is therefore never the loop's concern** — a
question of which language rule wins belongs to the language, and a loop arbitrating
between parsers would be a second place to encode it.

## The loop

Each position is offered to the parser exactly once.

```
for st.Pos() < len(src) {
    pos, n := st.Pos(), st.NodeCount()
    parser.Parse(st)
    if st.Pos() == pos && st.NodeCount() == n {
        st.Advance(1)           // nothing recognized here, so this byte is text
    }
}
```

There is nothing to flush at the end: every byte is already in the array.

**Nothing recognized is tested on *both* the position and the node count**, and
neither alone is sound.

- **Position alone misses a zero-length node.** A parser may emit a marker that
  consumes no bytes — an indent change back to column 0 has no whitespace to own. The
  loop would read that as no match, advance, and take the next byte as text. That byte
  is usually the line's content, but it can be a bracket opener: a string, a
  parenthesized expression or a list display at column 0 are all statements.
- **Node count alone is sufficient only by accident.** A parser may advance without
  emitting — an escape inside a restricted group consumes itself and the byte after
  it, both literal text. It is invisible to this loop only because restricted regions
  run inside the bracket parser's own loop.

The failure either shortcut permits is a **skipped marker rather than lost bytes**:
the source still round-trips, the array still tiles, and only a recognition count
sees it.

**A parser has two ways to consume.** It may recognize one thing and return, leaving
the loop to offer the next position; or it may **take the loop**, recursing until its
own terminator. Taking the loop is what a bracket group already does, and it is what
makes suppression structural: a parser registered only in the outermost loop is never
offered a position inside a group, so *indentation is significant only at bracket
depth 0* needs no check, and a restricted group's exclusivity needs no flag.

**Progress is a byte or a node.** The loop always advances when nothing was
recognized, so it terminates — but a parser that emits at one position without ever
advancing would spin. Not guarding this is deliberate: the violation is a parser bug
rather than a data condition, and the guard would cost a check at every position of
every parse to catch something a first test run makes obvious.
