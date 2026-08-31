# Carve: a simple DOM, and the mini-spec parser over it

Mini-spec reads and writes its own documents through scattered regexes and
hand-rolled scanners that disagree with each other. The repair is a **simple
DOM** — a parse that models only what the tool operates on, keeps every other
byte exactly where it was, and re-emits the source with nothing but the intended
change in it. This carve is that DOM, built clean, in **two packages**: a lexical
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
- [ ] **Item 2 — the bracket lexer.** **OPEN (#2.)**
- [ ] **Item 3 — regex compounds.** **OPEN (not queued.)**
- [ ] **Item 4 — declarations, as a post-pass.** **OPEN (not queued.)**
- [ ] **Item 5 — indent scope.** **OPEN (not queued.)**
- [ ] **Item 6 — the traceability reader.** **OPEN (not queued.)**

## Decisions

**DECIDED (Bill, 2026-08-27): two packages.** `sdom` carries the protocol and the
bracket and indent parsers; `minispecParser` carries mini-spec's readers. The
split is so microfts2 can consume the lexical half — which means **nothing below
the reader layer may know what a CRC card is**, and the boundary is enforced by
the compiler rather than by discipline. The export surface this forces
(constructors and accessors for what is currently package-internal) is part of
the work, not an obstacle to it.

**Module `github.com/zot/simple-dom`** (Bill, 2026-08-27). Package `sdom` lives in
`sdom/`, leaving the module root for the reader package and a command later.

**DECIDED (Bill, 2026-08-27): a bracket group is NOT a node.** An opener,
everything between it and its closer, and the closer are **siblings in the flat
array**. The scanner keeps the nesting on a stack; the emitted stream has none.

Nesting is expressed by **links**, owned by the **lexicon's context** and not by
`Doc` — not every document has brackets, and a markdown DOM would carry two dead
maps forever:

- an opener knows its **closer** and its **enclosing opener**
- a closer knows its **opener**
- any other node knows its enclosing opener — and a forward scan that skips whole
  bracket pairs finds the same answer independently, which is what makes the index
  checkable rather than merely believed

The context therefore **outlives the parse**: it carries the language, the
scanner's stack during scanning, and these links afterwards.

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
lexicon that needs no derived state carries none. A *content* edit does not bump
the generation, because node membership is unchanged and an index over structure
survives one.

**And reading the generation refuses inside the mutation window**, with the same
typed sentinel the core indices use. That is what makes the guard reach a
*layer's* index **without the layer knowing the guard exists**, since checking
freshness is the one call every stamped index must make. Poisoning the comparison
instead — returning a value no stamp can match — would let the layer rebuild from
a half-edited tree, which is a different wrong answer rather than a refusal.

**DECIDED (Bill, 2026-08-27): the readers are post-passes over the lexer's
output, never a re-parse of raw text.** A reader that re-parses bytes can *consume
tokens the lexer already produced*: a declaration regex matching
`func (s *Store) Index` from text swallows the receiver's parens, so the same
file scanned with and without readers has different bracket structure. Grouping
existing nodes is **purely additive**: the flattened token stream is identical
with and without the readers, and only the top-level stream differs — which is
what nesting means, and is *not* the same as a group having children.

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
segment lazily.** The rule is not *never normalise*; it is **never normalise what
you did not touch**. `R5-8` stays `R5-8` until something edits it — then
reformatting the edited part is fine and the unedited parts stay byte-intact.
Adding `R12` to `R5-R7, R10` must not flatten the range on its way past.

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

A **table-driven** bracket lexer. A string is not a special case: it is a bracket
group with scanning turned off, which is why interpolation, word brackets and
shell `if`/`then`/`fi` all fall out of one mechanism.

```go
type BracketGroup struct {
    Open, Separators, Close []string
    Escape                  string
    AllowedInner            []string // nil = code mode; non-nil = scan-restricted
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
derailing the scan; and a scan that **always consumes at least one byte**, so
nothing stalls on input it does not understand.

Output is **flat and document-order** — see the group-is-not-a-node decision
above, which this part is the main consumer of. The scanner keeps the nesting on a
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

A declaration node models the **keyword through the name** and nothing more.
Everything after stays opaque. This is not a language parser; it is a way to
*address* declarations.

**It is built by grouping nodes the lexer produced** — splitting text nodes at the
declaration's edges and nesting the span — so brackets inside a receiver survive
as children. See the second decision at the top of this document for why.

**Indentation is captured, not required to be empty.** Go's methods happen to sit
at column 0; Python's, Java's and Smalltalk's do not, and requiring column 0 finds
`class Widget` and misses every method in it. The indent capture must be
`[ \t]*` and never `\s*`: `\s` matches newlines, so it would swallow blank lines
and let a declaration match several lines further down, breaking both the line
lookup and the backward walk that finds its comment.

### Item 4's decisions

**DECIDED (Bill, 2026-08-27): only the fields a tool *writes* are nodes.** The
indent and the name are nodes; the keyword and the receiver are read-only
derivations over the render. *Editability sets the granularity* used as a budget
rather than a maximum.

@undecided: does this part live in `sdom` or in `minispecParser`? Nothing in it
knows what a CRC card is, and microfts2 would want it for symbol indexing, which
argues for `sdom` with the declaration regex moving onto a `DeclLang` layer
between the indent language and the reader language. Settled by whether a second
consumer is real or hypothetical when the part is scheduled.

@undecided: which declarations *ought* to be anchored is a policy and belongs in a
reader, not here. Unfiltered, the query is mostly noise: over one ordinary Go
package it reported 0 of 32 declarations anchored, nearly all of them protocol
boilerplate that should never want a comment. Candidates: exported
names only; types and functions but not interface-satisfying methods; only files
that already carry at least one traceability comment. Settled by running the
candidates over a real corpus and reading the noise.

**Known and accepted:** scope for a brace language's *locals* nests by indentation
rather than by brace, which is wrong. Every mis-nested declaration is a local,
which the anchoring policy discards, and every top-level declaration comes out
correct. Fix by nesting on bracket depth if something ever needs local scope.

## Item 5

Indentation and brackets **compose under one rule** rather than being two
mechanisms selected per language:

> **Indentation is significant only at bracket depth 0.**

Python needs it because a continuation line inside `foo(1,` carries no scope, and
YAML needs it because flow style *is* JSON. A brace language is the degenerate
case.

**This part relates declarations, so it follows Item 4.**

Scope is a **derived index**, not a lexicon: a scope frame is opened by a bracket,
or by a significant indent increase, and each frame remembers the declaration on
the line that opened it. A declaration's parent is the nearest enclosing frame
that *has* one — which is what steps correctly over an `if {` or a `for {` that is
not a declaration. Tab expansion advances to the next multiple of a configured
width.

### Item 5's decisions

**DECIDED (Bill, 2026-08-27): YAML is its own DOM, reusing this mechanism — not a
mode on the markdown or code one.** It shares the indent rule and shares no
lexicon. It also needs bracket parsing, for flow style. Mini-spec's original
comment-eating `--repair` bug was YAML, so the motivating instance already exists;
this carve does not schedule it.

## Item 6

Mini-spec's reader: the traceability comment, its fields, and the anchored /
unanchored query.

    // CRC: crc-Store.md | Seq: seq-crud.md#1.4 | R4, R5

The comment is a compound whose children tile its bytes. Field keys and
requirement lists are **derived from the literals, never stored** — nothing is
normalized on the way in, so `//CRC:crc-Index.md|Seq:...` and
`// CRC:   crc-Odd.md   |   R7` both parse and both render back byte-exact.

An anchor is a **two-ended link** and can dangle at either end. The query wants
all declarations with status, not only the failures: expected-but-missing,
present-but-unanchored, present-and-anchored, and **anchored to a requirement or
artifact that no longer exists**. A retired requirement is a forwarding address
rather than a break, so the tool can tell a hop from a genuine dangle.

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
4. **The structural round-trip over a *mutated* tree**, re-parsed into a document
   with a **different base offset**, so passing also proves `Equals` ignores
   provenance.
5. **The one-field delta** — mutate one node and require exactly that node and its
   ancestors to lose faithfulness, and every untouched sibling to keep it.
6. **A recognition count.** This is the one that is easy to omit and it catches
   what nothing else does: a sub-lexicon that quietly stops recognizing something
   falls back to opaque text, which **loses no bytes and breaks no round-trip**.
   Only counting what you expected to find sees it.
7. **The additive property** — the flattened token stream is identical with and
   without the readers.

**And the alarms must be pulled.** One case decides the shape of the suite: a
`Render` that hands back the retained source leaves the **byte round-trip green**
over every document, no matter how little was modelled. Removing a node from the
tree and requiring the output to lack it is the only injection that reaches that,
which is why it cannot be folded into the corpus test.
