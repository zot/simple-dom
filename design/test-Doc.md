# Test Design: Doc and the mutation window
**Source:** crc-Doc.md, crc-MutationWindow.md

## Test: navigation is by node, in document order
**Purpose:** R41 — and that the ends of the document are answerable
**Input:** a document of several nodes
**Expected:** `Next` walks them in order and returns nothing past the last; `Prev`
walks back and returns nothing before the first
**Refs:** crc-Doc.md
**Code:** sdom/doc_test.go

## Test: the generation bumps on membership and not on content
**Purpose:** R43, R44 — the distinction the whole stamping protocol rests on
**Input:** a document; one mutation that splits a node, one that only rewrites a
node's bytes
**Expected:** the generation differs after the first and is unchanged after the
second
**Refs:** crc-Doc.md, seq-stamp.md#1.2
**Code:** sdom/doc_test.go
**Alarm:** 1
**Fire alarm:** Bump unconditionally in `Doc.rebuild`, dropping the `d.dirty` test. Red: the content-edit half of this test. Over-bumping is the invisible failure — every layer simply rebuilds more often, the answers stay correct, and only a test that asserts the *absence* of a bump can see it.
**Inject:** sdom/doc.go:Doc.rebuild
**Pulled:** 2026-08-30 — rang. Only this test failed. `TestStaleStampDrivesTheRebuild`
held, because two refreshes with no mutation between them still read the same
generation — over-bumping is invisible to everything except an assertion that a
bump did *not* happen.

## Test: a stale stamp is what a layer sees, with no registration
**Purpose:** R45, R46 — the layer pulls; `Doc` pushes nothing
**Input:** a fake layer holding a stamp and a rebuild counter; a membership change
between two freshness checks
**Expected:** fresh on the first check, stale on the second, exactly one rebuild,
and `Doc` holding no reference to the layer at any point
**Refs:** crc-Doc.md, seq-stamp.md
**Code:** sdom/doc_test.go

## Test: navigation inside the window refuses
**Purpose:** R49, R50 — every read that could observe a half-edited document
**Input:** a mutation whose function calls, in turn, `Prev`, `Next`, the position
lookup, the line lookup, and the generation read
**Expected:** each refuses; `Mutate` returns an ordinary error for each, and the
process does not die
**Refs:** crc-MutationWindow.md, seq-mutate.md#1.4.3
**Code:** sdom/doc_test.go
**Alarm:** 2
**Fire alarm:** Make `Doc.guard` return without panicking. Red: all six reads return stale answers instead of refusing. Violated, this is silent by construction — a stale index answers plausibly, and a removed node's -1 is indistinguishable from end-of-document.
**Inject:** sdom/mutate.go:Doc.guard
**Pulled:** 2026-08-30 — rang. Only this test failed, and it failed six times —
once for each of `IndexOf`, `Next`, `Prev`, `Line`, `LineCount` and `Generation`.
The other 29 held.

## Test: a foreign panic is re-raised unchanged
**Purpose:** R51 — the guard must not swallow real bugs
**Input:** a mutation function that panics with an unrelated value
**Expected:** the panic escapes `Mutate` with its value and its stack intact,
rather than arriving as an error
**Refs:** crc-MutationWindow.md, seq-mutate.md#3.2.2
**Code:** sdom/doc_test.go
**Alarm:** 3
**Fire alarm:** In `Doc.Mutate`, convert every recovered value to an error instead of re-panicking on anything that is not the sentinel. Red: the panic never reaches the test's deferred recover. Swallowing panics reads as robustness and breaks nothing visible, which is why it needs its own alarm.
**Inject:** sdom/mutate.go:Doc.Mutate
**Pulled:** 2026-08-30 — rang. Only this test failed;
`TestEscapingFailurePoisons` held, so the two `Doc.Mutate` alarms are
independent rather than one property counted twice.

## Test: a nested Mutate is a pass-through
**Purpose:** R53 — save-and-restore rather than counting
**Input:** `Mutate` called from inside a mutation function, editing in both levels
**Expected:** both edits land, the inner call opens no second window, and the
window is closed exactly once — after the outer call returns
**Refs:** crc-MutationWindow.md
**Code:** sdom/doc_test.go

## Test: a nested Mutate keeps the outer window open
**Purpose:** R53 — the window belongs to the *outermost* call, which is the half
of the pass-through that node counts and generation bumps cannot see
**Input:** an outer mutation that runs an inner `Mutate` to completion and then,
still inside the outer function, reads the structural generation
**Expected:** the read still refuses, so `Mutate` returns the sentinel as an error
**Refs:** crc-MutationWindow.md
**Code:** sdom/doc_test.go
**Alarm:** 6
**Fire alarm:** Delete the `if d.mutating { return f() }` early return from
`Doc.Mutate`, so a nested call opens and closes its own window. Red: this test,
because the inner close sets `d.mutating = false` while the outer function is
still running and the guard lifts for the rest of it.
*Found by probing past the alarm list, 2026-08-30.* Before this test existed that
same injection left **all 30 tests green**, `TestNestedMutateIsAPassThrough`
included — it asserts five nodes and one generation bump, and both still hold: the
inner close rebuilds and bumps, then the outer close finds `d.dirty` already
cleared and does not bump again. The arithmetic survives while the guard silently
lifts, which is why the property needed an assertion of its own.
**Inject:** sdom/mutate.go:Doc.Mutate
**Pulled:** 2026-08-30 — rang. Only this test failed, on `got <nil>` where the
sentinel error was required; the other 30 held.

