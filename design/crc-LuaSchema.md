# LuaSchema
**Requirements:** R135, R136

The case that breaks anything written against Go. Lua announces a declaration two
ways at once, and one of them is not text at all.

## Knows
- `local` — an ordinary keyword, in a text node
- `function` — a **bracket opener**, closing with `end`, so no text pattern sees it
- a keyword-less global assignment, `NAME =`, with whitespace permitted
- its comment markers: `--` and `--[[`

## Does
- matches one alternation with a `kw` branch and a `name` branch, over top-level
  text
- also treats a `function` opener met at a statement start as an announcement
- walks forward to the name, which for `function` lies **inside the group the
  keyword opened**

## Constraints
- **One alternation, not two passes.** The branch that did not participate yields
  the **zero `Loc`**, which is how the schema tells which fired with no extra
  machinery — absence is the zero value, and this is the case `Group` was built
  for
- **Lua's `NAME =` permits whitespace and Shell's does not**, so the two schemas
  do not share a pattern however alike they look
- **Dotted and colon names count.** `M.f = 2` and `M:m` are how a Lua module
  defines its members, so the name character class is not `\w+`
- **Newline-insensitive, and this is required rather than tolerated.** The layout
  Go rejects for semicolon insertion — a keyword, blank lines, then the name — is
  legal Lua and runs. A shared rule stopping the walk at a separator would be
  wrong for both languages at once

## Collaborators
- DeclSchema: the role this instantiates
- BracketLang: `LangLua`

## Sequences
- seq-declare.md
