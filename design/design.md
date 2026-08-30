# Design: sdom

## Intent

`sdom` is the lexical core of the simple DOM: a parse that models only what a
tool operates on, keeps every other byte where it was, and re-emits the source
with nothing but the intended change in it.

It carries the node protocol, locations, and the document — `Node`, `Text`,
`Compound`, `Loc`, `Doc`, the structural generation, and the mutation window.
Nothing in it knows what a CRC card is, so another tool can consume it; that
boundary is enforced by the compiler rather than by discipline.

Source: [carves/simple-dom.md](../carves/simple-dom.md), part `#1`.

## Artifacts

### CRC Cards
- [x] crc-Node.md → `sdom/node.go`
- [x] crc-Text.md → `sdom/node.go`
- [x] crc-Compound.md → `sdom/node.go`
- [x] crc-Loc.md → `sdom/loc.go`
- [x] crc-Doc.md → `sdom/doc.go`
- [x] crc-MutationWindow.md → `sdom/mutate.go`

### Sequences
- [x] seq-mutate.md → `sdom/mutate.go`, `sdom/doc.go`
- [x] seq-stamp.md → `sdom/doc.go`

### Test Designs
- [x] test-Node.md → `sdom/node_test.go`
- [x] test-Loc.md → `sdom/loc_test.go`
- [x] test-Doc.md → `sdom/doc_test.go`
- [x] test-roundtrip.md → `sdom/roundtrip_test.go`

## Gaps

- [ ] I1: R12 (each schema's parse context is a concrete type, not an interface) has design
  coverage but no inline ref in any code file, because no parse context exists yet. Nothing in
  `sdom` parses from text — the lexer is Item 2 of carves/simple-dom.md, and `Parse` is
  deliberately off the `Node` interface. Closes when Item 2 lands a schema with a parse context.
- [ ] O1: R28 understates the rule the code implements. It says a compound is altered if any of
  its children is; `Compound.Location` also requires the children's spans to run **contiguously
  from the compound's own offset**, because children that are individually faithful but no
  longer tile the compound's source span would otherwise let it report `Faithful()` while
  rendering something else. Repairing this gap edits R28 and the sentence in specs/location.md
  that spawned it — R28 carries a back-link. No API in Item 1 can reach the case (nothing
  removes a child from a compound), so this is a latent correctness rule rather than a live
  defect; it becomes reachable as soon as a reader groups a non-contiguous selection.
- [ ] O2: `Doc.Nodes()` returns the live backing slice with "must not be modified" stated only
  in prose. The document's two derived indices are computed from it, so an in-place write by a
  caller corrupts them silently. Options: `slices.Clone` (an allocation per call), or an
  `iter.Seq[Node]` which enforces it for free but changes the API. Worth deciding once the call
  sites are known.
- [ ] O3: `Doc.Split` and `Doc.Merge` accept `Node` but are defined only for `*Text`, refusing
  anything else with a runtime `%T` error. That is honest while `*Text` is the only splittable
  kind, but the mutation vocabulary is then not expressible over the protocol the `Node` doc
  comment advertises. Item 2's lexer lands marker kinds; decide then whether re-granulation
  belongs on the interface or stays a leaf operation.
- [ ] O4: The structural round-trip (test-roundtrip.md) reconstructs its comparison tree by the
  same construction rather than re-parsing it, because nothing recovers a `Compound` from bytes
  until Item 2's lexer. It therefore proves `Equals` ignores provenance but does not prove a
  parse is stable. The re-parse form of the test belongs with Item 2 and should replace the
  reconstruction there.
- [ ] O5: Structural edits locate their nodes with a linear scan (`Doc.find`), because the node
  index refuses inside the mutation window and is stale there anyway. That is O(n) per edit and
  O(n^2) for a window making many edits to a large document. Splicing needs the position
  regardless, so the scan is not pure overhead — but a window that edits a whole document node
  by node would feel it. Measure before optimising; no consumer exists yet.
