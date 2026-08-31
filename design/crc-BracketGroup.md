# BracketGroup
**Requirements:** R61, R63, R64, R65, R66, R67, R71

One entry in a language's table: a set of matching markers, and the two fields
that decide what may be recognized inside it and where it may be recognized at
all. Code brackets, strings and comments are all this one type.

## Knows
- `Open`, `Separators`, `Close`: its markers
- `Escape`: the sequence that consumes itself and the byte after it
- `AllowedInner`: what is recognized inside — **nil is code mode**, non-nil (even
  empty) is scan-restricted
- `AllowedParent`: where it may be recognized — nil is anywhere, non-nil only
  inside the listed openers

## Does
- reports whether it is restricted, which is exactly `AllowedInner != nil`
- matches one of its markers at a position, **honouring word boundaries**: a
  marker whose first byte is a word character must be neither preceded nor
  followed by one, so `do` does not fire inside `download`

## Constraints
- **nil and an empty slice are different**, in both mode fields. nil is "no
  restriction"; empty is "a restriction with an empty list". The distinction is
  load-bearing — it separates code mode from pure raw mode
- Nesting is not a field. A block comment nests only if its own opener appears in
  its `AllowedInner`
- Carries no name and no identity a node could point at — see crc-Marker.md

## Collaborators
- BracketLang: holds it, and resolves an opener back to the group that owns it
- BracketParser: asks it what matches here

## Sequences
- seq-scan.md
