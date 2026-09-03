# Design: sdom

## Intent

`sdom` is the parsing core of the simple DOM: a parse that models only what a
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
- [x] crc-Loc.md → `sdom/loc.go`, `sdom/origin.go`
- [x] crc-Doc.md → `sdom/doc.go`
- [x] crc-MutationWindow.md → `sdom/mutate.go`, `sdom/alloc_test.go`
- [x] crc-BracketGroup.md → `sdom/bracket.go`
- [x] crc-BracketLang.md → `sdom/bracket.go`, `sdom/lang.go`
- [x] crc-Marker.md → `sdom/marker.go`
- [ ] crc-ParserState.md → `sdom/parser.go`
- [ ] crc-Parser.md → `sdom/parser.go`
- [x] crc-BracketParser.md → `sdom/bracket_parser.go`
- [ ] crc-IndentLang.md → `sdom/indent.go`, `sdom/lang.go`
- [ ] crc-IndentParser.md → `sdom/indent.go`
- [ ] crc-IndentContext.md → `sdom/indent.go`
- [x] crc-BracketContext.md → `sdom/context.go`
- [x] crc-StencilBuilder.md → `sdom/stencil.go`
- [x] crc-Bool.md → `sdom/bound.go`
- [x] crc-TodoItem.md → `sdom/stencil_test.go`
- [x] crc-List.md → `sdom/list.go`
- [x] crc-RequirementList.md → `sdom/list.go`
- [x] crc-TraceabilityComment.md → `minispecsdom/comment.go`
- [x] crc-Declaration.md → `sdom/declaration.go`
- [x] crc-DeclSchema.md → `sdom/schema/schema.go`
- [x] crc-GoSchema.md → `sdom/schema/golang.go`
- [x] crc-LuaSchema.md → `sdom/schema/lua.go`
- [x] crc-ShellSchema.md → `sdom/schema/shell.go`
- [ ] crc-PythonSchema.md → `sdom/schema/python.go`

### Sequences
- [x] seq-mutate.md → `sdom/mutate.go`, `sdom/doc.go`
- [x] seq-stamp.md → `sdom/doc.go`
- [x] seq-collaborate.md → `sdom/parser.go`, `sdom/bracket_parser.go`
- [x] seq-parse.md → `sdom/bracket_parser.go`
- [ ] seq-indent.md → `sdom/indent.go`
- [x] seq-pair.md → `sdom/context.go`
- [x] seq-stencil.md → `sdom/stencil.go`
- [x] seq-declare.md → `sdom/declaration.go`, `sdom/schema/schema.go`
- [x] seq-anchor.md → `sdom/list.go`, `minispecsdom/comment.go`

### Test Designs
- [x] test-Node.md → `sdom/node_test.go`
- [x] test-Loc.md → `sdom/loc_test.go`
- [x] test-Doc.md → `sdom/doc_test.go`, `sdom/alloc_test.go`
- [x] test-roundtrip.md → `sdom/roundtrip_test.go`
- [x] test-BracketParser.md → `sdom/parser_test.go`
- [x] test-BracketContext.md → `sdom/context_test.go`, `sdom/schema/declaration_test.go`
- [x] test-Languages.md → `sdom/lang_test.go`
- [x] test-Stencil.md → `sdom/stencil_test.go`
- [ ] test-Protocol.md → `sdom/protocol_test.go`
- [ ] test-Indent.md → `sdom/indent_test.go`
- [ ] test-PythonSchema.md → `sdom/schema/python_test.go`
- [x] test-Declaration.md → `sdom/declaration_test.go`, `sdom/schema/declaration_test.go`
- [x] test-DeclSchema.md → `sdom/schema/declaration_test.go`
- [x] test-List.md → `sdom/list_test.go`
- [x] test-TraceabilityComment.md → `minispecsdom/comment_test.go`

## Gaps

