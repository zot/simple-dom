# BracketParser
**Requirements:** R57, R72, R73, R74, R75, R76, R77, R78, R165, R166, R192

The parser. It walks a source once and appends nodes to the document in the
order it meets them.

## Knows
- its language table, its `BracketContext`, and its stack of open openers — all
  three being **its own** rather than the walk's, since the walk holds no language
- the group currently open, which it receives rather than looks up — the node
  driving the loop *is* the enclosing context

The source and the position belong to `ParserState`, which it is handed.

## Does
- parses in **code mode**: recognizes every group's openers subject to
  `AllowedParent`, then the open group's closers, then its separators
- parses in **parse-restricted mode**: recognizes only the open group's `Close`,
  its `Escape`, and the openers named in its `AllowedInner`; every other byte
  accumulates as literal text
- emits `Opener`, `Closer`, `Separator` and `Text`, appending each to the
  document's flat array

## Constraints
- **Output is flat and in document order.** An opener, everything between it and
  its closer, and the closer are **siblings**. The nesting is real while the parse
  is happening — it lives on the call stack — and is simply absent from the data
- **The parse always consumes at least one byte**, so nothing stalls on input it
  does not understand. This holds structurally rather than by a check: the text
  branch is only reached once no marker matched here, so its first advance
  always fires
- **An any-close fallback.** When nothing else matches, any code-mode group's
  closer is recognized, so a stray `}` lands as a bracket rather than derailing
  the parse
- **A group left open at end of input closes there**, with no bytes dropped
- **Whitespace is not a node of its own.** A text run is everything between two recognized
  markers, not a run of non-whitespace. A layer wanting line or indent boundaries
  contributes a parser to the same pass and emits them where they occur, rather than
  splitting text afterwards
- **It is a `Parser`**, and it **takes the loop** on an opener — recursing until the
  matching closer. That is what makes a restricted group's exclusivity structural
  rather than a flag, and what leaves a parser registered only in the outermost loop
  with no position offered inside a group at all

## Collaborators
- ParserState: the walk it is handed; the source, the position and `Emit`
- BracketLang / BracketGroup: the table it parses by
- BracketContext: the index it hands to a caller, and which it does not write to
- Doc: receives the nodes, in order

## Sequences
- seq-parse.md
- seq-collaborate.md
