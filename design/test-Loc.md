# Test Design: Loc
**Source:** crc-Loc.md

## Test: the zero value has no provenance
**Purpose:** R24, R26 — absence is the zero value, not offset 0
**Input:** a `Loc` nobody set, and a `Loc` deliberately at offset 0
**Expected:** the first reports `Offset() == -1` and is not faithful; the second
reports `Offset() == 0`. The two are distinguishable
**Refs:** crc-Loc.md
**Code:** sdom/loc_test.go

## Test: a faithful node renders its own source span
**Purpose:** R25 — the property the whole faithfulness distinction exists to name
**Input:** a document, every node in it that reports `Faithful()`
**Expected:** each node's render equals `source[Offset : Offset+Length]`
**Refs:** crc-Loc.md, crc-Doc.md
**Code:** sdom/loc_test.go

## Test: an altered node keeps its offset
**Purpose:** R27, R29 — provenance survives the edit that costs faithfulness
**Input:** a faithful node at a known offset, then a content write
**Expected:** `Altered()` true, `Faithful()` false, `Offset()` unchanged, `Length()`
reporting the **new** extent
**Refs:** crc-Loc.md
**Code:** sdom/loc_test.go

## Test: merged faithfulness, over the whole table
**Purpose:** R34 — "unfaithful" covers altered *and* no-provenance
**Input:** every pairing of {faithful, altered, no provenance} as `a` and `b`
**Expected:** faithful only where both operands are faithful; the seven other
combinations all unfaithful
**Refs:** crc-Loc.md
**Code:** sdom/loc_test.go
**Alarm:** 1
**Fire alarm:** Key `mergeLocs` on `Altered()` rather than on `Faithful()`. Red: merging a faithful node with a **synthesized** one yields a location claiming faithfulness at an offset whose bytes it does not render — the two no-provenance cells of the nine flip. This is the injection worth having, because the altered cases keep passing and only the no-provenance ones fail.
**Inject:** sdom/loc.go:mergeLocs
**Pulled:** 2026-08-30 — rang. Only this test failed, at `(faithful, none)` and
`(none, faithful)`; the other seven cells and the other 29 tests held.

## Test: merged offset takes the leftmost provenance
**Purpose:** R35
**Input:** `(a with provenance, b with)`, `(a with, b without)`, `(a without, b
with)`, `(a without, b without)`
**Expected:** `a`'s offset, `a`'s offset, `b`'s offset, and −1
**Refs:** crc-Loc.md
**Code:** sdom/loc_test.go

## Test: merging a run is associative
**Purpose:** R35 — the property that makes the offset rule safe to fold
**Input:** three nodes with provenance present or absent in each of the eight
patterns; merged left-to-right and right-to-left
**Expected:** the two groupings agree on offset and on faithfulness, in all eight
**Refs:** crc-Loc.md
**Code:** sdom/loc_test.go
**Alarm:** 2
**Fire alarm:** In `mergeLocs`, when **both** operands have provenance take the
offset of the one with the greater length, ties to `a`; keep taking the only
available offset when just one has provenance. Red: this test, because the
groupings compare different lengths — `merge(merge(a,b),c)` weighs an 8-byte left
against a 4-byte right, while `merge(a,merge(b,c))` weighs 4 against 8. Violating
associativity is otherwise silent: every individual merge still returns a
plausible offset, and only comparing two groupings can see it.
*Superseded 2026-08-30.* The first injection here — prefer `b`'s offset — did not
ring: rightmost-provenance-wins is associative for exactly the reason
leftmost-wins is, so it broke the leftmost rule (caught elsewhere) and left this
property untested. An injection has to be non-associative to reach it.
**Inject:** sdom/loc.go:mergeLocs
**Pulled:** 2026-08-30 — rang, on patterns 011, 101 and 111. Crucially
`TestMergedOffsetTakesLeftmostProvenance` **passed**, which is what proves the
injection reached associativity itself rather than the leftmost rule beneath it —
the discriminator the superseded injection lacked.

## Test: adjacency is checked exactly where it can be
**Purpose:** R33 — checked for a faithful pair at the call site, not checked otherwise
**Input:** a non-adjacent pair of faithful nodes; then the same pair with one altered
**Expected:** the first is refused at the call; the second is not refused, because
an altered node's offset is historical and proves nothing
**Refs:** crc-Loc.md, crc-MutationWindow.md, seq-mutate.md#2.2
**Code:** sdom/loc_test.go
**Alarm:** 3
**Fire alarm:** Drop the `la.Faithful() && lb.Faithful()` guard so `Merge` checks adjacency unconditionally. Red: the altered-operand case is refused, because an altered node's historical offset proves nothing. The happy path never exercises this, so nothing else objects.
**Inject:** sdom/mutate.go:Doc.Merge
**Pulled:** 2026-08-30 — rang. Only this test failed, refusing the altered pair
with `Merge: 0+2 is not adjacent to 90`; the other 29 held.

## Test: split then merge is the identity
**Purpose:** R30, R36 — re-granulation moves boundaries and nothing else
**Input:** a faithful node, split at each interior offset in turn, then merged back
**Expected:** the original span, offset and faithfulness recovered every time
**Refs:** crc-Loc.md
**Code:** sdom/loc_test.go
