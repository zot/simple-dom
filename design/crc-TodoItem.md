# TodoItem
**Requirements:** R96, R115, R116

A markdown todo line — `- [ ] label` — and the worked example of what a schema
writes. Real syntax rather than an invented fixture, and the case `Bool` exists
for.

## Knows
- `checked`: a `Bool` over the checkbox text
- `label`: the `Text` after it
- its children, through the embedded `Compound`

## Does
- `Parse`: runs the builder, decides what each group becomes, keeps its own
  references, and takes the children

## Constraints
- Its regex captures **only what it binds**: `- [`, `] ` and the line's end are not
  in the pattern at all and become glue
- **`Parse` is not on the `Node` interface.** Parsing constructs a concrete node
  and then sends it `parse`, so the interface is never involved at that moment
- Whether `label` deserves to be a bound field is an **editability** question, not
  a parsing one — a tool that only ever toggles the checkbox would leave the label
  as glue. It is bound here because a fixture needs two groups to prove the gap
  *between* them is computed, which is a test-shape reason and is recorded as one

## Collaborators
- StencilBuilder: the machinery it drives
- Bool: its checkbox
- Compound: what it embeds for tiling, render and extent

## Sequences
- seq-stencil.md
