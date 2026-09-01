# Test Design: the standing properties
**Source:** crc-Doc.md, crc-Node.md

Written once and inherited by every node kind, so a new kind is tested **by
existing**. Tests 6 (the recognition count) and 7 (the additive property) from the
carve need a schema and readers, and land with Items 2 and 6.

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

## Test: Equals ignores the parse a node came from
**Purpose:** R10, R89 — `Equals` never consults `Location`, and that now includes
the origin
**Input:** the same nested structure built twice with **different origins** and
identical offsets; then one leaf edited on each side in turn
**Expected:** equal, then unequal after one edit, then equal again after the same
edit on both
**Note:** this replaces a test that varied the document's *base*. That proved
nothing — a base never enters a location, so both trees had identical offsets and
the fixture had been offsetting them by base **by hand** to fake a difference the
real thing does not have. Found by an alarm that failed to ring; see gap O9.
**Refs:** crc-Node.md, crc-Loc.md
**Code:** sdom/roundtrip_test.go

## Test: structural round-trip through a re-parse
**Purpose:** R10, R57 — the real form of the property. Passing proves both that
the parse is stable under its own output and that `Equals` ignores provenance
**Input:** a Go source parsed, one text node's content rewritten inside a
mutation, the document rendered, and that output **re-parsed at a different
base**
**Expected:** the two node arrays compare equal, node for node, despite every
offset differing and the edited node being altered on one side and freshly
faithful on the other
**Refs:** crc-Node.md, crc-BracketParser.md
**Code:** sdom/roundtrip_test.go
**Alarm:** 2
**Fire alarm:** Make `Opener.Equals` compare locations as well as bytes — add
`&& o.Location() == x.Location()` to its return. Red: only this test, because it
is the only one comparing nodes across two documents with different bases.
`TestEqualsIgnoresProvenance` stays green, since it exercises `*Text` rather than
a marker kind — which is exactly the gap a per-kind `Equals` discipline opens.
**Inject:** sdom/marker.go:Opener.Equals
**Pulled:** 2026-08-30 — re-verified after this test was **accidentally deleted
and restored**. An index-to-index text replacement while rewriting the `nest`
fixture swallowed the whole function, and the count hid it: seven tests added and
one destroyed came to exactly the total expected without the deletion. Nothing in
`validate` noticed, because artifact checkboxes are per FILE — the file still
existed. Caught by a reviewer reading the design against the code. Rings alone
again on the restored test.
**Pulled:** 2026-08-30 — rang on the second attempt, and the first attempt is the
record worth keeping. As first written this test re-parsed **at a different
`base`** and the injection left all 57 tests green: node offsets are relative to
the document's own source, so a different base changes no location at all, and the
only nodes whose offsets differed were after the edit — none of them openers. The
test was rewritten to shift offsets with a prefix instead, and now fails alone,
with `node 1 differs after a re-parse: "(" vs "("` — identical bytes, different
provenance. Tracked as a gap, because two other documents assume base flows into
offsets.

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
**Pulled:** 2026-08-30 — rang. **Only this test failed, out of 30.**
The byte round-trip over the whole corpus, the tiling test, every faithful-span
check, the structural round-trip and the one-field delta all stayed green against
a `Render` that models nothing at all. This is the measurement the carve
predicted, and it is why this test cannot be folded into the corpus test.
