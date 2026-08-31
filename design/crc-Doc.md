# Doc
**Requirements:** R1, R2, R3, R37, R38, R39, R40, R41, R42, R43, R44, R45, R46, R118, R119

A parsed source: the bytes, the flat document-order node array over them, and two
derived indices. It holds **nothing schema-specific** and has **no knowledge of
what exists above it**.

## Knows
- the source bytes
- its **base** offset — its own position within an outer document. **Metadata
  about the source, not part of any node's location**: offsets are relative to
  this document's own source, and a consumer wanting a position in the outer
  document adds the base itself
- `dom` — the nodes, flat and in document order, tiling the source
- `data any` — an open slot for a consumer
- two derived indices over `dom` and nothing else: **node → position**, and
  **lines**
- a monotonic **structural generation**

## Does
- `Prev(n) / Next(n)`: navigate **by node**; positions are never a currency the
  caller carries
- reports the structural generation, so a layer can check its own stamp
- bumps the generation whenever **node membership** changes — never for a content
  edit, since membership is unchanged and an index over structure survives one
- rebuilds its own two indices

## Constraints
- **Stamped, not registered.** `Doc` keeps no registry of derived indices and
  issues no invalidation callbacks. Layers pull; the document does not push
- **The array tiles the document** — contiguous, half-open, first at 0, last at
  the end of the source. This is what makes "every other byte stays where it was"
  a checkable property rather than an intention
- Anything derived beyond the two indices is owned by the layer that needs it, so
  a document type needing none carries none

## Collaborators
- Node: owns them, in document order
- MutationWindow: the only way its contents change
- any layer above: stamps itself with the generation and rebuilds when stale

## Sequences
- seq-stamp.md
- seq-mutate.md