## Test: merging faithful nodes shares the source's storage
**Purpose:** R120 — the observable half of the fast path. Two strings with equal
bytes are `==` whether one was sliced or freshly built, so only a check of
*storage* can tell which path ran
**Input:** two adjacent faithful nodes merged; then the same pair with one altered
**Expected:** the faithful merge's text points into the document's source; the
altered one's does not, because those bytes are not in the source at all. Both
render the same
**Refs:** crc-MutationWindow.md, crc-Doc.md
**Code:** sdom/alloc_test.go
**Alarm:** 7
**Fire alarm:** Make `Merge` always concatenate, dropping the `bothFaithful`
branch. Red: only this test. Every byte is identical and the render is unchanged —
what is lost is that the result stopped sharing the source's storage, which no
comparison of values can see.
*Note what this alarm does NOT cover.* The original implementation built the
concatenation and then discarded it in the faithful case; this test stayed green
through that, because the discarded string still left a correct slice behind. That
half of the claim is gap O12, and it is not asserted here.
**Inject:** sdom/mutate.go:Doc.Merge
**Pulled:** 2026-08-31 — rang, and **alone out of 79**. The puller checked the
value-comparing tests exhaustively and every one stayed green: both corpus
round-trips, the array tiling, the stencil child tiling, every faithful-span
check, the structural round-trip through a re-parse, split-then-merge, merge
associativity, and the todo-list round-trip. Not one byte changes under this
injection, so nothing that compares bytes can see it. This is the sharpest
measurement in the project of what a round-trip does not prove.

## Test: an escaping failure poisons the document
**Purpose:** R54, R55 — no rollback, and no reset
**Input:** a mutation function that applies one edit and then returns an error;
separately, one that applies an edit and then panics
**Expected:** in both cases the edit is **still applied** — nothing is rolled back —
and the document refuses further use rather than offering recovery
**Refs:** crc-MutationWindow.md, seq-mutate.md#3.3
**Code:** sdom/doc_test.go
**Alarm:** 4
**Fire alarm:** Remove the `d.poisoned = true` assignments from `Doc.Mutate`'s deferred close. Red: the document keeps answering after a failed mutation. Nothing else in the suite touches a failed document, so the whole guard can vanish while the suite stays green.
**Inject:** sdom/mutate.go:Doc.Mutate
**Pulled:** 2026-08-30 — rang. Only this test failed, four times —
both the error path and the panic path, each failing on Render and on the
follow-up Mutate.

## Test: Line rejects offsets outside the document
**Purpose:** R39 — the contract says 0 for an offset outside the rendered
document, and that has to hold at the top end as well as the bottom
**Input:** offsets of -1, exactly the rendered length, and beyond it; then the
last valid byte; then offset 0 of an empty document
**Expected:** 0 for every out-of-range offset, a real line for the last byte, and
0 for the empty document
**Refs:** crc-Doc.md
**Code:** sdom/doc_test.go
**Alarm:** 5
**Fire alarm:** Drop the `offset >= d.renderedLen` bound from `Doc.Line`. Red:
every past-the-end offset returns `LineCount()` instead of 0, because
`sort.Search` falls off the end and reports the length. This is how the defect
originally shipped — the surrounding test only probed `[0, len(rendered))`, so a
plausible wrong answer at the boundary broke nothing visible.
**Inject:** sdom/doc.go:Doc.Line
**Pulled:** 2026-08-30 — rang. Only this test failed, on all three
past-the-end offsets and on the empty document, which returned line 1 for offset 0.

## Test: the indices are correct after the window closes
**Purpose:** R52 — one rebuild at the exit, and it is a *correct* rebuild
**Input:** a mutation that splits, merges and rewrites several nodes
**Expected:** afterwards, every node's position lookup and every line lookup agrees
with a freshly computed index over the same array
**Refs:** crc-Doc.md, crc-MutationWindow.md, seq-mutate.md#1.6
**Code:** sdom/doc_test.go

## Test: Insert places before a node or at the end, and bumps the generation
**Purpose:** R257, R43
**Input:** a parsed `ab|cd`; `Insert` a synthetic `X` before the marker, then a synthetic `Y` with `before` nil, each in a window; then `Insert` before a node not in the document
**Expected:** render `abX|cdY`; the generation advanced twice; the third call errors and, having escaped the window, poisons the document
**Refs:** crc-MutationWindow.md
**Code:** sdom/doc_test.go
**Alarm:** 8
**Fire alarm:** make `Insert` place *after* the named node. Red: `ab|Xcd…`.
**Inject:** sdom/mutate.go:Doc.Insert
