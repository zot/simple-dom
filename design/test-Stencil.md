# Test Design: StencilBuilder, Bool and TodoItem
**Source:** crc-StencilBuilder.md, crc-Bool.md, crc-TodoItem.md

## Test: the glue is computed from the gaps
**Purpose:** R98 — the author writes what they bind, and the rest becomes text
**Input:** `- [x] write the spec` under a regex naming only `checked` and `label`
**Expected:** children of `- [`, the checkbox text, `] `, and the label — the two
unnamed spans present as `Text` despite the pattern never mentioning them
**Refs:** crc-StencilBuilder.md, seq-stencil.md#1.2.2
**Code:** sdom/stencil_test.go
**Alarm:** 1
**Fire alarm:** In `Done`, drop the `glue` calls so only bound groups become children. Red: this test, the tiling test and the round-trip — because this reproduces exactly the failure computed glue dissolves. `- [` and `] ` vanish from the render though the pattern never claimed them, which is the four-bytes-eaten case the old outer-group tiling requirement existed to catch.
**Inject:** sdom/stencil.go:StencilBuilder.Done
**Pulled:** 2026-08-31 — rang, on six tests. The literal injection did **not
compile**: removing the two calls strands the `glue` closure and the `at` cursor,
so the puller extended to the minimal compiling form of the same change and said
so. Every corpus test stayed green, which is not evidence of weakness — the corpus
path runs the lexer, and no corpus document contains a stencil.

## Test: the children tile the match
**Purpose:** R98, R110 — contiguous, half-open, no byte unowned
**Input:** several todo lines, including `- [] no space`, `- [    ] padded`, and
one with an empty label
**Expected:** for each, the children run contiguously from the match's start to its
end, and the stencil renders the line byte-exact
**Refs:** crc-StencilBuilder.md
**Code:** sdom/stencil_test.go

## Test: a named group left unfilled panics
**Purpose:** R99, R105 — a group exists because a schema binds it
**Input:** a builder over a two-group regex where only one group is filled
**Expected:** `Done` panics, naming the group
**Refs:** crc-StencilBuilder.md, seq-stencil.md#1.4.1
**Code:** sdom/stencil_test.go

## Test: a mis-spanned plug panics
**Purpose:** R106 — the only way a schema can break tiling from here
**Input:** a node whose location is not its group's, patched in
**Expected:** `Done` panics rather than producing a compound that would report
itself altered
**Refs:** crc-StencilBuilder.md, seq-stencil.md#1.4.2
**Code:** sdom/stencil_test.go
**Alarm:** 2
**Fire alarm:** Remove the span comparison from `Done`. Red: **only this test**. A mis-spanned child leaves every byte present and the render correct, so the round-trip and the tiling assertions stay green; the compound simply reports itself *altered* forever after — a plausible wrong answer rather than a refusal, and invisible to everything that does not ask about faithfulness.
**Inject:** sdom/stencil.go:StencilBuilder.Done
**Pulled:** 2026-08-31 — rang, and **alone out of 79**. `TestUnfilledGroupPanics`
stayed green, so the two checks in `Done` are independent rather than one property
counted twice.

## Test: omitting a group is the same as never naming it
**Purpose:** R103 — the falsifiable meaning of `Omit`, rather than an intention
**Input:** the same text parsed twice — once with a three-group regex whose middle
group is omitted, once with a two-group regex that never names it
**Expected:** the two child lists are **identical**, node for node, in count, kinds,
bytes and locations. In particular the omitted span is merged with the glue on both
sides rather than standing as a third `Text`, so no two adjacent children are both
plain glue
**Refs:** crc-StencilBuilder.md, seq-stencil.md#1.3.2
**Code:** sdom/stencil_test.go
**Alarm:** 3
**No fire alarm — the property is structural, and that was measured rather than
assumed.** The designed injection was to make `Omit` a no-op. Pulled 2026-08-31,
it does not reach this test's claim at all: with `Omit` inert the group stays
*bound*, so `Done`'s nil-slot check fires and the test dies in its own setup
before comparing anything. With that test skipped, **78 of 78 pass** — nothing
else in the package uses `Omit`; `todo.go` never calls it.

The reason no small injection reaches this is the implementation. **There is no
merge step to break**: `bounds()` simply does not treat an omitted group as a
boundary, so the glue arithmetic spans it. An injection would have to *add* code
to violate it, which is the absence of a test result rather than a red one.

