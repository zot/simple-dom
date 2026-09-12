# Carve: an opener never closed is text

> **DRAFT (2026-09-07).** Opened the evening mini-spec's second opinion — an independent line
> scan its `validate` now runs over every document our readers own — reported on three
> repositories that an inline marker with no closer takes the rest of the file with it, and
> that a `## Test:` heading's title stops at its first code span. Every instance was
> reproduced here against `2a9921c` before the scope below was written.

`carves/done/backtick-runs.md` gave the markdown base CommonMark's run rule and made the
parser *say* when a group is still open at end of input. Saying so was the right first half:
the reader lists the opener and its line, and the second opinion is what found the rest.
What it found is that the report is the whole remedy today — a run that never closes still
swallows every heading and list item after it, so the base reads ark's requirements as 1419
of about 2800 and five of ark's test designs short by up to thirteen entries each. The
reader is honest and the document is still unread.

CommonMark resolves this by construction: a backtick string with no matching string is not a
code span, and an emphasis delimiter run with no closer is literal text, because inline
delimiters are resolved after the paragraph is read. This carve is that rule for the base,
and one reader defect of the same shape a level up.

**Provenance.** Both parts are mini-spec's, 2026-09-07:
`~/work/mini-spec/requests/unmatched-run-swallows.md` and
`~/work/mini-spec/requests/testdoc-title-stops-at-code-span.md`. Item 3 is mini-spec's too,
2026-09-12: `~/work/mini-spec/requests/pending-rule-inside-code-span.md`, folded in the same
day (Bill). Our acknowledgements are in `requests/`.

**What the report got wrong, measured 2026-09-07.** The request names two base defects. The
first — *an emphasis marker inside a code span opens a group* — does not exist: an asterisk
inside a code span is literal here, probed on `` a `*Read` b `` and on line 110 of
`design/test-Pending.md` in isolation, both clean. The two lines it cites are the *second*
defect wearing a different face. mini-spec's requirements line 157 quotes `specs/*.md` bare,
no code span around it, so the asterisk opens emphasis before `.md` and nothing closes it.
Our test-Pending line 110 is a symptom three paragraphs downstream: line 95 carries an odd
count of lone backticks, the last one pairs with the first backtick of line 99, and from
there every backtick in the file is flipped from open to close until the asterisk on line
110 stands outside a span. One defect, four instances, and the same repair for all of them.

## Status

