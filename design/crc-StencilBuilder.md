# StencilBuilder
**Requirements:** R96, R97, R98, R99, R100, R101, R102, R103, R104, R105, R106, R107, R108, R109, R110, R117

The machinery a schema **drives** to turn a regex match into a tiled child list.
It is a builder, never a `Node` — "stencil" alone names the *region* a tool writes
into, and this is the tool that cuts it.

## Knows
- the match, and the text and provenance of each named group
- the children under construction: `Text` for the glue, a **nil** in each
  participating group's slot
- where the match ended, so the caller can be told what is left

## Does
- `Group(name)`: the matched text and its `Loc` — the **zero `Loc`** for a group
  that did not participate
- `Put(name, node)`: patches a node the schema built into that group's slot
- `Omit(name)`: fills the slot with a `Text` and marks the group as glue
- `Done()`: **merges each omitted group's text with its neighbours**, then hands
  back the children and the unconsumed remainder

## Constraints
- **The glue is computed, never required.** A regex names only the groups its
  schema binds; the head, the tail and every gap between groups become `Text`. A
  pattern therefore cannot silently eat bytes, which is why no parse-time tiling
  check exists — that failure is **unrepresentable**, not detected
- **A group starts nil rather than defaulting to `Text`.** A group exists because a
  schema binds it, so one named and never filled should not have been named; a nil
  makes the omission loud where a default would swallow it
- **`Done` panics** on a nil slot, and on a plugged node whose span does not match
  its group's. Both are programming errors, and the second is the only way a schema
  can break tiling from here — it would otherwise surface as the compound reporting
  itself *altered*, which is a plausible wrong answer rather than a refusal
- **Two ways a group goes unfilled, and only one is the schema's.** A
  non-participating group has no bytes, so the builder gives it no slot and asks
  nothing. A participating group the schema does not want as a field is what `Omit`
  is for
- **`Done` leaves no two adjacent plain-glue children.** An omitted group would
  otherwise be three `Text` nodes where one belongs, implying a boundary nothing
  writes into. Which gives `Omit` a falsifiable meaning: omitting a group must
  produce the **identical** child list to a regex that never named it
- **Binding is by name, never by position** — alternation gives branches different
  group counts, so a fixed index is right for one input and out of range for
  another. Nested groups are skipped
- **No name-to-node map comes back.** The schema assigned every node to its own
  field as it built it; returning the same references would be a second copy of
  something that only flows one way

## Collaborators
- Compound: receives the children this assembles
- Loc: the zero value is what distinguishes a non-participating group
- Text: what all the glue is

## Sequences
- seq-stencil.md
