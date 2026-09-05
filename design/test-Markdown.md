# Test Design: the markdown base
**Source:** crc-MarkdownParser.md

## Test: the fixture round-trips and its structure is recognized
**Purpose:** R226, R298, R228, R229, R230 — a committed sample shaped like the trajectory files, read by count
**Input:** `testdata/trajectory-sample.md`: headings at three levels, a status block with nested checkbox items, a struck bold head, a marker span, a fenced example containing `- [ ]`, and a code span containing `**`
**Expected:** `render == src`; the headings' levels are `1, 2, 3`; the `ListItem` and `Checkbox` counts match the hand count; `Heading` nodes appear only where `Last()` was an `Indent` or a newline-ended `Text`
**Refs:** crc-MarkdownParser.md, seq-markdown.md#1.3
**Code:** sdom/schema/markdown_test.go
**Alarm:** 1
**Fire alarm:** make `lineHead` return true for any node, so `- ` and `## ` are recognized anywhere. Red: `a - b` mid-line yields a `ListItem` and the counts overshoot.
**Inject:** sdom/schema/markdown.go:lineHead
**Pulled:** 2026-09-03 — rang: `items 6 boxes 4, want 5 and 4` — the mid-line `a - b` became an item, only that test.

## Test: a checkbox is recognized only after a list item
**Purpose:** R230
**Input:** `- [x] done`, `- [ ] open`, and `see [x] in prose`
**Expected:** two `Checkbox` nodes; the third `[x]` is text
**Refs:** crc-MarkdownParser.md, seq-markdown.md#1.3.2
**Code:** sdom/schema/markdown_test.go
**Alarm:** 2
**Fire alarm:** in `head`, try the checkbox match at any line head as well as after a `ListItem`. Red: the fixture's line beginning `[x] not an item` becomes a `Checkbox`, and this test's fourth line too.
**Inject:** sdom/schema/markdown.go:MarkdownParser.head
**Pulled:** 2026-09-03 — rang: `boxes 5` in the fixture and `boxes 3 items 2` here — the `[x] not an item` lines both became checkboxes.

## Test: code hides structure
**Purpose:** R232 — structural, not a rule
**Input:** a fence containing `- [ ] inside` and `## not a heading`; a code span containing `**`
**Expected:** no `ListItem`, `Checkbox` or `Heading` inside the fence; no bold opener inside the span; the fence and span interiors are single `Text` nodes
**Refs:** crc-MarkdownParser.md, seq-markdown.md#1.4
**Code:** sdom/schema/markdown_test.go
**Alarm:** 3
**Fire alarm:** give the fence `AllowedInner: nil` (code mode) instead of the empty slice. Red: `**` inside the fence opens a group, and the fence interior is no longer one text node. The line-head markers stay hidden even then, since the wrapper is still not offered positions inside — which is why this test asserts the interior's node count, not only the markers.
**Inject:** sdom/schema/markdown.go:LangMarkdown
**Pulled:** 2026-09-05 — re-pulled after backtick-runs Item 1 touched the site; rang: `bolds:2` inside code and `a code interior is not a single text node` twice, only `TestCodeHidesStructure`; the fence is now the one code group, and the injection is its `AllowedInner`. Previously 2026-09-03 — rang: `bolds:1` inside the fence and `a code interior is not a single text node`, and the fixture lost a heading to the fence's swallowed close — three tests.

## Test: no line-head marker shares a first byte with an opener
**Purpose:** R231 — the rule that makes delegate-then-check sound
**Input:** every `Open` string in `LangMarkdown`, and its `OpenRegex`
**Expected:** none begins with `#`, `-` or `[`, and the pattern matches at none of them
**Refs:** crc-MarkdownParser.md
**Code:** sdom/schema/markdown_test.go
**Alarm:** 4
**Fire alarm:** add a link group `{Open: ["["], Close: ["]"]}` to the table. Red: this test, and the checkbox test — `[x]` opens a group before the wrapper sees it.
**Inject:** sdom/schema/markdown.go:LangMarkdown
**Pulled:** 2026-09-05 — re-pulled after backtick-runs Item 1 touched the site; rang: four tests — this one (`opener "[" shares a first byte`), both checkbox counts at zero, and NodeType. Previously 2026-09-03 — rang in four tests: this one (`opener "[" shares a first byte`), both checkbox counts at zero, and NodeType seeing two markers where three were.

## Test: NodeType agrees with Parse
**Purpose:** R233
**Input:** the fixture, asking `NodeType` at every position a marker was emitted and at a text position
**Expected:** `heading`, `list`, `checkbox` where those were emitted; `ok == false` at plain text; bracket kinds pass through
**Refs:** crc-MarkdownParser.md
**Code:** sdom/schema/markdown_test.go
**Alarm:** 5
**Fire alarm:** make `NodeType` delegate only. Red: `ok == false` at a heading position.
**Inject:** sdom/schema/markdown.go:MarkdownParser.NodeType
**Pulled:** 2026-09-03 — rang: `NodeType at 0 = "", want "heading"` and the two other markers, only that test.

## Test: the escape hatches pair
**Purpose:** R298 — found by injecting past the alarm list, not by design
**Input:** a part line: strike around bold, then a marker span holding two code spans
**Expected:** one paired `~~`, two paired `**`, two paired `` ` ``
**Refs:** crc-MarkdownParser.md
**Code:** sdom/schema/markdown_test.go
**Alarm:** 6
**Fire alarm:** remove the code group's pattern from bold's `AllowedInner`. Red: the code spans inside the marker span are text, so `` ` `` pairs zero times. The fixture test stayed green under this injection on 2026-09-03 because it counts line-head markers, not what pairs inside a span.
**Inject:** sdom/schema/markdown.go:LangMarkdown
**Pulled:** 2026-09-05 — re-pulled after backtick-runs Item 1 touched the site; rang: `"`": 0 paired groups, want 2`, only this test. Previously 2026-09-03 — rang: `` "`": 0 paired groups, want 2 ``, only this test; pulled by hand on the uncommitted test with a targeted reverse edit.

## Test: backtick runs
**Purpose:** R298 — the one code group reads CommonMark's runs, and entry-like lines after them survive
**Input:** `testdata/backtick-runs.md`: a two-run span holding one backtick followed by a list item; a four-run fence enclosing a three-run fence followed by a list item; a three-run inside a two-run span; a span closing on the file's last byte
**Expected:** `render == src`; the `ListItem` count matches the hand count; every code opener pairs with a closer of the same length
**Refs:** crc-MarkdownParser.md
**Code:** sdom/schema/markdown_test.go
**Fire alarm:** restore the two literal groups, three backticks and one, in place of the pattern group. Red: the list-item count falls, since the two-run flips parity and the items after it land inside spans.
**Inject:** sdom/schema/markdown.go:LangMarkdown
**Pulled:** 2026-09-05 — rang: `items 3, want 4` and `code openers 10, want 4` — every backtick became its own opener — only `TestBacktickRuns`.
