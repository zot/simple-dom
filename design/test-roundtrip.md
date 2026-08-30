# Test Design: the standing properties
**Source:** crc-Doc.md, crc-Node.md

Written once and inherited by every node kind, so a new kind is tested **by
existing**. Tests 6 (the recognition count) and 7 (the additive property) from the
carve need a lexicon and readers, and land with Items 2 and 6.

## Test: byte round-trip over the real corpus
**Purpose:** R2 — bytes the parse does not model come back unchanged
**Input:** the project's own files, this package's source included — not fixtures,
since what a lossy parse eats is precisely what nobody thought to put in one. Each
parsed, then split at many arbitrary interior offsets to give the property teeth
while `Text` is the only leaf
**Expected:** the rendered document is byte-identical to the source, before and
after the splitting
**Refs:** crc-Doc.md
**Code:** sdom/roundtrip_test.go

## Test: the array tiles the document
**Purpose:** R3 — contiguous, half-open, first at 0, last at the end
**Input:** every corpus document, before and after arbitrary splits and merges
**Expected:** each node's span begins where the previous ended, with no gap and no
overlap; the first begins at 0 and the last ends at the length of the source
**Refs:** crc-Doc.md
**Code:** sdom/roundtrip_test.go

## Test: every faithful node renders its own span
**Purpose:** R25 — across the whole corpus rather than one constructed case
**Input:** every node of every corpus document that reports `Faithful()`
**Expected:** the node's render equals the source bytes at its location
**Refs:** crc-Loc.md
**Code:** sdom/roundtrip_test.go

## Test: structural round-trip at a different base
**Purpose:** R10 — passing also proves `Equals` ignores provenance
**Input:** a mutated tree, and the same structure rebuilt over a document with a
**different base offset**.
**Note:** in this item the comparison tree is *reconstructed* by the same
construction, not re-parsed, because `sdom` has no lexer yet — nothing recovers a
`Compound` from bytes until Item 2. The re-parse form of this test lands there
**Expected:** the two compare equal despite every offset differing
**Refs:** crc-Node.md, crc-Doc.md
**Code:** sdom/roundtrip_test.go

## Test: the one-field delta
**Purpose:** R28 — exactly the edited node and its ancestors lose faithfulness
**Input:** a document containing nested compounds; one leaf's content rewritten
**Expected:** that leaf and every compound above it report `Altered()`; **every
untouched sibling and its subtree still reports `Faithful()`**
**Refs:** crc-Compound.md, crc-Loc.md
**Code:** sdom/roundtrip_test.go

## Test: a deleted node is visible in the output
**Purpose:** the alarm the corpus test structurally cannot reach — a `Render` that
hands back retained source leaves the byte round-trip green over every document,
no matter how little was modelled
**Input:** a document with one node removed from the array
**Expected:** the rendered output **lacks** that node's bytes, and equals the
source with exactly that span excised
**Refs:** crc-Doc.md, crc-MutationWindow.md
**Code:** sdom/roundtrip_test.go
**Alarm:** 1
**Fire alarm:** Make `Doc.Render` return `d.source` instead of concatenating the nodes' renders. Red: **only this test.** The byte round-trip over the entire corpus stays green, the tiling test stays green, and every faithful-span check stays green — a Render that hands back retained source satisfies all of them no matter how little was modelled. This is the alarm that decides the shape of the suite.
**Inject:** sdom/doc.go:Doc.Render
