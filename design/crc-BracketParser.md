# BracketParser
**Requirements:** R57, R72, R73, R74, R75, R76, R77, R78, R165, R166, R192, R294, R297, R309, R346, R347, R353, R356, R358

The parser. It walks a source once and appends nodes to the document in the
order it meets them.

## Knows
- its language table, its `BracketContext`, and its stack of open groups — each frame
  the group and the text that opened it, pushed and popped by `open`, empty when the
  parse ends — all three being **its own** rather than the walk's, since the walk holds
  no language
- the group currently open, which it receives rather than looks up — the node
  driving the loop *is* the enclosing context

The source and the position belong to `ParserState`, which it is handed.

## Does
- parses in **code mode**: recognizes every group's openers subject to
  `AllowedParent`, then the open group's closer, then its separators
- parses in **parse-restricted mode**: recognizes only the open group's closer,
  its `Escape`, and the openers of the groups its `AllowedInner` names; every other
  byte accumulates as literal text
- **carries the opener's text down to the closer check**, so a `CloseIsOpen` group
  closes on exactly the bytes that opened it — checked before any opener in either
  mode — and every opener is matched under its group's `AfterOpen`, every closer under
  its `BeforeClose`
- **matches a `CloseRegex` closer only where its named groups agree** with the opened
  text, matched against `OpenRegex` again rather than remembered; a match that disagrees
  is content, and the parse moves one byte on, so an overlapping real closer is still
  found. A pattern closer never lands as a stray
- **takes a run that is not the opener's text whole**, inside a close-is-open pattern
  group: shorter as content, longer as content or — with `RejectLongerCloses` — as an
  unbalanced closer that ends the group; the hatches are tried first, so emphasis nests
- **closes an enclosing group from inside a child**, in code mode: a closer that is not
  the open group's but closes a group further down its stack, the nearest first, makes
  the loop return unclosed without consuming; each group in between ends as at end of
  input, and the frame whose closer it is closes on it
- compiles each group's `OpenRegex`, `AfterOpen` and `BeforeClose` once, at construction;
  markers stay byte comparisons
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
- **An any-close fallback.** When nothing else matches and no enclosing group closes
  here, any code-mode group's
  **literal** closer is recognized, so a stray `}` lands as a bracket rather than
  derailing the parse; a close-is-open marker outside its group is an opener and has
  already matched as one
- **A group left open at end of input closes there**, with no bytes dropped — unless it
  demotes: then the opener was text, and the parser **rewinds** to the byte after it, drops
  the opener and everything since, and returns to the enclosing loop, which re-reads the
  bytes in its own mode; a `BlankLineBound` group demotes at a blank line the same way,
  unless `LineHeadUnbound` and its opener stood at a line head — decided once, at the
  opener, and carried down with the opener's text. The
  indent parser and the markdown wrapper are never offered a position inside a group, so
  their state needs no rewind
- **records each demotion on the context** at the rewind — marker, the run it folded into,
  its offset there — and drops the records made inside a group it then demotes itself
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
