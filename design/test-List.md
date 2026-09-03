# Test Design: List and RequirementList
**Source:** crc-List.md

## Test: items derive from the literal and the literal is preserved
**Purpose:** R199, R200, R202 — nothing normalised on the way in
**Input:** `a,b`, ` a ,  b `, `crc-Store.md, crc-Index.md`
**Expected:** each reads its two items trimmed; each renders back byte-exact; each node has exactly one child; `ParseList` on `a, b | rest` returns remainder ` | rest`
**Refs:** crc-List.md
**Code:** sdom/list_test.go
**Fire alarm:** make `Items` trim nothing — ` a ,  b ` reads `[" a " "  b "]`.
**Inject:** sdom/list.go:List.Items

## Test: SetItems rewrites canonically and refuses what would not read back
**Purpose:** R201, R203 — the guarded write, here and only here
**Input:** parse ` a ,  b `; `SetItems([x y z])`; then `SetItems([a,b c])` and `SetItems([p q])`
**Expected:** after the first, the literal is `x, y, z` and the node is altered; the second and third return an error and the literal is still `x, y, z`
**Refs:** crc-List.md, seq-anchor.md#3
**Code:** sdom/list_test.go
**Fire alarm:** drop the re-parse and write unconditionally — `SetItems([a,b c])` succeeds and `Items` reads back four items where two were set. The byte round-trip is blind to it, which is the whole point of the guard.
**Inject:** sdom/list.go:List.SetItems

## Test: requirement ranges expand, reversed ranges contribute the low ref
**Purpose:** R204, R206
**Input:** `R4, R5-7, R10-R12, R9-R8`; and `ParseList` over `R5-7`
**Expected:** `Items()` is `[4 5 6 7 10 11 12 8]`; the plain parser reads `R5-7` as the single item `R5-7`, a string
**Refs:** crc-RequirementList.md
**Code:** sdom/list_test.go
**Fire alarm:** expand a reversed range as an empty run rather than its low ref — `8` goes missing.
**Inject:** sdom/list.go:RequirementList.Items

## Test: SetItems condenses
**Purpose:** R205 — maximally condensed, sorted, de-duplicated
**Input:** `SetItems([12 7 8 9 4 4 10 11])`; `SetItems([7 8])`
**Expected:** `R4, R7-12`; `R7, R8`
**Refs:** crc-RequirementList.md
**Code:** sdom/list_test.go
**Fire alarm:** condense runs of two — `R7, R8` becomes `R7-8`.
**Inject:** sdom/list.go:RequirementList.SetItems
