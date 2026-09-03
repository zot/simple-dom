# Location: provenance and faithfulness

A node's location answers two different questions, and `sdom` keeps them
separate because they stop agreeing the moment anything is edited:

- **Provenance** — where in the source this node was read from.
- **Faithfulness** — whether it still renders the bytes it was read from.

A node that has been changed keeps its provenance. Clearing it would discard
what a diagnostic still wants, and would force a deferred walk to invalidate
ancestors.

## The API

```go
type Loc struct{ ... }

func (l Loc) Offset() int    // source position, or -1 for no provenance
func (l Loc) Length() int    // current rendered extent, in bytes
func (l Loc) Altered() bool  // read from Offset, no longer rendering it
func (l Loc) Faithful() bool // has provenance and is not altered
```

**The zero value means "no provenance," never "offset 0."** Offset 0 is the first
byte of the file — a plausible wrong answer that nothing would be forced to
correct. Absence is the zero value, so a location nobody set cannot be mistaken
for a location at the start of the document.

## Offsets are relative to the document's own source

An offset indexes the source of the document the node belongs to. It starts at 0
for the first byte of that source, whatever the document's `base` may be — **the
base never enters a location.** A consumer wanting a position in an outer document
adds the base itself.

This is what makes `Faithful` checkable: a faithful node renders exactly
`source[Offset : Offset+Length]` of its own document, with no adjustment.

`Faithful` is the question callers should actually be asking. A faithful node
renders exactly the source span at its offset; an unfaithful one does not, and
the two are not distinguishable from `Offset` alone.

## Origin: which parse this came from

An offset alone is not globally meaningful. Two nodes at offset 10 from different
parses are indistinguishable, so a location also carries the **parse it came
from**:

```go
// Origin identifies one parse, not one file. Two parses of the same source are
// two Origins.
type Origin struct {
    Name string // a path, a URL, or whatever the caller finds useful
}

func (l Loc) Origin() *Origin  // nil when unknown
func (l Loc) In(o *Origin) Loc // the parser's chained setter
```

**It is the parse context, not the document.** The context exists before the nodes
do — a parse mints it, runs, and only then builds the document from what it
emitted — so a document reference would need back-patching over every node.

**It is a concrete type, not an interface.** Each schema's parse context is its
own concrete type, so there is no single one for a location to hold; a small token
that every parser mints and its context keeps is what makes this work without an
abstraction, and it doubles as a lookup key. It carries a field rather than being
an empty marker because Go may give every zero-size allocation the same address,
and an empty `Origin` would fail as an identity exactly where it was needed.

**Setting it changes no existing call site.** `Source(pos, n)` is unchanged, for
the many places with no origin to give; a parser writes `Source(pos, n).In(o)`.

**A nil origin is unknown, not different.** A synthesized node has none, and that
is compatible with anything — the same way no provenance already is.

`Equals` still never compares `Location`, so this reference cannot make two
documents unequal. That is what distinguishes it from a node holding a pointer to
its bracket group, which is forbidden for exactly that reason.

## `Altered` is derived, not stored

A compound is altered if **any** of its children is. This is computed on read,
not stamped at edit time and propagated upward.

Deriving it removes the invalidation walk entirely, and that walk carried a quiet
defect: a fold that short-circuits leaves later compounds claiming spans they can
no longer honour while the root's answer stays correct. The read has to visit
every child to sum lengths anyway, so nothing can be skipped and nothing is saved
by stamping.

**Caveat, and it must be carried into every diagnostic.** For an altered node
`Offset` is historical while `Length` is current, so the pair names a span the
node never owned. **Use `Offset` alone unless `Faithful()`.**

## `Split` and `Merge` are re-granulation

They move **boundaries**. Bytes do not move and provenance does not move — the
same source, described at a different granularity.

**`Split`** divides one node's span into two, each keeping the provenance of the
part of the source it now covers.

**`Merge`** joins two into one, and **requires adjacency**. Both are `Doc`
methods, because both change node membership.

Adjacency is **checked when both operands are faithful** — their two locations
prove it on their own, at the call, with no lookup. Otherwise it is **not
checked**. An altered node's `Offset` is historical while its `Length` is current,
so the arithmetic proves nothing about it, and carrying adjacency past that point
would mean tracking it across queued edits. The requirement still holds; only the
check is absent.

**Two present, differing origins never meet.** A document refuses nodes of two
parses when it is built (see [document.md](document.md)), and `Merge` joins only
nodes already in one document, so the merge arithmetic is never asked about two
coordinate systems and carries no check for it. A nil origin on either side is
absent rather than different.

That also settles adjacency across documents: two faithful nodes from different
parses whose offsets happen to satisfy `10+5 == 15` are never in one document to be
merged by accident.

The merged node's location follows two rules:

- It is **faithful only if both operands are.** "Unfaithful" covers both ways an
  operand can fail — altered, or having no provenance at all — and either one is
  enough to make the result unfaithful, because the merged bytes are then not the
  source span at any single offset.
- It takes **the first operand's offset if it has one, and the second's
  otherwise** — the leftmost provenance in the merged span. That makes merging a
  run of nodes independent of how the merges are grouped: the answer is the same
  left-to-right as right-to-left.

Because neither operation invents or destroys bytes, a split followed by a merge
of the same pair returns the original span with its provenance intact.
