# Carve: backtick runs in the markdown base

> **DRAFT (2026-09-05).** Opened the morning mini-spec reported that a two-backtick code span
> in its `DONE.md` cost the done reader 41 of 58 entries with nothing listed as unread. The
> scope below is settled, and the mechanism was decided the same day.

`LangMarkdown`'s bracket table knows two backtick groups, the three-backtick fence and the
one-backtick span, and nothing between or beyond. CommonMark's rule is that a run of N backticks
opens a code span that only a run of exactly N closes, for any N. A two-run today is read as a
one-span that opens and closes at once; the lone backtick *inside* the intended span then opens
a real one, and from there every backtick in the file that should open a span closes one
instead. The observable is silent: entry headers land inside spans, list items stop being list
items, and the reader sees no further entries while `Unread()` lists nothing — because nothing
entry-like survived to be unread.

This carve holds the two halves of the repair: read backtick runs as CommonMark does, and say
so when a group is still open at end of input, which is the same silence one layer down.

**This document is written without literal backtick runs longer than one**, spelled in words
instead, because the tool that reads carves parses them with the table this carve repairs:
the first draft carried a four-run and two two-runs, and `minispec query carves` reported it
as having no status block. A carve about a reader defect that the reader cannot read is the
defect demonstrated, not a curiosity. The sentence goes when mini-spec's tool reads carves
through a version of this module that carries Item 1 — landing here is not enough, since the
tool pins its own dependency.

**Provenance.** Both parts come from mini-spec, 2026-09-05: the report is
`~/work/mini-spec/requests/double-backtick-span-absorbs.md` and the same-day correction and
proposal is `~/work/mini-spec/requests/backtick-run-groups.md`; our acknowledgements are in
`requests/`.
@ark-response-ref: requests/RESP-double-backtick-span-absorbs.md
@ark-response-ref: requests/RESP-backtick-run-groups.md

## Status

