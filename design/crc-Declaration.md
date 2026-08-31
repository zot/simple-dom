# Declaration: DeclarationType, DeclarationName
**Requirements:** R122, R123, R124, R148

The two leaf kinds a declaration pass adds. A declaration is **not** a node: it is
a keyword node and the name nodes it introduces, all ordinary siblings, with the
structure between them left exactly where the parse put it.

## Knows
- the bytes it matched
- its location

## Does
- `Render`: returns its bytes
- `Kids`: returns none
- `Equals`: asserts its own kind, then compares bytes

## Constraints
- **No span, and no compound.** A declaration's two writable parts are separated
  by structure — a receiver group, a parameter group — so one node covering both
  would drag those groups in as children. Narrow siblings keep every byte with the
  owner the parse gave it
- **A `DeclarationType` holds no reference to its names.** No node kind holds
  state its children do not carry; the link lives on the schema's context. Same
  rule that keeps a `Marker` from holding its group
- **A pass only splits and re-types.** Nothing is consumed that the parse already
  claimed, so the flattened array is identical with and without a declaration pass
  and no marker stops being recognized. The additive property is structural here
  rather than a rule a schema must honor
- **A name is sliced out of the middle.** It arrives inside a text node carrying
  whitespace on both sides — `Text " Index "` — so both edges are split. Retyping
  the node whole would put the spaces inside the name, which renders identically
  and is wrong

## Collaborators
- Node: the protocol both satisfy
- Text: the kind they are carved from and the kind they embed
- Doc: `Split` cuts the edges, `Replace` changes the kind
- BracketContext: owns the links from a keyword to its names

## Sequences
- seq-declare.md
