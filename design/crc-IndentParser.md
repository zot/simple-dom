# IndentParser
**Requirements:** R174, R177, R178, R179, R180, R181, R182, R183, R184

A `Parser` that turns line starts into scope. It holds a `BracketParser` and hands
off whenever it does not match, so one pass produces both kinds of node.

## Knows
- its `IndentLang`
- the stack of open columns
- its `BracketParser`, and its own `IndentContext`

## Does
- at a **line start**, decides whether the level changed and emits an `Indent`
  carrying that line's leading whitespace
- emits the **root** `Indent("")` when nothing has been emitted yet
- **delegates** to its bracket parser everywhere else
- records **nothing**: the frames are implied by the columns it emits

## Constraints
- **A node at every change and none where the level is unchanged.** Consecutive
  lines at one column share the frame the last change opened, and their leading
  whitespace stays ordinary text — so the same bytes are a node on one line and
  text on the next, which is what keeps a node meaning *something happened here*
- **Two of them are zero-length**: the root, and a return to column 0, which has no
  whitespace to own. Half-open indices make both free
- **A dedent is no node of its own.** The column *is* the level, so closing several
  levels at once is one node with a smaller column
- **The column is derived from the text**, tabs expanded by `Tab`, never stored
- **Blank lines and `Transparent`-only lines do not change the level.** Deciding
  needs `NodeType` from the delegate — a comment and a string both open a
  restricted group, and only the `Kind` separates them
- **A `Continuation` marker counts only at bracket depth 0**, which needs no rule
  of its own: one inside a comment group is at depth 1 and does not continue, and
  one inside a string needs nothing because the group is still open at the next
  line start
- **It is registered only in the outermost loop.** That, not a check, is what makes
  indentation significant only at bracket depth 0
- **The root is emitted here, not by the walk.** Guarding on `NodeCount() == 0`
  keeps `ParserState` ignorant of indent and gives a non-indent parse no root at
  all. An empty source produces no nodes, since the loop never calls `Parse`

## Collaborators
- ParserState: the walk it is handed
- BracketParser: held, delegated to, and asked `NodeType`
- IndentContext: receives the frame links
- IndentLang: `Tab`, `Transparent`, `Continuation`

## Sequences
- seq-indent.md
- seq-collaborate.md
