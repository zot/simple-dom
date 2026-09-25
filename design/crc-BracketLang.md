# BracketLang
**Requirements:** R58, R59, R60, R62, R68, R70, R121, R169, R170, R171, R172, R173, R207, R208, R209, R295, R296, R354, R359

A language's whole bracket table, and nothing else. Supporting a new language is
adding an entry, not writing code.

## Knows
- its `BracketGroup`s
- its `Comment` style — `Prefix`, `Suffix`, `Kind` — which is how a comment is
  **written**, not how one is recognized

## Does
- resolves a name back to the group that owns it — a string one of its `Open` lists,
  or its `OpenRegex` — which is how `AllowedInner`, `AllowedParent` and `GroupFor`
  reach a group; matching then uses that group's own opener, never the name as a
  prefix
- is checked when a parser is constructed: a pattern that does not compile, an
  `OpenRegex` beside a non-empty `Open`, `CloseIsOpen` beside a `Close`, a `CloseRegex`
  beside a `Close` or `CloseIsOpen`, or a `CloseRegex` naming a different set of groups
  from the opener's panics naming the group, and a test over every shipped table keeps that panic from a
  consumer
- answers `Check`, the same construction check returned as an error rather than
  panicked, for a table that is caller input — a consumer's configuration — whose
  errors belong to the caller; an `IndentLang` answers it through the embedding
- offers its groups in order, which is the precedence the BracketParser parses by

## Constraints
- **No comment configuration exists for *recognition*.** A line comment is a group
  closing on a newline; a block comment is a group closing on its terminator. Both
  are parse-restricted, which is what makes a comment non-nesting with a literal
  interior. Strings are the same shape with different markers. *(Narrowed
  2026-09-02, Item 6.2: `CommentStyle` is configuration for **construction** — the
  `// ` a writer wants where the group's opener is `//`. Its `Kind` must equal the
  recognizing group's, guarded by a test per shipped language, never a runtime check;
  a language with several comment forms designates one.)*
- **No indent parameters and no flag.** A language needing indent scope is a type
  that **embeds** `BracketLang` and adds what it needs, and *the type is the
  flag* — so a brace language carries no indent parameters rather than
  meaningless zeroed ones
- **Tables are Go values, not a file format**, and no loader ships. The deciding
  reason is that nil versus empty is semantically distinct and is precisely what
  a document format loses, an absent key and an empty list being the same thing
  to most readers of most formats. A consumer that loads tables from its own format
  (mini-spec's TOML, whose decoder keeps nil and empty apart) owns that loader and
  validates each table with `Check`

## Ships
`LangGo`, `LangShell`, `LangPascal`, `LangJavaScript`, `LangTypeScript`,
`LangLua` — chosen for **two jobs**. They still cover the mechanism, so that no
field of `BracketGroup` is dead code and the recognition count has languages that
recognize something; and they now serve **the languages mini-spec reads**, with
Python following the indent parser. `LangPascal` earns its place under the first
job alone.

- **`LangPython` is shipped, and is the first `IndentLang`.** It is also the only
  table modelling string **prefixes**, and only the `f` forms get groups: they
  interpolate, so they are restricted with `{` as the one escape hatch. `r`, `b`
  and `u` need none — the prefix letter falls through as text and the quote that
  follows opens the ordinary group
- **Raw does not mean unescaped.** A backslash escapes the closing quote even in a
  raw string, so every Python string group carries the same `Escape` and raw needs
  no group of its own
- **The `f` groups come first, longest quote form first**, so `f"""` is matched
  before `f"`. A plain `"""` never competes: at the `f` it cannot match at all
- **Interpolation needs no group.** `AllowedInner` names Python's own code brace,
  so the inside of `{…}` is full code mode and a dict display parses like any other

## Collaborators
- BracketGroup: its entries
- BracketContext: carries the language through the parse

## Sequences
- seq-parse.md
