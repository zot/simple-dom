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
**Fire alarm:** In `ParserState.run`, delete the node-count term **and its variable** — keep `pos := st.pos`, drop `n`, and test `if st.pos == pos`. Both edits: removing only the term leaves `n` declared and unused, which is a **compile error rather than a red test**, and a build failure proves nothing while reading the same in a terminal. Red: the emission is read as a non-match, so the walk advances over it and the offered positions become `[0, 1]` instead of `[0, 0, 1]` — and the byte it swallows is text here but a bracket opener after a real dedent to column 0. Every byte is still emitted somewhere, so the round trip and the tiling stay green; only a recognition count sees it.
**Inject:** sdom/parser.go:ParserState.run
**Pulled:** 2026-09-03 — re-pulled after the live text run replaced the flush and rang: `offered [0 1], want [0 0 1]`, plus the dedent-to-column-zero opener test; three failures in all. Previously 2026-09-01 — rang, `offered [0 1], want [0 0 1]`, exactly as predicted — **and the consequence the prose describes arrived with it, three levels deep.** The walk swallowed a `"` after a dedent to column 0, so `TestADedentToColumnZeroBeforeABracketOpenerKeepsTheOpener` saw `*sdom.Text("\"a string statement")` where an `Opener` belonged; the string group therefore never opened; its interior became top-level text; and in the other package `TestADefInsideADocstringIsNotADeclaration` returned `def[notreal] def[real]` — **a `def` inside a docstring recognized as a real declaration.** The round trip and the tiling stayed green the whole way, which is the point: one dropped term in a loop condition reaches from a byte offset to a wrong answer about someone's source code, and nothing that counts bytes can see it.

*First attempt did not build:* dropping the term alone leaves `n` declared and unused, which is a compile error rather than a red test. The prescription now names both edits.

## Test: a parser that advances without emitting is not read as a match
**Purpose:** R164 — the half of the rule that the node count alone misses
**Input:** a parser that advances two bytes from position 1 — inside a live run, where
extending it changes no node count — and emits nothing
**Expected:** the walk does not also take a byte, so exactly those two bytes land
in the live text run and the position is where the parser left it; offered
positions are `0, 1, 3`
**Refs:** crc-ParserState.md, seq-collaborate.md#1.4
**Code:** sdom/protocol_test.go
**Alarm:** 2
**Fire alarm:** In `ParserState.run`, delete the position term **and its variable** — keep `n := len(st.out)`, drop `pos`, and test `if len(st.out) == n`. Both edits, for the reason above: dropping the term alone will not compile. Red: a parser that advanced two bytes is read as having done nothing, so the walk takes a third — offered positions become `[0, 1]` rather than `[0, 1, 3]` — the extra advance runs off the end. (Since 2026-09-03 the fixture advances from position 1: at 0 `Advance` creates the live run and the count moves, which would let a count-only walk pass for the wrong reason.) The skipped byte can be a marker, and nothing but a recognition count would notice.
**Inject:** sdom/parser.go:ParserState.run
**Pulled:** 2026-09-03 — re-pulled after the live text run replaced the flush and rang: `offered [0 1], want [0 1 3]` — the walk's extra advance ran off the end of the four-byte source, so the prose's `[0 1 4]` was one position optimistic; alone, as before. Previously 2026-09-01 — rang, `offered [0 3], want [0 2 3]`, and **alone** — the other package stayed green and so did every other test in this one. The narrower of the two halves, and the one nothing else in the suite guards.

*First attempt did not build*, for the mirror of the reason above: dropping the position term alone leaves `pos` declared and unused.

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

## Test: the last node is the live text
**Purpose:** R224, R225, R156 — the array is the only parse state, and its last node is current at every offer; a lookahead consumes nothing
**Input:** `ab|cd` with a recording parser that notes `Last()` at every offer and emits a marker at 2; and a parser that peeks two bytes ahead with `SetPos` and restores
**Expected:** at 0 `Last` is nil; at 1 and 2 it is the text `a` then `ab`, ending exactly at the position; at 3 it is the marker; at 4 the text `c`; the final array is `ab`, `|`, `cd`; the peeking parser leaves the text run unchanged
**Refs:** crc-ParserState.md, seq-collaborate.md#1.4
**Code:** sdom/protocol_test.go
**Alarm:** 3
**Fire alarm:** In `ParserState.Advance`, create the run on the first byte but never extend it — delete the extension branch after the `return`. Red: `Last` at 2 reads `T:a` rather than `T:ab`, the run's end lags the position, and because the run is the array, the round trips lose every byte after the first of each run. The property this test alone guards is the *during*-the-parse one: a run that lagged by a byte and was patched up at `Emit` would pass every round trip and fail only here. *(The alarm this replaces injected a flush order into `Emit`; there is no flush to reorder now, and the property it pinned — document order — is carried by every round trip.)*
**Inject:** sdom/parser.go:ParserState.Advance
**Pulled:** 2026-09-03 — rang as written: `Last at each offer: ["<nil>" "T:a" "T:a!lagging" "M:|" "T:c"]`, plus 32 more as the round trips lost bytes. Pulled by hand on the uncommitted tree with a targeted reverse edit; a first attempt restored with `git checkout` and reverted the item's whole file — recorded so nobody repeats it.

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
