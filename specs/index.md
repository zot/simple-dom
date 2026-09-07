# Spec index

Every spec in this project, by system. The per-feature specs are canonical; this
index is a pointer, not a copy.

## sdom — the parsing core

Module `github.com/zot/simple-dom`, package `sdom`. A parse that models only what
a tool operates on, keeps every other byte where it was, and re-emits the source
with nothing but the intended change in it. **Nothing in this system knows what a
CRC card is** — the boundary is enforced by the compiler, so another tool can
consume the parsing half.

- [node-protocol.md](node-protocol.md) — what a node is, the `Node` interface,
  per-kind equality, and the kinds `sdom` defines.
- [location.md](location.md) — provenance separated from faithfulness, and
  `Split` / `Merge` as re-granulation.
- [document.md](document.md) — `Doc`, the flat array, the structural generation,
  and the mutation window.
- [parser-protocol.md](parser-protocol.md) — one source, one pass, several parsers:
  the walk, the `Parser` interface, and the entry point.
- [bracket-parser.md](bracket-parser.md) — the table-driven parser, its two mode
  fields, and the pairing links the schema's context owns.
- [indent-parser.md](indent-parser.md) — indentation as scope, significant only at
  bracket depth 0, and the frame links its context owns.
- [declarations.md](declarations.md) — the declaration machinery: the two typed
  node kinds, `Replace`, and the links a schema owns.
- [stencils.md](stencils.md) — compounds that parse by regex: the builder a schema
  drives, computed glue, and bound values.
- [lists.md](lists.md) — the comma-separated field as a one-child stencil, its
  guarded whole-field write, and the numbered list that alone accepts ranges.

## sdom/schema — the bundled language schemas

A separate package under `sdom`, because a schema is tightly coupled to the
machinery and anyone parsing Go wants Go's schema. Nothing here knows what a CRC
card is either; being a separate package is what keeps `sdom`'s export surface
honest.

- [declaration-schemas.md](declaration-schemas.md) — how each language announces a
  declaration: keyword and keyword-less forms, groups, and the bundled Go,
  TypeScript, JavaScript, Lua and Shell schemas.
- [markdown.md](markdown.md) — the markdown base the trajectory file schemas embed:
  a narrow `IndentLang` and a one-pass parser wrapping `IndentParser` that emits
  line-head markers.

## minispecsdom — mini-spec's readers

A sibling package of `sdom` in this repo. The only system that may know what a CRC
card is, and the only one that will ever be used by mini-spec alone. The boundary is
enforced by the compiler rather than by discipline.

- [traceability-comment.md](traceability-comment.md) — the anchor comment as one
  node over the whole group: grammar, `Parse`, the second pass, and construction.
- [part-line.md](part-line.md) — a carve status line as one node: bound checkbox
  and key, marker spans, strike behind an accessor, deviations as the contract.
- [carve-schema.md](carve-schema.md) — the first file schema: owns a carve's DOM,
  reads the `## Status` region into parts with depth, writes markers and landings.
- [testdoc-schema.md](testdoc-schema.md) — the first design-document reader: a test
  design's `## Test:` entries, the five alarm fields, and the three alarm writes
  (pulled, inject, number-alarms) through node references.
- [gaps-schema.md](gaps-schema.md) — the `## Gaps` section of a design document: typed,
  numbered entries with checkbox or permanence, nesting, and the add, resolve and
  approve writes.
- [requirements-schema.md](requirements-schema.md) — `requirements.md`: sections at any
  level with their own content and source, live and retired entries, and the add and
  retire writes.
- [pending-schema.md](pending-schema.md) — the queue file: entries as views over
  heading regions, placement by position never by splice, removal, and what could
  not be read.
- [done-schema.md](done-schema.md) — the ledger: entries at `- **`, the identifier
  slot between em dash and colon, the part pointer, prepend after the rule.
- [current-schema.md](current-schema.md) — the resume buffer: exactly one `## Active`,
  its region set or reset as a unit, standing sections never touched.

## Summary specs

None yet.

## Cross-cutting themes

- **The array is flat.** Nesting is never node children; it is links owned by the
  layer that needs them. A bracket group is not a node.
- **One source, one pass.** A layer that needs its own nodes contributes a parser to
  the same walk rather than splitting and re-carving the array afterwards. Nesting a
  parser inside another is how a language composes them; the walk arbitrates nothing.
- **Derived state is stamped, not registered.** A layer holding an index over
  document structure stamps itself with the document's structural generation and
  rebuilds when stale. The document knows nothing of what is above it.
- **Correctness and enforcement are different things.** An invariant can be
  correct without a mechanism guarding it. A guard earns its place only where the
  invariant is easy to violate silently, the violation is expensive, and the
  guard is cheap and local.
