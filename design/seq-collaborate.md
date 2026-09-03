# Sequence: one pass, several parsers
**Requirements:** R155, R159, R162, R163, R164, R165, R166, R167, R224

How a position is offered, what counts as nothing having happened, and how one
parser hands off to another inside the same walk.

## 1. The walk

1. `Parse` builds the state and turns the loop
   1.1. `ParserState` is minted with the source, position 0, and **one `Origin`**
        for the whole pass
   1.2. While the position is inside the source, remember the position and the
        node count, then hand the state to the parser
   1.3. The parser recognizes something and emits it, advancing past it — or
        recognizes nothing and touches nothing
   1.4. **Nothing happened** means the position is unchanged **and** the node count
        is unchanged; only then does the walk **advance** one byte — creating the live
        text run on the first declined byte, extending it on every later one
        1.4.1. Position alone would miss a **zero-length** node, and the next byte
               can be a bracket opener rather than the line's content
        1.4.2. Node count alone would miss a parser that **advances without
               emitting**, as an escape inside a restricted group does
   1.5. Each position is offered **exactly once**: whatever the parser did, the
        walk either advanced or the parser did
   1.6. At the end the document is built from the emitted nodes — nothing is flushed,
        since every byte is already in the array — and the walk returns the document
        alone, because each parser already owns its context

## 2. Handing off

2. One parser, delegating
   2.1. The state holds **one** parser; an indent parser holds a bracket parser as
        a field
   2.2. The indent parser tries its own rule at this position
   2.3. It does not match, so it calls its bracket parser with the same state —
        **precedence is decided here**, in the language, never by the walk
   2.4. The bracket parser matches an opener and **takes the loop**, recursing
        until the matching closer
        2.4.1. Inside that recursion the indent parser is never offered a position,
               because it is registered only in the outermost loop
        2.4.2. So *indentation is significant only at bracket depth 0* holds
               structurally, with no depth check anywhere
        2.4.3. A restricted group's exclusivity holds the same way, with no flag
   2.5. The group closes, the recursion returns, and the outermost loop resumes
        offering positions to the indent parser again

## 3. Asking without consuming

3. A parser looks ahead
   3.1. The indent parser reaches a line start and must decide whether this line
        changes the level
   3.2. It asks its delegate for the **kind** of node that would be emitted here,
        which emits nothing and moves nothing
   3.3. *No node* and *a node with no kind* come back as different answers, because
        a line of code and a line opening a string are both significant while a
        comment-only line is not
