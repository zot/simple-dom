# The node protocol

`sdom` parses a document into a **flat, document-order array of nodes**. A node is
a span of the document that something wants to *address* — read it, compare it,
render it back, or write into it. Everything the tool does not operate on stays
where it was, as bytes, and comes back out unchanged.

Module `github.com/zot/simple-dom`, package `sdom` in `sdom/`.

The array is flat because searching and splicing are the operations this DOM
exists for. Nesting, where a layer needs it, is expressed as links owned by that
layer — never as node children.

## The interface

```go
type Node interface {
    Kids() []Node
    Location() Loc
    Render() (string, error)
    Equals(Node) bool
}
```

- **`Kids`** — the node's children, in document order. A leaf has none.
- **`Location`** — where the node was read from and how much it currently is.
  See [location.md](location.md).
- **`Render`** — the bytes the node stands for *now*.
- **`Equals`** — structural equality.

### `Render` and `Equals` take no context

`Render` is not given the document. A node renders from what it actually
modelled, and cannot satisfy a round-trip by reaching for the source span it was
read from.

`Equals` never compares `Location`. Two nodes read from different files with
identical content are equal. Including provenance would make the structural
round-trip false for every mutated node, which is precisely the tree that test
most needs to be true of.

### `Parse` is not on the interface

Parsing constructs a *concrete* node and then asks it to parse, so the interface
is not involved. Only the compounds that genuinely parse from text carry the
method — the traceability comment and its fields. `Node` is about document
structure and nothing else.

Each schema has its own **parse context**, a concrete type rather than an
interface, carrying parse-time-only state shaped for that schema.

## Equality, per kind

**Every concrete node kind declares its own `Equals`**, and the minimum body is a
type check that delegates:

```go
func (d *Decl) Equals(other Node) bool {
    o, ok := other.(*Decl)
    return ok && d.Compound.Equals(&o.Compound)
}
```

A kind holding state the children do not already carry compares that state in the
same expression. A kind whose state is derived from its children compares nothing
extra, because the children already settle it.

Declaring it is a discipline, not a compiler-enforced one, but **forgetting it is
loud**: an undeclared kind inherits `Compound.Equals`, whose type assertion fails
against the new kind, so two identical nodes of that kind compare unequal and the
structural round-trip goes red on the first document containing one.

## The kinds this package defines

- **`Text`** — a leaf. Holds bytes, has no children.
- **`Compound`** — a node whose children **tile its span**. It renders by
  concatenating its children's renders. It does no parsing.

`Compound` is embedded by every compound the layers above define — the regex
compound, the declaration, the traceability comment — each of which differs only
in **how its children are computed**. That difference is theirs; the tiling,
concatenation, summed extent and propagated alteration are defined once, here.

## What is not a node

**Compound nodes exist only for stenciling** — a region a tool writes *into*.

A **bracket group is not a node**. An opener, everything between it and its
closer, and the closer are siblings in the flat array. Modelling groups as
compounds would make every span query a traversal and every edit a re-parent. No
kind defined here or above may be a bracket group, or be named one.
