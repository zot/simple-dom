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
**Fire alarm:** In `NewIndentParser`, initialize `levels` to `[]int{0}`, **and** remove the `NodeCount() == 0` branch from `Parse`. Both edits, because that branch does two jobs: it emits the root *and* seeds the level stack. Removing it alone panics in `change` on an empty `levels` — a nil dereference rather than the property, which is not a proof. Red: no root node exists, so the first frame is the column-2 one and reports a nil parent, and `Children(root)` has nothing to ask. The document still round-trips exactly, the root being zero-length, which is precisely why nothing else could catch it.
**Inject:** sdom/indent.go:NewIndentParser, sdom/indent.go:IndentParser.Parse
**Pulled:** 2026-09-01 — rang, and **wider than predicted**: seven test functions, because `shape()` renders the root like any other frame, so every shape assertion loses its leading `0` as well as the parenting and root tests failing on their own terms. The round trip stayed exactly green throughout, as predicted — the root contributes no bytes, which is the whole reason nothing else could catch its absence.

*First attempt did not reach the property:* removing the branch alone panicked in `change` with `index out of range [-1]`, because that branch also seeds the level stack. A nil dereference reads like a red test and proves nothing. The prescription now names both edits, and separates the root's existence from the stack's initialization.

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
**Pulled:** 2026-09-01 — rang, and the prediction held **to the character**: the comment-only case came back `shape "0 4 8 0", want "0 4 0"`. Only this test failed, in only that one of its three cases — the blank line and the docstring both stayed correct, so the injection reached the `Transparent` test and nothing adjacent to it.

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
**Pulled:** 2026-09-01 — rang exactly as predicted: `shape "0", want "0 4"` on the backslash-ending-a-comment case, and on nothing else. The real continuation and the string cases both stayed correct, so the pending-text test is doing precisely the depth-0 work claimed for it and not more.

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
**Pulled:** 2026-09-01 — rang, on all three of this test's assertions, and **the cross-derivation caught it too** — `TestTheFrameIndexAgreesWithAnIndependentWalk` failed on a corpus file. That is the widening working: only the *links* moved, so bytes, tiling and the frame shape all stayed correct exactly as predicted, and the independent walk was the only other thing in the suite that could see it.

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

## Test: IndentParser reports what it would emit
**Purpose:** R160 — the contract method nothing in the library calls
**Input:** an `IndentParser` asked for `NodeType` at the start, at a level change, at
a comment opener, and at plain text
**Expected:** a node with no kind for the first two, the delegate's `"comment"` for
the third, and no node for the fourth
**Refs:** crc-IndentParser.md, seq-collaborate.md#3
**Code:** sdom/indent_test.go
**Alarm:** 5
**Fire alarm:** Make `IndentParser.NodeType` panic on entry. **This test is the only
thing that calls it** — an `IndentParser` is always the root parser and the walk only
ever calls `Parse` on that, so before this test existed a panic there left the entire
suite green. A library prices its API by contract rather than by demand, and an
untested implementation of a contract method is a liability the moment someone nests
one parser inside another.
**Inject:** sdom/indent.go:IndentParser.NodeType
**Pulled:** 2026-09-01 — **found by injecting past the alarm list**, which is what that
step is for: the panic was silent across 124 tests. With this test it panics inside
it, and nothing else. Neither the recorded alarms nor any other test reached this
method.
