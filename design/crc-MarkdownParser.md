# MarkdownParser
**Requirements:** R226, R228, R229, R230, R231, R232, R233, R234, R235, R312

The markdown base's parser: one pass, wrapping an `IndentParser`, emitting line-head
markers where the delegate emitted nothing. With `LangMarkdown` and the three marker
kinds it is the whole base — narrow, and embedded by the file schemas rather than used
alone.

## Knows
- its `IndentParser` delegate, built over `LangMarkdown`
- the three line-head shapes: `#{1,6} `, `- `, and `[ ]` / `[x]` after a `ListItem`

## Does
- `Parse`: delegates; if nothing was emitted and nothing moved, reads whether the
  position is a **line head** from `Last()` — an `Indent`, or a `Text` ending in a
  newline followed only by spaces or tabs — and emits a `Heading` or `ListItem`; immediately after a `ListItem`, a
  `Checkbox`
- `NodeType`: answers for the three kinds, otherwise delegates
- `Done`: delegates
- `Heading.Level`: derived from the bytes

## Constraints
- **Delegate first, then check.** Sound only because no line-head marker shares a
  first byte with a bracket opener — `#`, `-`, `[` open nothing, and the code pattern
  matches at none of them. Links are kept out of the table for exactly this reason; a
  test over the table guards it
- **No post-pass, no `Replace`.** A heading or list item is a line-head marker like
  `Indent`, holding only its bytes. Extents are a consumer's derivation
- **Line-headness is read from the array**, never kept as state — the live text run
  is what makes that possible
- **Emphasis and strike are restricted with escape hatches, never code-mode groups**, so
  that only their hatches are recognized inside them; each names the code group by its
  pattern and so admits every run
- **Code hides everything inside it structurally**: the bracket parser takes the loop,
  so this parser is never offered a position there
- **The fence and the code span are one group**: a pattern opener of one or more
  backticks, `CloseIsOpen`, `RejectLongerCloses` — CommonMark's run-of-N rule in one entry,
  with a longer run inside rejected rather than content, by decision.
- **Emphasis is a run of asterisks that nests by flanking**: `AfterOpen` and `BeforeClose`
  non-whitespace, naming itself in its hatches, so a bold title with bold inside reads whole. Two groups, for three backticks and one, read a two-run as
  an empty span and flipped every later backtick from open to close, losing most of a
  file with nothing unread

## Collaborators
- IndentParser: the delegate, and through it BracketParser
- ParserState: `Last`, `Emit`, `At`
- Heading, ListItem, Checkbox: the kinds it emits

## Sequences
- seq-markdown.md