- [x] ~~**Item 1 — an opener with no closer is demoted to text.**~~ **LANDED (`a261869`, 2026-09-12 — `#37`.)**
- [ ] **Item 2 — the test-entry title reads to the end of its line.** **OPEN (#38.)**
- [ ] **Item 3 — a rule is a line of its own, never a code span's interior.** **OPEN (#39.)**

## Decisions

**DECIDED (inherited from `carves/done/sdomification.md`, Bill, 2026-09-03): committed tests
rely only on code in this repository.** The three repositories' documents are the throwaway
comparison that says whether Item 1 is done; the committed fixtures are synthetic and live in
`sdom/schema/testdata`, with expectations written by hand from CommonMark's rule and the byte
round-trip over every one of them.

**DECIDED (inherited from `carves/done/backtick-runs.md`, Bill, 2026-09-05): a longer run
inside a code group is a rejected closer, against CommonMark.** Unchanged by this carve. A
rejected closer ends its group and is listed by `Unpaired()`; demotion is about an opener
whose group *never* ends, and the two do not meet.

**DECIDED (Bill, 2026-09-07): demotion is a per-group flag, not a parser rule.** An unclosed `{` at the end of a Go file is a real error and a reader should keep
seeing it in `Unclosed()`; an unclosed asterisk in a markdown paragraph is a typographer's
asterisk. The table says which, one field on `BracketGroup`, set by markdown's three groups
and by nothing in Go, Lua, Shell or Python. `Unclosed()` keeps its meaning for the groups
that do not set it, which is why R299–R303 stay as they are.

**DECIDED (Bill, 2026-09-07): an inline group does not cross a blank line.** CommonMark's inline delimiters live inside one paragraph, and that bound is what
makes its resolution cheap: an unclosed opener is settled at the paragraph's end, not the
file's. Without it, demotion is correct but its cost is a re-parse from the opener to end of
input, nested once per unclosed opener inside — ark's test-Matcher line 29 opens two. With
it, the re-parse is bounded by the paragraph and the flag can double as the bound: a group
that demotes at end of input demotes at a blank line too. The fork this leaves is the fence.
CommonMark says a fence at a line head that never closes runs to the end of the document,
and the base's one code group cannot tell a fence from a span without a line-head test it
does not have. Either the code group takes the bound like the others — a fence must be
closed, which every fence in our three repositories is — or the flag learns a line-head
exemption. The first is one field; the second is the seam for it if a document ever needs
it. **The first, decided with the bound (Bill, 2026-09-07, on Daneel's recommendation):**
the code group takes the bound, and a fence is closed or it is text. **SUPERSEDED the same
evening (Bill, 2026-09-07), before any code was written.** Measured across the three
repositories' design, spec and carve files: 739 fences, 133 with a blank line inside. A code
group bounded by a blank line would demote every one of them and read the list items and
headings inside as structure — the failure *code hides structure* exists to prevent. So a
fenced block must be allowed to contain blank lines, and **the code group demotes at end of
input only**; emphasis and strike take the blank-line bound. A backtick run then fails to
terminate in exactly two ways: end of input, which demotes it; and bad nesting — a longer
run inside it — which the rejected-closer rule already ends it on, the trailing run opening
afresh and demoting at end of input in turn. An unclosed inline span costs one re-parse of
the file's tail, once per such defect; one measured in ark, one in each of our two test
designs. No line-head test is needed. **AMENDED again the same evening (Bill, 2026-09-07),
after the build measured it:** with no bound on the code group, one odd backtick makes every
later lone backtick pair with the next one across blank lines and across the headings between
them — nothing is unclosed, nothing demotes, and the span is simply wrong. Our
test-BracketParser read 17 of 26 and test-Pending 9 of 11 with demotion in place; ark's
test-Secretary 2 of 6. Measured over the three repositories: 33,195 inline spans, 264 wrapped
across one line break, 17 across a blank line — the 17 are exactly the wrong pairings. So the
code group takes the blank-line bound **except when its opener stands at a line head**, with
only whitespace before it on its line: that is a fence, bounded by end of input alone, and
CommonMark's own distinction. Blank lines, not newlines — a span wrapped at the column limit
stays a span, and the 264 stay read. One more field on the group says it. **And every demoted opener is reported (Bill,
2026-09-07):** the opener becomes text and the document also says so — first said for the
fence, then widened to anything unclosed, since the reader cannot tell a typographer's
asterisk from a forgotten marker and should not guess. The base has no log; the report is the
readers' `Unread` line, the surface R300–R303 already gave the unclosed case, and the context
keeps the record because the rewind has dropped the node. An `Unread` line is document-level
and gates no write (measured 2026-09-07: only an entry's own deviations refuse), so mini-spec's
documents with a glob in prose read whole and stay writable, and are listed.

## Item 1

**The rule.** When a group whose flag is set reaches end of input — or, for emphasis and
strike, a blank line — without its closer, its opener was never an opener. The parse
rewinds to the byte after the opener's text, drops every node emitted since the opener
including the opener itself, folds the opener's bytes into the live text run that preceded
it, and continues in the *enclosing* group's mode from there. Every marker the failed group
had swallowed is offered to the parser again, so a heading or list item inside it is read
as one, and a later opener of the same group opens afresh.

**Why a rewind and not a post-pass.** The base is one pass by decision (`specs/markdown.md`,
*One pass, no post-pass*), and the nesting lives on the call stack: `parseBody` recurses and
the emitted array is flat because nothing flattens it. A demoted group is one frame
returning *unclosed* to `open`, which then rewinds and returns to its caller's loop; the
caller was already standing in the right mode. Nothing above the bracket parser needs to
know it happened — the indent parser and the markdown wrapper are never offered a position
inside a group (R174, R232), so their state at the opener is their state after the rewind,
with no bookkeeping. **This is the structural fact that makes the part small**, and the
build should state it in a test rather than trust it: a heading inside a demoted span at a
changed indentation must read as a heading at the right depth.

**What the parser state needs.** `ParserState` can move its position (`SetPos`) but cannot
shorten its array or restore its live text run, and R225 is stated over the promise that no
parser backs up. A rewind is a new verb on the state — to a node count and a position — that
truncates the array and points the live run at the last node again when that node is a
`Text`, so the opener's bytes and every byte after them extend it as declined bytes do. R225
is restated rather than retired: no parser emits over bytes in the live run, and the one
parser that backs up does so through the verb that keeps the run true. Whether a dropped
node leaves anything behind in the origin's location minting is a question for the build; if
it does, the rewind is where it is returned.

**What it costs, measured 2026-09-07 before landing.** Over 20 parses: mini-spec's done file
789 ms at `ceafd0c`, 413 ms with demotion; ark's requirements 1.24 s against 2.07 s — and the
whole of that increase is the 630 KB of ark's file, 77 percent of its bytes, that the old
parse never read because it sat inside one code span opened at line 2157. On the 190 KB
before that line, which both parses read in full, the new build takes 370–400 ms against
410–420 ms. Reading the tail costs what reading it costs; the demotions themselves are
bounded by a paragraph and cost nothing visible.

**What it reads, measured the same day.** ark's requirements 3402 of 3402 by grep (was 1419);
ark's 69 test designs all at their grep count (six were short); mini-spec's requirements and
16 test designs whole; ours, 353 requirements and 24 test designs whole (test-BracketParser
and test-Pending were short). Every remaining `Unread` line in those files names a real
defect in the document — an odd backtick, a bare glob — read as text and listed.

**What changes in the reports.** Nothing is *unclosed* in a markdown document any more,
since every group that could be is demoted. `Unclosed()` still answers for languages whose
groups do not set the flag, and `Unpaired()` still lists a rejected longer run. The context
gains a third list beside them, the **demoted** openers — their text and location, recorded
at the rewind because the opener node no longer exists — and the readers' `unbalanced` lines
(R300–R303) read it as they read the other two: every demoted opener is listed as never
closed and read as text. The tests that pin the unclosed lines move to a demoted fixture and
gain the assertion that the entries after the opener are read.

**Acceptance.** Committed fixtures, each with its expected node stream written by hand and
the byte round-trip: a lone backtick before a heading, the heading read; a three-run in
prose before a list item, the item read; a bare asterisk in a glob before a `- ` line; two
nested emphasis openers that neither close, both demoted innermost first; a span that *does*
close after a demoted asterisk inside it, so a demotion does not cost the enclosing group
its closer; a span open across a blank line before a heading, the heading read; and a
three-run at a line head with no closer, the lines after it read and the run listed.
Then the throwaway: ark's requirements read whole by `grep -c` against `Entries()`, and
every test design in the three repositories reads the count its second opinion counts. Fire
alarms: the rewind made to keep the opener node, which must lose the heading after it; and
the rewind made to skip the enclosing mode, so a closer of the enclosing group inside the
demoted one is missed; and the demoted list left unfilled at the rewind, which must lose
every never-closed line.

## Item 2

`TestDoc.scan` reads an entry's title from the single text node after its heading marker
(`headingText`, shared with the carve reader's head lines), so a code span in the heading
ends the title at the span's opener: *the pre-`track` refusal asks the intent and stops*
reads as *the pre-*. Twenty-five of mini-spec's test headings carry a code span. The part
line's title already follows emphasis to its own close (`carves/done/sdomification.md`,
Item 8); this is the same read one level down — the title is every byte from after `Test:`
to the end of the heading's line, across whatever markers the line carries, rendered as
source. The entry's `Title` field keeps its type; only its extent changes.

**Acceptance.** A fixture entry whose heading holds a code span and one whose heading holds
bold; both titles read whole; the `Test:` prefix and the trim as before. Fire alarm: the
read cut back to the first text node, which must truncate the code-span title.

This part is display only — the census names alarms by number — and mini-spec said so. It
sits second for that reason and lands in one small item.

## Item 3

`Pending.regionEnd` ends an entry's region at a `---` rule by testing each flat text node
for a line that is exactly three dashes. A code span's interior is its own text node, so a
code span holding three dashes in the middle of a prose line is a node whose whole text is
three dashes, and the region ends there. `Remove` then takes the entry's head and leaves its tail — from the closing
backtick to the entry's end — standing in the file, well-formed enough that `validate
trajectory` lists it as one unread line and nothing else objects. Measured 2026-09-12 by
mini-spec on `pending finish 55` at `2a9921c`, and reproduced here the same day on a
two-entry document: the render after `Remove` begins with the three dashes, the span's closing
backtick, and the rest of that sentence.

The spec already says what a rule is (R259: a `---` *line* outside a fence); the reader
tests a node where it should test a line. The repair is the test's subject, not a new rule:
a candidate `---` counts only when it is the whole of its document line — the node's text
starts at a line head, or the bytes before it on the line are blank, and the bytes after it
on the line are blank. A code span's interior fails the first half, since a backtick
precedes it on the line. `Place` shares `ruleAt` for the header's rule and gets the same
fix for free; the sibling in `Done` (`prependDoneEntry`) is checked in the build and fixed
if it tests the same way.

Same family as Items 1 and 2 — the reader's view of a markdown fact stops at a node boundary
where CommonMark's fact is a line — and the same shape mini-spec keeps naming: a write that
stops early and leaves a file nothing reads as wrong.

**Acceptance.** A fixture entry whose body carries a three-dash code span mid-line and a second entry
after it; the first reads whole and `Remove` of it leaves the second untouched and no tail;
a `---` line in a fence does not end a region (already true by construction, pinned); and
the existing rule-ends-region test still passes. Fire alarm: the line-head test removed from
`ruleAt`, which must leave the tail after `Remove`.