- [x] I1: R12 (each schema's parse context is a concrete type, not an interface) has design
  coverage but no inline ref in any code file, because no parse context exists yet. Nothing in
  `sdom` parses from text — the parser is Item 2 of carves/simple-dom.md, and `Parse` is
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
  comment advertises. Item 2's parser lands marker kinds; decide then whether re-granulation
  belongs on the interface or stays a leaf operation.
- [x] O4: The structural round-trip (test-roundtrip.md) reconstructs its comparison tree by the
  same construction rather than re-parsing it, because nothing recovers a `Compound` from bytes
  until Item 2's parser. It therefore proves `Equals` ignores provenance but does not prove a
  parse is stable. The re-parse form of the test belongs with Item 2 and should replace the
  reconstruction there.
- [ ] O5: Structural edits locate their nodes with a linear scan (`Doc.find`), because the node
  index refuses inside the mutation window and is stale there anyway. That is O(n) per edit and
  O(n^2) for a window making many edits to a large document. Splicing needs the position
  regardless, so the scan is not pure overhead — but a window that edits a whole document node
  by node would feel it. Measure before optimising; no consumer exists yet.
- [ ] O6: `TestLangShell` and `TestLangPascal` assert with `strings.Contains` over the rendered
  stream rather than comparing it exactly, so they check that certain markers appear rather than
  that the parse is right. Measured 2026-08-30: under the `Restricted() == false`
  injection both stayed green while their dumps showed visibly wrong parses, because the
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
- [ ] O8: `matchOpen` and `matchAnyClose` walk the entire bracket table at every parse position,
  so parsing costs positions × groups × markers with no dispatch on the first byte. Fine for
  the corpus today (the whole suite parses every project file under four languages in well under
  a second) but it is the obvious hot spot if a large file or a big table ever appears. A
  first-byte index over the markers would collapse most of it. No consumer needs it yet; measure
  before optimising.
- [x] O9: A `Doc`'s `base` never enters a node's `Loc`: the parser emits `Source(pos, len)` with
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
- [ ] O11: `Parse` mints an unnamed `Origin`, and there is no way to name it at the call. A
  caller wanting a useful name must reach through `ctx.Origin().Name = path` afterwards, which
  works but means the common case — two parses colliding — reports two unnamed
  identities in the panic message rather than two paths. `Origin.String` degrades gracefully (it
  prints the pointer when unnamed) so the diagnostic is still usable, but it is worse than it
  needs to be. The repair is an API change: either `Parse` takes a name, or it takes an `*Origin`
  the caller minted. Deferred because no consumer names its parses yet, and because whichever
  shape is right will be obvious once one does.
- [ ] O12: Whether `Doc.Merge`'s fast path actually avoids work is unestablished, as distinct
  from whether its result is a slice — which R120 now asserts and
  `TestMergeSlicesTheSourceWhenItCan` checks. The original implementation built
  `ta.text + tb.text` unconditionally and discarded it in the faithful case, which a reviewer
  caught as a defect. Measured 2026-08-31 with `testing.AllocsPerRun` over 200 runs: 13.0
  allocations for the faithful path and 13.0 for the altered path, **and the same 13.0 with the
  discarded concatenation re-introduced**. Either Go's SSA sinks the concatenation into the
  branch that uses it — making the source-level defect cosmetic and the machine code identical
  — or the probe was wrong; it ignored errors and is not trustworthy. Both readings are worth
  knowing and neither is established. Repair: a careful benchmark that isolates `Merge` from the
  window's index rebuild, or an inspection of the generated code. Until then the requirement
  claims only what has been observed.
- [x] O13: `StencilBuilder.Omit` has no caller outside its own test, and **that is not a defect** — the original wording of this gap applied an application standard to a library. A library package exports API for *consumers*; its tests are the in-package exercise of that API, and there is no reason it would call its own exports internally. Recorded here rather than deleted because the mistake is worth not repeating: "unused outside tests" is a real signal in application code and a meaningless one in a library. What remains true and worth knowing is narrower — `Omit` exists because a schema will want a group it does not bind, carve Items 4 and 6 both have shapes that need it, and the alternative was forcing a schema to `Put` a `Text` it does not want purely to satisfy the nil check.
- [x] O14: `TodoItem` binds `label` for a test-shape reason rather than an editability one,
  which is a knowing deviation from the minimality rule the same item establishes. The rule says
  only what a tool *writes into* is a bound field; a tool working on todo lists almost certainly
  toggles the checkbox and may never rewrite the label. `label` is bound because a fixture needs
  two named groups to prove the **gap between them** is computed — with one group only the
  head and tail spans are exercised. The CRC card records this, but it means the worked example
  a reader learns from demonstrates a binding its own rule would reject. Repair when a real
  consumer exists: either something writes labels, and the binding is justified, or the fixture
  grows a second genuinely-written field and `label` becomes glue.
  **Resolved 2026-09-03 (Bill):** the rule was wrong, not the binding. Binding serves access
  as well as editing — a DOM exists to make a file easier to read *and* to edit — so `label`
  is bound because a tool reads it. R115 retired to R223; the rule is rewritten at its source
  in `specs/stencils.md`. `TodoItem` moves to `stencil_test.go`: it is an example, not shipped
  code.
- [x] O15: Two guards now cover one property, and the inner one is unreachable. `mergeLocs`
  panics when two locations carry different origins; `New` panics when a document is built from
  nodes of two parses. Since `New` is the only way foreign nodes enter a document — `Split`
  inherits the origin, `Merge` refuses a mismatch, `Remove` takes nothing in — `Doc.Merge` can
  never present `mergeLocs` with a cross-origin pair. R95's test can only be built by corrupting
  a document from inside the package, and it says so. The inner guard is kept because
  `mergeLocs` guards a **function** rather than a path and could acquire a second caller, but
  this is the fourth guard added in three days and the density is worth watching: a codebase is
  a prompt, and a reader will take it as the local idiom. Revisit if a fifth appears, or if
  `mergeLocs` still has one caller when the readers land.
  **Resolved 2026-09-02 (Bill, Item 6.2):** the fifth guard appeared — `List.SetItems`, a
  refusing write rather than a panic — and the revisit went the other way: the unreachable
  inner guard in `mergeLocs` is gone, with R94 and R95 retired to R118. Unreachable code is
  cognitive load for a human and noise in a context for a model.
- [ ] O16: `minispec validate trajectory` fails on this repository, and has since before this
  session: *item numbers in no readable entry: #4 #5 #6*. Two halves of one tool defect, and
  only the first was recorded. The **minter** reads `#N` anywhere in the done ledger, so
  `part #7` inside #3's completion entry counted as an item that had been handed out, and the
  next mint jumped from #3 to #8 — leaving 4, 5 and 6 never issued. The **validator** then
  objects to the hole the minter made. Measured 2026-08-31: `query next-id item` reports
  `DONE.md 7` against four done entries, and the three numbers it names were never minted, so
  there is nothing to restore and the skill's usual repair for a numbering gap — put the
  missing entry back — does not apply. Like `O10`, the fix belongs in the mini-spec tool
  rather than here, so this records the hole and the evidence: the minter should count only the
  identifier a done entry owns, not every `#N` in its prose.
- T1: R69 retired by R121 (2026-08-31 carve Item 4: the shipped set grows to six with
  LangTypeScript and LangLua, and its selection rule gains a second job — serving the
  languages mini-spec reads, not only covering every BracketGroup field.)
- [ ] O17: `R129` says the package bundles declaration schemas for Go, TypeScript, JavaScript,
  Python, Lua and Shell. **Four exist** — `Go`, `Lua`, `Shell` and, since carve Item 5,
  `Python` — and there is still no `TypeScript` or `JavaScript` entry point. `validate` is green throughout, because `R129` has a design ref and
  an inline ref in `schema.go`: coverage is formally satisfied while the behaviour is absent,
  which is the failure mode requirement coverage cannot see. TypeScript and JavaScript are
  **not** a table swap over the Go schema: their keyword sets differ (`function`, `class`,
  `let`, `interface`, `enum`, and `export` prefixing any of them), so each needs its own pattern
  and its own `Lang`, even though the walk is Go's shape. Repairing this gap edits `R129` — it
  carries a back-link — because the requirement should either be met or should say what is
  bundled today and what is pending. It is recorded rather than rushed because the three that
  exist were chosen to cover the three distinct *shapes* (keyword in text, keyword as bracket
  marker, no keyword at all), and TS and JS add a fourth of nothing; but a library's contract is
  not sized by what its author found interesting, which is the lesson `O13` already records in
  its other form.
- [ ] O18: The declaration pass runs **one declaration per mutation window and re-scans the
  whole document each time**, which is O(n^2) in declarations × nodes. The reason is structural
  rather than lazy: targets must be resolved *before* entering a window, because navigation
  refuses inside one, and carving invalidates the node references a later hit in the same node
  would have needed. Measured 2026-08-31: the corpus test does 241 declarations over 24 files in
  about 0.07s, so nothing is felt yet. The alternative is threading `carve`'s right-remainder
  through hits that may span four nodes and two languages' walks, which is exactly the
  shared-driver machinery carve Item 9 defers until three schemas exist to generalize from.
  Measure before optimising; no consumer exists yet, and `O5` records the same judgement about
  `Doc.find`. **Second instance, 2026-09-01:** carve Item 5's Python schema has the identical
  shape for the identical reason, so this is a property of every declaration pass rather than of
  Go's — which is what carve Item 9's shared driver would fix once, if it fixes it at all.
- [ ] O19: `R149` — skipped whitespace may contain statement separators, so the walk crosses
  them — has **two halves living in two places**, and the test named for it covers only one.
  Go's `func` ⏎ `foo(x int)` resolves inside a single text node, so its newline is crossed by
  `goNameAfter`'s in-node prologue; the `skipForward` half is only reached when a
  whitespace-only node sits between comments, as in `func /* a */  /* b */` ⏎ `/* c */ foo`.
  Measured 2026-08-31 while pulling `test-DeclSchema.md#4`: a delegate found `isSpace` is
  invoked **zero times** during `TestADeclarationMaySpanLines`, so the test named for the
  property never exercises its main mechanism. Both halves are in fact covered — the second by
  `TestCommentsAnywhereInASignature` — so this is not a coverage hole; it is a naming and
  design one, and it was invisible until an injection was aimed at the site the alarm named.
  Worth repairing by splitting the test so each half is asserted where it lives.
- [ ] O20: **The alarm census cannot see a change made on the same day as the pull that preceded
  it.** It compares a `**Pulled:**` date against the date git last changed the `**Inject:**`
  symbol, and says so itself — *"compared by date until re-pulled"*. So a symbol pulled in the
  morning and rewritten in the afternoon reports **verified** while its proof is void. Measured
  2026-08-31: `test-Declaration.md#4` was pulled that morning against
  `BracketContext.Declarations`; the afternoon's index consolidation rewrote that function's
  body (`bc.declaration[kw]` → `bc.info[kw].declaration`, visible in commit `70f13e6`); the
  census still listed it among the verified, and only knowing what had been edited caught it.
  Three other alarms on the same rewrite *were* reported stale, because their previous pulls
  were dated 2026-08-30 — so the blind spot is exactly one day wide and opens whenever a pass
  pulls and then keeps working, which is the normal shape of a mini-spec item. The fix belongs
  in the mini-spec tool rather than here, and it is the same shape as the `unsealed` note the
  census already prints: a pull record that named its commit could be compared against a commit
  rather than a date, and the census would not need to guess. Like `O10` and `O16`, this records
  the hole and the evidence rather than proposing a repair in this repository.
- A1: `Separators` and `DeclarationNames` return the context's **own slice**, so a caller that
  appends to or writes through the returned value corrupts the index in place, and the
  corruption survives until a rebuild happens to overwrite it. This is `O2`'s problem in a
  second location — there it is `Doc.Nodes()` returning the live backing array with "must not
  be modified" stated only in prose — and the options are the same: `slices.Clone` at an
  allocation per call, or an `iter.Seq[Node]` which enforces it for free and changes the API.
  The aliasing predates this item for the declaration links and is new for the separators, and
  deciding it in one place for both is better than twice. **Approved 2026-09-02 (Bill, carve
  Item 6.1, R196):** the slices stay live. Every consumer of `sdom` is fire-and-forget — mini-spec
  is a CLI that builds a DOM, uses it and exits; microfts2 (and Ark through it) indexes and
  searches and caches no slices — so there is no lifetime in which aliasing is a hazard, and a
  copy per call buys nothing. Documented at the accessor. `O2` is the same question for
  `Doc.Nodes()` and the same reasoning applies.
- [ ] O22: The consolidated index costs about **2.3× the heap** of the three maps it replaced,
  and that was accepted deliberately rather than overlooked. *Measured 2026-08-31 over `sdom`'s
  own 14 files, 15,747 nodes:* three maps 1,067 KB against one map of flat structs 2,463 KB, and
  the struct has since grown from 72 to 96 bytes with the declaration links folded in, so the
  real figure is higher. What decided it was **allocations, not size** — 111 against the three
  maps' 275, and against 15,858 for a boxed `NodeInfo` hierarchy, on an index `rebuild`
  recreates at every structural change. The cost comes from mixing two kinds of fact: `opener`,
  `closer` and `separators` are properties of a **group**, of which there are 4,557, while
  `enclosing` is a property of **any node**, of which there are 8,965 — so every plain text
  node carries 96 bytes to hold one 16-byte field. If the heap ever matters, the levers in order
  are: key a group-shaped index by its opener and keep `enclosing` separate (≈1.6× rather
  than 2.3×), or move `separators` to a side map, since `LangGo` produces **zero** separators
  and pays 24 bytes an entry for them regardless. Measure before optimising; 1 MB of index for
  100 KB of source is not a constraint mini-spec or microfts2 will feel.
- [ ] O23: An alarm's `**Fire alarm:**` prose can go stale without the census noticing, because
  the census compares the `**Pulled:**` date against the date git last changed the `**Inject:**`
  symbol — it tracks whether the *code* moved, never whether the *prediction* is still true.
  Measured 2026-09-01 while re-pulling for an unrelated rename: `test-BracketContext.md#1` says
  the injection "is silent everywhere else", which was accurate when written on 2026-08-30 and
  became false the moment carve Item 4 landed a declaration pass that reads *top level* as
  `Enclosing(n) == nil`. An index where every opener encloses itself leaves nothing top-level,
  so the pass sees an empty document: the injection now fails **13 tests across two packages**
  instead of two in one, with `TestADeclarationPassIsAdditive` reporting 0 declarations over 24
  files against an independent count of 251. Nothing detected the drift — the alarm's own
  subject (`parser.open`) had not changed, so the census called it verified for two days, and it
  surfaced only because a rename forced a re-pull. This instance drifted in the **safe**
  direction, a prediction understating its blast radius. The opposite drift is equally invisible
  and far worse: an alarm whose injection has stopped reaching the property it names still reads
  as a passing proof. Like `O10`, `O16` and `O20`, the repair belongs in the mini-spec tool
  rather than in this repository, so this records the hole and the evidence. It is also the
  argument for re-pulling after a pure rename, which is otherwise hard to justify: the pull is
  what re-reads the prediction.
- [ ] O24: `BracketContext.refresh` and `IndentContext.refresh` are the same eight lines —
  read the document's generation, rebuild if the stamp is stale or nothing has been built, then
  re-stamp. Unifying them means an embedded `doc`/`built`/`stamp` struct with a rebuild hook,
  which is machinery to save eight lines and would restructure two types whose fields carry
  their own commentary. The two copies also hold *different* load-bearing prose: the bracket one
  is where the `R87` reasoning lives for why `built` is checked and not just the stamp. The
  simplification pass flagged this deliberately and left it, which is recorded here rather than
  in a commit message so the next reader meets the judgement rather than re-making it. Revisit
  if a third stamped context appears — two copies is a coincidence, three is a pattern.
- [ ] O25: `IndentParser.NodeType` has no caller inside the library, and the delegation path it
  exists for is exercised only by its own test. An `IndentParser` is always the root parser and
  the walk calls only `Parse` on that, so nothing nests one inside another parser that looks
  ahead before delegating. **This is not the `O13` mistake of pricing a library export by
  demand** — the method is required by the `Parser` interface and must exist. What is worth
  recording is narrower: *found by injecting past the alarm list on 2026-09-01, a `panic` as its
  first statement left all 124 tests green.* It now has a test and an alarm, but the shape it
  serves — one parser delegating to an `IndentParser` — has no worked instance, so the
  *composition* remains unproven even though the method does not. The first consumer that nests
  one will be the real test.
- [ ] O26: A list item containing the comment's own | separator passes the list
  guard.
`List.SetItems` guards against an item that would not read back as a list item — a
  comma or whitespace — but a `|` inside an item is legal to the list and is the **enclosing**
  `TraceabilityComment`'s field separator, so `c.CRC().SetItems([]string{"a|b"})` renders fine,
  round-trips its bytes, and reparses as two fields. The 2026-08-27 guarded-write decision says
  the guard re-parses *that field from its own render*; the field here is the list, and the list
  does not know `|`. The repair is at the comment: either the comment hands out lists whose
  write path re-parses the whole comment, or the item grammar the comment asks `ParseList` for
  excludes `|`. Left open deliberately (Item 6.2, 2026-09-02) rather than teaching the generic
  list a separator that belongs to one consumer.
- T2: R94 retired by R118 (2026-09-02 Item 6.2 (Bill): the inner of two guards was unreachable
  through the public API, since New refuses mixed parses; unreachable code is removed rather
  than kept.)
- T3: R95 retired (2026-09-02 Item 6.2 (Bill): the panic it described no longer exists; Mutate's
  re-raise of foreign panics is unchanged but nothing in sdom produces one on this path.)
- T4: R115 retired by R223 (2026-09-03 (Bill): binding serves access as well as editing — the
  point of a DOM is to make file access *and* editing easier — so 'only what a tool writes
  into' was the wrong test.)
