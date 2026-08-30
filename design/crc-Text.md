# Text
**Requirements:** R15

A leaf. The kind every unmodelled byte ends up in, and the reason a parse that
models very little still round-trips.

## Knows
- its bytes
- its location

## Does
- `Kids`: returns none
- `Render`: returns its bytes
- `Equals`: asserts `*Text`, then compares bytes
- carries `Altered` as **stored** state, since it has no children to derive it from

## Collaborators
- Loc: reports provenance and faithfulness
- Doc: splits and merges runs of it

## Sequences
- seq-mutate.md
