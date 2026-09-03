# Sequence: one pass over markdown
**Requirements:** R228, R229, R230, R231, R232

## 1. A position is offered

1. The walk offers a position to `MarkdownParser`
   1.1. It delegates to its `IndentParser`, which emits an `Indent` on a level change
        or hands the position to its `BracketParser`
   1.2. Something was emitted or the position moved: return — the walk re-offers or
        continues
   1.3. Nothing happened. Read `Last()`:
        1.3.1. an `Indent`, or a `Text` ending in a newline — this is a **line head**:
               match `#{1,6} ` and emit a `Heading`, or `- ` and emit a `ListItem`
        1.3.2. a `ListItem` — match `[ ]` or `[x]` and emit a `Checkbox`
        1.3.3. anything else — return, and the walk takes the byte as text
   1.4. The check can run after delegation because `#`, `-` and `[` open no bracket
        group; inside a fence or code span this parser is never offered the position
        at all
