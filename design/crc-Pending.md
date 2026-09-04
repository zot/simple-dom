# Pending
**Requirements:** R258, R259, R260, R261, R262, R263, R264, R265, R283, R284

The pending file schema: embeds the markdown base, owns the document, and adds the
queue entry — a view over a heading region, since nothing in an entry is a field a
tool writes into.

## Knows
- its `Doc`, the markdown parser and its contexts
- its entries, in file order, each with its run of nodes and derived values
- the level-2 headings that are not entries, each with its line
- each entry's line at parse time

## Does
- `ParsePending`: parses with the base, walks level-2 headings, and for each whose
  text opens `N.` collects the run to the region's end and derives the values
- `Entries`, `Entry(id)`, `MaxID`, `Unread`, `Render`
- `Place(e, pos)`: renders the canonical entry as one synthetic text and `Insert`s it
  before the entry at `pos`, or at the end; refuses a position out of range
- `After(id)`: the position following a live entry
- `Remove(id)`: drops the run, splitting a shared tail text at the region's end
- re-reads the document from its bytes after every write and re-derives the entries

## Constraints
- **A view, not a node.** No re-cutting, no compound: the run is the base's nodes and
  the values are read from their bytes at the format's positions
- **Placement is a node placement**, never a byte splice; refused, never clamped
- **A fence is the entry's**, by construction of the base
- **Say what could not be read**: a heading that is not an entry is listed

## Collaborators
- schema.MarkdownParser, schema.Heading: the base
- Doc: `Insert`, `Remove`, `Split`

## Sequences
- seq-pending.md