*Worth recording as a design consequence.* The original sketch — *`Done` zips over
the children and merges omitted groups with their neighbours* — **would** have
been alarmable, because the merge would have been a step that could be removed.
Recomputing the spans instead traded a testable step for a structural guarantee.
The test still earns its place as the specification of what `Omit` means; it is
simply not falsifiable by injection.

## Test: a non-participating group owes nothing
**Purpose:** R100, R101 — the losing branch of an alternation has no bytes
**Input:** a regex with two alternative named groups, matched against text
exercising each branch in turn
**Expected:** `Done` does not panic though one group is never filled; `Group`
returns the **zero location** for the branch that did not participate, and a real
one for a participating group even when its match is empty
**Refs:** crc-StencilBuilder.md, crc-Loc.md
**Code:** sdom/stencil_test.go
**Alarm:** 4
**Fire alarm:** In `Group`, return `b.span(0, 0)` instead of the zero `Loc` when a group did not participate. Red: this test. The distinction it destroys — absent versus present-but-empty — is exactly the one `Loc`'s zero value exists to make, and nothing else in the suite asks for it.
**Inject:** sdom/stencil.go:StencilBuilder.Group
**Pulled:** 2026-08-31 — rang alone. The failure printed the whole `Loc`, showing
`offset:1 length:0` where the zero value belongs — absence rendered as a real
position, which is exactly the distinction being destroyed.

## Test: binding is by name, not position
**Purpose:** R108 — alternation gives branches different group counts
**Input:** a regex whose branches capture a different number of groups, matched on
each branch
**Expected:** the same names resolve correctly on both, where a fixed index would
be right for one and out of range for the other
**Refs:** crc-StencilBuilder.md
**Code:** sdom/stencil_test.go

## Test: a checkbox reads without normalising
**Purpose:** R111, R112 — nothing is stored, so nothing is normalised
**Input:** `[ ]`, `[]`, `[    ]`, `[x]`, `[X]`
**Expected:** the first three read false and the last two true, and **every one
renders back byte-exact**, padding intact
**Refs:** crc-Bool.md
**Code:** sdom/stencil_test.go

## Test: setting a checkbox keeps its offset and contracts
**Purpose:** R113, R114 — the dual port, and provenance surviving an edit
**Input:** `- [    ] task` with the checkbox set true
**Expected:** the line renders `- [x] task`; the checkbox text keeps its original
offset, reports altered, and its length is now 1; the stencil is altered because a
child is; and the `Bool` still points at the same node in the child list
**Refs:** crc-Bool.md, seq-stencil.md#2
**Code:** sdom/stencil_test.go
**Alarm:** 5
**Fire alarm:** In `Bool.Set`, build a **new** `Text` and point at that instead of writing through the existing one. Red: this test, on the assertion that the `Bool` still points at the node in the child list. The render is unaffected — the child list keeps the old text — so a document would silently stop reflecting what the tool believed it had written.
**Inject:** sdom/bound.go:Bool.Set
**Pulled:** 2026-08-31 — rang alone, on four of its assertions at once. The puller
also ran a second variant without `.alter()` and got byte-identical output, which
isolates node **identity** as the thing that failed rather than the alteration flag.

## Test: setting a value already held changes nothing
**Purpose:** R113 — an edit reformats what it touched, and confirming a value
touches nothing
**Input:** `- [X] a` with the checkbox set to true, which it already is
**Expected:** the text is not rewritten, so the node stays faithful and the `X`
survives
**Refs:** crc-Bool.md
**Code:** sdom/stencil_test.go
**Alarm:** 6
**Fire alarm:** Remove the early return from `Bool.Set` so it always writes. Red:
only this test. The document still renders correctly — `x` for `X` — and the
round-trip is blind to it, because the bytes it compares are the ones that were
just written. What is lost is faithfulness: a node that was a faithful view of the
source becomes altered for no reason, and every diagnostic downstream believes the
line was edited.
**Inject:** sdom/bound.go:Bool.Set
**Pulled:** 2026-08-31 — rang alone. `TestCheckboxReadsWithoutNormalising` stayed
green because it never writes, and `TestSettingACheckboxKeepsItsOffsetAndContracts`
because it sets a value the text does not already hold. The property needs its own
test precisely because the two neighbouring ones cannot reach it.

## Test: a todo item round-trips a real markdown list
**Purpose:** R96 — the whole mechanism over real syntax rather than a fixture
**Input:** a markdown todo list with mixed indentation, padding, empty labels and
one line that is not a todo at all
**Expected:** every todo line parses, the non-todo line does not match and is left
to the caller, and the document renders byte-exact
**Refs:** crc-TodoItem.md
**Code:** sdom/stencil_test.go
