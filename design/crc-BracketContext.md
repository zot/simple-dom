# BracketContext
**Requirements:** R80, R81, R82, R83, R84, R85, R86, R87, R91

The schema's parse context: a concrete type, not an interface. It carries the
language through the scan and **outlives the parse** to own the pairing links.

## Knows
- the language, while scanning
- the **`Origin` it minted for this parse**, which every node it produces carries
- the pairing links, afterwards:
  - an **opener** knows its closer and its enclosing opener
  - a **closer** knows its opener
  - **any other node** knows its enclosing opener
- the structural generation it was built against

## Does
- mints one origin per parse, before any node exists, and hands it to the parser
- hands the parser the group currently open
- records the pairing as the scan discovers it
- reports whether its links are fresh, and rebuilds them when its stamp is stale

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

## Collaborators
- Doc: supplies the nodes and the generation this stamps against
- BracketParser: the scan that populates it
- Marker: the nodes the links pair

## Sequences
- seq-scan.md
- seq-pair.md
