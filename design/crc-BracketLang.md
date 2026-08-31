# BracketLang
**Requirements:** R58, R59, R60, R62, R68, R70, R121

A language's whole lexical table, and nothing else. Supporting a new language is
adding an entry, not writing code.

## Knows
- its `BracketGroup`s, and nothing besides

## Does
- resolves an opener string back to the group that owns it, which is how
  `AllowedInner` reaches a nested group
- offers its groups in order, which is the precedence the BracketParser scans by

## Constraints
- **No comment configuration exists.** A line comment is a group closing on a
  newline; a block comment is a group closing on its terminator. Both are
  scan-restricted, which is what makes a comment non-nesting with a literal
  interior. Strings are the same shape with different markers
- **No indent parameters and no flag.** A language needing indent scope is a type
  that **embeds** `BracketLang` and adds what it needs, and *the type is the
  flag* — so a brace language carries no indent parameters rather than
  meaningless zeroed ones
- **Tables are Go values, not a file format**, and no loader ships. The deciding
  reason is that nil versus empty is semantically distinct and is precisely what
  a document format loses, an absent key and an empty list being the same thing
  to most readers of most formats

## Ships
`LangGo`, `LangShell`, `LangPascal`, `LangJavaScript`, `LangTypeScript`,
`LangLua` — chosen for **two jobs**. They still cover the mechanism, so that no
field of `BracketGroup` is dead code and the recognition count has languages that
recognize something; and they now serve **the languages mini-spec reads**, with
Python following the indent parser. `LangPascal` earns its place under the first
job alone.

## Collaborators
- BracketGroup: its entries
- BracketContext: carries the language through the parse

## Sequences
- seq-scan.md
