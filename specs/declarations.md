# Declarations — the machinery

`sdom` knows what a declaration **is**: a keyword node, and the name nodes that
keyword introduces, carved out of an already-parsed document and left in place.

It does **not** know how any language announces one. Keyword sets, keyword-less
forms, where a name sits relative to its keyword — all of that is a fact about one
language, and it lives in a schema. See [declaration-schemas.md](declaration-schemas.md).

## Typed nodes, not a span

There is **no declaration node and no declaration span.** A declaration is a
`DeclarationType` sliced out of the keyword and one or more `DeclarationName`s
sliced out of the names, all ordinary siblings in the flat array:

```
func (s *Store) Index(k string) int {

[ Text{"\n"}, DeclarationType{"func"}, Text{" "}, Opener{"("},
  Text{"s *Store"}, Closer{")"}, Text{" "}, DeclarationName{"Index"},
  Opener{"("}, … ]
```

**The writable parts are separated by structure**, so one span covering them would
drag a receiver group in as children — which the minimality rule forbids and the
span rule would then oblige. Narrow typed nodes with the structure between them as
ordinary siblings is the shape.

```go
// DeclarationType is a declaration's keyword.
type DeclarationType struct{ Text }

// DeclarationName is a name that keyword introduces.
type DeclarationName struct{ Text }
```

**Nothing is consumed that the parse already claimed.** Text nodes are split and
some halves change kind, so the flattened array is identical with and without a
declaration pass, and no marker stops being recognized. That property is
structural rather than a rule a schema must honor: a pass that only splits and
re-types cannot violate it.

## Re-granulating a node into a typed one

Slicing a keyword or a name out of a text node is `Split`, then a replacement, so
the document gains one structural verb:

```go
// Replace swaps one node for another in the document, preserving position.
// Membership changes, so it bumps the structural generation and requires an
// open mutation window.
func (d *Doc) Replace(old, new Node) error
```

It is `sdom`'s and not a schema's for a reason that is not preference: Go does not
permit a method on `*Doc` to be declared from another package.

## Declaration links

**A `DeclarationType` does not hold its names.** No node kind holds state its
children do not carry — the same rule that keeps a bracket marker from holding its
group.

The **map lives on `BracketContext`**, beside the pairing links, because storage
for "which names does this keyword declare" is machinery. **Filling it in is a
schema's job**, because only a schema knows what announces a declaration.

```go
// in BracketContext, beside the pairing links:
declaration []*DeclarationName   // on a DeclarationType's entry: every name it declares
```

The link is **typed at both ends**. A schema records it with
`SetDeclarations(map[*DeclarationType][]*DeclarationName)` and a consumer reads it
with `DeclarationNames(t *DeclarationType) ([]*DeclarationName, error)`. Pointers,
not values: the names returned are the very nodes in the document, so a consumer can
match one against a node it skimmed. It is a **method on the context**, not an
interface — every schema stores its links here, so an interface would have one
implementer. Where it is stored is not part of the contract; that a keyword answers
for its names is.

**The relation is one-to-many**, because a grouped declaration declares several
names: one entry for a plain declaration, several for a group.

**Freshness is answered at the accessor, and it refuses.** The bracket links are
*stamped, not registered* — safe because `rebuild` re-derives them from the finished
array. Declaration links cannot be re-derived: `sdom` does not know what announces a
declaration. So the schema's write stamps the links against the document's structural
generation, and `DeclarationNames` on a document that has changed since returns
`ErrDeclarationsStale` rather than the old links or an empty answer. An empty answer
would read identically to *this keyword declares nothing*; the error says *re-run the
schema's pass*, which is recoverable.
