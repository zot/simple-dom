# Test Design: the parser protocol
**Source:** crc-ParserState.md, crc-Parser.md

Fire alarms are added in the Implementation phase, when the sites they inject into
exist.

## Test: a zero-length node is not read as nothing happening
**Purpose:** R164 — the half of the no-change rule that `pos` alone misses
**Input:** a parser that emits a zero-length node at position 0 and nothing after
**Expected:** the node is in the array, and the byte at 0 is still the first byte
of the following text — not consumed as text *in addition to* the emission
**Refs:** crc-ParserState.md, seq-collaborate.md#1.4
**Code:** sdom/protocol_test.go
**Alarm:** 1
**Fire alarm:** In `ParserState.run`, drop the node-count term from the no-change test, leaving `st.Pos() == pos`. Red: the emission is read as a non-match, so the walk advances over it and the offered positions become `[0, 1]` instead of `[0, 0, 1]` — and the byte it swallows is text here but a bracket opener after a real dedent to column 0. Every byte is still emitted somewhere, so the round trip and the tiling stay green; only a recognition count sees it.
**Inject:** sdom/parser.go:ParserState.run

## Test: a parser that advances without emitting is not read as a match
**Purpose:** R164 — the half of the rule that the node count alone misses
**Input:** a parser that advances two bytes and emits nothing
**Expected:** the walk does not also take a byte, so exactly those two bytes land
in the pending text run and the position is where the parser left it
**Refs:** crc-ParserState.md, seq-collaborate.md#1.4
**Code:** sdom/protocol_test.go
**Alarm:** 2
**Fire alarm:** In `ParserState.run`, drop the position term instead, leaving `len(st.out) == n`. Red: a parser that advanced two bytes is read as having done nothing, so the walk takes a third — offered positions become `[0, 3]` rather than `[0, 2, 3]`. The skipped byte can be a marker, and nothing but a recognition count would notice.
**Inject:** sdom/parser.go:ParserState.run

## Test: every position is offered exactly once
**Purpose:** R163 — and that a parser declining twice at one position terminates
**Input:** a recording parser over a short source, declining everywhere
**Expected:** the positions it was offered are `0..n-1` with no repeats and no
holes, and the whole source arrives as one text node
**Refs:** crc-ParserState.md, seq-collaborate.md#1.5
**Code:** sdom/protocol_test.go

## Test: one parse, one origin
**Purpose:** R157 — the property that makes merging across parsers legal
**Input:** a source parsed with an indent parser holding a bracket parser, taking
one node emitted by each
**Expected:** both carry the same `Origin`, and merging their locations does not
panic
**Refs:** crc-ParserState.md, crc-Loc.md
**Code:** sdom/protocol_test.go

## Test: pending text is flushed before an emitted node
**Purpose:** R156 — the array is in document order without anyone ordering it
**Input:** text, then a marker, then text
**Expected:** three nodes in that order, and the text node's span ends exactly
where the marker's begins
**Refs:** crc-ParserState.md, seq-collaborate.md#1.6
**Code:** sdom/protocol_test.go
**Alarm:** 3
**Fire alarm:** In `ParserState.Emit`, append the node before flushing the pending text rather than after. Red: the array is out of document order — the marker precedes the text that came before it — so the spans no longer tile even though every byte is still present exactly once. The corpus round trip is blind to it, since Render concatenates whatever order it is given.
**Inject:** sdom/parser.go:ParserState.Emit

## Test: NodeType distinguishes no-node from an unlabelled node
**Purpose:** R160 — two different claims, not one string
**Input:** a position where a group with no `Kind` opens, and a position where
nothing matches
**Expected:** the first reports a node with an empty kind; the second reports no
node. `Parse` is not called and nothing moves in either case
**Refs:** crc-Parser.md, seq-collaborate.md#3
**Code:** sdom/protocol_test.go

## Test: a delegate taking the loop hides positions from the outer parser
**Purpose:** R166 — suppression is structural, not a depth check
**Input:** a recording indent-like parser over a source containing a bracket group
**Expected:** it is offered no position between the opener and the closer
**Refs:** crc-Parser.md, seq-collaborate.md#2.4
**Code:** sdom/protocol_test.go
