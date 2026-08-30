# Test Design: Node, Text and Compound
**Source:** crc-Node.md, crc-Text.md, crc-Compound.md

## Test: Equals ignores provenance
**Purpose:** R10 — two nodes with identical content but different offsets are equal
**Input:** the same bytes built as `Text` at offset 0 and at offset 400
**Expected:** equal, in both directions
**Refs:** crc-Node.md, crc-Loc.md
**Code:** sdom/node_test.go

## Test: Equals distinguishes kinds
**Purpose:** R13 — a type check precedes any structural comparison
**Input:** a `Text` and a single-child `Compound` that render the same bytes
**Expected:** not equal, in both directions
**Refs:** crc-Node.md
**Code:** sdom/node_test.go

## Test: a kind that forgets its own Equals is loud
**Purpose:** R13 — the omission fails closed, and the failure is visible rather than silent
**Input:** a test-only kind embedding `Compound` and declaring no `Equals`; two
instances with identical children
**Expected:** they compare **unequal** — the promoted `Compound.Equals` asserts
`*Compound` and the assertion fails against the outer type
**Refs:** crc-Node.md, crc-Compound.md
**Code:** sdom/node_test.go
**Alarm:** 1
**Fire alarm:** Drop the `ok` check from `Compound.Equals` so it compares children without asserting the kind. Red: two `*forgetful` values with identical children compare **equal**, and TestEqualsDistinguishesKinds goes red alongside it. Nothing else in the suite notices, which is the point — a kind-blind Equals loses no bytes and breaks no round-trip.
**Inject:** sdom/node.go:Compound.Equals

## Test: a leaf has no children
**Purpose:** R7, R15
**Input:** a `Text` node
**Expected:** `Kids()` returns nothing; `Render()` returns exactly its bytes
**Refs:** crc-Text.md
**Code:** sdom/node_test.go

## Test: Compound renders by concatenation and sums its children
**Purpose:** R16 — the tiling contract, in both the render and the extent
**Input:** a `Compound` over three `Text` children
**Expected:** `Render()` is the three renders joined; `Location().Length()` is the
sum of the three lengths
**Refs:** crc-Compound.md
**Code:** sdom/node_test.go

## Test: Compound derives Altered from its children
**Purpose:** R28 — computed on read, not stamped at edit time
**Input:** a `Compound` over three faithful children; then one child altered
**Expected:** faithful before, altered after, with nothing having propagated
anything upward; alteration of **any** child is sufficient
**Refs:** crc-Compound.md, crc-Loc.md
**Code:** sdom/node_test.go
**Alarm:** 2
**Fire alarm:** Make `Compound.Location` return `c.loc` with only the lengths summed, dropping the faithfulness walk. Red: the compound still reports `Faithful()` after a child is rewritten, here and in the one-field delta.
**Inject:** sdom/node.go:Compound.Location
