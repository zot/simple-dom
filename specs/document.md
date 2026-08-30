# The document

A `Doc` is a parsed source: the bytes, the flat document-order array of nodes
over them, and two derived indices. It holds **nothing lexicon-specific**.
Anything else derived — bracket links, a scope relation, an anchor index — is
owned by the layer that needs it, so a document type that needs no such state
carries none.

A `Doc` also carries a **base** offset, its own position within an outer
document, so the same text can be parsed as a sub-document without its nodes
lying about where they came from.

## What `Doc` holds

- the **source** bytes
- the **base** offset
- **`dom`** — the nodes, flat and in document order
- **`data`** — an open slot for whatever a consumer wants to hang here
- two derived indices over `dom` and nothing else: one from **node to position**,
  one over **lines**

## Navigation

```go
func (d *Doc) Prev(n Node) Node
func (d *Doc) Next(n Node) Node
```

Navigation is by node, not by index. A caller holds nodes and asks the document
where they sit; positions are never a currency the caller carries.

## The structural generation

`Doc` exposes a **monotonic generation**, bumped whenever **node membership**
changes. A content edit does not bump it: membership is unchanged, and an index
over structure survives one.

Any layer holding state derived from the document's structure **stamps itself**
with the generation it was built against, and rebuilds when the stamp is stale.
`Doc` therefore needs no registry of indices, no invalidation callbacks, and **no
knowledge of what exists above it**. The layers pull; the document does not push.

## Mutation

```go
func (d *Doc) Mutate(f func() error) error
func (d *Doc) Split(n Node, at int) (Node, Node, error)
func (d *Doc) Merge(a, b Node) (Node, error)
```

`Mutate` brackets a set of edits. Inside it:

- **Edits are direct.** A content write changes the node; a structural change
  changes the array. Nothing is queued and nothing is deferred.
- **Navigation is forbidden.** `Prev`, `Next`, the position lookup, the line
  lookup and **reading the generation** all refuse, by panicking with a typed
  sentinel that `Mutate` converts to an error. A panic from anywhere else is
  re-raised untouched, so real bugs keep their stack.
- **Resolve your targets before you enter.** Navigation is legal outside the
  window, so a caller works out which nodes to edit there and carries **node
  references** in — those survive whatever else the mutation does. A position
  would not.
- The derived indices rebuild **once**, at the exit.

**The derived indices are updated only after a mutation completes.** That is why
every index-backed read refuses while the window is open — there is nothing
correct for it to return. `Doc.Render` and a node's own accessors stay open,
because they consult no index: under direct edits, what you have written is
simply there.

**Why refusal rather than a stale answer.** Rebuilding an index mid-window would
not rescue the caller: if the node they hold has been removed, a position lookup
returns −1 and `Next` hands back nothing, which is indistinguishable from
end-of-document. Refusing says what happened. Making the *generation* read refuse
is what extends the guard to a **layer's** index without the layer knowing the
guard exists, since checking freshness is the one call every stamped index makes.

### Nesting, failure, and the absence of rollback

`Mutate` **saves and restores** rather than counting, so a nested call is a
pass-through.

There is **no rollback**. Edits landed as they were made, so there is nothing to
roll back to, and a partial restore would produce a plausible wrong state rather
than an obvious one.

**An error or a panic escaping the mutation function poisons the document.** Both
mean something the caller could not or did not handle got out, so what was
applied by the time it did is unknown. A caller that *can* handle a failure
handles it inside the function rather than propagating it. There is no reset:
recovery is to re-parse, because a poisoned document means a bug in the code that
wrote to it.

## What this deliberately does not do

There is no operation log, no queued edit plan, no transaction and no undo. A
later layer is free to build any of them **on top of** this: the mutation window
is the seam it would attach to, and edits keyed by node identity rather than
position are already the primitive such a system needs.
