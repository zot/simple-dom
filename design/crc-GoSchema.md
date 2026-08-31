# GoSchema
**Requirements:** R137, R139, R140, R151

The worked example, and the shape TypeScript and JavaScript follow. Every Go
declaration keyword is ordinary text, which is what makes it the easy case — and
what makes a schema written only against it fail to generalize.

## Knows
- its keyword set: `func`, `var`, `type`, `const`
- its statement separators: `\n` and `;`
- its comment markers: `//` and `/*`

## Does
- matches `(?:^|[\n;])[ \t]*(?P<kw>func|var|type|const)\b` over top-level text
- treats an opener met after the keyword as a **receiver group**, jumping it
- for a grouped declaration, runs a second pass over the single text node inside
  the parens

## Constraints
- **The pattern consumes its separator and ends on `\b`.** Go's regexp supports
  neither lookbehind nor lookahead, so `(?<=…)` and `(?=\s)` do not compile — and
  `\b` is the better rule anyway, rejecting `constant` while admitting a keyword
  followed by punctuation
- **A `func`'s name reaches its opening parenthesis without crossing a newline.**
  That is what semicolon insertion makes true, and it lets this schema reject what
  Go rejects. **Not abutment** — `func foo (x int) {` and `func bar /* c */ (x int)
  {` are both legal, while `func baz` ⏎ `(x int)` is not, and neither is a comment
  carrying a newline between them, since Go treats one as a newline. The test is
  therefore over the **source span**, where comment bytes are included by
  construction. It binds `func` alone
- **A name list is captured whole and split.** A repeated capture reports only its
  last iteration, so `B, C, D` would yield `B` and `D` and lose `C`. One group for
  the list, identifiers out of it — which covers `A = 1`, `B, C = 2, 3` and
  `D, E int` alike

## Collaborators
- DeclSchema: the role this instantiates
- BracketLang: `LangGo`, and `LangTypeScript` for the same shape

## Sequences
- seq-declare.md
