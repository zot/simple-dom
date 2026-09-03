# ParserState
**Requirements:** R155, R156, R157, R158, R161, R162, R163, R164, R167, R224, R225

One pass over one source. It owns the walk and the loop; it knows nothing about
any language.

## Knows
- the source and the position
- the nodes emitted so far — as a count, an appender and the **last node**, never as
  an array; the last node is the **live text run** while the walk is inside one
- the **`Origin`** for this parse, which every node it produces carries
- the one `Parser` driving it

## Does
- offers each position to the parser **exactly once**
- treats a position as unrecognized only when **neither** the position **nor** the
  node count changed, and then **advances** one byte — creating the text run on the
  first declined byte and extending it on every later one, a substring and a length
  bump, so the array is in document order without anyone ordering it
- distinguishes **`Advance`**, which consumes bytes as text, from **`SetPos`**, which
  only moves — a lookahead goes forward and back and changes nothing
- ends the live run at `Emit`; the next declined byte starts a new one
- builds the document from what was emitted, and returns it alone

## Constraints
- **`Emit`, `NodeCount` and `Last`, never the slice.** A parser appends, asks how
  many, and looks at the node before the position; it never needs the array, and
  handing out the live one is the aliasing shape two gaps already record
- **A parser never emits over bytes already in the live run** — stated at `Emit`,
  not guarded. None does, and the run is not shrunk there
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
