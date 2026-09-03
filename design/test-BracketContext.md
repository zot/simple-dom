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
**Fire alarm:** In `BracketParser.open`, push the opener onto the stack *before* emitting it rather than after. Red: an opener records **itself** as its own enclosing opener instead of the one containing it. The bytes, the tiling and the pairing are untouched. ~~so this is silent everywhere else~~ — **corrected 2026-09-01: it is now the loudest alarm in the suite**, because the rest of this sentence came true: it *would quietly corrupt any layer walking enclosure to find scope*, and Item 4 built one.
**Inject:** sdom/bracket_parser.go:BracketParser.open
**Pulled:** 2026-09-01 — re-pulled after the vocabulary pass and rang **far louder than at any previous pull** — not because the property moved, but because the coverage grew into the hazard this alarm had already named. **13 failures across both packages:** this test, the corpus cross-check, and **eleven declaration tests** in `sdom/schema`, ending with `TestADeclarationPassIsAdditive` reporting **0 declarations over 24 files** against an independent count of 251. The mechanism is exactly the one the prose predicted: the declaration pass reads *top level* as `Enclosing(n) == nil`, so an index where every opener encloses itself leaves **nothing** top-level and the whole layer sees an empty document. On 2026-08-30 that consumer did not exist. Previously 2026-08-31 — re-pulled after the parser rename and rang again, on the same two tests — this one and the cross-check. The rename moved no property; only symbols changed name. Originally 2026-08-30 — rang, and wider than designed. This test failed and so
did the cross-check, **in the opposite column** from the alarm above: pairs equal
at 316, enclosings 974 vs 962. The blast radius is larger than predicted, because
`take` flushes pending text *before* emitting: pushing first means the text
**preceding** an opener is attributed to it as well. The failure output also
showed this test's message was lossy — it printed nil-ness rather than which
node, so a real failure could read `got true, want true`. Message rewritten
afterwards to name the node; the assertion is unchanged, so this record stands.

## Test: an independent derivation agrees, over the corpus
**Purpose:** R87 — the check that makes the index a fact rather than an assertion
**Input:** every corpus file under each shipped language; every link the context
reports through its **public accessors**, against the same links derived by a stack
walk written in the test rather than in the library
**Expected:** the two agree for every node of every file. **This is the test that
would catch a wrong index**; nothing else in the suite derives the links twice
**Note:** the second derivation lives outside the library on purpose. An index
checked by code sharing its author, its file and its helpers is checked by
something liable to share its misconceptions. What the library owes is that the
answer is reproducible from the array; proving it is a consumer's job, and a test
is a consumer.

## Test: a rescan per node agrees, on a fixture
**Purpose:** R87 — the same guarantee by a genuinely different algorithm
**Input:** for each node, a rescan from the start of the document counting depth,
taking the innermost opener still unclosed when that node is reached
**Expected:** it agrees with the index, and a closer reports **no** enclosing
opener at all
**Note:** O(n) per node, so a fixture rather than the corpus — and that is the
trade worth making. The corpus check shares an *idea* with the library even though
it shares no code; this shares neither, so it is the one that would catch a
mistake common to both.
**Refs:** crc-BracketContext.md, seq-pair.md#2
**Code:** sdom/context_test.go
**Alarm:** 2
**Fire alarm:** Remove the `bc.closes(o, n)` condition from `rebuild`, so the stack walk pairs any closer with whatever opener is on top. **This is the real defect, hit while implementing:** on `( { )` the parse emits `)` unpaired via the any-close fallback while the walk pairs it with `{`. Red: the two derivations disagree, on most corpus files under most languages. Every other test stays green, because each derivation is individually self-consistent.
**Inject:** sdom/context.go:BracketContext.rebuild
**Pulled:** 2026-09-02 — re-pulled after Item 6.1 typed the pair fields and rang: `indent.go under shell: the two derivations disagree on a closer`, only that test. Previously 2026-09-01 — re-pulled after the vocabulary pass and rang again, at `stencil_test.go under shell: the parse recorded 655 entries; the independent walk found 664`. **Only this test failed**, both packages otherwise green — the claim that each derivation stays individually self-consistent, confirmed a third time. (That message now says *the parse recorded*; the quotes further down keep the wording actually printed on their own dates.) Previously 2026-08-31 — re-pulled after the index consolidation and rang again,
disagreeing at the first corpus file it reached: *the scan recorded 192 entries; the
independent walk found 194*. The message reads differently from the 2026-08-30 pull
because the comparison is now over whole `BracketInfo` entries rather than two
separate maps. Originally 2026-08-30 — rang. **Only this test failed**, out of 56, and the
message named the divergence exactly: `doc.go under shell: scan recorded 268
enclosings / 77 pairs; the independent walk found 268 / 86`. Enclosings equal,
pairs differing by nine — the stray closers the fallback leaves unpaired. Each
derivation stays individually self-consistent, which is why nothing but the
comparison can see it.

