# MarkdownParser
**Requirements:** R226, R227, R228, R229, R230, R231, R232, R233, R234, R235

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
  first byte with a bracket opener — `#`, `-`, `[` open nothing. Links are kept out of
  the table for exactly this reason; a test over the table guards it
- **No post-pass, no `Replace`.** A heading or list item is a line-head marker like
  `Indent`, holding only its bytes. Extents are a consumer's derivation
- **Line-headness is read from the array**, never kept as state — the live text run
  is what makes that possible
- **Bold and strike are restricted with escape hatches, never code-mode groups.** A
  symmetric marker in code mode reopens rather than closes, because openers are tried
  before the enclosing closer there; restricted groups check the closer first
- **A fence or code span hides everything inside it structurally**: the bracket parser
  takes the loop, so this parser is never offered a position there

## Collaborators
- IndentParser: the delegate, and through it BracketParser
- ParserState: `Last`, `Emit`, `At`
- Heading, ListItem, Checkbox: the kinds it emits

## Sequences
- seq-markdown.md
