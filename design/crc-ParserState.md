# ParserState
**Requirements:** R155, R156, R157, R158, R161, R162, R163, R164, R167

One pass over one source. It owns the walk and the loop; it knows nothing about
any language.

## Knows
- the source, the position, and where the pending text run began
- the nodes emitted so far — as a count and an appender, never as an array
- the **`Origin`** for this parse, which every node it produces carries
- the one `Parser` driving it

## Does
- offers each position to the parser **exactly once**
- treats a position as unrecognized only when **neither** the position **nor** the
  node count changed, and then takes one byte as pending text
- flushes pending text before any node is emitted, so the array stays in document
  order without anyone ordering it
- builds the document from what was emitted, and returns it alone

## Constraints
- **`Emit` and `NodeCount`, never the slice.** A parser appends and asks how many;
  it never needs the array, and handing out the live one is the aliasing shape two
  open gaps already record
- **One parse, one `Origin`.** With several parsers collaborating there is still
  one pass. A per-context origin would mint two for a single document, and merging
  a location from one with a location from the other is defined to panic
- **The walk arbitrates nothing.** It holds one parser; which language rule wins is
  that parser's business, and a loop deciding between parsers would be a second
  place to encode precedence
- **Progress is a byte or a node**, so the loop terminates. A parser that emits at
  one position without ever advancing would spin — a parser bug, stated rather
  than guarded, since the check would cost every position of every parse to catch
  what a first run makes obvious

## Collaborators
- Parser: the one it drives, which may delegate to others
- Doc: receives the emitted nodes, in order
- Loc / Origin: every node's provenance comes from this pass

## Sequences
- seq-collaborate.md
