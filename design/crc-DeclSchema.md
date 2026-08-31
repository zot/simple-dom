# DeclSchema
**Requirements:** R129, R130, R131, R132, R133, R134, R138, R141, R142, R143, R144, R145, R146, R147, R149, R150

The role, not a type. A schema knows how **one language** announces a declaration,
and drives `sdom`'s machinery with that knowledge. Bundled: Go, TypeScript,
JavaScript, Lua and Shell, with Python following the indent parser.

## Knows
- its language's bracket table, and the context a scan of it produced
- what announces a declaration in that language
- **which of its groups are comments** — the table cannot say, since a comment and
  a string are both scan-restricted, and there is deliberately no comment
  configuration

## Does
- finds what announces each declaration, over the top-level nodes
- walks **forward in document order** to the name
- splits the text at the keyword's and the name's edges, and re-types the halves
- fills in the declaration links its context holds

## Constraints
- **There is no shared recognition rule.** Each schema recognizes its own
  language's declarations from the parse it is given. What is written here is what
  the bundled schemas happen to share, not a contract
- **The announcement may be text or a node.** A keyword may be a substring of a
  text node, or a bracket marker — Lua's `function` opens a group closing with
  `end` — and a schema handles whichever its language uses
- **Skip before every decision, not once.** Whole comment groups and
  whitespace-only text are stepped over, in any number and any interleaving, both
  **backward** when testing whether a match begins a statement and **forward** when
  finding a name. A comment goes anywhere a space goes
- **Skipped whitespace may contain separators.** A declaration may span lines, so
  the forward walk does not stop at one — while the backward test uses one to
  decide a statement began. Two questions about the same byte
- **The forward walk has no terminator.** Harmless on valid input, where a keyword
  is always followed by its name; unbounded on truncated input. A schema wanting a
  bound imposes its own
- **How strict the walk is, is the schema's choice.** `sdom` neither validates nor
  requires laxity; nothing in it is a syntax checker
- **Commented-out code cannot match**, and this costs nothing — a comment is a
  bracket group, so its interior is not a top-level node
- **`import` is in no keyword set.** An import is not a declaration a tool anchors
- **Indentation is not part of a declaration.** Bracket depth already pinpoints
  them, so indent gets no node
- **Which declarations *ought* to be anchored is not here.** That is mini-spec's,
  and it starts from unanchored *requirements* rather than by filtering
  declarations
- **The duplication between schemas is deliberate.** Each carries its own pass;
  lifting them into a shared tool waits until three exist

## Collaborators
- BracketContext: the scan's links, and the declaration links this fills
- Declaration: the kinds it produces
- Doc: `Split` and `Replace`

## Sequences
- seq-declare.md
