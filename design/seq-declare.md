# Sequence: the declaration pass
**Requirements:** R124, R131, R132, R134, R139, R145, R147, R148, R149

Two diagrams: a schema walking a document, and what it does to the nodes it found.
The pass runs **after** a parse, over the nodes that parse produced.

## 1. A schema finds a declaration

1. A schema walks a parsed document
   1.1. It takes the **top-level** nodes — a comment's interior is inside its
        group, so commented-out code is excluded without a check
   1.2. For each top-level text node it matches its own pattern, and for each hit
        asks whether that hit **begins a statement**
        1.2.1. Hit is inside the text: the pattern's own separator branch decided
        1.2.2. Hit is at the node's start: walk **backward**, stepping over whole
               comment groups and whitespace-only nodes, and ask whether what
               remains ends with a separator. Nothing before it means the document
               began, which counts
        1.2.3. A comment group is skipped backward in **one hop**, because a
               closer names its opener
   1.3. A language may announce a declaration with a node rather than a substring —
        Lua's `function` is an opener — so a schema may also test the node kind at
        a statement start
   1.4. From the announcement it walks **forward in document order**, and the skip
        of 1.2.2 runs **before every decision**, not once
        1.4.1. An opener met after skipping is a **receiver group**: jump to its
               closer and skip again
        1.4.2. The first text with content holds the name
        1.4.3. Skipped whitespace may contain a separator, and the walk crosses it:
               a declaration may span lines
        1.4.4. A schema may be stricter than this — Go requires a `func`'s name to
               reach its opening paren without crossing a newline — and `sdom`
               neither requires nor prevents it

## 2. What it does to the nodes

2. The pass re-granulates what it found
   2.1. The keyword is **split out** of its text node at both edges, unless it was
        already a node of its own
   2.2. `Replace` swaps that half for a `DeclarationType`, keeping its position;
        membership changed, so the generation bumps
   2.3. The name is split out of the **middle** of its node — it carries whitespace
        on both sides — and replaced by a `DeclarationName`
   2.4. A grouped declaration repeats 2.3 for each name, over the single text node
        inside the parens, its name list captured whole and split into identifiers
   2.5. The schema records the links: the keyword to every name it declared
   2.6. Nothing was consumed that the parse had claimed. Only splits and kind
        changes happened, so the flattened array is unchanged and no marker stopped
        being recognized
