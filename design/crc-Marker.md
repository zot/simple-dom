# Marker: Opener, Closer, Separator
**Requirements:** R78, R79

The three leaf kinds the parser adds. Each holds the bytes it matched and nothing
more.

## Knows
- the bytes it matched
- its location

## Does
- `Render`: returns its bytes
- `Kids`: returns none
- `Equals`: asserts its own kind, then compares bytes

## Constraints
- **A marker holds no pointer to its group.** Holding one would make `Equals`
  compare pointers, so two documents parsed with independently constructed
  languages would never be equal and the structural round-trip would fail on
  every bracketed document. The active group comes from the parse context
- **Three kinds, not one with a role field.** A symmetric group's `"` is
  byte-identical opening and closing, so the role cannot be derived from the
  bytes — and a stored role would be state the children do not carry, which
  `Equals` would then have to compare. As separate types the discrimination is
  the type assertion every `Equals` already performs, and nothing extra is stored
- Everything that is not a marker is `Text`, whitespace included

## Collaborators
- Node: the protocol all three satisfy
- BracketContext: owns the links that pair them

## Sequences
- seq-parse.md
- seq-pair.md
