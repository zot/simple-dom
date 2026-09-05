# BracketGroup
**Requirements:** R61, R63, R64, R66, R67, R71, R168, R290, R291, R292, R293

One entry in a language's table: a set of matching markers, and the two fields
that decide what may be recognized inside it and where it may be recognized at
all. Code brackets, strings and comments are all this one type.

## Knows
- `Open`, `Separators`, `Close`: its markers — literal openers, mid-group markers, and
  the **one** closer
- `OpenRegex`: a pattern opener, anchored where the parse stands, exclusive with `Open`;
  the marker is whatever it matched
- `CloseIsOpen`: the closer is the text that opened this instance — one word for a
  symmetric group, the only way to say it for a pattern group
- `Lookahead`: an anchored pattern the bytes after an opener or closer must satisfy,
  satisfied at end of input
- `Escape`: the sequence that consumes itself and the byte after it
- `AllowedInner`: what is recognized inside — **nil is code mode**, non-nil (even
  empty) is parse-restricted
- `AllowedParent`: where it may be recognized — nil is anywhere, non-nil only
  inside the listed openers

## Does
- reports whether it is restricted, which is exactly `AllowedInner != nil`
- matches one of its markers at a position, **honouring word boundaries**: a
  marker whose first byte is a word character must be neither preceded nor
  followed by one, so `do` does not fire inside `download` — and a pattern opener
  honours the rule on the bytes it matched
- matches an opener or closer only where its `Lookahead` holds after it, so a run of
  backticks is a marker only where no further backtick follows — on either edge, in
  any table order
- closes on its literal `Close`, or with `CloseIsOpen` on the opener's own text, both
  under the lookahead

## Constraints
- **nil and an empty slice are different**, in both mode fields. nil is "no
  restriction"; empty is "a restriction with an empty list". The distinction is
  load-bearing — it separates code mode from pure raw mode
- Nesting is not a field. A block comment nests only if its own opener appears in
  its `AllowedInner`
- Carries no name and no identity a node could point at — see crc-Marker.md. It is
  *named* in `AllowedInner` and `AllowedParent` by any literal opener or by its pattern
- **`Open` and `OpenRegex` are exclusive, and `CloseIsOpen` contradicts a non-empty
  `Close`**; either is a construction error the language's table test sees, never a
  consumer

- **`Kind` is a label this layer never reads.** An uninterpreted string carried for
  a layer above — indent scope needs to know which groups are transparent to the
  level, and no property of a group's *shape* answers that, a comment and a string
  being the same shape with different markers. Storing and ignoring it is not a
  mode: the parsing rules are identical whatever it says

## Collaborators
- BracketLang: holds it, and resolves an opener back to the group that owns it
- BracketParser: asks it what matches here

## Sequences
- seq-parse.md
