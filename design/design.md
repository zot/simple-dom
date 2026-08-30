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
