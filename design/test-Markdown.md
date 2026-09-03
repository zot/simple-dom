# Test Design: the markdown base
**Source:** crc-MarkdownParser.md

## Test: the fixture round-trips and its structure is recognized
**Purpose:** R226, R227, R228, R229, R230 — a committed sample shaped like the trajectory files, read by count
**Input:** `testdata/trajectory-sample.md`: headings at three levels, a status block with nested checkbox items, a struck bold head, a marker span, a fenced example containing `- [ ]`, and a code span containing `**`
**Expected:** `render == src`; the headings' levels are `1, 2, 3`; the `ListItem` and `Checkbox` counts match the hand count; `Heading` nodes appear only where `Last()` was an `Indent` or a newline-ended `Text`
**Refs:** crc-MarkdownParser.md, seq-markdown.md#1.3
**Code:** sdom/schema/markdown_test.go
**Alarm:** 1
**Fire alarm:** make `lineHead` return true for any node, so `- ` and `## ` are recognized anywhere. Red: `a - b` mid-line yields a `ListItem` and the counts overshoot.
**Inject:** sdom/schema/markdown.go:lineHead

## Test: a checkbox is recognized only after a list item
**Purpose:** R230
**Input:** `- [x] done`, `- [ ] open`, and `see [x] in prose`
**Expected:** two `Checkbox` nodes; the third `[x]` is text
**Refs:** crc-MarkdownParser.md, seq-markdown.md#1.3.2
**Code:** sdom/schema/markdown_test.go
**Alarm:** 2
**Fire alarm:** in `head`, try the checkbox match at any line head as well as after a `ListItem`. Red: the fixture's line beginning `[x] not an item` becomes a `Checkbox`, and this test's fourth line too.
**Inject:** sdom/schema/markdown.go:MarkdownParser.head

## Test: code hides structure
**Purpose:** R232 — structural, not a rule
**Input:** a fence containing `- [ ] inside` and `## not a heading`; a code span containing `**`
**Expected:** no `ListItem`, `Checkbox` or `Heading` inside the fence; no bold opener inside the span; the fence and span interiors are single `Text` nodes
**Refs:** crc-MarkdownParser.md, seq-markdown.md#1.4
**Code:** sdom/schema/markdown_test.go
**Alarm:** 3
**Fire alarm:** give the fence `AllowedInner: nil` (code mode) instead of the empty slice. Red: `**` inside the fence opens a group, and the fence interior is no longer one text node. The line-head markers stay hidden even then, since the wrapper is still not offered positions inside — which is why this test asserts the interior's node count, not only the markers.
**Inject:** sdom/schema/markdown.go:LangMarkdown

## Test: no line-head marker shares a first byte with an opener
**Purpose:** R231 — the rule that makes delegate-then-check sound
**Input:** every `Open` string in `LangMarkdown`
**Expected:** none begins with `#`, `-` or `[`
**Refs:** crc-MarkdownParser.md
**Code:** sdom/schema/markdown_test.go
**Alarm:** 4
**Fire alarm:** add a link group `{Open: ["["], Close: ["]"]}` to the table. Red: this test, and the checkbox test — `[x]` opens a group before the wrapper sees it.
**Inject:** sdom/schema/markdown.go:LangMarkdown

## Test: NodeType agrees with Parse
**Purpose:** R233
**Input:** the fixture, asking `NodeType` at every position a marker was emitted and at a text position
**Expected:** `heading`, `list`, `checkbox` where those were emitted; `ok == false` at plain text; bracket kinds pass through
**Refs:** crc-MarkdownParser.md
**Code:** sdom/schema/markdown_test.go
**Alarm:** 5
**Fire alarm:** make `NodeType` delegate only. Red: `ok == false` at a heading position.
**Inject:** sdom/schema/markdown.go:MarkdownParser.NodeType
