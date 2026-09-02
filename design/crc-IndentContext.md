# IndentContext
**Requirements:** R185, R186, R187, R188

The parse context for an indent language. It carries the language during the parse
and **outlives** it to own the frame links.

## Knows
- the language, and the document
- **one index**, from a node to what this context knows about it

## Does
- answers a frame's **parent** and its direct **children**
- rebuilds when its stamp is stale, re-deriving the links from the columns

## Constraints
- **One index, and it is the outer one.** Brackets cannot contain an indent region,
  so the indent context is above the bracket context rather than beside it — and a
  brace language constructs none of this and pays nothing
- **A frame's parent is the nearest preceding `Indent` with a strictly smaller
  column.** So a dedent past several levels re-parents to the level it lands in,
  and two frames at one column under one parent are siblings — which is what a flat
  array expresses without a tree
- **The index is checkable rather than believed.** Every link is derivable from the
  flat array alone — an `Indent` carries its own whitespace, so a walk with a stack
  of open columns needs no language knowledge to reach the same answers. The pass
  records nothing; this derivation is the only one, and what checks it is written
  outside the library
- **An unmatched dedent is not refused.** A column matching no open level parents
  to the nearest smaller one; `sdom` is no syntax checker and a schema may object

## Collaborators
- IndentParser: records the links as it goes
- BracketContext: the inner context, for the bracket links
- Doc: the structural generation this stamps against

## Sequences
- seq-indent.md
