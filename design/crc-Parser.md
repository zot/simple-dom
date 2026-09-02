# Parser
**Requirements:** R159, R160, R165, R166, R192

What recognizes something at the head of the input. An interface, because the walk
holds one and a language composes several.

## Knows
- its own language, and its own context — each parser owns the context it fills,
  so the entry point returns only the document

## Does
- **`Parse`** — recognizes something here and emits it, or does nothing at all
- **`NodeType`** — reports what `Parse` *would* emit here without emitting it, and
  reports separately that it would emit nothing
- **`Done`** — told once that the document now exists

## Constraints
- **Two ways to consume.** Recognize one thing and return, leaving the walk to
  offer the next position; or **take the loop**, recursing until your own
  terminator. Taking the loop is what makes suppression structural: a parser
  registered only in the outermost loop is offered no position inside a group, so
  *significant only at bracket depth 0* needs no check and a restricted group's
  exclusivity needs no flag
- **`Done` is a callback so the caller needs no knowledge.** It fires once the
  document is built, which is the only moment a derived index can bind to it — the
  nodes exist before the document does
- **A parser records nothing while it parses.** Whatever it could record is already
  implied by the array it is building, so a parse-time copy is a second statement of
  one fact, and a second statement can disagree with the first
- **`NodeType` returns two values.** One string would conflate *no node here* with
  *a node whose group carries no kind*, and those are different claims — the same
  reason a location's offset is stored biased so absence is not offset 0
- **Delegation is the parser's own business.** An indent parser holds a bracket
  parser and hands off when it does not match. Precedence lives with the language,
  never with the walk
- **This is not a node's `Parse`.** The node protocol deliberately has none: during
  a parse you construct a concrete node and the interface is not involved

## Collaborators
- ParserState: what it is handed, and all it may touch
- BracketParser, IndentParser: the implementations

## Sequences
- seq-collaborate.md
