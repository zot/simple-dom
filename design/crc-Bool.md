# Bool
**Requirements:** R111, R112, R113, R114

A typed view over a `Text` node — the first bound value, and the shape every other
one follows.

## Knows
- the `Text` it is a view of, and nothing else

## Does
- `Value()`: derives the value from the text, on every read
- `Set(v)`: writes the text through, which keeps its offset and becomes altered

## Constraints
- **The text is the storage.** Nothing is stored twice, so nothing can drift, and
  `Equals` needs no special case — a kind whose state is derived from its children
  compares nothing beyond them
- **It points at the node in the child list; it does not replace it.** The schema
  puts a `Text` in the children and hands the same pointer here
- **Nothing is normalised on the way in.** `[    ]`, `[]` and `[ ]` all read false
  and all render back byte-exact
- Setting it contracts or grows the span while the node keeps its provenance, so
  the enclosing stencil reports altered because a child is

## Collaborators
- Text: the storage
- StencilBuilder: puts the text in the child list that this views

## Sequences
- seq-stencil.md
