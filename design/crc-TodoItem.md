# TodoItem
**Requirements:** R96, R223, R116

A markdown todo line — `- [ ] label` — and the worked example of what a schema
writes. A **test fixture**, living in `stencil_test.go`: `sdom` ships no markdown
schema, and an example that is not shipped code belongs with the tests. Real syntax
rather than an invented fixture, and the case `Bool` exists for.

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
- `label` is bound because a tool **reads** it as a value — binding serves access as
  well as editing (R223) — and because two groups prove the gap *between* them is glue

## Collaborators
- StencilBuilder: the machinery it drives
- Bool: its checkbox
- Compound: what it embeds for tiling, render and extent

## Sequences
- seq-stencil.md
