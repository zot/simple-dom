# BracketContext
**Requirements:** R80, R81, R82, R83, R84, R85, R86, R87, R91, R126, R127, R128, R152, R153, R154

The schema's parse context: a concrete type, not an interface. It carries the
language through the parse and **outlives** it to own the pairing links.

## Knows
- the language, during the parse
- the **`Origin` it minted for this parse**, which every node it produces carries
- the pairing links, afterwards:
  - an **opener** knows its closer and its enclosing opener
  - a **closer** knows its opener
  - an **opener** also knows the **separators** belonging to its group
  - a **separator** knows its opener
  - **any other node** knows its enclosing opener
- the **declaration links**: a keyword node maps to every name it declares — one
  entry for a plain declaration, several for a group
- the structural generation it was built against

## Does
- mints one origin per parse, before any node exists, and hands it to the parser
- hands the parser the group currently open
- records the pairing as the parse discovers it
- reports whether its links are fresh, and rebuilds them when its stamp is stale
- **holds** the declaration links; it does not derive them

## Constraints
- **`Doc` does not own these links.** Not every document has brackets, and a
  markdown DOM would carry two dead maps forever. The layer that needs the state
  owns it
- **Stamped, not registered.** The context stamps itself with the document's
  structural generation and rebuilds when stale — so `Doc` keeps no registry and
  issues no callbacks, and reading the generation inside a mutation window refuses,
  which reaches this index without it knowing a guard exists
- **The index is checkable, not merely believed.** A forward scan that skips whole
  bracket pairs finds a node's enclosing opener **independently**, and must agree.
  That second path is the point: an index nothing can contradict is an assertion
- **Separators join that cross-derivation rather than riding along.** They are
  recorded by the parse *and* recovered by the same stack walk that recovers the
  pairing, and the two must agree. Letting the parse be the only writer is precisely
  how a representation change loses `R87` without anything going red
- **One index, not four.** The pairing, the enclosure and the declaration links live
  in a single `map[Node]BracketInfo` rather than four maps. It is a **flat value
  struct**, and it won on allocations rather than size: a boxed `NodeInfo` hierarchy
  is the smaller index — plain nodes carry 16 bytes rather than 96 — but an interface
  value cannot hold a struct inline, so every entry becomes its own heap object.
  Measured over 15,747 nodes: **111 allocations against 15,858**, on an index
  `rebuild` recreates at every structural change. The flat struct also allocates less
  than the three maps it replaces
- **How the links are stored is not part of the contract.** What the context owes is
  the answers; one map or several is its own business, which is what makes the
  consolidation a free change
- **The declaration links are the one index this context cannot rebuild.** The
  pairing is recoverable from the finished array; declarations are not, because
  `sdom` does not know what announces one in any language. Storage is here and
  **freshness is answered at the accessor**, which is the one call every reader
  makes — and a stale accessor **refuses** rather than returning the old map or an
  empty one. An empty answer reads identically to *this keyword declares nothing*,
  which is the plausible wrong answer `IndexOf` already refuses on the same grounds

## Collaborators
- Doc: supplies the nodes and the generation this stamps against
- BracketParser: the parse that populates it
- Marker: the nodes the links pair

## Sequences
- seq-parse.md
- seq-pair.md
