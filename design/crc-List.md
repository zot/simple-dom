# List
**Requirements:** R199, R200, R201, R202, R203, R206

A comma-separated field as a stencil: a compound with **one child**, the `Text` of
the whole field. The `CRC:`, `Seq:` and `Test:` lists of a traceability comment, and
reusable by any schema that has a comma list.

## Knows
- its one `Text` child, through the embedded `Compound`

## Does
- `ParseList`: consumes `WS? item (WS? , WS? item)* WS?` from the head of a text and
  returns the node, the remainder, and whether anything matched
- `Items`: derives the values from the literal on every call — split on commas,
  trimmed — storing nothing
- `SetItems`: renders the canonical literal (`", "`-joined), **re-parses it**, and
  writes it through only if the re-parse consumed all of it and yielded the same
  items; otherwise refuses with an error and leaves the literal alone

## Constraints
- **One child, not one per item.** Its only write is whole-field replace, so per-item
  boundaries would imply an edit nobody makes. A tool that adds an item reads,
  appends, and sets
- **The guard is here and only here.** A node-level write changes one literal, so
  rollback is free; the document-level mutation has no guard for the same reason it
  would not be free there
- An item containing the *enclosing* stencil's separator is legal to the list and is
  that stencil's corner. This kind knows commas and whitespace, nothing else
- `Items` is not lazy — small files, derive on every call

## Collaborators
- Text: the literal it derives from and writes through
- Compound: tiling, render and extent
- RequirementList: embeds it and re-types the items

## Sequences
- seq-anchor.md
