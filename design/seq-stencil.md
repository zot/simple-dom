# Sequence: parsing a stencil
**Requirements:** R96, R98, R99, R100, R102, R103, R104, R105, R106, R113

Two diagrams: a schema driving the builder, and what `Set` does afterwards.

## 1. A schema parses itself

1. A todo line is parsed
   1.1. The schema constructs a concrete node and sends it `Parse` — the `Node`
        interface is not involved at this moment, which is why `Parse` is not on it
   1.2. `Parse` opens a builder over its regex, the text and the location
        1.2.1. No match: the builder reports false, and what that means is the
               schema's decision
        1.2.2. A match: the builder lays out the whole span — `Text` for the head,
               for every gap between participating groups, and for the tail
        1.2.3. Each **participating** named group gets a **nil** slot; a group that
               did not participate gets no slot and owes nothing
   1.3. For each group it binds, the schema asks the builder for the text and
        provenance, builds whatever node that group should become, keeps its own
        reference, and patches it in
        1.3.1. A `Bool` is not patched in — the schema puts the `Text` in the
               children and points the `Bool` at it
        1.3.2. A group whose bytes should be plain glue after all is `Omit`ted —
               the builder fills the slot with a `Text` and marks it as glue
   1.4. `Done` merges each omitted group's text with its neighbours, so no two
        adjacent children are both plain glue, then walks what it assembled
        1.4.1. A slot still nil — **panic**: a group was named and never bound
        1.4.2. A plugged node whose span is not its group's — **panic**: the only
               way a schema can break tiling from here
        1.4.3. Otherwise it hands back the children and the unconsumed remainder
   1.5. The schema takes the children; the compound now tiles its span, renders by
        concatenation, and derives its own faithfulness from them

## 2. A tool writes into the checkbox

2. `- [    ]` becomes `- [x]`
   2.1. The tool calls `Set(true)` on the stencil's `Bool`
   2.2. The `Bool` writes through to the `Text` it views — nothing is stored twice,
        so there is no second copy to keep in step
   2.3. The `Text` keeps its offset and becomes **altered**: provenance survives an
        edit, and the span contracts from four bytes to one
   2.4. The stencil reports altered because a child is, derived on read rather than
        propagated
   2.5. The document renders the new bytes; the offset still points at where the
        old ones were, which a diagnostic uses **alone**, because the pair now
        names a span the node never owned
