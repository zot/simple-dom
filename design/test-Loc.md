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
**Pulled:** 2026-08-31 — re-pulled and rang again. Its earlier proof expired when
`Doc.Merge` grew the source-slicing branch; re-earned against the function as it
now stands, failing alone with the same refusal. Previously 2026-08-30, when it
also rang alone against 29 other tests.

## Test: split then merge is the identity
**Purpose:** R30, R36 — re-granulation moves boundaries and nothing else
**Input:** a faithful node, split at each interior offset in turn, then merged back
**Expected:** the original span, offset and faithfulness recovered every time
**Refs:** crc-Loc.md
**Code:** sdom/loc_test.go

## Test: offsets are relative to the document's own source
**Purpose:** R88 — a document's base is metadata about its source, not part of any
location. This is what gap O9 was about
**Input:** a document parsed with a base of 500
**Expected:** its nodes still begin at offset 0 and tile to the length of the
source, and each renders exactly the source at its own offset with no adjustment
**Refs:** crc-Loc.md, crc-Doc.md
**Code:** sdom/loc_test.go
**Alarm:** 4
**Fire alarm:** In `Parse`, thread the base into the parser and add it in `at`, so
locations come out base-relative. Red: **only this test** — measured twice, on
2026-08-30 and 2026-08-31. Every offset in the document is wrong by the base and
nothing else objects, because `Render` never consults an offset. ~~and the tiling
assertions with it, loud rather than silent~~ — that prediction was wrong both
times it was checked, and the tiling tests parse at base 0, where the injection
changes nothing at all.
**Inject:** sdom/parser.go:Parse, sdom/parser.go:ParserState.At
**Pulled:** 2026-09-01 — re-pulled after the vocabulary pass and rang a **third** time, still **only this test**, out of a larger suite. The anchor resolved under the new name `Parse` — the one thing a rename can genuinely break, and the reason this pull was worth taking when the property could not have moved. Previously 2026-08-31 — re-pulled after the parser rename and rang again, and again **only this test failed** — the second measurement agreeing with the first, and with neither agreeing with the prescription above. The rename moved no property; only symbols changed name. Originally 2026-08-30 — rang, and less loudly than predicted. **Only this test
failed**, out of 64. Every offset in the document was wrong by 500 and the corpus
round-trip, the tiling test and the faithful-span check all stayed green, because
`Render` never consults an offset. The prediction of "loud" was wrong.

## Test: an origin is set by chaining
**Purpose:** R89, R92 — and that `In` does not mutate the location it is chained
onto, which a value receiver gives for free and a pointer receiver would lose
**Input:** a location built without an origin, then chained into one
**Expected:** the original still reports no origin; the result reports the given
one and differs in nothing else
**Refs:** crc-Loc.md
**Code:** sdom/loc_test.go

## Test: distinct origins are distinct
**Purpose:** R90 — the field is what makes `Origin` usable as an identity
**Input:** two separately minted `&Origin{}` values
**Expected:** they are not the same identity, and locations attributed to them are
distinguishable
**Alarm:** 5
**Fire alarm:** Make `Origin` an empty struct — `type Origin struct{}` — dropping
`Name`. Red: this test, **if the hazard is real.** Go *may* give every zero-size
allocation the same address; whether it does here is the thing being measured. A
green result would mean the field is justified by `Name` being useful rather than
by the address hazard, and the reasoning in the decision should be narrowed to say
so.
**Inject:** sdom/origin.go:Origin
**Pulled:** 2026-08-30 — **the hazard is real and this test could not see it.**
The empty-struct injection reddened four tests, so zero-size `Origin`s genuinely
collide — but *this* test, the one whose comment names the hazard, stayed GREEN.
Escape analysis is why: `a, b := &Origin{}, &Origin{}` as locals do not escape, so
the compiler gives them distinct stack slots and they compare distinct with or
without a field. Production mints through `newBracketContext`, which escapes to
the heap, where every zero-size allocation lands on `runtime.zerobase` and compares
equal — measured directly at `0x726840` for all of them. The test now mints through
a context so it observes the production path, and re-pulling against the corrected
version reddens five tests including this one.
**Code:** sdom/loc_test.go

## Test: one origin per parse, carried by every node
**Purpose:** R91 — the question the whole item exists to answer: same file,
different parser
**Input:** the same source parsed twice with the same language
**Expected:** two distinct origins; every node of each parse carries its own; and
nodes from the two are distinguishable despite identical bytes and offsets
**Refs:** crc-BracketContext.md
**Code:** sdom/loc_test.go
**Alarm:** 6
**Fire alarm:** Make `ParserState.At` return `Source(pos, length)` without `.In(...)`.
Red: this test and `TestOneParseHasOneOrigin`, which asks the same question from the
protocol side. Everything else works exactly as well without origins — the bytes, the
tiling, the pairing, the round-trips are all untouched — which is why attribution
needs an assertion of its own.
**Inject:** sdom/parser.go:ParserState.At
**Pulled:** 2026-09-01 — re-pulled after `at` moved onto `ParserState` and rang again, now on **two** tests: this one and `TestOneParseHasOneOrigin`, which did not exist at the last pull. The alarm outgrew its own prediction of *alone*, in the safe direction — the protocol test asks the same question from the other side, so one parser dropping the origin is now caught twice. Everything else stayed green exactly as claimed. Previously 2026-08-31 — re-pulled after the parser rename and rang again, and again alone. The rename moved no property; only symbols changed name. Originally 2026-08-30 — rang, and **alone**: 63 of 64 tests passed with the
parser attributing nothing at all. The bytes, the tiling, the bracket pairing, the
cross-derivation check and every corpus round-trip are all indifferent to
provenance, which is why attribution needs an assertion of its own.

## Test: a nil origin is absent, not different
**Purpose:** R93 — a synthesized node has no origin, and merging it into parsed
material is legitimate
**Input:** a parsed location merged with a synthesized one, in both orders; and
two synthesized ones
**Expected:** the known origin survives from either side; two unknowns merge to
unknown
**Refs:** crc-Loc.md
**Code:** sdom/loc_test.go
**Alarm:** 7
**Fire alarm:** In `mergeLocs`, drop the fallback that takes `b`'s origin when
`a`'s is nil, so the result keeps `a`'s nil. Red: this test. Silent elsewhere —
the merged node simply loses its attribution and nothing else looks.
**Inject:** sdom/loc.go:mergeLocs
**Pulled:** 2026-08-30 — rang, on exactly one row of its table: *synthesized on
the left*. The other two rows are unaffected by the injection and correctly stayed
quiet, which is the table earning its keep — a single combined assertion would have
said less.

## Test: merging across origins panics
**Purpose:** R94, R95 — a programming error, not a data condition; and it must
escape `Mutate` with its stack rather than arriving as an ordinary error
**Input:** two locations with different origins merged directly; then two such
nodes merged inside a mutation window
**Expected:** a panic naming the two parses; and a panic that escapes `Mutate`
rather than being converted
**Refs:** crc-Loc.md, crc-MutationWindow.md
**Code:** sdom/loc_test.go
**Alarm:** 8
**Fire alarm:** Remove the cross-origin check from `mergeLocs` entirely. Red: both
of these. Silent everywhere else — the merge produces a perfectly plausible
location that names a position in one parse for bytes drawn from two, and the
round-trip is blind to it because no byte moves.
**Inject:** sdom/loc.go:mergeLocs
**Pulled:** 2026-08-30 — rang, on both of its tests and nothing else. 62 of 64
passed while a cross-parse merge silently produced a location naming a position in
one parse for bytes drawn from two. No byte moves, so the round-trip is blind to it.