## Test: an opener knows its separators, and each names it back
**Purpose:** R152, R153 — the direction the contract was missing
**Input:** shell's `for x in a b; do echo $x; done` and
`if p; then q; elif r; then s; else t; fi`
**Expected:** the `for` opener reports `in` and `do`, **in document order**; the `if`
opener reports `then`, `elif`, `then`, `else`; and every separator names its own
opener back. A group with no separators reports none rather than nil-versus-empty
confusion
**Refs:** crc-BracketContext.md, seq-pair.md#1.4
**Code:** sdom/context_test.go
**Alarm:** 3
**Fire alarm:** Record separators only from the parse and drop them from `rebuild`'s
independent walk. ~~Red: **not immediately** — the links are right until a structural
edit makes the stamp stale, and then the rebuilt index has none.~~ — **corrected
2026-08-31 by pulling it:** red **immediately**, and in two places at once. See below.
**Inject:** sdom/context.go:BracketContext.rebuild
**Pulled:** 2026-09-02 — re-pulled after Item 6.1 typed the pair fields and rang: in the same three places as before — the cross-check, `separators [], want [in do]`, and `before the edit: 0 separators, want 2`. Previously 2026-09-01 — re-pulled after the vocabulary pass and rang in **three** places rather than two: this test (`separators [], want [in do]`), the corpus cross-check, and `TestSeparatorsRefreshesLikeEveryOtherAccessor`, which did not exist when the two-place record below was written — it came out of Item 11's own inject-past-the-list probe, so that probe is still earning its keep. Previously 2026-08-31 — rang, and **more loudly than predicted, which is the
widened cross-derivation paying off.**

This test failed as designed — `separators [], want [in do]` — and so did
`TestIndexAgreesWithTheIndependentDerivation`, immediately, at 43 entries against 42.
The prediction of a *delayed* failure assumed separators sat outside `R87`'s
cross-check and would only surface once a stale stamp forced a rebuild. They do not:
the corpus comparison is over whole `BracketInfo` entries, so a separator recorded by
the parse and not by the walk is a disagreement the corpus test sees at once, with no
stale stamp needed.

*That is the change worth keeping from this item.* Comparing the entry rather than
two chosen maps is what put separators inside the cross-derivation **by
construction**, and it is why this alarm could not stay quiet. A field added to
`BracketInfo` later is covered the same way, without anyone remembering to extend
anything.

## Test: a stale stamp rebuilds, and a fresh one does not
**Purpose:** R86 — the context is a derived index like any other
**Input:** a parsed document; a freshness check, a membership change, another
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
**Input:** a markdown file parsed with a language whose table is empty
**Expected:** the document is all `Text`, and no link storage is allocated
**Refs:** crc-BracketContext.md
**Code:** sdom/context_test.go

## Test: Separators refreshes like every other accessor
**Purpose:** R86, R152 — found by injecting past the alarm list, not by design
**Input:** `for x in a b; do echo $x; done`, then remove one separator inside a
mutation window and ask the opener again **without** forcing a rebuild
**Expected:** one separator, not two
**Refs:** crc-BracketContext.md, seq-pair.md#1.5
**Code:** sdom/context_test.go
**Alarm:** 4
**Fire alarm:** Remove `bc.refresh()` from `Separators`. Red: the accessor answers
from an index the document has moved past — *2, want 1*.

**This test exists because that injection was silent.** After the seven recorded
alarms were pulled, the same injection was tried against the suite as it then stood
and **nothing failed**: `TestAnOpenerKnowsItsSeparators` calls `rebuild` itself, so it
never depended on the accessor refreshing, and no other caller existed. Silence at an
unalarmed site is a missing alarm rather than a passing one, which is the whole reason
for injecting past the list.

Note the edit has to **remove** a separator. A merely unrelated structural change
leaves the parse's record still correct, so a missing refresh would return the right
answer for the wrong reason and the test would prove nothing.
**Inject:** sdom/context.go:BracketContext.Separators
**Pulled:** 2026-08-31 — rang, *2, want 1*, with the tree restored clean.

