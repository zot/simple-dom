# Sequence: a line start
**Requirements:** R174, R177, R178, R179, R180, R183, R184, R185, R186, R187, R188

What the indent parser does at the head of a line, and how a frame finds its
parent.

## 1. Deciding

1. The indent parser is offered a position
   1.1. **Nothing emitted yet?** Emit the zero-length root `Indent("")` and return.
        The walk sees the node count move, does not advance, and offers this same
        position again — where the level stack now holds column 0 and this branch
        does not fire twice
   1.2. Not at a line start — the previous byte is not a newline — so return
        without touching anything, and the delegate gets the position
   1.3. **The previous line ended with a `Continuation` marker at depth 0**: this
        line is not indented. Return
        1.3.1. A marker inside a comment group was at depth 1 and does not count,
               so the line after `# … \` **is** indented
        1.3.2. A marker inside a string needs no rule: that group is still open, so
               this position is never offered here at all
   1.4. Measure the leading whitespace, expanding tabs by `Tab`, and look past it
        1.4.1. A newline follows — a blank line. Return
        1.4.2. Ask the delegate for the kind here; it is `Transparent`. A
               comment-only line. Return
        1.4.3. Anything else is content, including a string, whose line **does**
               change the level
   1.5. The measured column equals the top of the stack — no change. Return, and
        the leading whitespace becomes ordinary text
   1.6. The column differs. Emit an `Indent` over exactly that whitespace, advance
        past it, and adjust the stack
        1.6.1. Greater: push. The frame's parent is the node on top before the push
        1.6.2. Smaller: pop until the top is below this column. **One node**, however
               many levels closed, because the column is the level
        1.6.3. Smaller and matching no open level: parent to the nearest smaller
               column and carry on. Not an error here
        1.6.4. Column 0: the same, and the node is **zero-length**

## 2. Checking the answer

2. The links, recorded and re-derived
   2.1. The parse records each frame's parent as it goes, from its own stack
   2.2. A consumer asks for a parent or the children, and the context compares its
        stamp against the document's structural generation
   2.3. Stale: rebuild by walking the finished array, keeping a stack of open
        columns and reading each `Indent`'s own whitespace
        2.3.1. This needs **no language knowledge** — the node carries its column,
               so the walk is over the array rather than over the source
   2.4. The two derivations must agree. Letting the parse be the only thing that
        records a frame is exactly what a representation change loses quietly
