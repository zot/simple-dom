# RequirementList
**Requirements:** R204, R205, R206

The numbered list: `List` with items that are requirement refs, where a range is
one item and `Items` expands it.

## Knows
- what `List` knows

## Does
- `ParseRequirementList`: the list grammar with items `Rn`, `Rn-Rm` or `Rn-m` —
  the **only** parser that accepts a range
- `Items() []int`: expands every range; a reversed range contributes only its low ref
- `SetItems([]int)`: sorts, de-duplicates, and emits maximally condensed — a run of
  three or more as `R7-12`, a pair as `R7, R8`; cannot be refused, since integers
  carry no separator

## Constraints
- **A range exists only through this parser**, so a plain list cannot contain one by
  construction — an invariant that needs no check anywhere downstream
- **A write re-emits the whole list canonically.** `R5-R7, R10` plus `12` becomes
  `R5-7, R10, R12`; preserving an unedited range inside an edited field was judged
  too fancy for this case (Bill, 2026-09-02)

## Collaborators
- List: what it embeds

## Sequences
- seq-anchor.md
