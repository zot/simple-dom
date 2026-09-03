# Test Design: TraceabilityComment
**Source:** crc-TraceabilityComment.md

## Test: the fuzzed DOM round-trip
**Purpose:** R210, R211, R212, R214, R215, R218 — the whole grammar, and under-modelling caught where a byte round-trip cannot see it
**Input:** Go native fuzzing over a seeded corpus: every field order, `//CRC:x|R7` with no spaces, wide spacing, all three description separators, ranged refs, `Seq` steps, and Lua's `-- CRC: x | R4 -- desc`; each seed wrapped in each shipped language's `Comment`
**Expected:** for every input that `Comments` recognizes, `render(doc) == src` and `Comments(parse(render)).Equals(node)`; the node's children are the original opener and closer by identity; the field accessors read the expected items
**Refs:** crc-TraceabilityComment.md, seq-anchor.md#2
**Code:** minispecsdom/comment_test.go
**Fire alarm:** make the segment walk keep the whole interior as one glue `Text` and set no fields. Bytes round-trip perfectly; the DOM compare goes red because the reference carries typed fields and the accessors read nil.
**Inject:** minispecsdom/comment.go:TraceabilityComment.Parse

## Test: recognition is consumption
**Purpose:** R215, R216, R219 — leading with a keyword is not enough, and nothing else is touched
**Input:** a Go file with `// CRC: crc-A.md | R1`, `// Test: a repaint frame round-trips (R3136).`, `// see R5`, `// (R5)`, and `x := 1 // R7`
**Expected:** `Comments` returns two nodes — the first and the trailing `// R7`; the other three remain ordinary groups; `Parse` on the `Test:` comment returns false and the document is unchanged by the attempt
**Refs:** crc-TraceabilityComment.md, seq-anchor.md#1
**Code:** minispecsdom/comment_test.go
**Fire alarm:** return true whenever a segment matched, ignoring the remainder — the `Test:` prose comment is recognized and the count goes to three.
**Inject:** minispecsdom/comment.go:TraceabilityComment.Parse

## Test: the splice reuses the markers and the context survives it
**Purpose:** R217, R219
**Input:** `func f() {}\n// CRC: crc-A.md | R1\nfunc g() {}\n` under Go; take the comment opener and closer before the pass
**Expected:** after `Comments`, the node's first and last children are those very pointers; the flat array holds one node where three were; the context still pairs the two `{ }` groups around it, since the splice bumped the generation and the next read rebuilt cleanly
**Refs:** crc-TraceabilityComment.md, seq-anchor.md#1.2.2
**Code:** minispecsdom/comment_test.go
**Fire alarm:** build fresh `NewOpener` / `NewCloser` from the originals' text — `Equals` still holds and the identity check fails.
**Inject:** minispecsdom/comment.go:TraceabilityComment.Parse

## Test: New parses back Equals, in every language
**Purpose:** R208, R220, R221 — construction agrees with reading, and the comment style's kind holds per language
**Input:** `New(lang, Fields{CRC: [crc-A.md], Seq: [seq-b.md#1.2], Refs: [4 5 6 9], Description: "note"})` for every shipped `BracketLang` with a non-empty `Comment.Prefix`
**Expected:** the render is `<Prefix>CRC: crc-A.md | Seq: seq-b.md#1.2 | R4-6, R9 -- note<Suffix>`; parsing that source with the language and running `Comments` yields one node `Equals` to the constructed one; the constructed node's location has no origin
**Refs:** crc-TraceabilityComment.md, crc-BracketLang.md
**Code:** minispecsdom/comment_test.go
**Fire alarm:** write refs before `Seq` in `New` — the render changes, and the parse-back still `Equals`, which is why the render is asserted literally.
**Inject:** minispecsdom/comment.go:New

## Test: a field write through the node
**Purpose:** R201, R203, R214 — the write reaches one literal and the descsep is preserved
**Input:** parse `// R5: desc` and `// CRC: a.md — note`; `Refs().SetItems([5 6 7])` on the first
**Expected:** the first renders `// R5-7: desc` — the `:` separator untouched; the second's `Description` reads ` note` and renders back with the em dash
**Refs:** crc-TraceabilityComment.md, seq-anchor.md#3
**Code:** minispecsdom/comment_test.go
**Fire alarm:** normalise the separator to `--` on read — the second renders `-- note`, the byte round-trip fails.
**Inject:** minispecsdom/comment.go:TraceabilityComment.Parse
