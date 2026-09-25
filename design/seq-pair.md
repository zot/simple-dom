# Sequence: the pairing links
**Requirements:** R81, R82, R83, R84, R85, R86, R87, R355, R357

How the bracket structure is recovered from a flat array, and how the answer is
checked rather than believed.

## 1. Recording, and answering

1. The context builds and serves the links
   1.1. During the parse, an `Opener` is appended and becomes the context currently
        open
   1.2. When its `Closer` is appended, the context records the pair **both ways** —
        opener to closer, closer to opener
   1.3. Every node appended while that opener is open records it as its **enclosing
        opener**; an opener records the one enclosing *it*
   1.4. When the parse ends, the context stamps itself with the document's
        structural generation
   1.5. A consumer asks for a node's enclosing opener
        1.5.1. The context reads the document's generation — and **inside a
               mutation window that read refuses**, which reaches this index
               without the context having written a guard
        1.5.2. Stamps equal: the links stand and are used
        1.5.3. Stamps differ: the context rebuilds from the document's nodes and
               re-stamps with the generation it just read
   1.6. `Doc` learns nothing of any of this — no registration, no callback. A
        document with no brackets carries no links rather than two dead maps

## 2. The independent answer

2. The same question, asked from outside
   2.1. A consumer walks the flat array from the start, keeping its own stack of
        open openers — the source and the table are all it needs
   2.2. At each node the innermost opener still unclosed is that node's enclosing
        opener; a closer pops instead, and records none
   2.3. Whether a closer *belongs* to the opener on top is decided from the
        **opener's own bytes** through the table. Without that a stray closer,
        which the any-close fallback emits unpaired, is paired here and the two
        answers differ on every unbalanced file
        2.3.1. A closer that does not belong, but is a longer run the opener's
               `RejectLongerCloses` group rejected — a whole match of its pattern,
               longer than the opener's text — pops the opener unpaired, because the
               parse ended the group there. Without it the ended opener stays on the
               stack and every later closer is tested against the wrong group
        2.3.2. A closer that neither belongs to the opener on top nor was rejected by
               it, when that opener's group is in code mode, pairs with the nearest
               opener further down that it closes; every opener above that one leaves
               the stack unpaired, as the parse ended them there (R356)
   2.4. No index is consulted anywhere in this, which is what makes the answer
        independent of the one it checks
   2.5. It must agree with what the context reports. **That is the point** — an
        index nothing outside can contradict is an assertion rather than a fact,
        and tools that query documents by walking links need the stronger thing
