# Stencils: compounds that parse by regex

A **stencil** is a compound a tool writes *into* — a checkbox, a field, a name.
It parses itself from text with a regex whose **named groups are the fields it
binds**, and its children tile its span so nothing it matched can go missing.

## The builder

Parsing a stencil is the same computation every time — find the match, lay out
the bytes, tile the children — while *what each group becomes* differs per
schema. So the machinery is a **builder the schema drives**, not a type the
schema inherits from:

```go
func NewStencilBuilder(re *regexp.Regexp, text string, loc Loc) (*StencilBuilder, bool)

func (b *StencilBuilder) Group(name string) (string, Loc) // the matched text and its provenance
func (b *StencilBuilder) Put(name string, n Node)         // patch a built node into the children
func (b *StencilBuilder) Omit(name string)                // this group is plain text after all
func (b *StencilBuilder) Done() (kids []Node, remain string)
```

It is a **builder**, never a `Node`. "Stencil" alone would collide with the
established name for the *region* a tool writes into; this is the tool that cuts
it.

`NewStencilBuilder` reports `false` when the regex does not match, and what that
means is the schema's business, not the builder's.

**On construction it lays out the whole span**: a `Text` for the head of the match,
one for every gap between consecutive named groups, one for the tail — and a
**nil** in each named group's slot. The schema fills the nils.

**The glue is computed, never required.** A regex names only the groups its schema
binds; everything else becomes `Text` from the gaps between them. A pattern
therefore cannot silently eat bytes, which is why no parse-time tiling check
exists — the failure is unrepresentable rather than detected.

**A named group starts nil rather than defaulting to `Text`.** A group exists
*because* a schema binds it, so a group named and never filled is a group that
should not have been named — and a nil makes that omission loud where a default
would swallow it.

**`Done` panics on either failure it can see.** A slot left nil, and a plugged node
whose span does not match its group's. Both are programming errors, and the second
is the only way a schema can break tiling from here — it would otherwise surface
as the compound reporting itself *altered*, a plausible wrong answer rather than a
refusal.

### Two ways a group goes unfilled, and only one is the schema's business

**A group that did not participate in the match** — the losing branch of an
alternation — has no bytes. The builder sees that for itself and gives it no slot,
no glue and no nil, so nothing is owed and nothing panics.

**A group that participated but the schema does not want as a field** is what
`Omit` is for. Those bytes exist and must go somewhere, so `Omit(name)` fills the
slot with a `Text` and marks the group as glue; **`Done` then merges it with its
neighbours**. Without it a schema would have to `Put` a `Text` it does not care
about purely to satisfy the nil check — which would make the child list claim a
field where there is none, against the minimality rule.

**The invariant `Done` leaves: no two adjacent children are both plain glue.** An
omitted group would otherwise sit between the glue before it and the glue after it
as three `Text` nodes where one belongs, implying a boundary that nothing writes
into. The merge is ordinary re-granulation — same origin, adjacent, both faithful —
so provenance survives it.

**Which gives `Omit` a checkable meaning**: omitting a group must produce the
**identical** child list to a regex that never named it. That is what "omit" means,
stated as something a test can falsify rather than as an intention.

The two are distinguishable through `Group` without a third return value, by the
rule `Loc` already has: a non-participating group yields the **zero `Loc`** — no
provenance — while a participating-but-empty group yields a real one at a real
offset. Absence is the zero value.

**Binding is by group name, never by position.** With alternation the branches have
different group counts, so a fixed index is right for one input and out of range
for another. **Nested groups are skipped**: they cannot double-count and they
produce nothing.

Indices are **half-open** throughout — contiguity is `next.start == prev.end`, with
no `+1` anywhere, and a zero-length group is `[k,k)` needing no special case. A
zero-length node is *useful*: an insertion point a tool can later write into
without moving its neighbours.

There is no `byName` result. The schema assigned every node it built to its own
field on the way past, so handing the same references back in a map would be a
second copy of something that only flows one way.

## Bound values

A stenciled field is a **typed value over a `Text` node**, reachable through the
child list where it tiles and renders, and through a typed pointer where a tool
reads and writes it.

```go
type Bool struct{ txt *Text }

func (b *Bool) Value() bool // derived on read
func (b *Bool) Set(v bool)  // written through to txt
```

**The text is the storage; the value is a view of it.** Nothing is stored twice,
so nothing can drift, and `Equals` needs no special case — a kind whose state is
derived from its children compares nothing beyond them.

A `Bool` does not *replace* the node in the child list. It **points at** the `Text`
the schema put there:

```go
str, loc := st.Group("checked")
txt := NewText(str, loc)
t.checked = &Bool{txt: txt}
st.Put("checked", txt)
```

**`Set` writes through and keeps the offset.** Flipping `- [    ]` to true writes
`x`: the span contracts, the node keeps its provenance and becomes altered, and
the enclosing stencil is altered because a child is.

**Reading is a derivation, never a lookup of stored state.** `[    ]`, `[]` and
`[ ]` all read false and all render back byte-exact, because nothing is normalised
on the way in.

## What a schema writes

A markdown todo item is the worked example:

```go
var todoRe = regexp.MustCompile(`- \[(?P<checked>[^\]]*)\] (?P<label>.*)`)

type TodoItem struct {
    Compound
    checked *Bool
    label   *Text
}
```

Its `Parse` builds what each group becomes, keeps its own references, and hands
the children to the compound. `Parse` is **not** on the `Node` interface: parsing
constructs a concrete node and then sends it `parse`, so the interface is never
involved at that moment.

Note what `todoRe` does not capture. `- [`, `] ` and the line's end are not in the
pattern at all; they become `Text` from the gaps. The author writes what they bind.

## Minimality

**Editability chooses the span, and the span chooses the children.** Only what a
tool *writes into* is a bound field. Within a span, tiling forces children to cover
every byte, glue included — so minimality constrains the **span's width**, never
the child count inside it. When stenciled parts sit far apart, make several stencil
nodes rather than one wide span that drags the text between them in as children.

The test to apply to any child: *would a tool ever write into this?* If not, it
should have been a sibling and the span was too wide.