- [x] ~~**Item 1 — backtick runs as one group.**~~ **LANDED (`c42cd24`, 2026-09-05 — `#26`.)**
- [ ] **Item 2 — a group open at end of input is reported.** **OPEN (#27.)**

## Decisions

**DECIDED (Bill, 2026-09-05, at mini-spec): bracket groups for runs of five backticks down to
one.** A native run-of-the-same-byte opener would express CommonMark's any-N rule in one entry,
but five suffices in practice: measured across every markdown file ark indexes — 3,171 files
over all our projects and notes — 9,522 runs of three, 21 runs of four, none of five or more.
Five is a free margin over the corpus, and the seam for more is one more table entry.
**SUPERSEDED the same day by the decision below**: with a pattern opener the enumeration and
the margin are unnecessary; the measurement stands as the record of why five would have been
enough.

**DECIDED (Bill, 2026-09-05): one bracket group whose opener is a pattern, whose closer is the
text that opened it, and whose markers must not be followed by their own byte.** Four changes
to `BracketGroup`, each earning its place:

- `Close` becomes a `string`. Every shipped group has exactly one closer, and a list said
  nothing about which closer paired with which opener; the one multi-closer group in the
  repository is a test convenience (`sdom/parser_test.go`, the word-boundary test), which
  splits into two groups with its expected stream unchanged.
- `CloseIsOpen bool`: the closer is whatever text opened this instance. For a symmetric group
  this says in one word what a repeated marker said in two; for a run group it is the only
  way to say it.
- `Lookahead string`: an anchored pattern the bytes after a marker must satisfy, or the marker
  does not match there — satisfied at end of input, since a closer is often a file's last
  byte. It applies to openers and to `CloseIsOpen` closers alike, so a longer run on either
  edge is literal text. It is additive to the word-boundary rule, which checks the leading
  edge too and which no lookahead can express.
- `OpenRegex string`: a pattern opener, anchored, exclusive with `Open`. This is the native
  run rule: CommonMark's any-N in one entry, with no list to order and no corpus margin.

**DECIDED (inherited from `carves/sdomification.md`): committed tests rely only on code in
this repository.** mini-spec's probe — parse its `DONE.md`, count `Entries()` against
`grep -c '^- \*\*'`, expect 58 — is the throwaway comparison that says whether the part is
done; it is not a test. The committed fixture is synthetic and lives in
`sdom/schema/testdata`.

## Item 1

The table's two backtick groups become one:

```go
{OpenRegex: "`+", Lookahead: "[^`]", CloseIsOpen: true, AllowedInner: []string{}, Kind: "code"},
```

The fence and the span are the same shape here — a restricted run closed by the same run — so
a four-backtick fence around a fenced sample, which is what the corpus's 21 four-runs are,
needs nothing the span does not. One kind stays, so a consumer that skips code skips one label.

**Why the two mechanisms in the code needed all of this, as read 2026-09-05.** `matchOpen` is
first-match in table order and `closeGroup` matches a closer by byte prefix through `matchAny`;
so an enumeration of run lengths would have made the order load-bearing, and would still have
let the first two bytes of a three-run close a two-span. The pattern opener removes the list,
`CloseIsOpen` picks the closer's length, and `Lookahead` refuses a longer run on either edge.
A global rule that a marker never extends into a longer run was considered and is wrong: Go's
`///`, Lua's `---` and markdown's own `***` are all longer runs of a shipped marker that must
still match, which is why the refusal is a per-group field.

**Consequences the part carries:**

- **The parser must remember which opener opened.** `open` has the marker; `parseBody` and
  `parseRestricted` receive only the group. The matched text rides down one level, or sits on
  the context the parser already keeps per open group.
- **`AllowedInner` names openers and resolves through `groupFor`.** A regex group has no
  literal opener to name, so it is named by its pattern string, and `groupFor` compares
  against `OpenRegex` as well as `Open`. `matchInner` then matches the resolved group's own
  opener — pattern or list — rather than the naming string as a prefix, or a two-run inside
  bold would open the group with the wrong marker. Bold's hatch names the backtick group once
  and admits every run.
- **The markers stay byte compares; only `OpenRegex` and `Lookahead` are regexes**, compiled
  once. Both run on the hot path at every position, so the cost is measured on the existing
  corpus before the part lands, not assumed either way.
- **`Open` and `OpenRegex` are exclusive, and `CloseIsOpen` with a non-empty `Close` is a
  contradiction.** Both are construction errors caught by the per-language table test, so no
  table can say two things about its markers.
- **The word-boundary test splits** into `do`/`fi` and `begin`/`end`; the stray `end` still
  lands through the any-close fallback, so R71 is exercised as before.

Whether bold and strike are rewritten to `CloseIsOpen` is cosmetic and decided when the part
is built.

**Two rules the build added, 2026-09-05 (Daneel; Bill to confirm).** Both surfaced as red
tests, not as design:

- **A close-is-open group checks its closer before any opener, in either mode.** In code mode
  the parser tries openers first, so a symmetric marker reopened instead of closing — the
  reason bold was made restricted. With `CloseIsOpen` the intent is unambiguous, so the check
  order follows it; R291 says so.
- **Inside a close-is-open pattern group, a match of the pattern that is not the opener's text
  is literal, consumed whole.** The lookahead guards the trailing edge only; stepping one byte
  into a three-run inside a two-span left a two-run followed by text, which closed the span.
  Go has no lookbehind, and none is needed: the run is taken as one piece, so the parse never
  stands inside it. R293 says so.

Landed `c42cd24`: R290–R298, `specs/bracket-parser.md` "Runs, and markers that close themselves",
`specs/markdown.md`'s table; R65 → R294 (T6), R227 → R298 (T7).

**Measured 2026-09-05, before landing.** mini-spec's `DONE.md` (210 KB, 60 entries by grep —
the file has grown since the request's 58): the old reader returned 17 entries, the new one
60, none unread. Parse time over 20 runs: 315 ms before, 569 ms after — 16 ms against 28 ms
per parse. The profile puts the regex and the pattern matcher together near a tenth of the
run; the rest is the extra dispatch in the per-byte opener loop, where three groups now pass
through a function instead of a byte compare. Recorded rather than optimized.

**Acceptance.** A committed fixture holding: a two-run span containing one backtick followed
by an entry-like line; a four-run fence enclosing a three-run fence followed by an entry-like
line; a three-run inside a two-run span; a closer as the file's last byte. Expectations written
by hand from CommonMark's rule; the byte round-trip over the fixture. Fire alarms: the
lookahead removed from the group, and the closer check made to ignore the opener's length,
each of which must make the list-item count go red.

## Item 2

The bracket parser closes a group left open at end of input silently — its own comment says
*the live run already holds every byte, so nothing drops*, which is true of bytes and false
of structure. A fence or span that runs to end of file takes every later heading and list
item with it, and no reader above the base can tell, because there is nothing left to list
as unread. This is the failure `Unread()` exists to prevent, arriving one layer below it.

Needed: the parse reports every group closed by end of input rather than by its closer —
the opener's text and its line — on a surface the schemas over the base can carry into their
own reports. This is the base layer's input to `carves/sdomification.md` Item 1, *say what
could not be read*, and should land in whatever shape that item settles on rather than
inventing a second one; if this part is scheduled first, it sets the shape and Item 1
inherits it.

**Acceptance.** A fixture whose last span has no closer; the report names its opener and
line; the byte round-trip is unchanged, since reporting is not repair.
