# Test Design: indent scope
**Source:** crc-IndentParser.md, crc-IndentContext.md

Fire alarms are added in the Implementation phase.

## Test: a node at every change and none where the level is unchanged
**Purpose:** R178 — the rule that makes a node mean *something happened here*
**Input:** the worked example — `a` / `  b` / `    c` / `  d` / `    e`
**Expected:** five `Indent` nodes: the root, then columns 2, 4, 2, 4. The second
line at column 2 gets one because the level *changed*; a line repeating a column
with no change in between gets none, and its whitespace stays text
**Refs:** crc-IndentParser.md, seq-indent.md#1.5
**Code:** sdom/indent_test.go

## Test: the root is emitted once, and never for an empty source
**Purpose:** R179 — the guard that keeps the walk ignorant of indent
**Input:** a one-line source, and an empty one
**Expected:** the first opens with a zero-length `Indent` at offset 0 and has
exactly one root; the second has **no nodes at all**
**Refs:** crc-IndentParser.md, seq-indent.md#1.1
**Code:** sdom/indent_test.go
**Alarm:** 4
**Fire alarm:** In `IndentParser.Parse`, remove the `NodeCount() == 0` branch so no root is emitted. Red: top-level frames have no parent to link to, so `Children(root)` has nowhere to live and the first column-2 frame reports a nil parent. The document still round-trips exactly, because the root is zero-length and contributes no bytes at all — which is precisely why nothing else could catch it.
**Inject:** sdom/indent.go:IndentParser.Parse

## Test: a return to column 0 is zero-length, and closes several levels at once
**Purpose:** R180, R181 — the column is the level
**Input:** `def a():` / `  if x:` / `    p` / `b = 1`
**Expected:** one `Indent` at the last line, rendering `""`, parented to the root
— not two dedent markers, and not a node carrying whitespace it does not have
**Refs:** crc-IndentParser.md, seq-indent.md#1.6.2
**Code:** sdom/indent_test.go

## Test: a dedent to column 0 before a bracket opener keeps the opener
**Purpose:** R164, R180 — the case the two-signal rule exists for
**Input:** an indented block followed at column 0 by `"a string statement"`
**Expected:** the zero-length `Indent`, then an `Opener` for the quote — the quote
is **not** text
**Refs:** crc-IndentParser.md, seq-collaborate.md#1.4.1
**Code:** sdom/indent_test.go

## Test: indentation is inert inside brackets
**Purpose:** R174 — measured against CPython, which emits no indent tokens there
**Input:** `foo(1,` / wildly indented continuation lines / `)` then a real block
**Expected:** no `Indent` node anywhere between the parentheses, and the block
after them indents normally
**Refs:** crc-IndentParser.md, seq-collaborate.md#2.4.2
**Code:** sdom/indent_test.go

## Test: blank and comment-only lines do not change the level
**Purpose:** R183 — and that a *string*-only line does
**Input:** a suite containing a blank line, a comment-only line at a deeper column,
and a docstring line
**Expected:** no `Indent` for the blank or the comment; one for the docstring —
the distinction coming from the group's `Kind` and nothing else
**Refs:** crc-IndentParser.md, seq-indent.md#1.4
**Code:** sdom/indent_test.go
**Alarm:** 1
**Fire alarm:** In `IndentParser.change`, drop the `transparentAt` test. Red: a comment-only line at a deeper column opens a frame, so every commented block nests one level further and the shape of that case becomes `0 4 8 0` instead of `0 4 0`. No byte moves and the array still tiles — the whitespace is a node instead of text, which only a test counting frames can see.
**Inject:** sdom/indent.go:IndentParser.change

## Test: a continuation marker counts only at bracket depth 0
**Purpose:** R184 — three cases, one predicate
**Input:** `a = 1 + \` then an indented line; `# c \` then an indented line; and a
string spanning a line break
**Expected:** no `Indent` for the first; an `Indent` for the second, because that
backslash is inside the comment group; and for the third the position is never
offered, the group still being open
**Refs:** crc-IndentParser.md, seq-indent.md#1.3
**Code:** sdom/indent_test.go
**Alarm:** 2
**Fire alarm:** In `IndentParser.continued`, test the whole source behind the position — `st.Src()[:st.Pos()]` — rather than the pending text. Red: a backslash that a comment group consumed now suppresses the following line's indent, so `# c \` before an indented line yields shape `0` instead of `0 4`. CPython treats that indent as significant and rejects it, which is only reachable because it IS significant.
**Inject:** sdom/indent.go:IndentParser.continued

## Test: a frame parents to the nearest smaller column
**Purpose:** R185 — and that two frames at one column are siblings
**Input:** the worked example
**Expected:** the column-4 frame under the first column-2 frame; the second
column-2 frame under the **root**, not under the first; and the root's children
being exactly the two column-2 frames
**Refs:** crc-IndentContext.md, seq-indent.md#1.6
**Code:** sdom/indent_test.go
**Alarm:** 3
**Fire alarm:** In `IndentContext.rebuild`, pop while the top column is strictly greater than this one rather than greater-or-equal. Red: two frames at one column stop being siblings and nest instead, so the root's children shrink to one and the second column-2 frame parents to the first. Nothing about the nodes changes — only the links — so bytes, tiling and the frame *shape* all stay correct.
**Inject:** sdom/indent.go:IndentContext.rebuild

## Test: an unmatched dedent parents rather than refusing
**Purpose:** R188 — `sdom` is no syntax checker
**Input:** `a` / `    b` / `  c` — a dedent to a column never opened
**Expected:** the column-2 frame parents to the root, and nothing errors
**Refs:** crc-IndentContext.md, seq-indent.md#1.6.3
**Code:** sdom/indent_test.go

## Test: the independent walk agrees, over the corpus
**Purpose:** R187 — the links are checkable rather than believed
**Input:** every project file, parsed with `LangPython`
**Expected:** the links the pass recorded and the links a stack walk re-derives
from the columns alone are equal, entry for entry
**Refs:** crc-IndentContext.md, seq-indent.md#2
**Code:** sdom/indent_test.go

## Test: the column is derived, with tabs expanded
**Purpose:** R182 — the text is the storage
**Input:** lines indented with tabs, spaces, and both, under a known `Tab`
**Expected:** frames nest by expanded column rather than by byte length, and each
node still renders exactly its own whitespace
**Refs:** crc-IndentParser.md
**Code:** sdom/indent_test.go
