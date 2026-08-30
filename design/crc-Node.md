# Node
**Requirements:** R4, R5, R6, R7, R8, R9, R10, R11, R12, R13, R14, R18

The protocol every node kind satisfies. It is about **document structure and
nothing else** — parsing is not part of the contract.

## Knows
- nothing: `Node` is an interface with no state of its own

## Does
- `Kids() []Node`: the node's children in document order; a leaf returns none
- `Location() Loc`: the node's provenance and current extent
- `Render() (string, error)`: the bytes the node stands for now
- `Equals(Node) bool`: structural equality, never comparing `Location`

## Constraints
- **`Render` takes no context.** Denying it the document is what stops a node
  satisfying the round-trip with the source span it was read from
- **`Parse` is not on the interface.** Parsing constructs a concrete node and
  then asks it to parse, so the interface is not involved. Only kinds that
  genuinely parse from text declare the method
- **Each schema's parse context is a concrete type**, not an interface. Code that
  needs only the document takes `*Doc`
- **Every concrete kind declares its own `Equals`**, minimum body a type check
  that delegates. Go embedding promotes without dispatching, so a promoted
  `Equals` cannot see the outer type — but the omission is loud, not silent
- **Nesting is never children.** Where a layer needs it, nesting is links that
  layer owns
- **No kind may model a bracket group or be named one**

## Collaborators
- Loc: every node reports one
- Doc: owns the nodes and is the only thing that navigates between them

## Sequences
- seq-mutate.md
