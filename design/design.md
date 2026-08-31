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
- [x] crc-BracketGroup.md → `sdom/bracket.go`
- [x] crc-BracketLang.md → `sdom/bracket.go`, `sdom/lang.go`
- [x] crc-Marker.md → `sdom/marker.go`
- [x] crc-Lexer.md → `sdom/lexer.go`
- [x] crc-BracketContext.md → `sdom/context.go`

### Sequences
- [x] seq-mutate.md → `sdom/mutate.go`, `sdom/doc.go`
- [x] seq-stamp.md → `sdom/doc.go`
- [x] seq-scan.md → `sdom/lexer.go`
- [x] seq-pair.md → `sdom/context.go`

### Test Designs
- [x] test-Node.md → `sdom/node_test.go`
- [x] test-Loc.md → `sdom/loc_test.go`
- [x] test-Doc.md → `sdom/doc_test.go`
- [x] test-roundtrip.md → `sdom/roundtrip_test.go`
- [x] test-Lexer.md → `sdom/lexer_test.go`
- [x] test-BracketContext.md → `sdom/context_test.go`
- [x] test-Languages.md → `sdom/lang_test.go`

## Gaps

- [x] I1: R12 (each schema's parse context is a concrete type, not an interface) has design
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
- [x] O4: The structural round-trip (test-roundtrip.md) reconstructs its comparison tree by the
  same construction rather than re-parsing it, because nothing recovers a `Compound` from bytes
  until Item 2's lexer. It therefore proves `Equals` ignores provenance but does not prove a
  parse is stable. The re-parse form of the test belongs with Item 2 and should replace the
  reconstruction there.
- [ ] O5: Structural edits locate their nodes with a linear scan (`Doc.find`), because the node
  index refuses inside the mutation window and is stale there anyway. That is O(n) per edit and
  O(n^2) for a window making many edits to a large document. Splicing needs the position
  regardless, so the scan is not pure overhead — but a window that edits a whole document node
  by node would feel it. Measure before optimising; no consumer exists yet.
- [ ] O6: `TestLangShell` and `TestLangPascal` assert with `strings.Contains` over the rendered
  stream rather than comparing it exactly, so they check that certain markers appear rather than
  that the tokenization is right. Measured 2026-08-30: under the `Restricted() == false`
  injection both stayed green while their dumps showed visibly wrong tokenization, because the
  markers they look for still appeared somewhere. `TestLangGo` and `TestLangJavaScript` use
  exact-stream comparison and both failed. Not a hole — the per-marker recognition count now
  covers what these miss — but the two tests read stronger than they are, and a reader would
  trust them further than the evidence warrants.
- [ ] O7: `BracketContext.rebuild` recomputes the whole index on any membership change: there is
  no incremental path, so a document edited structurally many times rebuilds its bracket links
  in full each time the stamp goes stale. That is the price of the stamped-not-registered design
  and it is the right default — an incremental rebuild would need `Doc` to describe what
  changed, which is exactly the coupling the design refuses. Revisit only if a consumer with a
  large document and frequent structural edits appears; measure before optimising.
- [ ] O8: `matchOpen` and `matchAnyClose` walk the entire bracket table at every scan position,
  so scanning costs positions × groups × markers with no dispatch on the first byte. Fine for
  the corpus today (the whole suite scans every project file under four languages in well under
  a second) but it is the obvious hot spot if a large file or a big table ever appears. A
  first-byte index over the markers would collapse most of it. No consumer needs it yet; measure
  before optimising.
- [ ] O9: A `Doc`'s `base` never enters a node's `Loc`: the lexer emits `Source(pos, len)` with
  positions relative to the document's own source, and `base` is metadata about where that
  source sits in an outer document. Nothing states this, and two documents assume the opposite.
  The carve's Item 1 test 4 says the structural round-trip is "re-parsed into a document with a
  different base offset, so passing also proves `Equals` ignores provenance" — which is false
  as built, since a different base leaves every offset identical. And Item 1's `nest()` fixture
  bakes base into offsets by hand, so `TestStructuralRoundTripAtADifferentBase` passes for a
  reason unrelated to how a parse actually assigns locations. Measured 2026-08-30: an alarm on
  `Opener.Equals` comparing locations did not ring against a re-parse at a different base, and
  rang immediately once offsets were shifted by a prefix instead. Repair: state the rule in
  specs/document.md and add a requirement for it (this gap ADDS a requirement rather than
  editing one — R38 stays true, it is merely not the whole story), then rewrite `nest()` to
  stop faking the shift, and correct the carve's test-4 wording at its source.
- [ ] O10: A `test-*.md` entry can specify a test that no longer exists in the code, and nothing
  detects it. Artifact checkboxes are per FILE, so `sdom/roundtrip_test.go` stayed checked while
  one of the tests the design specifies for it was gone; `validate` stayed green, requirement
  coverage stayed green, and the alarm anchored to that test had nothing to ring. Measured
  2026-08-30: `TestStructuralRoundTripThroughAReparse` was destroyed by an index-to-index text
  edit and the test count masked it — seven added and one deleted totalled exactly what was
  expected without the deletion. It was found by a reviewer reading the design against the code,
  which is the only mechanism that currently can. A checkable repair exists: a `test-*.md` entry
  names its `**Code:**` file, and its `## Test:` headings could be required to correspond to
  test functions in that file. That belongs in the mini-spec tool rather than here, so this gap
  records the hole and the evidence rather than proposing a fix in this repository.
