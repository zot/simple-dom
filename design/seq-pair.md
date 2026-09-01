# Sequence: the pairing links
**Requirements:** R81, R82, R83, R84, R85, R86, R87

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

2. The same question, asked a second way
   2.1. Scan forward from the node, skipping **whole bracket pairs** rather than
        descending into them
   2.2. The first unmatched opener encountered is the enclosing opener
   2.3. That answer is reached without consulting the index at all
   2.4. It must agree with the index. **This second path is the point** — an index
        that nothing can contradict is an assertion rather than a fact
