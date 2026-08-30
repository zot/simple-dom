# Compound
**Requirements:** R16, R17, R19, R28

A node whose children **tile its span**. It does no parsing: the layers above
supply only *how their children are computed*, and embed this for everything
else.

## Knows
- its children, in document order
- its own provenance

## Does
- `Kids`: returns the children
- `Render`: concatenates the children's renders
- `Location`: sums the children's lengths and **derives `Altered` as their OR**
- `Equals`: asserts `*Compound`, then compares children pairwise

## Constraints
- **`Altered` is derived on read, never stamped and propagated.** The read must
  visit every child to sum lengths anyway, so nothing is saved by stamping — and
  the deferred walk it replaces carried a defect, where a short-circuiting fold
  left later compounds claiming spans they could no longer honour
- **Compounds exist only for stenciling** — a region a tool writes into. Never
  for bracket structure
- Embedded by the regex compound (Item 3), the declaration (Item 4) and the
  traceability comment (Item 6). Each declares its own `Equals`

## Collaborators
- Node: its children are any kind
- Loc: the summed extent and OR'd alteration are reported through one

## Sequences
- seq-mutate.md
