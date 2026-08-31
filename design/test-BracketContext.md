# Test Design: BracketContext
**Source:** crc-BracketContext.md

## Test: pairing is recorded both ways
**Purpose:** R82, R83
**Input:** `a {b [c] d} e`
**Expected:** each `Opener` names its own `Closer` and each `Closer` its own
`Opener`, for both pairs, with no crossing
**Refs:** crc-BracketContext.md, seq-pair.md#1.2
**Code:** sdom/context_test.go

## Test: every node knows its enclosing opener
**Purpose:** R84 — including text, separators and nested openers
**Input:** a nested document with text at each depth
**Expected:** each node reports the innermost opener containing it; nodes at top
level report none
**Refs:** crc-BracketContext.md, seq-pair.md#1.3
**Code:** sdom/context_test.go
**Alarm:** 1
**Fire alarm:** In `parser.open`, push the opener onto the stack *before* emitting it rather than after. Red: an opener records **itself** as its own enclosing opener instead of the one containing it. Nothing about the bytes, the tiling or the pairing changes, so this is silent everywhere else — and it would quietly corrupt any layer walking enclosure to find scope.
**Inject:** sdom/parser.go:parser.open
**Pulled:** 2026-08-31 — re-pulled after the parser rename and rang again, on the same two tests — this one and the cross-check. The rename moved no property; only symbols changed name. Originally 2026-08-30 — rang, and wider than designed. This test failed and so
did the cross-check, **in the opposite column** from the alarm above: pairs equal
at 316, enclosings 974 vs 962. The blast radius is larger than predicted, because
`take` flushes pending text *before* emitting: pushing first means the text
**preceding** an opener is attributed to it as well. The failure output also
showed this test's message was lossy — it printed nil-ness rather than which
node, so a real failure could read `got true, want true`. Message rewritten
afterwards to name the node; the assertion is unchanged, so this record stands.

## Test: the independent forward scan agrees, over the corpus
**Purpose:** R87 — the check that makes the index a fact rather than an assertion
**Input:** every corpus file under each shipped language; for every node, the
enclosing opener from the index, and the one found by scanning forward while
skipping whole bracket pairs
**Expected:** the two agree for every node of every file. **This is the test that
would catch a wrong index**; nothing else in the suite reads the links twice
**Note:** what this protects is a fact written in two places on purpose. The
scan's rule — *everything but a closer records an enclosing opener* — and
rebuild's rule — *closers get no enclosing* — are the same statement, expressed
once in `parser.emit` and once in `BracketContext.rebuild`. That duplication is
inherent to having two independent derivations, and this test is exactly the
thing that catches them drifting apart.
**Refs:** crc-BracketContext.md, seq-pair.md#2
**Code:** sdom/context_test.go
**Alarm:** 2
**Fire alarm:** Remove the `bc.closes(o, n)` condition from `rebuild`, so the stack walk pairs any closer with whatever opener is on top. **This is the real defect, hit while implementing:** on `( { )` the scan emits `)` unpaired via the any-close fallback while the walk pairs it with `{`. Red: the two derivations disagree, on most corpus files under most languages. Every other test stays green, because each derivation is individually self-consistent.
**Inject:** sdom/context.go:BracketContext.rebuild
**Pulled:** 2026-08-30 — rang. **Only this test failed**, out of 56, and the
message named the divergence exactly: `doc.go under shell: scan recorded 268
enclosings / 77 pairs; the independent walk found 268 / 86`. Enclosings equal,
pairs differing by nine — the stray closers the fallback leaves unpaired. Each
derivation stays individually self-consistent, which is why nothing but the
comparison can see it.

## Test: a stale stamp rebuilds, and a fresh one does not
**Purpose:** R86 — the context is a derived index like any other
**Input:** a scanned document; a freshness check, a membership change, another
check, and a third
**Expected:** fresh, then stale-and-rebuilt exactly once, then fresh again — and
`Doc` holds no reference to the context throughout
**Refs:** crc-BracketContext.md, seq-pair.md#1.5
**Code:** sdom/context_test.go

## Test: the context inherits the mutation guard without writing one
**Purpose:** R86 — reading the generation is the one call every stamped index
makes
**Input:** a freshness check attempted inside a mutation window
**Expected:** it refuses, surfacing as an error from `Mutate`, and the context
contains no guard of its own
**Refs:** crc-BracketContext.md, seq-pair.md#1.5.1
**Code:** sdom/context_test.go

## Test: a document with no brackets carries no links
**Purpose:** R85 — the reason `Doc` does not own this
**Input:** a markdown file scanned with a language whose table is empty
**Expected:** the document is all `Text`, and no link storage is allocated
**Refs:** crc-BracketContext.md
**Code:** sdom/context_test.go
