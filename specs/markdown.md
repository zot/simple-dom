# The markdown base

The markdown mini-spec's own documents are written in, as an `sdom` schema — **the
subset those documents use, and only the structure a reader binds.** It is a base:
the trajectory file schemas embed it and add their own stencils, and nothing parses a
file with the base alone. It lives in `sdom/schema` beside Go and Lua because markdown
is a language and nothing here knows what a queue is.

**Narrow, on purpose.** Headings, list items, the checkbox, the fence, the code span,
`**` and `~~`. As the parsers over it accumulate, commonality among them earns
generalization; nothing is generalized ahead of it. The seam for more is a
`BracketLang` entry.

## The table

```go
var LangMarkdown = sdom.IndentLang{ /* depth is a list item's indentation */ }
```

Its bracket groups, in match order:

| group | open / close | mode | kind |
|---|---|---|---|
| code | a run of backticks, closed by a run of the same length and no other | restricted: nothing inside is recognized | `code` |
| bold | `**` … `**` | restricted; admits code | |
| strike | `~~` … `~~` | restricted; admits bold and code | |

**The fence and the code span are one group**, a pattern opener of one or more backticks
with `CloseIsOpen` and a lookahead refusing a following backtick — CommonMark's rule that a
run of N is closed only by a run of exactly N, for any N. Before this a two-run read as a
one-span that opened and closed at once, the lone backtick inside it opened a real span,
and every later backtick in the file flipped open for close: 41 of 58 done entries vanished
from one file with nothing listed as unread, because nothing entry-like survived to be
unread. A four-backtick fence around a fenced sample needs nothing the span does not.

**Bold and strike are restricted with escape hatches** — the template-literal shape —
rather than code-mode groups, so that only their hatches are recognized inside them. The
hatches are exactly what a part line uses — `~~**Item 1 …**~~` and a marker span holding
code — and no more; each names the code group once, by its pattern, and admits every
run.

The one code group carries the kind `code`, so a consumer that wants to skip code skips one
label. **Links are not groups.** `[doc](path)` is text; the stencil that wants the path
reads it out of the run. Keeping `[` out of the table is also what keeps the checkbox
recognizable — see the ordering rule below.

## The parser

```go
type MarkdownParser struct { /* holds an *sdom.IndentParser */ }

func NewMarkdownParser() *MarkdownParser
func (p *MarkdownParser) Indent() *sdom.IndentParser   // the delegate, for its contexts
// and the three Parser methods: Parse, NodeType, Done
```

One pass, no post-pass. `MarkdownParser` implements the `Parser` interface and holds an
`IndentParser` the way that one holds a `BracketParser`. Its `Parse` **delegates first**;
when the delegate emitted nothing and moved nothing, and the position is a **line
head**, it recognizes a line-head marker and emits it:

| marker | bytes it holds | recognized when |
|---|---|---|
| `Heading` | `# ` … `###### ` | at a line head |
| `ListItem` | `- ` | at a line head |
| `Checkbox` | `[ ]` or `[x]` | immediately after a `ListItem` |

**A line head is read from the array, not kept as state:** `Last()` is an `Indent`, or a
`Text` ending in a newline followed only by spaces or tabs — an indented line at an
*unchanged* level keeps its leading whitespace in the run, since an `Indent` is emitted
only on a change. That is what the live text run was landed for.

**A line-head marker shares no first byte with any bracket opener.** `#`, `-` and `[`
open no group, and the code group's pattern matches at none of them. This is the rule that lets the wrapper check *after* delegating: were
`[` an opener, the bracket parser would claim a checkbox before the wrapper saw it. It
is stated here and guarded by a test over the table.

**Inside a fence or a code span the wrapper is never offered a position.** The bracket
parser takes the loop for a restricted group, so `- [ ]` inside a fence is text with no
rule needed — the same structure that makes indentation significant only at depth 0.

Each marker holds only the bytes it matched. A heading's **level** is derived from
those bytes, and a checkbox's **`Checked`** likewise, with `SetChecked` writing the
interior through — the checkbox owns its own semantics, since it holds the brackets
and a bare `Bool` would not read it. Its **extent** — the rest of the line, the item's body, the region under a
heading — is a consumer's derivation from the array, as the indent frames are; the base
records nothing.

`NodeType` answers for the three line-head kinds and otherwise delegates, so lookahead
and transparency keep working through it. `Done` delegates.
