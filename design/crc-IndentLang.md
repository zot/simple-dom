# IndentLang
**Requirements:** R175, R176

A language whose indentation carries scope. It **embeds** `BracketLang` and adds
three settings — and **the type is the flag**, so a brace language carries none of
them rather than meaningless zeroed ones.

## Knows
- `Tab` — a tab advances to the next multiple of this
- `Transparent` — groups of this `Kind` do not affect the level
- `Continuation` — a line after this marker is never indented
- everything `BracketLang` knows, by embedding

## Constraints
- **`Transparent` names a `Kind`, not a syntax.** A language marks its comment
  groups with some label and names that label here, so `sdom` compares two
  configured strings and never learns what a comment *is*
- **The distinction has to come from a label**, because no property of a group's
  shape supplies it: a comment-only line does not change the level and a
  string-only line does, and both open a parse-restricted group
- **`LangPython` is the first of these**, and writing it is what forces the type

## Collaborators
- BracketLang: embedded, unchanged
- BracketGroup: carries the `Kind` that `Transparent` names
- IndentParser: reads all three

## Sequences
- seq-indent.md
