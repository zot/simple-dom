# ShellSchema
**Requirements:** R136

The case with no keyword at all, and the one whose assignment rule is stricter
than it looks.

## Knows
- `NAME=value` — assignment, with **no whitespace permitted** around `=`
- `foo() { … }` — a function, announced by a *shape* rather than a word
- its comment marker: `#`

## Does
- matches `(?:^|[\n;])[ \t]*(?P<name>\w+)=` over top-level text
- recognizes a function by nodes: a name in text, then an empty paren group, then
  a brace

## Constraints
- **The strict form is not a stylistic choice.** Shell assignment forbids
  whitespace around `=`; `NAME = value` is a **command invocation**. A lenient
  `NAME[ \t]*=` matches it and would report a command as a declaration
- **A function is recognized structurally, not by pattern.** No string matches
  `foo() {` usefully — the parse has already turned it into a text node, an opener,
  a closer, and another opener, and that sequence is what the schema looks for
- **Word brackets mean much of a shell file is not top level.** `if`, `while`,
  `for` and `case` open groups, so assignments inside them sit at depth and are
  not sought by a top-level scan

## Collaborators
- DeclSchema: the role this instantiates
- BracketLang: `LangShell`

## Sequences
- seq-declare.md