## Test: Opener and Closer are typed
**Purpose:** R193 — the accessors return `*Opener` / `*Closer`, nil for an unmatched marker, with no assertion at the call site.
**Input:** `a(b)c}` parsed with the Go schema.
**Expected:** `Closer(open)` is the `*Closer` for `)`; `Opener` of that closer is the same `*Opener`; `Opener` of the stray `}` is a nil `*Opener`.
**Refs:** crc-BracketContext.md
**Code:** sdom/context_test.go
**Alarm:** 5
**Fire alarm:** make `Opener` hop one entry too far — `return bc.info[bc.info[closer].enclosing].opener`, answering from the closer's enclosing entry rather than its own. A closer records no enclosing opener, so a real closer answers nil; the stray still answers nil. Red: `the ( ) pair does not agree through the typed accessors`. (A first attempt guarded on `bc.info[closer].closer == nil`, which a closer's entry always satisfies — a no-op that left the suite green and proved nothing; recorded so nobody writes it again.)
**Inject:** sdom/context.go:BracketContext.Opener
**Pulled:** 2026-09-02 — rang: `the ( ) pair does not agree through the typed accessors`, plus the pairing test, the cross-derivation, every separator's back-link, and `inner from closer: got ""` — the whole bracket contract reads through this one lookup.

## Test: InnerText and OuterText, from either end and past end of input
**Purpose:** R194, R195 — a group's interior and its whole extent, named by opener or closer, with the open-at-EOF rule.
**Input:** `x = (a, [b]) // tail` parsed with the Go schema, and `f(a` parsed the same way.
**Expected:** `InnerText` of the `(` opener is `a, [b]`; of its `)` closer the same; `OuterText` is `(a, [b])`; for the comment, `InnerText` is ` tail` and `OuterText` is `// tail` (the closer is end of input, so both run to the end); for `f(a`, `InnerText(open)` is `a` and `OuterText` is `(a`.
**Refs:** crc-BracketContext.md
**Code:** sdom/context_test.go
**Alarm:** 6
**Fire alarm:** make `OuterText` stop before the closer — it equals `InnerText` plus the opener and the open-at-EOF case still passes, but `(a, [b])` loses its `)`.
**Inject:** sdom/context.go:BracketContext.OuterText
**Pulled:** 2026-09-02 — rang: `outer: got "(a, [b]", want "(a, [b])"`, only that case; the open-at-EOF cases stayed green exactly as the prose predicts.

## Test: DeclarationNames returns the document's own nodes
**Purpose:** R197, R198 — the typed accessor preserves identity, so a returned name `==` the node skimmed from the array, and it still refuses when stale.
**Input:** `func Foo() {}\nvar Bar = 1` through the Go declaration pass; skim the array for `*DeclarationName`.
**Expected:** for each `*DeclarationType`, `DeclarationNames` returns pointers identical to the skimmed nodes; after a structural mutation the call returns `ErrDeclarationsStale`.
**Refs:** crc-BracketContext.md, seq-declare.md
**Code:** sdom/schema/declaration_test.go
**Alarm:** 7
**Fire alarm:** have `DeclarationNames` return `slices.Clone` of copies built as `&DeclarationName{Text: n.Text}` — the values are equal and the identities are not.
**Inject:** sdom/context.go:BracketContext.DeclarationNames
**Pulled:** 2026-09-02 — rang: `"Foo" is not the node in the document`, `"Bar" is not the node in the document`; the `sdom` package stayed green, since only the identity test can see a faithful copy.

## Test: Opener and Closer refresh like every other accessor
**Purpose:** R86, R193 — found by injecting past the alarm list after Item 6.1, not by design
**Input:** `a(b)(c)`; remove the second closer inside a mutation window and ask its opener again, then remove the first opener and ask its closer — both **without** forcing a rebuild
**Expected:** nil both times: the group is open now, and the closer is stray
**Refs:** crc-BracketContext.md, seq-pair.md#1.5
**Code:** sdom/context_test.go
**Alarm:** 8
**Fire alarm:** Remove `bc.refresh()` from `Opener`. Red: `Opener(close)` still names the removed opener. This is the same hole `Separators` had on 2026-08-31: every other pairing test rebuilds through some other accessor first, so nothing depended on these two refreshing themselves. The edit must **remove** a marker, for the reason the Separators entry gives.
**Inject:** sdom/context.go:BracketContext.Opener
**Pulled:** 2026-09-02 — rang: `Opener(close) = &{{( …}}, want nil`, only this test, both packages otherwise green; the closer half stayed green under this injection since it removes the refresh from `Opener` alone.
