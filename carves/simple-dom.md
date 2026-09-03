# Carve: a simple DOM, and the mini-spec parser over it

Mini-spec reads and writes its own documents through scattered regexes and
hand-rolled scanners that disagree with each other. The repair is a **simple
DOM** — a parse that models only what the tool operates on, keeps every other
byte exactly where it was, and re-emits the source with nothing but the intended
change in it. This carve is that DOM, built clean, in **two packages**: a parsing
layer another tool could consume, and mini-spec's readers on top of it.

**Provenance.** Opened 2026-08-27. A throwaway prototype answered the feasibility
questions and is discarded; **no part of this carve is discharged by porting it.**
Everything a stranger needs is restated here rather than linked, because the
working notes live under a gitignored `.scratch/` — the case mini-spec's own
reference-discipline carve exists to prevent. Those notes are private and are
**not** the authority. This document is.

## Status

Ordered by intent; position is the priority and the number is only the
identifier.

- [x] ~~**Item 1 — the node protocol and the document.**~~ **LANDED (`77fa5f4`, 2026-08-30 — `#1`.)**
- [x] ~~**Item 2 — the bracket parser.**~~ **LANDED (`57fa312`, 2026-08-30 — `#2`.)**
- [x] ~~**Item 7 — provenance carries its origin.**~~ **LANDED (`baec358`, 2026-08-30 — `#3`.)**
- [x] ~~**Item 3 — regex compounds.**~~ **LANDED (`4b903a9`, 2026-08-31 — `#8`.)**
- [x] ~~**Item 4 — declarations, as a post-pass.**~~ **LANDED (`df7a987`, 2026-08-31 — `#9`.)**
- [x] ~~**Item 10 — one bracket index.**~~ **LANDED (`efef190`, 2026-08-31 — `#11`.)**
- [x] ~~**Item 11 — separator links, completing the bracket contract.**~~ **LANDED (`efef190`, 2026-08-31 — `#11`.)**
- [x] ~~**Item 12 — the vocabulary, second pass: it is a parse, not a scan.**~~ **LANDED (`f9008c6`, 2026-09-01 — `#12`.)**
- [x] ~~**Item 5 — indent scope.**~~ **LANDED (`fe5e90e`, 2026-09-02 — `#13`.)**
- [ ] **Item 6 — the traceability reader.** **OPEN (not queued.)**
  - [ ] **6.1 — `BracketContext` accessors: typed names, opener/closer, inner/outer text.** **OPEN (#14.)**
  - [ ] **6.2 — the traceability reader: list compound, `CommentStyle`, `minispecsdom`.** **OPEN (#15.)**
- [ ] **Item 9 — generalize the schema work into `sdom` tools.** **OPEN (not queued.)**
- [x] ~~**Item 8 — the vocabulary: it is a parser, not a lexer.**~~ **LANDED (`145ee96`, 2026-08-31 — `#10`.)**

## Decisions

**DECIDED (Bill, 2026-08-27): two packages.** `sdom` carries the protocol and the
bracket and indent parsers; `minispecsdom` carries mini-spec's readers. The
split is so microfts2 can consume the parsing half — which means **nothing below
the reader layer may know what a CRC card is**, and the boundary is enforced by
the compiler rather than by discipline. The export surface this forces
(constructors and accessors for what is currently package-internal) is part of
the work, not an obstacle to it.

**Module `github.com/zot/simple-dom`** (Bill, 2026-08-27). Package `sdom` lives in
`sdom/`, leaving the module root for the reader package and a command later.

**DECIDED (Bill, 2026-08-27): a bracket group is NOT a node.** An opener,
everything between it and its closer, and the closer are **siblings in the flat
array**. The parser keeps the nesting on a stack; the emitted stream has none.

Nesting is expressed by **links**, owned by the **schema's context** and not by
`Doc` — not every document has brackets, and a markdown DOM would carry two dead
maps forever:

- an opener knows its **closer** and its **enclosing opener**
- a closer knows its **opener**
- any other node knows its enclosing opener — and every one of these links is
  derivable from the flat array alone, so a consumer walking it reaches the same
  answers, which is what makes the index checkable rather than merely believed

The context therefore **outlives the parse**: it carries the language, the
parser's stack during the parse, and these links afterwards.

**Compound nodes exist only for stenciling** — a region a tool writes *into*, such
as a traceability comment, one of its fields, or a declaration. **Never for
bracket structure.** Modelling groups as compounds makes every span query a
traversal and every edit a re-parent, and the array is flat precisely because
searching and splicing are the operations this DOM exists for. If a design pass
produces group nodes with children, it has gone wrong here.

**DECIDED (Bill, 2026-08-27): a layer's derived index is stamped, not
registered.** `Doc` exposes a monotonic **structural generation**, bumped whenever
node membership changes. Any layer holding derived state — the bracket links, a
scope relation, an anchor index — stamps itself with that generation and rebuilds
when the stamp goes stale.

So `Doc` needs no registry of indices, no invalidation callbacks, and **no
knowledge of what exists above it**: the layers pull, `Doc` does not push. A
schema that needs no derived state carries none. A *content* edit does not bump
the generation, because node membership is unchanged and an index over structure
survives one.

**And reading the generation refuses inside the mutation window**, with the same
typed sentinel the core indices use. That is what makes the guard reach a
*layer's* index **without the layer knowing the guard exists**, since checking
freshness is the one call every stamped index must make. Poisoning the comparison
instead — returning a value no stamp can match — would let the layer rebuild from
a half-edited tree, which is a different wrong answer rather than a refusal.

**DECIDED (Bill, 2026-08-27): the readers are post-passes over the bracket
parse's output, never a re-parse of raw text.** A reader that re-parses bytes can
*consume nodes the bracket parse already produced*: a declaration regex matching
`func (s *Store) Index` from text swallows the receiver's parens, so the same
file parsed with and without readers has different bracket structure. Grouping
existing nodes is **purely additive**: the flattened node array is identical
with and without the readers, and only the top-level stream differs — which is
what nesting means, and is *not* the same as a group having children.

*Vocabulary corrected 2026-08-31; the claim is unchanged.* This said "the lexer's
output", "tokens the lexer already produced" and "the flattened token stream". The
bracket layer is a **parser** — it matches nested structure well beyond tokens —
and there are no tokens here, only nodes. The words had imported the
lexer-then-parser model, and that cost a session's design work before it was
caught. The **names** moved with Item 8 the same day.

**DECIDED (Bill, 2026-08-27): correctness and enforcement are different things.**
An invariant can be correct without a mechanism guarding it. Total enforcement in
a non-trivial system ends up requiring case logic as a substitute for dynamic
dispatch, and the escape hatch back to openness breaks the guarantee anyway — so
the guarantee was never total, only total within a region, with the seam
unmarked. Enforcement earns its place only where all three hold: the invariant is
easy to violate **silently**, the violation is **expensive**, and the guard is
**cheap and local**. Otherwise state the rule and move on. This governs every part
below and is the reason several plausible mechanisms are absent from them.

**DECIDED (Bill, 2026-08-27): a codebase is a prompt.** A mechanism added once is
read as the local idiom and reproduced, so the most expensive complexity is the
kind that looks principled — a hack gets deleted, a pattern gets propagated.

**DECIDED (Bill, 2026-08-30): a node's children are exactly what falls inside its
own span.** One rule for brackets and stencils alike. A bracket opener's span is
the opener bytes; the group's contents are **siblings**; the closer ends the
region. A stencil's span is its own marked-up text; what it introduces but does
not cover is siblings, and the region is delimited by marker nodes — either the
next typed node, or an explicit **zero-length marker** for a region with no
natural successor.

It forecloses a rendering bug rather than guarding against one: if a stencil's
span covered a region whose content also sat in the document list, those bytes
would be emitted twice. Under this rule every byte has exactly one owner and the
array tiles.

**And the span rule does not choose the span — editability does.** Only what a
tool *writes into* is a stenciled field, and the span is the narrowest one
covering those fields. Note what "minimum" constrains: inside a span, tiling
**forces** children to cover every byte, glue included, so minimality is a
constraint on the span's **width**, never on the child count within it. When
stenciled parts are far apart, make **several stencil nodes** rather than one wide
span that drags the intervening text in as children. The test to apply to any
compound: *would a tool ever write into this child?* If not, it should have been a
sibling and the span was too wide.

**DECIDED (Bill, 2026-08-30): a bound value is derived from its text, never
stored.** A stenciled field is a typed value over a `Text` node — a checkbox, a
gap name (which manages both its kind and its number), a requirement number, a
requirement list. The text is the **storage**; the literal and the typed value are
two views of it, which makes "dual-ported" exact in the hardware sense: one
storage, two access ports.

This generalises Item 6's rule — *field keys and requirement lists are derived
from the literals, never stored* — to **every** field, and what it buys is the
absence of an invalidation protocol: no second copy, no staleness, no question of
which representation is authoritative.

**The lineage is TCL, and we are choosing its earlier half.** Pre-8.0 Tcl was
pure strings — everything re-parsed at every reference, no internal
representation and so nothing to invalidate. `Tcl_Obj`, with a typed internal rep
cached beside the string and each invalidating the other, arrived with the
bytecode compiler in 8.0 (1997), because by then that re-parsing was the
bottleneck. We do not have that constraint. **The absent cache is a decision, not
an oversight** — and if a profile ever says otherwise, adding one is a local
change behind the same two accessors.

Its consequence reaches the protocol: no node kind holds state its children do not
carry, so `Equals` is uniformly *assert the type, compare children*. The
local-`Equals` escape hatch stays in the rule for a future kind that needs one, and
nothing in this design does.

**DECIDED (Bill, 2026-08-30): an edit reformats only what it touched, and fields
segment lazily.** *Narrowed 2026-09-02 by Item 6.2a: a field's only write is whole-list
`SetItems`, so a field is one `Text` and there is no lazy segmentation to do. The
granularity rule still holds between fields — editing the refs never reformats the
CRC list.* The rule is not *never normalise*; it is **never normalise what
you did not touch**. `R5-8` stays `R5-8` until something edits it — then
reformatting the edited part is fine and the unedited parts stay byte-intact.
~~Adding `R12` to `R5-R7, R10` must not flatten the range on its way past.~~
**Reversed 2026-09-02 (Bill):** a list write re-emits the whole list canonically, so
that becomes `R5-7, R10, R12`. Preserving unedited items inside an edited field is
too fancy here; unedited *fields* still stay byte-exact.

That is the granularity rule one level inside a field. A field edited as a whole
can be one `Text`; a field whose parts are edited independently must be a compound
over its **segments**. **Lazily**: the field holds one `Text`, reading derives the
value and creates no nodes, and segment nodes come into existence only because
something is about to **write** — at which point `Split` puts a boundary where the
edit needs one. The common case, a field nobody edits, costs one node.

The property is already tested. After such an edit the edited segment is altered,
its siblings still report `Faithful()`, and the enclosing compound is altered
because a child is — which is the **one-field delta**, written before anyone
noticed this is what it was for.

*Worked examples of all three live in `.scratch/PARSING.md`, which is a working
note. The calls are here.*

**DECIDED (Bill, 2026-08-30): `Loc` carries an `Origin`, and merging across two
of them panics.** Provenance is currently document-relative and therefore not
globally meaningful: node offsets are relative to a document's own source, `base`
never enters a location, and two nodes at offset 10 from different parses are
indistinguishable. `Loc` gains a reference to the parse it came from.

```go
// Origin identifies one PARSE, not one file. Two parses of the same source are
// two Origins, which is what makes "same file, different parser" answerable.
type Origin struct {
    Name string // a path, a URL, or whatever the caller finds useful
}
```

**The parse context, not the `Doc`** — and not merely because context identity is
the question being asked. The context **exists before the nodes do**: a parse mints
it, runs, and only then builds the `Doc` from what it emitted. Holding a `*Doc`
would need a back-patching pass over every node once the document existed.

**Concrete, not an interface.** Each schema's context is its own concrete type, so
there is no single `Ctx` for `Loc` to hold; a small core token that every parser
mints and its context keeps is what makes this work without an abstraction. It
doubles as a lookup key. It carries a field rather than being an empty marker
because Go may give every zero-size allocation the same address, and an empty
`Origin` would fail as an identity exactly where it was needed.

**This does not reopen the group-pointer ban.** A marker holding its group was
refused because `Equals` would compare it, so two documents parsed with
independently built languages would never be equal. A reference inside `Loc`
cannot do that: `Equals` never compares `Location` at all.

**Set by chaining, so no existing call site changes**: `Source(pos, n)` stays as
it is for the many places with no origin to give, and a parser writes
`Source(pos, n).In(origin)`.

**`mergeLocs` panics when the two origins differ.** Merging across parses is a
programming error rather than a data condition — the offsets are in different
coordinate systems and no result names anything true. It is a foreign panic, so
`Mutate` re-raises it with its stack rather than converting it, which is right for
a caller error. It also subsumes the cross-document adjacency false positive:
`Merge` never reaches the adjacency arithmetic. **A nil origin is unknown rather
than different, and compatible with anything** — settled by Item 7 as it landed
(2026-08-30), and migrated to where it belongs: `specs/location.md`, `R93`, and the
guard in `sdom/loc.go:mergeLocs`.

## Item 1

The protocol every other part stands on. **LANDED — the specification now lives
where a stranger can find it**, and this section keeps only the decisions and why
they were made:

- [specs/node-protocol.md](../specs/node-protocol.md) — `Node`, its four methods,
  the two that take no context, per-kind `Equals`, and the kinds `Text` and
  `Compound`.
- [specs/location.md](../specs/location.md) — `Loc`, provenance separated from
  faithfulness, and `Split` / `Merge`.
- [specs/document.md](../specs/document.md) — `Doc`, the flat array, the
  structural generation, and the mutation window.
- `design/` carries the six CRC cards, two sequence files and four test designs;
  `sdom/` carries the code, anchored to them by `// CRC:` comments.

**Two sentences that stood here were wrong by the time the item landed**, and are
recorded rather than deleted because a reader of Items 2–6 may remember them:

- *"`Loc` is `{Offset, Length int; Altered bool}`"* — it is accessors over an
  unexported offset **stored biased by one**, which is what makes the zero value
  mean absence rather than offset 0. There are no exported fields; a biased field
  would leak the bias.
- *"Edits resolve outside the mutation window and apply as one rebuild of `dom`"* —
  this read as a queued plan, and there is none. It means: **resolve your targets
  before you enter**, where navigation is still legal, then edit **directly**. The
  "one rebuild" is the derived indices, rebuilt once at the exit. "Ops keyed by
  node identity, never by index" survives intact, and is about what stays valid
  across that boundary.

**DECIDED (Bill, 2026-08-27): Item 1 lands `Text` and `Compound`.** Settled by its
own criterion — the tests cannot be written against `Text` alone. The one-field
delta requires *a node and its ancestors* to lose faithfulness, and `Text` has no
ancestors, so the derived-`Altered` decision would ship untested and `Kids()`
would never be exercised by anything.

`Compound` is not a test fixture. It is a node whose children **tile its span**,
rendering by concatenation, summing their extents and propagating their
alteration — and doing no parsing. That is exactly what remains of Item 3's regex
compound, Item 4's declaration and Item 6's traceability comment once you subtract
*how each computes its children*, so it is defined once here and embedded by all
three. It is not a bracket group and must never be used or named as one.

**Every concrete kind declares its own `Equals`**, minimum body a type check that
delegates: `o, ok := other.(*T); return ok && t.Compound.Equals(&o.Compound)`. Go
embedding promotes without dispatching, so a promoted `Equals` cannot see the
outer type — but **forgetting to declare one is loud rather than silent**: the
inherited assertion fails against the new kind, two identical nodes of it compare
unequal, and the structural round-trip goes red on the first document containing
one.

### Item 1's decisions

**DECIDED (Bill, 2026-08-27): the offset is stored biased by one**, so absence is
the zero value. A sentinel fails because Go's zero struct then reads as *offset
0, the first byte of the file* — a plausible wrong answer that no constructor is
forced to correct.

**DECIDED (Bill, 2026-08-27): an altered node keeps its offset.** Clearing it
discards provenance a diagnostic still wants, and forces a deferred walk to
invalidate ancestors — which carries a quiet defect, because a fold that
short-circuits leaves later composites claiming spans they can no longer honour
while the root's answer stays correct. Deriving `Altered` on read removes the walk
entirely: a composite is altered if any child is, and the walk must visit every
child to sum lengths anyway, so nothing can be skipped. *Caveat to carry:*
for an altered node `Offset` is historical while `Length` is current, so the pair
names a span the node never owned; a diagnostic uses `Offset` alone unless
`Faithful()`.

**DECIDED (Bill, 2026-08-27): `Render` and `Equals` take no context.** Denying
`Render` the document is what stops it satisfying the round-trip with bytes it
already has. `Equals` never compares `Location`: two nodes read from different
files with identical content are equal, and including provenance makes the
structural round-trip false for *every* mutated node.

**DECIDED (Bill, 2026-08-27): each schema has its own parse context, and it is a
concrete type rather than an interface.** It carries parse-time-only state and
each schema shapes it for its own needs — the items in a carve, the list of gaps —
which is why it belongs in `Parse` and matters more the more document types there
are. Counting its call sites measures the wrong thing.

An interface here would exist only to let one signature serve heterogeneous node
kinds, which nothing requires once `Parse` is concrete: each schema's method takes
its own context type directly, and generic core code that needs only the document
takes `*Doc`. **Watch for the shape that says an abstraction is unwanted** — code
that accepts an interface and immediately type-asserts back to the concrete type
is buying nothing.

**DECIDED (Bill, 2026-08-27): navigating an index inside the mutation window
panics, via a typed sentinel `Mutate` converts to an error.** Rebuilding the index
does not rescue it: if the node you hold was deleted, `IndexOf` returns −1 and
`Next` hands back nil, which is indistinguishable from end-of-document. A
foreign panic is re-raised rather than swallowed, so real bugs keep their stack.

**DECIDED (Bill, 2026-08-27): an error *or* a panic escaping the mutation
function poisons the document.** Both mean something the caller could not or did
not handle got out, so what it had done by then is unknown. A caller that *can*
handle them does so inside the function rather than propagating. There is no
reset — recovery is to re-parse, because a poisoned document means a bug in the
code that wrote to it.

**DECIDED (Bill, 2026-08-27): `Mutate` saves and restores rather than counting**,
so the nested call is a pass-through, and it has **no rollback**. Partial rollback
would be worse than none: restoring the node array would not undo content written
through node doors, producing a plausible wrong state rather than an obvious one.

**DECIDED (Bill, 2026-08-27): edits are direct, and navigation is not allowed
inside a mutation.** A content edit writes the node; nothing is queued. The second
rule this fork was weighing — *when does your own edit become readable* — does not
arise, because inside the window you cannot navigate to read anything.

**The navigation guard is what makes the document never observably half-edited.**
Deferring edits would be a second mechanism for the same property, and a plan with
queue-time validation and rollback is a transaction engine inside a document model
whose whole point is being *simple*. Item 6's guarded write therefore keeps its own
rollback, and that `DECIDED` stands unsuperseded.

**DECIDED (Bill, 2026-08-27): `Parse` is not on the interface.** During parsing
you construct a *concrete* node and then send `Parse` to it, so the interface is
not involved at that point. Only the compounds that genuinely parse from text keep
the method — here, the traceability comment and its fields.

Its consequence is worth more than the method: **it removed the reason for a
whole abstraction.** `Node` becomes purely about document structure, parsing is
not part of the contract, and the parse context stops needing to be an interface
at all — see the decision above. The two were coupled and nothing said so.

**DECIDED (Bill, 2026-08-27): `Merge` and `Split` are `Doc` methods because they
change node membership, and `Merge` checks adjacency by offset arithmetic when
both operands are faithful and not at all otherwise.** The arithmetic is free and
catches the ordinary mistake at the call site. `d.Next(a) == b` is not the basis:
navigation panics inside the mutation window, so it cannot run at the call, and
running it when the plan resolves would mean deciding adjacency against a `dom`
the other queued ops are about to change. Following adjacency through mutation is
unjustifiably expensive and baroque for what it buys, so past that point the rule
is stated and left unenforced. `Split` needs no adjacency at all.

**DECIDED (Bill, 2026-08-27): a merged node is faithful only if both operands are,
and takes the first operand's offset when it has one and the second's otherwise.**
Keying the rule on *unfaithful* rather than on *altered* also catches an operand
with no provenance at all, which would otherwise merge into a node claiming
faithfulness at an offset whose bytes it does not render. Leftmost-provenance-wins
makes the offset rule associative, so merging a run of nodes gives the same answer
however the merges are grouped.

**A standing note for every part.** Four methods have already left this interface,
each added for a reason that was real when written and quietly stopped being true,
with nothing flagging any of them. **Expect it to happen again and check for it
rather than defending what is there** — and after removing something, ask what was
only there because of it, since a removal cascades and nothing announces that
either.

## Item 2

A **table-driven** bracket parser. A string is not a special case: it is a bracket
group with parsing turned off, which is why interpolation, word brackets and
shell `if`/`then`/`fi` all fall out of one mechanism.

```go
type BracketGroup struct {
    Open, Separators, Close []string
    Escape                  string
    AllowedInner            []string // nil = code mode; non-nil = parse-restricted
    AllowedParent           []string // nil = anywhere
}
```

`AllowedInner` non-nil (**even empty**) means only this group's `Close`, its
`Escape`, and the listed openers are recognized inside; everything else is
literal. `AllowedParent` is its dual and is **not optional**: with a flat bracket
table, code mode recognizes every group's openers, so without it `${` fires at top
level where it is really a `$` followed by a `{`.

Also required: **word-boundary matching** for alphanumeric markers, so `do` does
not fire inside `download` nor `fi` inside `file`; `Separators` for mid-group
markers; an **any-close fallback** so a stray `}` lands as a bracket rather than
derailing the parse; and a parse that **always consumes at least one byte**, so
nothing stalls on input it does not understand.

Output is **flat and document-order** — see the group-is-not-a-node decision
above, which this part is the main consumer of. The parser keeps the nesting on a
stack and emits `Open`, `Close` and `Sep` as ordinary siblings; the pairing that
recovers the structure is a derived index, not a node graph.

### Item 2's decisions

**DECIDED (Bill, 2026-08-27): a bracket marker node holds the bytes it matched
and no pointer to its group.** Holding the group makes `Equals` compare pointers,
so two documents parsed with independently constructed languages are never equal
and the structural round-trip fails on every bracketed document. The active group
comes from the parse context instead.

**DECIDED (Bill, 2026-08-27): `IndentLang` embeds `BracketLang` rather than
replacing it, and the type is the flag.** There is no `IndentScope` boolean, so a
brace language carries no indent parameters at all rather than meaningless zeroed
ones. See Item 5 for why the two compose.

**Prior art, and it is close.** `~/work/microfts2/specs/chunk-bracket.md` and
`bracket_chunker.go` implement this design already, including the shell and
Pascal word-bracket configs. Read them before writing: that design is the target,
not a starting point to improve on.

## Item 7

`Loc` gains a reference to the parse it came from. **Added 2026-08-30**, after
Item 2 — a part discovered rather than planned, which is why its number runs out
of sequence.

**Discharges gap O9.** Provenance is currently document-relative and therefore
not globally meaningful: node offsets are relative to a document's own source,
`base` never enters a location, and two nodes at offset 10 from different parses
are indistinguishable. Two documents already assume otherwise — Item 1's test 4
wording above, and the `nest()` fixture, which bakes base into offsets by hand so
its test passes for a reason unrelated to how a parse assigns them.

The design is settled: see **the `Origin` decision** in `## Decisions`. What is
left for the item is the pass — spec, requirement, design, code, alarms — plus
three repairs the gap names: state in `specs/location.md` that offsets are
relative to the document's own source, retire R38's back-link when a real
requirement replaces it, and rewrite `nest()` to stop faking a shift it never had.

The one question left open when this item was written — whether a **nil** origin
counts as different or merely absent — was settled by the item itself: **unknown
rather than different, and compatible with anything.** It now lives in
`specs/location.md`, `R93`, and `sdom/loc.go:mergeLocs`.

## Item 3

Compound nodes parse by regex. ~~and the regex checks itself.~~ — **superseded
2026-08-30: there is no check, because the failure it caught is now
unrepresentable. See the computed-glue decision below.**

~~**Every byte of a match must land in an outer capture group.**~~ A regex that
leaves bytes uncaptured would drop them from the render silently — `^- \[([ xX])\]
(.*)$` looks reasonable and eats four bytes — which is exactly the failure the
computed glue dissolves.

Indices are **half-open**: contiguity is `next.start == prev.end`, with no `+1`
anywhere. A zero-length group is `[k,k)` and needs no special case, which is what
settles the convention — and a zero-length node is *useful*, being an insertion
point a tool can later write into without moving its neighbours.

**Nested groups are skipped, not rejected.** They cannot double-count and they
produce nothing. **Binding is by group name, never by position**: with alternation
the branches have different group counts, so a fixed index is right for one input
and out of range for another.

### Item 3's decisions

**DECIDED (Bill, 2026-08-27): the parse-time check is the outer-group index
tiling, not a render comparison.** ~~Once the tiling holds, concatenating the
groups re-proves what was just established.~~ **Superseded 2026-08-30 — there is
no tiling check any more.** The half that survives is why a render comparison was
rejected: it is a *different* claim, catching a node that renders something other
than what it captured, and it belongs in the corpus round-trip where it already is.

**DECIDED (Bill, 2026-08-30): the glue is computed, not required.** A stencil's
regex names groups only for the fields it **binds**. The machinery synthesizes
`Text` nodes for the head of the match, the tail, and every gap between
consecutive groups, then pushes the whole ordered run into the stencil's child
list.

Tiling then holds **by construction**, which *dissolves* the problem the check
above existed to catch rather than detecting it. A regex cannot eat bytes, because
whatever it does not capture becomes `Text`. So the requirement that every byte
land in an outer capture group is **lifted**, the parse-time tiling check is
**removed**, and this part gets smaller.

Unchanged: binding is **by group name, never by position**, since with alternation
the branches have different group counts and a fixed index is right for one input
and out of range for another. Nested groups are skipped. Indices stay half-open.

## Item 4

Declarations are addressable so a tool can ask which carry no traceability
comment, and can insert one above a declaration that does not.

~~A declaration node models the **keyword through the name** and nothing more.~~
~~**It is built by grouping nodes the lexer produced** — splitting text nodes at
the declaration's edges and nesting the span.~~ — **superseded 2026-08-31: there is
no declaration node and no span.** What survives is the framing: this is not a
language parser; it is a way to *address* declarations.

### The design (Bill, 2026-08-31)

**DECIDED: declarations post-process a bracket-parsed or indent-parsed document.**
You parse a code file, get its basic structure, and find the declarations in that.
The 2026-08-27 post-pass call stands; the shape it produces is what changed.

~~**DECIDED: recognition is forward to a brace, then backwards over structure.**~~
**Superseded within the day (Bill, 2026-08-31), and it never survived contact with
a type.** Walking back from a brace to the nearest identifier gives `Index` for
`func (s *Store) Index(…) int {` and gives `struct` for `type Foo struct {` and
`BracketLang` for `var LangGo = BracketLang{`. It also missed three of this
repository's own anchored declarations outright, `var X = f(...)` having no brace
at all.

**DECIDED (Bill, 2026-08-31): recognition is a keyword at the start of a
statement, matched over the top-level text nodes.** In a whitespace-insensitive
language a declaration is announced by its keyword, and each language brings its
own keywords and its own statement separators — Go and TypeScript separate on
`\n` and `;`. From the keyword the pass searches **forward** for the name: within
the same text for `var`, `type` and a plain function, and into the **forward
siblings** for a method, whose receiver group sits between the keyword and the
name.

```go
`(?:^|[\n;])[ \t]*(?P<kw>func|var|type|const)\b`
```

**`\b`, and a consuming prefix, are forced rather than preferred.** Go's RE2
supports neither lookbehind nor lookahead — *verified 2026-08-31: `(?<=[\n;]\s*)`
and `(?=\s)` are both rejected at compile* — so the separator is a non-capturing
group and the trailing boundary is `\b`. That is also the better rule: `\b`
rejects `constant` while admitting a keyword followed by punctuation.

**Comments cost nothing, because the parse already excluded them.** A comment's
interior is inside its group, so it is not a top-level text node and commented-out
code cannot match.

**But the comment's closing newline is a `Closer`, not a byte in the text** —
which is the one hazard here, and it is the common case rather than an edge.
*Verified 2026-08-31 on `// CRC: …\nfunc Index(k string) int {`:*

```
2  *sdom.Closer  top=true   "\n"          ← the comment's Close marker
3  *sdom.Text    top=true   "func Index"  ← starts with func; no \n in front of it
```

Nearly all 183 declarations in `sdom/` carrying a `// CRC:` comment sit in exactly
that position, so a purely in-text separator would miss most of them. **Statement
separation is therefore partly structural:** a match at the start of a top-level
text node begins a statement when the **preceding top-level content ends with a
separator**. The in-text `[\n;]` branch handles the rest, since `}\nfunc b` keeps
its newline in the text.

**And "preceding content" means after skipping — comments go anywhere a space
goes** (Bill, 2026-08-31), interleaved with whitespace-only text in any number. A
schema walking the array steps over **whole comment groups and whitespace-only
text**, both **backward** when testing a statement start and **forward** when
finding a name. *Verified 2026-08-31, and each direction has a case that fails
without it:*

```
/* c */ func Foo() {
[ Opener "/*", Text " c ", Closer "*/", Text " func Foo", … ]
```

The pattern matches `func`; the backward test then finds `*/`, sees no separator,
and **rejects a real declaration** — until the comment group is skipped, leaving
the document start.

```
func /* hi */ Index(a) int {
[ Text "func ", Opener "/*", Text " hi ", Closer "*/", Text " Index", … ]
```

The name is three nodes past the keyword, and `func /* a */ /* b */ Index` puts it
arbitrarily further. Skipping a comment backward is one hop, since a closer names
its opener.

*Which groups are comments is the language layer's to know*, as recognition always
is. A comment and a string are the same shape with different markers, so no property
of a group could answer it — which is why Item 5 made it a **label**: the table marks
its comment groups with a `Kind` and a schema reads that back. ~~so a schema matches an
opener against the markers it knows~~ — superseded 2026-09-01, when a second consumer
appeared and both were re-deriving by string-match what the table can state once.

**And the skip runs before every decision in the walk, not once at its start**
(Bill, 2026-08-31). `func /* a */ foo /* b */ (…) /* c */ {` puts a comment at each
position, and the method form adds one more decision to skip before:

```
skip*                          comments and whitespace-only nodes
if the next node is an Opener  it is a receiver group — jump to its Closer
skip*                          again
the next text with content     the name is the identifier inside it
```

**Two different whitespace problems, and only one is skipping.** A whitespace-only
node is stepped over; but the name arrives as `Text " Index "`, carrying whitespace
on **both sides**, so it is sliced out of the middle by splitting that node at both
edges. Taking the node whole would put the spaces inside the `DeclarationName` —
which renders identically and is wrong, the class of failure this project keeps
finding.

**And skipped whitespace can contain a separator** (Bill, 2026-08-31). *Verified:*
`func /* a */  /* b */\n/* c */ foo …` yields `Text "   \n"` between two comments —
whitespace-only, and a newline. A declaration may span lines, so the forward walk
**cannot stop at a separator**, while the backward test uses one to decide a
statement began. Not a contradiction: one asks *did a statement begin here*, the
other *where is the name of the declaration already announced*.

The consequence is that the forward walk has **no terminator** — harmless on valid
input, where a keyword is always followed by its name, and unbounded on truncated
input. Stated rather than guarded, per the enforcement rule at the top of this
document.

**Newline-insensitivity is required rather than tolerated**, and one case settles
it. *Verified 2026-08-31 against both compilers:* `func` ⏎⏎ `foo` ⏎ ` (x int)` ⏎ `{`
is **rejected by Go** — `expected '(', found newline`, a semicolon being inserted
after the identifier `foo` — and the same layout in Lua **runs and returns its
value**. So a shared rule stopping the walk at a separator would be wrong for both
languages at once; and even in Go, `func` ⏎ `foo(x int) {` is legal and vets clean.

**And how strict the walk is, is the schema's choice** (Bill, 2026-08-31) — each
schema does its own, so the general walk above is the *loosest* one permitted
rather than the one everybody runs. **Go's requires a `func`'s name to reach its
opening paren without crossing a newline**, which is what semicolon insertion makes
true, and which lets the Go schema reject the shape Go rejects. It binds `func`
alone. Lua's schema imposes no such rule, because Lua has none.

*This was first written as **abutment**, and the implementation caught it.* A pass
built to that rule rejected `func /* a */ (s *S) /* b */ Index /* c */ (x int) {`,
which is legal Go. Measured against the compiler afterwards: `func foo (x int) {`
and `func bar /* c */ (x int) {` are legal; `func baz` ⏎ `(x int)` is not; and
neither is `func qux /*` ⏎ `*/ (x int)`, Go treating a comment that carries a
newline as one. So the test is over the **source span** between name and paren,
where comment bytes are in it by construction — which makes the last case fall out
rather than need a rule. The stricter rule had been generalized from a single
failing input.

`sdom` itself neither validates nor requires laxity — it only promises never to
refuse text on a schema's behalf, a DOM that rejected malformed input being useless
for editing. **Nothing in `sdom` is a syntax checker; whether a schema is, is up to
it.**

**DECIDED: the product is two narrow typed nodes, not one wide compound.** Slice
the keyword out and replace it with a `DeclarationType`; slice the name out and
replace it with a `DeclarationName`. `DeclarationType` knows how to find its name.
Everything between them — receiver group, parameter group, return type — stays
exactly where it is in the document array.

Three consequences, each of which had been a live worry:

- **The additive property holds by construction.** Nothing consumes a byte the
  bracket parse already claimed: two text nodes are split, and two halves change
  kind. The flattened array is identical with and without the pass, so the
  recognition count cannot drop either.
- **This is what the minimality rule already said.** Two writable fields separated
  by a run of structure want two narrow nodes with that run as an ordinary
  sibling — not one span that drags a receiver group in as children.
- **`StencilBuilder` is not involved, but its idea is.** No compound is built and
  no glue is computed, so the builder itself has no part here — yet recognition is
  a regex over text and the keyword is located by its **named group**, which is
  Item 3's binding-by-name reused at a different layer. ~~never bound from a regex~~
  — *corrected 2026-08-31, when the brace walk gave way to a keyword match.*

**The one new primitive is `Doc.Replace(old, new Node) error`**, a sibling of
`Split` / `Merge` / `Remove`. `Split` already yields two `*Text`; turning a half
into a typed node is the only structural verb missing.

**DECIDED (Bill, 2026-08-31): `const` is in, and grouped declarations yield several
names.** `const`, `var` and `type` may take a group, and one group declares many:

```go
const (
    A = 1
    B, C = 2, 3
)
```

*Verified 2026-08-31:* the parse puts the group's whole interior in **one text node
inside the parens**, so the names come from a second pass over it — an identifier
list at the start of a line within the group. `import` stays out: it is not a
declaration a tool anchors.

**A name list is captured whole and split, never matched with a repeated group.**
*Verified the same day:* a repeated capture reports only its last iteration, so
`B, C, D` yields `B` and `D` and loses `C` outright. One group for the list, and
the identifiers come out of it — which covers `A = 1`, `B, C = 2, 3` and
`D, E int` alike.

**DECIDED (Bill, 2026-08-31): the declaration links live in the schema's parse
context, and the relation is one-to-many.** `DeclarationType` does **not** hold its
names — no node kind holds state its children do not carry, the rule that already
keeps a bracket marker from holding its group. The links go where
`BracketContext`'s `closerOf` / `openerOf` / `enclosing` go: a map on the schema's
context, stamped against the document's structural generation and rebuilt when the
stamp is stale, so `Doc` keeps no registry and issues no invalidation. Reading one
refuses inside a mutation window for free, because checking freshness is the one
call every stamped index makes.

**One-to-many is forced by the group form**, and it is worth stating as the reason
rather than as a shape: `map[Node][]Node` from a keyword to every name it declares
— one entry for a plain declaration, several for a group.

~~**Indentation is captured, not required to be empty.** The indent capture must
be `[ \t]*` and never `\s*`.~~ — **superseded 2026-08-31: there is no capture,
because there is no regex.** What survives is the observation that motivated it:
Go's declarations sit at column 0 and Python's, Java's and Smalltalk's do not, so
anything keyed on column 0 finds `class Widget` and misses every method in it.
That now bears on the depth question below rather than on a capture group.

### Item 4's decisions

**DECIDED (Bill, 2026-08-27): only the fields a tool *writes* are nodes.** ~~The
indent and the name are nodes; the keyword and the receiver are read-only
derivations over the render.~~ *Editability sets the granularity* used as a budget
rather than a maximum.

**Partly superseded 2026-08-31, and the remainder is a real question.** The keyword
*is* a node now — `DeclarationType` — and it is not a field any tool writes into.
So either the rule reads "only what a tool writes is a **field**", with an *anchor*
being a different thing that may also be a node, or the rule takes an exception.
Worth settling in words before the spec is written, because the same distinction
decides the indent.

~~**DECIDED (Bill, 2026-08-31): this part lives in `sdom`, on a `DeclLang`
layer.**~~ **Superseded the same day — see the decision below.** The reasoning is
kept because it was measured, and because the measurement stays true: microfts2's
`specs/`, `design/`, `notes.md` and its queue mention no declaration or symbol
indexing anywhere, so the second consumer for a declaration layer is
**hypothetical**. What was argued against that — Item 5's scope index being
`sdom`'s and its frames remembering a declaration — turns out to weigh less than
the sequencing below, and that argument was mine rather than Bill's.

**DECIDED (Bill, 2026-08-31): the schemas live in `sdom/schema`, and only
mini-spec's own readers are `minispecsdom`'s.** A language schema is tightly
coupled to the machinery and **bundles with `sdom`** — anyone parsing Go wants Go's
schema, and none of it knows what a CRC card is. What stays in its own package is
the part that is genuinely mini-spec's and will only ever be used by mini-spec: the
traceability comment and its readers.

*This corrects an ambiguity carried through the morning*, in which "the schemas" and
"`minispecsdom`" were treated as the same place. Three packages, not two:

| package | holds |
|---|---|
| `sdom` | the protocol, the bracket parser, the declaration machinery |
| `sdom/schema` | the bundled Go, TypeScript, JavaScript, Lua and Shell schemas |
| `minispecsdom` | the traceability comment, Item 6 — a sibling of `sdom` |

**The boundary argument survives the move.** `sdom/schema` is a separate package, so
driving the machinery from it still exercises every accessor and constructor the
export surface is missing — which was the point of not building the schemas inside
`sdom`.

**DECIDED (Bill, 2026-08-31): the machinery is `sdom`'s and the language is the
schema's.** The line runs between *what a declaration is* and *how this language
announces one*:

| `sdom` — the machinery | the schema — the language |
|---|---|
| `DeclarationType`, `DeclarationName` | the keyword set, or the keyword-less form |
| `Doc.Replace`, the re-granulation | the pattern that recognizes a statement start |
| the declaration links and their stamp | how a name sits relative to its keyword |
| | the bundled Go, Lua, JS, Shell, TS schemas |

Every rule on the right is a fact about **one language** — Go's keywords, Lua's
keyword-less `NAME =`, Shell's whitespace-free `NAME=` — and none is a property of
documents in general. Everything on the left is true of a declaration in any
language, so a second consumer gets it without inheriting mini-spec's opinions.

**And the generalization is deliberately deferred rather than attempted now.** The
reusable half — whatever turns out to be a *tool* other schemas can build on —
gets extracted into `sdom` as **Item 9**, after declarations, indent, and Python
declarations have all landed. Three worked schemas is when there is enough
knowledge to know what generalizes; extracting from one is guessing, and the guess
would be built into `sdom`'s export surface where it is expensive to withdraw.

*Two consequences, stated because they are structural rather than matters of
taste:*

- **`Doc.Replace` still lands in `sdom`**, and not by preference. Go does not
  permit a method on `*Doc` to be declared from another package, so the one new
  structural verb is `sdom`'s wherever the rules live. `Split`, `Merge` and
  `Remove` are already exported, so the pass can call them from above.
- **Writing the schema one package up is what exercises the split.** Driving
  `sdom`'s machinery from `sdom/schema` will find every accessor and constructor
  the export surface is still missing — which the two-package decision at the top
  of this document already called part of the work rather than an obstacle to it,
  and which building the whole thing inside `sdom` would have hidden completely.
- **`BracketContext` stays in `sdom`, a schema inherits from it, and Item 4 adds
  the declaration map to it — but the schema does the linking** (Bill,
  2026-08-31). The map is machinery and goes where the other links go; filling it
  in requires knowing what a declaration looks like, which is language work.

  ```go
  // in sdom, beside closerOf / openerOf / enclosing:
  declaration map[Node][]Node   // a keyword -> every name it declares
  ```

  One-to-many because a grouped declaration declares several names. **A plain field
  for now**: Item 10 consolidates it into `BracketInfo` along with the pairing
  links.

  **First export-surface finding, 2026-08-31:** `BracketContext` exposes
  `Language`, `Origin`, `Closer`, `Opener` and `Enclosing` — and **no `Doc()`**.
  A schema building on it therefore holds the document `Parse` returned. Workable,
  possibly right (the context is deliberately not a document handle), and recorded
  because it is exactly what the two-package decision predicted would surface only
  once something outside `sdom` tried to build on it.

  **DECIDED (Bill, 2026-08-31): the declaration accessors notice.** Every other
  index here is *stamped, not registered*, which works because
  `BracketContext.rebuild` can re-derive the bracket links by walking the finished
  array. It cannot re-derive declaration links — `sdom` does not know what
  announces a declaration in any language — so the freshness check moves to the
  **accessor**, which is the one call every reader of these links must make. Same
  placement as everywhere else, and the same reason: checking freshness is what
  every consumer does anyway.

  *What a stale accessor does is the follow-on, and the house style already
  answers it.* It cannot rebuild, and returning the stale map — or an empty one,
  which reads identically to "this keyword declares nothing" — is the plausible
  wrong answer this project refuses on principle. `IndexOf` sets the precedent: it
  refuses inside a mutation window precisely because `-1` would be
  indistinguishable from end-of-document, and *"refusing says what happened
  instead."* So a stale declaration accessor refuses and names the repair: re-run
  the schema's pass. What is left to settle in Design is only the **shape** of the
  refusal — a typed sentinel like `inMutation`, or an ordinary error.

*What is deliberately left concrete:* the **algorithm** — scan the top-level text,
match, slice, link — is written in the schema for now, not lifted into a reusable
driver. Item 9 is where that becomes a tool, once three schemas have shown what
the tool should be.

**DECIDED (Bill, 2026-08-31): the bracket tables stay in `sdom`, and `LangLua` and
`LangTypeScript` join `lang.go`.** A `BracketLang` is not a declaration rule — it
is the bracket shape the parser already consumes — so the tables do not follow the
schemas up. This part adds the two missing ones, because a declaration schema for
a language with no bracket table cannot be tested.

*Consequences to carry into the Requirements phase:* **`R69` enumerates the four
shipped tables by name** and gives "every field of `BracketGroup` is exercised" as
the selection rule. Both change — the set grows to six and the rule gains a second
job, serving the languages mini-spec reads — so `R69` is **retired and replaced**
rather than edited. `specs/bracket-parser.md` is already rewritten to match.

**`for`/`while` … `do` … `end` needs no new mechanism — `Separators` is what it is
for**, and microfts2 already ships the shape: one group, two openers, `do` as a
separator rather than an opener.

```go
{Open: []string{"while", "for"}, Separators: []string{"do"}, Close: []string{"done"}},
```

*Recorded because this session got it wrong first:* a reading of `parseCode` showed
openers matched before the enclosing group's separators, and that was written up as
a conflict — on the assumption that `do` would need a group of its own. It does
not. The parse order is real and it is not a problem here.

**The residual is narrow and worth carrying: Lua's standalone `do … end` block.**
With `do` a separator of the enclosing group and `end` its closer, an inner
`do … end` reads as separator-then-close and ends the enclosing group early. No
shipped table has met this because **shell has no standalone `do`**. To settle in
Design, and small enough that leaving it unmodelled may be the answer.

**No prior art for Lua.** microfts2 configures Go, Java, C, JavaScript, Lisp,
nginx, Pascal and shell, and has **no Lua bracket table** — only a note that its
line comment is `--`. So unlike the bracket parser itself, where that repository is
the target to copy, Lua's table is genuinely new.

*Also to check in Design:* `elseif` must precede `else` among Lua's separators,
since `else` is a prefix of it. microfts2's shell config observes exactly this,
ordering `elif` before `else` — so the convention is real, but the prefix-ordering
rule is stated for **groups** and not for `Separators` within a group.

**DECIDED (Bill, 2026-08-31): whether to anchor is a mini-spec concern, not an
`sdom` one — and the question was posed backwards.** It was framed as *filtering*:
`sdom` reports every declaration, most of them noise, so which ones deserve a
comment? That invites a policy into the wrong package and makes the answer a
heuristic over declarations.

The real shape starts from the **other end**. Mini-spec finds its **unanchored
requirements**, derives the plausible declaration names for them — plausibly with
agent assistance, since that is a judgement about meaning rather than a match — and
uses `sdom` to get the **unanchored declarations** to match those against. Two
lists meeting in the middle, rather than one list being filtered.

So `sdom`'s side of it is small and already specified: report the declarations, and
which carry a traceability comment. Everything above that lives in mini-spec, and
mostly in the mini-spec **tool** rather than this repository.

*What this retires:* the noise measurement that motivated the filter — "0 of 32
declarations anchored over one ordinary Go package" — was measuring the wrong
thing, and this repository's own 183-of-225 says so. Neither number decides
anything now, because nothing is being filtered.

**DECIDED (Bill, 2026-08-31): the indent gets no node, and the reason generalizes
past Go.** Bracket groups already pinpoint the declarations, so indent is
irrelevant to **recognition** in any bracket-parsed document — depth does that
work. The 2026-08-27 requirement to capture it was an artifact of the regex
design: it argued that a pattern anchored at `^` with no indent capture finds
`class Widget` and misses every method in it, and with no regex that argument has
nothing to bite on.

Indent is still **read** where a tool emits a matching one while inserting a
comment above an indented declaration — a derivation over the preceding text, not
a field. And it is load-bearing in an *indent-parsed* document, where it is the
scope mechanism itself rather than a property of a declaration: Item 5.

*Measured 2026-08-31 over `sdom/*.go`, 183 declarations carrying a `// CRC:`
comment: **0** of them are indented.*

@undecided (2026-08-31): **what "top level" means when declarations nest.** The
trigger scans **top-level text nodes**, which is exactly right for Go: its
declarations are at bracket depth 0, and the 225/183/0 measurement above is that
scope. A Java method sits at depth 1 inside its class body, and a Go method value
or local `func` literal sits deeper still.

The keyword trigger makes this cheaper to answer than the brace walk did, because
scanning another depth is the *same* code against a different node set — no `if`,
`for` or `switch` false positives follow it in, since those are not keywords in
the table. What is left is a scope question rather than a mechanism one: which
depths a given language's declarations live at. Depth 0 covers this item's own
corpus completely; settle the general form when a language that needs depth 1
arrives.

~~@undecided: **declarations with no brace.**~~ **Resolved 2026-08-31 by the
keyword trigger**, which is also what settled the question the other way round: the
brace walk missed `var ErrPoisoned = errors.New(...)`, `ErrNotMutating` and
`todoRe` — three declarations already carrying `// CRC:` comments — because
`var X = f(...)` has no brace at all. A keyword at a statement start has no such
blind spot.

*Measured over `sdom/*.go` with the pattern and the structural-separator rule
above:* **225** declarations recognized, **183** of them anchored, and of those
anchored **183 recognized, 0 missed**. The 42 remaining are the unanchored set that
Item 6's query exists to report — **19%**, which is a far quieter signal than the
"mostly noise" this document feared when the policy question was written below.

@undecided (2026-08-31): **indent-parsed documents.** Python has no curly; the
trigger there is `:` plus an indent increase. The same walk with a different
trigger, or a different mechanism?

**Known and accepted:** scope for a brace language's *locals* nests by indentation
rather than by brace, which is wrong. Every mis-nested declaration is a local,
which the anchoring policy discards, and every top-level declaration comes out
correct. Fix by nesting on bracket depth if something ever needs local scope.

## Item 5

**Includes Python declarations** (Bill, 2026-08-31): the indent parser needs a
declaration schema to be tested against, so the two land together rather than as
separate parts. That also makes Item 5 the second of the three worked schemas
Item 9 reads.

Indentation and brackets **compose under one rule** rather than being two
mechanisms selected per language:

> **Indentation is significant only at bracket depth 0.**

Python needs it because a continuation line inside `foo(1,` carries no scope, and
YAML needs it because flow style *is* JSON. A brace language is the degenerate
case.

**This part relates declarations, so it follows Item 4.**

Scope is a **derived index**, not a schema: a scope frame is opened by a bracket,
or by a significant indent increase. Tab expansion advances to the next multiple of
a configured width, and the column is **derived from the node's text**, never
stored — the dual-ported rule again.

~~each frame remembers the declaration on the line that opened it. A declaration's
parent is the nearest enclosing frame that *has* one — which is what steps correctly
over an `if {` or a `for {` that is not a declaration.~~ — **superseded 2026-09-01
(Bill): not an invariant, and not the parser's job.** *"We parse. We do a second pass
to find declarations, slice text, and insert nodes. That's it."* The relation is
computable by a consumer from the scope index and Item 4's declaration links;
requiring the parser to encode it invented a requirement. Written 2026-08-27, before
declarations became a post-pass owning their own links.

### Item 5's decisions

**DECIDED (Bill, 2026-09-01): indent is parsed in ONE PASS, not post-processed — and
that needs a way for parsers to collaborate over text.** A `Parse` that only moves
forward when it finds a node at the head of the input, with an outer loop
accumulating pending text when none does.

**It is not a new mechanism.** `parseCode` already runs exactly that loop; this opens
its matcher list. The forcing reason is cost: a post-pass must split text nodes at
line starts, which is `O18`'s shape — one target per mutation window, `Doc.find`
re-scanning each time, O(n²) — with **per-line** targets instead of `O18`'s
per-declaration 251 over 24 files. The additive property also stops needing an
argument, since nothing is consumed and re-carved.

**The shapes.** `parser` becomes exported **`ParserState`**, which owns the loop, the
pending text, `flushText`, and the **`Origin`** — one parse, one origin, which is what
keeps `mergeLocs` from panicking when an indent node and a bracket node merge. A new
**`Parser`** interface carries `Parse(st *ParserState)` and `HasNode` for lookahead.
`ParserState` holds **one** `Parser` that may delegate: `IndentParser` owns its own
`BracketParser` and hands off when it does not match. **Precedence is the parser's
business, not `ParserState`'s.** Each parser owns its own context, so
`Parse(src, base, parser) *Doc` needs no `any` and nobody type-asserts — the shape
Item 1 says to watch for.

**Keep the recursion.** `open()` takes the loop until its closer rather than returning
a position, so *indentation is significant only at bracket depth 0* stops being a check
and becomes **structural**: the indent parser sits in the top-level loop's matcher list
and the recursive calls never offer it a position. Exclusivity inside a string falls out
the same way — two unrelated suppression rules from one structural choice.

**DECIDED (Bill, 2026-09-01): the loop's "no change" test reads BOTH `pos` and the node
count.** Neither alone is sound. `pos` alone misses a zero-length node, and a dedent to
column 0 may begin with a bracket opener — `"a string statement"`, `(1)`,
`[x for x in ()]` and `"""doc"""` are all legal Python statements there, so the opener
would be eaten as text. Node count alone is sufficient only *by accident*:
`parseRestricted` advances 1–2 bytes for an escape and emits nothing, and that is
invisible to the outer loop only because restricted regions run inside the
BracketParser's own loop. **The tiling invariant does not license the shortcut** —
contiguity is satisfied by *pending* text, so it never required a node per advance. One
integer comparison, and the failure it prevents is a **skipped marker** rather than lost
bytes: round-trip green, visible only to a recognition count.

**DECIDED (Bill, 2026-09-01): a node for every indent change; the column-0 return is
zero-length.** An indent node carries its line's leading whitespace; a return to column
0 has no bytes to own. Parent is the nearest enclosing **smaller** column, and the
document opens with a zero-length node for the root — which is what gives the root frame
an identity instead of a nil special case. Consecutive same-level lines share the frame
opened by the last change-node. Blank and comment-only lines do not change the level,
measured against CPython — which *does* indent on a docstring line, so the parser must
tell a comment from a string. See the `Kind` decision below, which is what supplies that.

**DECIDED (Bill, 2026-09-01): `IndentLang` carries a continuation marker**, and it earns
its place beyond Python. A line following one is never indented — settled 2026-09-01 — and
the *configuration* is now settled with it.

**It needs no new predicate: a continuation marker counts only at bracket depth 0**, the
same law indentation itself composes under. *Measured against CPython:* `a = 1 + \` before
a newline continues and emits no INDENT; **`# comment \` does not continue** — INDENT fires
and Python calls it an unexpected indent — because that backslash is inside the comment
group, at depth 1; and `a = "x \` needs no rule at all, since the string group is still open
at the next line start and depth-0 suppression already covers it. Three cases, one predicate.

**DECIDED (Bill, 2026-09-01): `ParserState` exposes `Emit(n)` and `NodeCount() int`; `Out`
stays private.** The call site is unchanged — `IndentParser.Parse` emits the root `Indent("")`
when `st.NodeCount() == 0` — and no live slice is handed out, which `O2` and `O21` are both
open gaps about. Nothing else needs to read the emitted array: the bracket parser needs its
`stack`, `NodeType` needs neither, and declarations do not join the protocol.

*And the rename is not a straight publicize.* Three of `parser`'s seven fields **move** rather
than going public: `lang` and `ctx` to `BracketParser`, since each parser owns its context, and
`stack`, which is bracket nesting rather than the walk's. `ParserState` keeps `src`, `pos`,
`textStart`, `out`, the `Origin` and the root parser. `emit` splits on the same line — the
append is `ParserState`'s, `ctx.enclose(n, stack-top)` is the bracket parser's — and `at()`
becomes `ParserState`'s outright once the origin lives there.

**DECIDED (Bill, 2026-09-01): the root `Indent("")` is emitted by `IndentParser.Parse`**,
guarded on `NodeCount() == 0` — not by the loop, which stays ignorant of indent, and not at
all for a non-indent parse. It is what lets top-level frames link somewhere and keeps their
list in `childIndent` rather than in a `roots` field beside the map, which would undo the
one-map decision below. *Consequence:* an **empty document gets zero nodes**, since the loop
never calls `Parse` — consistent with today.

**DECIDED (Bill, 2026-09-01): one map, in `IndentContext`** — brackets cannot contain
indents, so the indent context is the outer one and owns the whole index. The parse
records the links and `rebuild` re-derives them independently from the columns, so indent
gets `R87`'s checkable-not-believed treatment for free, wanting its own requirement and
its own alarm.

**DECIDED (Bill, 2026-09-01): this overrides part of Item 9's deferral, knowingly.**
Item 9 defers the shared driver because *"extracting from one is guessing, and the guess
would be built into `sdom`'s export surface."* That reasoning was about extracting with
**no forcing need** from **one** instance. This has a forcing need — indent is genuinely
bad as a post-pass — and **two** instances in hand, brackets and indent: extraction from
evidence rather than from a guess. Item 9 keeps the rest. The *declaration* driver still
waits for three worked schemas, and **declarations do not join the one-pass protocol**,
because a declaration's name sits forward of its keyword, past nodes another parser has
yet to produce.

**DECIDED (Bill, 2026-09-01): `BracketGroup` gains a `Kind` field, and no special cases.**
An indent language's parser marks its own comment groups `"comment"` and leaves the rest
unlabelled. This is what makes the `Parser` interface's lookahead **`NodeType`** rather
than `HasNode`: a comment opener and a string opener are both `*Opener`, so a lookahead
reporting mere presence cannot answer the question it exists for.

*Two heuristics were considered and refused.* Testing whether the group closes on a
newline needs no configuration and works for Python and YAML — and misses **every block
comment**, with Pascal already in the shipped tables having nothing but block comments.
A generalization exactly two languages wide. Injecting a predicate from the schema
avoids the reversal below but consolidates nothing.

**It reverses neither `R62` nor `R146`** — *this session claimed it reversed both, and Bill
asked the question that disproved it: does any bracket-parser logic involve comments?* It
does not. The only mentions of "comment" in `parser.go` sit inside one doc comment
explaining that restricted groups cover strings and comments alike.

- **`R62` stands untouched.** Its claim is that comments need no comment-specific
  *machinery* — they are ordinary parse-restricted groups, "a string is the same shape with
  different markers". `Kind` adds no parsing behaviour, so that stays exactly true.
- **`R146` keeps its number and gets edited text.** Its claim — *recognizing which groups
  are comments is the language layer's business, not `sdom`'s* — is **unchanged**; the
  indent parser or schema still does the marking. Only its *mechanism* moves, from
  string-matching an opener at read time to writing the fact into the table. Claim
  unchanged, wording changed: Item 8's `R76` precedent exactly.

**DECIDED (Bill, 2026-09-01): the kind string itself lives on `IndentLang`**, and the
bracket table's groups are marked with it. *"I suspect comment kind will always be
`comment`, but we can put it in indent lang."*

**That placement is what keeps `sdom` ignorant, which is a better reason than the value
varying.** Had `sdom`'s indent parser written `if g.Kind == "comment"`, the layering would
be gone — `sdom` would know what a comment is. Comparing two configured strings, it never
spells the word.

So the opacity rule is not *"`Kind` is never read by parser logic"*, which is now false, but:

> The **bracket** parser never reads `Kind`. The **indent** parser reads it only by
> comparing against a value its own language supplied. `sdom` therefore holds no comment
> knowledge — only *groups matching this configured kind are transparent to indentation*.

`sdom/schema` may spell `"comment"` freely, being the language layer; only `sdom` may not.
**No `KindComment` constant in `sdom`** — that would restore exactly the nominal knowledge
this avoids.

*Three consequences.* The **aliasing hazard dissolves** when kinds are set at declaration —
`LangPython` declares `Kind` inline, with no runtime write to a shared `Brackets` array; it
returns only if someone marks an existing brace table at runtime, so the rule is *mark at
declaration, or copy the slice first*. A **single string suffices** rather than a list,
since one kind value already covers several groups (Go's `//` and `/*` alike). And the
field's role is *the kind whose lines do not change the level*, so a name like
`TransparentKind` would put the last "comment" in `sdom` out of reach — cosmetic, and
unlike *lexer* it imports no wrong model.

**Export-surface finding, the third:** `BracketLang.groupFor` is **unexported**, so
`sdom/schema` cannot reach a group from an opener at all. Deleting `Lang.Comments` needs
`sdom` to export `GroupFor`, or a narrower `KindOf(opener string) string`. After
`BracketContext` having no `Doc()`, this is the two-package decision predicting itself
again — *driving the machinery from outside is what finds the accessors the export surface
is missing.*

What it still buys is the consolidation: it **deletes** `sdom/schema`'s `Lang.Comments`
rather than duplicating it onto `IndentLang`, because two independent consumers were
re-deriving by string-match what the table can state once.

*Two constraints on the shape, both from decisions already in force:*

- **`Kind` goes on the group, never on the node.** Item 2 refused a group pointer on
  marker nodes because `Equals` would compare it; a `Kind` copied onto a node has the
  identical defect. The lookup is `lang.groupFor(text).Kind`, the path `closes()` already
  takes.
- **Marking must not mutate a shared table.** `LangTypeScript = LangJavaScript` copies the
  struct header while both share one `Brackets` backing array, so a runtime write to one
  marks the other, silently and across consumers. Either the shipped tables carry `Kind` at
  declaration, or a parser that marks copies the slice first.

*What Item 5 owes when it implements this:* edit `R146`'s text, and reconcile the three
prose sources that state the old **mechanism** — `specs/declaration-schemas.md`,
`design/crc-DeclSchema.md`, `sdom/schema/schema.go` — plus a note at Item 4's decision in
this document. `specs/bracket-parser.md` and `sdom/bracket.go` say *there is no comment
configuration* about the parser, and both **stand**. No retirements.

*The reasoning, the measurements and the worked examples are in `.scratch/INDENTS.md`.
The calls are here.*

**DECIDED (Bill, 2026-08-27): YAML is its own DOM, reusing this mechanism — not a
mode on the markdown or code one.** It shares the indent rule and shares no
schema. It also needs bracket parsing, for flow style. Mini-spec's original
comment-eating `--repair` bug was YAML, so the motivating instance already exists;
this carve does not schedule it.

## Item 6

Mini-spec's reader: the traceability comment. This item is the **comment parser plus
the accessors a consumer needs to relate a comment to a declaration** — not the
resolving query. Whether a ref points at a live, retired or absent requirement is the
mini-spec tool's business, as a consumer of `sdom`; nothing here knows what a
requirement *is* beyond its number.

    // CRC: crc-Store.md | Seq: seq-crud.md#1.4 | Test: test-Store.md | R4, R5-7 -- note

The comment is a compound whose children tile its bytes. Field keys and values are
**derived from the literals, never stored** — nothing is normalized on the way in,
so `//CRC:crc-Index.md|R7` and `// CRC:   crc-Odd.md   |   R7` both parse and both
render back byte-exact.

### The design (Bill, 2026-09-02)

The item lands in **two subparts**. **6.1** re-touches landed `context.go` and
depends on nothing else here; **6.2** is the reader, in three pieces, and follows it.

**6.1 — typed and text accessors on `BracketContext`.**

- `BracketInfo.declaration` is `[]*DeclarationName`, not `[]Node`; `SetDeclarations`
  takes `map[*DeclarationType][]*DeclarationName`; a typed accessor
  `DeclarationNames(t *DeclarationType) ([]*DeclarationName, error)` returns pointers
  (values would copy the structs and break node identity) and keeps the
  `ErrDeclarationsStale` refusal, which must not silently become "declares nothing".
  A **method, not an interface**: every schema stores its links in `*BracketContext`,
  so an interface would have one implementer.
- `Opener(closer Node) *Opener` and `Closer(opener Node) *Closer` — typed, same move.
- `InnerText(n Node) string` and `OuterText(n Node) string`, mirroring `innerHTML` /
  `outerHTML`. A group left open at end of input: inner runs to the end of the source.
- **`O21` stays, by design.** Accessors hand back the context's own slices. Every
  `sdom` consumer is fire-and-forget — mini-spec is a CLI that builds a DOM, uses it
  and exits; microfts2 (and Ark through it) uses `sdom` for indexing and searching and
  caches no DOM slices — so aliasing has no lifetime to be a hazard in. `sdom` is a
  *simple* library that parses, renders and stencils; a sharp corner is documented in
  a sentence, and the one guarded class is a write that would silently corrupt bytes.

**6.2a — the list compound (`sdom`, generic).** One node kind holding one `Text` child;
`Items() []string` derives the comma-separated values from the literal on every call
(no laziness — small files), and `SetItems([]string)` rewrites the literal through the
guarded write. Whitespace around commas is glue and is preserved when unedited.
**Two parsers produce it:** a plain one (tokens only) and a ranged one, whose items
may also be `Rn-Rm` or `Rn-m` ranges. The **requirements node** is the ranged one's
product and is numeric: `Items() []int` flattens, expanding ranges (a reversed range
contributes only its low ref); `SetItems([]int)` sorts and emits maximally condensed,
runs of three or more as `R7-12`. Only requirements use the ranged parser, so a range
inside a doc list cannot exist by construction. No insert or delete.

**6.2b — `CommentStyle` on `BracketLang`.**

```go
type BracketLang struct {
    Brackets []BracketGroup
    Comment  CommentStyle       // how this language writes a comment
}
type CommentStyle struct{ Prefix, Suffix, Kind string }   // "// ", "\n", "comment" for Go
```

Prefix and suffix are the construction template — not redundant with the group's
`Open`/`Close`: the opener is `//` but a written comment wants `// `, and the closer
is a structural `\n`. `Kind` is what a constructed comment must parse back as, and
should equal the `Kind` of the group recognizing the prefix. **Guarded by a test, not a
runtime check:** for every shipped language, construct a comment, parse it, assert the
group's `Kind == lang.Comment.Kind`. A language with several comment forms designates
one for construction (Go's `//`, not `/*`); the others still parse. Three names touch
"comment" at three layers and stay separate: `BracketGroup.Kind` (recognition),
`IndentLang.Transparent` (indentation), `CommentStyle.Kind` (construction). `sdom`
still never spells the word in a branch.

**6.2c — `minispecsdom`, the first mini-spec-centric package.** A sibling of `sdom` in
this repo (it may move into mini-spec later). `sdom` never learns what a CRC card is.

*The grammar.* The interior is the bytes between the language's comment opener and
closer.

```
interior ::= WS? field ( WS? "|" WS? field )* ( WS? descsep description )?
field    ::= "CRC:"  WS? doclist  |  "Seq:" WS? doclist  |  "Test:" WS? doclist  |  rlist
doclist  ::= token ( WS? "," WS? token )*        token = one non-space run (a doc path)
rlist    ::= ref   ( WS? "," WS? ref   )*        ref   = Rn | Rn-Rm | Rn-m
descsep  ::= "--" | "—" | ":"     accepted on read;  "--" emitted on write
```

Four field kinds, **one occurrence each, every one a list, in any order** — a reader
classifies each `|`-segment by its lead. A `:` is a field-key colon only immediately
after `CRC`/`Seq`/`Test`; anywhere else it is the descsep, so `R5: desc` reads `R5`
then a description. A `Seq` value's `#step` splits from its path in the typed view.
Whitespace is glue everywhere; keywords, `|` and the descsep are computed glue, so the
pattern cannot silently eat bytes. The description is bound and writable; its
separator is preserved byte-exact when unedited and emitted as `--` on a fresh write.
Canonical write order is CRC, Seq, Test, refs, single spaces.

*The node.* Stencilled nodes have the shape `type T struct { FIELDS aliasing child
nodes; dom []Node }`, and this one is **one shared kind for every language** — only
the `CommentStyle` wrapping is language-specific. It **tiles the whole comment, opener
through closer**; a consumer that skims the flat DOM for the kind has the complete
comment as one node.

*The second pass.* `c.Parse(cmt *Opener, ctx *BracketContext) bool`. The pass visits
each comment group; a cheap lead filter skips obvious non-candidates; the rest get an
attempted parse of `ctx.InnerText(cmt)`, built off to the side. **Recognition is a
parse that consumes the whole interior:** `// Test: a repaint frame round-trips
(R3136).` leads with a keyword but leaves prose uncovered, so it is not a traceability
comment; nor are `// see R5` or `// (R5)`. On false the pass discards `c` and
advances; on true it rips the group's run of nodes out of the flat DOM and puts the
one compound in its place, **moving the original `*Opener` and `*Closer` inside it** —
reused, not recreated — so the context's identity-keyed maps stay valid. This is the
same test as the guarded write below: a parse that consumes less than everything is
the refusal. Locs derive from the opener's offset and `ctx.Origin()`.

*Inside `Parse`.* `StencilBuilder` is single-regex and cannot express arbitrary order
(a repeated capture reports only its last iteration), so the interior is a **segment
walk**: split on `|`, peel a trailing description off the first descsep that is not a
field-key colon, then a `StencilBuilder` per segment with an alternation regex so each
segment classifies itself by which group participated. List texts become 6.2a's list
nodes; results splice **flat** into the comment — no nested per-segment compounds.

*Construction.* There is no hand-built child list. A constructor assembles the
canonical string from card, seq, test, refs and description, wraps it in
`lang.Comment.Prefix … Suffix`, parses it, and calls `Parse` — one path, so nothing
can build a comment that disagrees with how one is read.

*Tests.* (1) A fuzzed source string over the whole grammar — any field order,
whitespace, all three descseps, ranged refs — asserting `render(dom) == src` and
`parse(render(dom)).Equals(dom)`; the DOM compare fails on under-modelling, which a
byte round-trip cannot see. Go native fuzzing gives reproducibility for free. Lua is
the case the generator must hit: its opener is `--`, the same token as the write
descsep. (2) The constructor's output parses back to an `Equals` DOM. (3) 6.2b's kind
test per language. (4) Fire alarm: make the parser return the interior as opaque text,
drop a field, or normalize a range — the DOM compare goes red.

### Item 6's decisions

**DECIDED (Bill, 2026-08-27): a write into a field is guarded by re-parsing that
field from its own render**, requiring the re-parse to consume *all* of it and
compare equal. Consuming less is the interesting failure: a value containing a
field separator renders fine and loses no bytes — the byte round-trip is blind to
it — but the re-parse stops at the separator, proving the replacement would read
back as two fields. Rollback is cheap **here and only here**, because a node-level
write changes one literal; that is why the document-level mutation has none and
this does.

**DECIDED (Bill, 2026-08-27): a refused write is handled where it happens** and
not propagated out of the mutation function, since anything escaping it poisons
the document. The guarded write is its own transaction.

**DECIDED (Bill, 2026-09-02): scope is the parser and the accessors, not the
resolving query.** Anchored / unanchored / retired / dangling is how a consumer uses
`sdom`, not what `sdom` does.

**DECIDED (Bill, 2026-09-02): order-independent reading, canonical writing.** Fields
appear in any order on read, at the price of a segment walk instead of one regex; a
human hand-editing these does not keep an order, and the walk is cheap.

**DECIDED (Bill, 2026-09-02): two list parsers, one list kind.** Ranges are legal only
where requirements are parsed. An invariant true by construction needs no check.

**DECIDED (Bill, 2026-09-02): `DeclarationNames` is a method on `*BracketContext`**,
not an interface, until a second provider exists.

## Item 8

The bracket layer is a **parser**, not a lexer, and five names say otherwise.
**Added 2026-08-31** — a part discovered rather than planned, which is why its
number runs out of sequence.

`specs/bracket-lexer.md`, `design/crc-Lexer.md`, `design/test-Lexer.md`,
`sdom/lexer.go`, `sdom/lexer_test.go`, and `R76`'s text ("whitespace is not a
token"). The **prose** in this document was corrected the day the part opened; the
names are what remains.

**Why this is a part and not a wording pass.** The word imports the
lexer-then-parser-over-tokens model, and that model is wrong here: a parse level
matches at the head of the input, decides which concrete node it is looking at, and
hands that node the string. *Measured 2026-08-31:* reading "lexer" literally
produced a design for Item 4 built on a second `StencilBuilder` binding over
existing nodes — a thing that should not exist — and the error survived a full
carve read, because every document says "lexer" too.

**What makes it a pass rather than a rename.** Moving `crc-Lexer.md` rewrites every
`// CRC: crc-Lexer.md` comment in `sdom/`, the Artifacts manifest row, and the
`**Inject:**` anchors in `design/test-Lexer.md`. Moving `specs/bracket-lexer.md`
rewrites the `**Source:**` lines and the root index entry. `R76` keeps its number
and gets edited text: the claim — whitespace folds into text rather than becoming a
node of its own — is unchanged, so this is **not** a retirement.

**Not to be renamed.** `Origin` is "a small core token that every parser mints", in
`specs/location.md`, `design/crc-Loc.md` and `sdom/context.go`. That is a
capability token, not a lexical one, and it is correct as it stands.

**DECIDED (Bill, 2026-08-31): the parser family, and `Scan` stays.**
**~~`Scan` stays~~ — superseded 2026-09-01 (Bill): `Scan` becomes `Parse`. See Item
12.** The rest of this decision is unchanged and landed as written. The words that
import the wrong model are *lexer* and *token*. **Scan** is a neutral verb for one
pass over a source, so the exported entry point keeps its name and everything
naming the **layer** becomes parser:

| from | to |
|---|---|
| `specs/bracket-lexer.md` | `specs/bracket-parser.md` |
| `design/crc-Lexer.md` | `design/crc-BracketParser.md` |
| `design/test-Lexer.md` | `design/test-BracketParser.md` |
| `sdom/lexer.go` | `sdom/parser.go` |
| `sdom/lexer_test.go` | `sdom/parser_test.go` |
| `type lexer` | `type parser` |

`Scanner` was considered and refused: in ordinary compiler usage a scanner **is** a
lexer, so it would have changed the spelling and kept the model — the exact failure
this part exists to fix.

**DECIDED (Bill, 2026-08-31): `lexicon` becomes `schema`.** Ten hits, and a
different word making a different claim — it names the language table rather than a
lexing pass. `schema` is not a synonym picked for freshness: it is **already this
project's word** for a document type, in `StencilBuilder`'s "a schema drives it",
in "each schema's parse context is its own concrete type", and throughout
`specs/stencils.md`. The rename leaves one word where there were two.

**DECIDED (Bill, 2026-08-31): the carve's status block moves; `DONE.md` does not.**
The status block is a live index people read to orient — the whole reason this part
goes first — so Item 2's title becomes "the bracket parser". The done ledger
records what the work was called when it landed, and rewriting it would falsify a
record rather than clarify one.

## Item 10

Replace `BracketContext`'s three link maps with one. **Added 2026-08-31**, and
**split the same day**: the separator links it was originally bundled with are a
contract matter and moved to Item 11, while this half is a representation change
and is **deferred**.

**DECIDED (Bill, 2026-08-31): Items 10 and 11 are done together, as one queue
item.** They rewrite the same function, and the decisive reason is that neither
ordering works apart: `BracketInfo` carries a `separators []Node` field, and nothing
derives separators until Item 11 — so Item 10 alone ships a field nothing fills,
while Item 10 without the field changes the struct twice and voids the measurement
below. Declared once, correctly, is only available together.

*The cost of splitting them is also concrete.* Three alarms anchor into exactly the
code both rewrite — `parser.open`, `BracketContext.rebuild`, and
`BracketContext.Declarations`, the last because this part folds the declaration map
in — so a split rewrites `rebuild` twice and pulls those alarms twice.

**But their justifications stay separate, and that is not bookkeeping.** This part is
a representation change, deferred on consumer grounds and legitimately so. Item 11 is
a **contract gap**, which consumer count does not get to defer. If the rewrite goes
badly, Item 11 still lands. Merging the work must not merge that.

**DEFERRED PAST ITEM 4 (Bill, 2026-08-31):** *"We can still defer `BracketInfo` to
later, since it seems like we don't need it for declarations."* Deferring a
representation change on consumer grounds is legitimate in a way that deferring a
contract gap is not — see Item 11. Placed immediately **after** Item 4 the same
day, so "later" means next, not someday.

**DECIDED (Bill, 2026-08-31): one map of flat structs.**

```go
type BracketInfo struct {
    opener, closer, enclosing Node
    separators                []Node
}

// in BracketContext, replacing closerOf / openerOf / enclosing:
info map[Node]BracketInfo
```

**Measured before deciding**, over `sdom`'s own 14 files — 15,747 nodes — building
each shape from one parse:

| shape | heap | allocations |
|---|---|---|
| three maps (as landed) | 1,067 KB | 275 |
| **one map of flat structs** | **2,463 KB** | **111** |
| one map of boxed `NodeInfo` values | 1,834 KB | 15,858 |

A boxed hierarchy — `NodeInfo` interface, `BaseNodeInfo` for plain nodes,
`BracketInfo` for bracket participants — was proposed and refused **on its
allocation count, not its size.** It is the smaller index, because a plain node
carries 16 bytes rather than 72; but an interface value cannot hold a struct
inline, so every entry becomes its own heap object. `rebuild` recreates this index
on **every structural change**, so 15,858 allocations is per-edit churn rather
than a one-time cost, and trading 630 KB for it is the wrong direction.

*The flat struct also has fewer allocations than the three maps it replaces* —
111 against 275 — because it is one map instead of three.

**The empty interface was the second reason.** `type NodeInfo interface{}`
dispatches nothing, so every use site type-asserts back to the concrete type —
exactly the shape Item 1's decision says to watch for. Had it been kept, it would
have wanted a real method set rather than nothing.

**And the extensibility it reached for is already provided.** A later layer
wanting its own per-node facts keeps its own map, stamped separately — the rule
already in force, and how the declaration links were going to work regardless. That
buys extensibility with no boxing and no assertions, and leaves `BracketInfo` about
brackets.

**Why it is its own part.** It rewrites landed Item 2 code, and `rebuild` and
`parser.open` both carry alarms that will need re-pulling. Item 4 is already
carrying two node kinds, `Doc.Replace`, the declaration map, `LangLua`,
`LangTypeScript`, and a new `sdom/schema` package with five schemas.

**DECIDED (Bill, 2026-08-31): the declaration links fold into `BracketInfo` — and
not yet.** Item 4 adds a plain `declaration map[Node][]Node` field to
`BracketContext`, which the schema populates; **this part consolidates it into
`BracketInfo`** with the rest.

*Recorded against a recommendation that went the other way.* The argument for
keeping it separate was that a `[]Node` of names is language-derived and widens
every entry — true, and it loses to the same reasoning that put the map in `sdom`
at all: storage is machinery, and one index answering everything about a node beats
one more map beside it.

*Consequence for the numbers above:* `BracketInfo` gains a fifth field and goes
from **72 to 96 bytes**, so the measured 2,463 KB was for the narrower struct. The
comparison it settled is unaffected — a field widens the boxed shape identically,
and the allocation counts that decided it do not move at all.

## Item 11

`BracketContext` answers **opener→closer**, **closer→opener** and
**node→enclosing**. It does not answer **opener→separators**. That asymmetry is the
part.

**Separator *nodes* already exist** — Item 2 emits `Separator` and links each one
to its opener through `enclosing`, so `for x in a b; do … done` parses with `in`
and `do` as separators inside the `for` group. What is missing is only the index in
the other direction, which a consumer can today only get by scanning forward.

**DECIDED (Bill, 2026-08-31): consumer count does not decide this.**

> we want bracket parsing to be correct, because simple dom is a library and
> that's the contract. The number of consumers doesn't matter -- it still needs to
> be there.

*Recorded because this session argued the other way twice*, deferring the direction
on the grounds that nothing needed it yet — which is `O13`'s mistake in its other
form. `O13` says *"unused outside tests"* is meaningless in a library; this is
*"no consumer yet"*, and it prices an export by demand where a library prices it by
contract. Demand orders work that is all going to happen; it does not decide what
belongs.

**The work, and `R87` is the property at risk.** `rebuild` collects separators from
the **same stack walk** that recovers the pairing, so its independent
cross-derivation survives — the whole point being that `rebuild` re-derives every
link a *second* way, from the finished array rather than from the recursion that
produced it. A representation change is exactly what can quietly lose that, by
letting the parse be the only thing that records a separator — that
property being the one most easily lost when this index changes. Then an accessor
in the shape of the three that exist, a requirement, a test, and an alarm.

**GROUPED WITH ITEM 10 (Bill, 2026-08-31)** — one queue item, for the reasons
recorded there. Neither part depends on the other in principle: one adds a link the
contract is missing, the other changes how the links are stored. In practice they
share a function and a struct field, so doing them apart declares `separators` before
anything fills it.

**What must not be merged is the justification.** Item 10 is deferrable and was
deferred; this is a contract gap and is not. Should the representation change stall,
this still lands on its own — the grouping is about the work, not the standing.

**Bill placed Item 10 first** (2026-08-31), immediately after Item 4.

*That order is the tidier one, and this document previously argued the reverse.*
With Item 10 first, `BracketInfo` is declared **with** its `separators []Node`
field and this part fills it; the field exists from the outset rather than being
added to a struct that just settled. Taken the other way round, Item 11 would have
minted a fourth map for Item 10 to immediately fold in.

## Item 9

Extract the reusable half of the schema work into **tools in `sdom`**, and retarget
the schemas onto them. **Added 2026-08-31**, at the moment the first schema was
about to be written, so that the deferral is a decision on the record rather than
an omission somebody notices later.

**It goes last on purpose, and the ordering is the whole point.** It runs after
declarations, indent, and Python declarations have all landed — because *three
worked schemas is when there is enough knowledge to know what generalizes.*
Extracting from one is guessing, and the guess would be built into `sdom`'s export
surface, where withdrawing it costs every consumer.

So until this part runs, the algorithm — scan the top-level text, match, slice,
link — is written **concretely in each schema**, and the duplication between them
is expected rather than regrettable. It is the evidence this part reads.

**What is already known to be `sdom`'s** and is therefore not waiting for this
part: the `DeclarationType` and `DeclarationName` kinds, `Doc.Replace`, and the
declaration links with their stamp. Those are true of a declaration in any
language. What waits is the **driver**: whatever shape lets a schema say *here is
my pattern, here is how my names sit* and get the pass for free.

**Watch for the generalization that is only two cases wide.** Go and TypeScript
will look alike, and a tool fitted to them is not a tool. Lua and Shell are the
useful evidence, because they announce a declaration with **no keyword at all**,
and they do not even agree with each other: Shell's assignment forbids whitespace
around `=` while Lua's expects it. Python arrives with a third shape and no braces.

**DECIDED (Bill, 2026-08-31): Python declarations are part of Item 5**, because
the indent parser needs them to be tested anyway. So this part's precondition is
Items 4, 5 and 6 landing — not a fourth part that never existed.

## Item 12

Item 8 fixed the **layer's** name and left the **pass's**. `Scan` kept its name on
the argument that it is a neutral verb; the prose then kept scanning, scanned,
scans and the scan throughout. **Added 2026-09-01**, when the residue was measured.

**DECIDED (Bill, 2026-09-01): `Scan` becomes `Parse`.** This reverses Item 8's
`Scan` stays, which is marked superseded at its source. Everything else Item 8
decided stands. Already done in the working tree when this part was opened:
`sdom.Parse`, its call sites and tests, and `seq-scan.md` to `seq-parse.md` with
every `**Refs:**` rewritten.

**DECIDED (Bill, 2026-09-01): every scanning name goes — *we do not have a scanner,
just a parser*.** `scanBody` and `scanCode` were called first, as misnomers —
`scanBody` in particular *recurses*, which is the thing a scan is not — and the call
then widened to `scanRestricted` and to the **term** `scan-restricted` itself, which
becomes `parse-restricted` in specs, design, code and this document. Also the two
test identifiers (`TestTheScanNeverStalls`, `fromScan`), the `Origin{Name:}` fixtures
and the diagnostic strings.

*This session argued for keeping `scan-restricted` and was wrong.* The case was that
the term names a mode rather than a phase and imports no two-phase model. But a
project with no scanner has no business owning a word for one, and the second half of
the argument — that leaving it costs no alarm re-pull — was pricing the vocabulary by
what it was convenient to avoid. `O13`'s mistake in a third form.

**And the rule that decides every remaining case (Bill, 2026-09-01):**

> **`parse` is the pass over bytes; `scan` is a walk over the finished array.**

So R87's forward cross-derivation, `Doc.find`'s linear scan, and the declaration
passes over top-level nodes all keep the word, **correctly and now unambiguously**.
That is the part worth more than consistency: before this pass "the scan" named both
the byte pass and R87's independent walk, whose entire point is that it does *not*
trust the byte pass. Renaming only one of them **separates two things that were one
word**, so R87 reads better than it did rather than merely differently.

*Measured 2026-09-01 over tracked files* (`git ls-files`, after a first count was
inflated three-fold by an editor's `~undo-tree~` backup beside the carve): about 180
occurrences of the scan family, in four kinds — the pass (~120), the term (18), the
parser internals (14), and R87's forward walk (6). Plus five stale `Scan` API
references in prose and seven uses of *lexical*. Everything but the last group moved.
`token` survives in four places, all of them `Origin`, which Item 8 ruled correct as a
capability token and this part does not reopen.

**One thing deliberately not rewritten: a dated `**Pulled:**` quote.**
`test-BracketContext.md` quotes the corpus test's failure message verbatim in two
pull records. The live format string moved to *the parse recorded*; the quotes keep
the wording that was actually printed on 2026-08-30 and 2026-08-31. A record reports
what happened, so rewriting it would falsify it rather than clarify it — the same
reason Item 8 left `DONE.md` alone.

**The one load-bearing site.** `design/test-Loc.md`'s alarm 4 anchors
`**Inject:** sdom/parser.go:Scan` and names `Scan` in its `**Fire alarm:**` prose.
That symbol no longer exists. The census reads verified only because the rename is
uncommitted and git still sees `Scan` at HEAD — so the anchor must be repaired in
the same commit, not after it.

*Define the survey population with `git ls-files`.* A first count over the
directories was inflated three-fold by an editor's `~undo-tree~` backup sitting
beside the carve, which is gitignored but not invisible to grep.

---

## The tests, and why they are not a separate item

Written once and inherited by every node kind, so a new kind is tested **by
existing**. They belong to each part's Implementation phase, not to an item of
their own.

1. **Byte round-trip over the real corpus**, not fixtures — a fixture contains only
   what its author thought to include, and what a lossy parse eats is precisely
   what nobody thought of. The corpus includes the parser's own source.
2. **The flat array tiles the document** — contiguous, half-open, first at 0 and
   last at the end.
3. **Every faithful node renders exactly its own source span.**
4. **The structural round-trip over a *mutated* tree**, re-parsed and compared
   against the original. ~~into a document with a **different base offset**~~ —
   **corrected 2026-08-30 (gap O9): a base never enters a location, so varying it
   changes no offset and proves nothing.** What makes the comparison bite is that
   the two trees carry **different origins** and, where the bytes shifted,
   different offsets — and `Equals` must still ignore both.
5. **The one-field delta** — mutate one node and require exactly that node and its
   ancestors to lose faithfulness, and every untouched sibling to keep it.
6. **A recognition count.** This is the one that is easy to omit and it catches
   what nothing else does: a sub-schema that quietly stops recognizing something
   falls back to opaque text, which **loses no bytes and breaks no round-trip**.
   Only counting what you expected to find sees it.
7. **The additive property** — the flattened node array is identical with and
   without the readers. ~~token stream~~ — **corrected 2026-08-31 (Item 8): there
   are no tokens here, only nodes.**

**And the alarms must be pulled.** One case decides the shape of the suite: a
`Render` that hands back the retained source leaves the **byte round-trip green**
over every document, no matter how little was modelled. Removing a node from the
tree and requiring the output to lack it is the only injection that reaches that,
which is why it cannot be folded into the corpus test.
