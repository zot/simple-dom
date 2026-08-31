# Test Design: StencilBuilder, Bool and TodoItem
**Source:** crc-StencilBuilder.md, crc-Bool.md, crc-TodoItem.md

## Test: the glue is computed from the gaps
**Purpose:** R98 — the author writes what they bind, and the rest becomes text
**Input:** `- [x] write the spec` under a regex naming only `checked` and `label`
**Expected:** children of `- [`, the checkbox text, `] `, and the label — the two
unnamed spans present as `Text` despite the pattern never mentioning them
**Refs:** crc-StencilBuilder.md, seq-stencil.md#1.2.2
**Code:** sdom/stencil_test.go

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

## Test: a non-participating group owes nothing
**Purpose:** R100, R101 — the losing branch of an alternation has no bytes
**Input:** a regex with two alternative named groups, matched against text
exercising each branch in turn
**Expected:** `Done` does not panic though one group is never filled; `Group`
returns the **zero location** for the branch that did not participate, and a real
one for a participating group even when its match is empty
**Refs:** crc-StencilBuilder.md, crc-Loc.md
**Code:** sdom/stencil_test.go

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

## Test: a todo item round-trips a real markdown list
**Purpose:** R96 — the whole mechanism over real syntax rather than a fixture
**Input:** a markdown todo list with mixed indentation, padding, empty labels and
one line that is not a todo at all
**Expected:** every todo line parses, the non-todo line does not match and is left
to the caller, and the document renders byte-exact
**Refs:** crc-TodoItem.md
**Code:** sdom/stencil_test.go
