# Carve: backtick runs in the markdown base

> **DRAFT (2026-09-05).** Opened the morning mini-spec reported that a two-backtick code span
> in its `DONE.md` cost the done reader 41 of 58 entries with nothing listed as unread. The
> scope below is settled; one mechanism fork is open.

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
defect demonstrated, not a curiosity, and the sentence goes when Item 1 lands.

**Provenance.** Both parts come from mini-spec, 2026-09-05: the report is
`~/work/mini-spec/requests/double-backtick-span-absorbs.md` and the same-day correction and
proposal is `~/work/mini-spec/requests/backtick-run-groups.md`; our acknowledgements are in
`requests/`.
@ark-response-ref: requests/RESP-double-backtick-span-absorbs.md
@ark-response-ref: requests/RESP-backtick-run-groups.md

## Status

- [ ] **Item 1 — runs of five backticks down to one.** **OPEN (not queued.)**
- [ ] **Item 2 — a group open at end of input is reported.** **OPEN (not queued.)**

## Decisions

**DECIDED (Bill, 2026-09-05, at mini-spec): bracket groups for runs of five backticks down to
one.** A native run-of-the-same-byte opener would express CommonMark's any-N rule in one entry,
but five suffices in practice: measured across every markdown file ark indexes — 3,171 files
over all our projects and notes — 9,522 runs of three, 21 runs of four, none of five or more.
Five is a free margin over the corpus, and the seam for more is one more table entry.

**DECIDED (inherited from `carves/sdomification.md`): committed tests rely only on code in
this repository.** mini-spec's probe — parse its `DONE.md`, count `Entries()` against
`grep -c '^- \*\*'`, expect 58 — is the throwaway comparison that says whether the part is
done; it is not a test. The committed fixture is synthetic and lives in
`sdom/schema/testdata`.

## Item 1

Five groups where there are two, all restricted with kind `code` and an empty `AllowedInner`,
so a consumer that skips code still skips one label. The fence and the span are the same
group shape here — a restricted run closed by the same run — so a four-backtick fence around
a fenced sample, which is what the corpus's 21 four-runs are, needs nothing the span does not.

Two details decide whether the table alone is correct, and both are live in the code as
read 2026-09-05, not hypothetical:

- **`matchOpen` is first-match in table order** (`sdom/bracket_parser.go`). Five must precede
  four must precede three, or a four-run opens a three-fence and leaves a stray backtick —
  today's failure one level up. With first-match the order is load-bearing, not cosmetic, and
  the table's comment must say so the way it already says why the fence precedes the span.
- **`closeGroup` matches a closer by prefix.** It calls `matchAny` over `g.Close`, and
  `matchAt` is a byte-prefix comparison with a word-boundary check that backticks never
  trigger. So with a two-run group open, the first two bytes of a three-run close it. Rare in
  prose, same class of bug, and the enumeration does not fix it by itself.

**OPEN fork — how a closer comes to match the whole run.** Two mechanisms:

1. A `BracketGroup` field marking a marker as a *run*, so `matchAt` refuses a match adjacent
   to its own byte — the word-boundary rule it already applies, chosen per group instead of
   per byte class. Applied to openers too, the table order becomes cosmetic: a four-run can
   only ever match the four group.
2. Nothing new in `BracketGroup`; the five entries plus ordering handle openers, and the
   closer stays a prefix match with the corner documented.

A *global* rule that a marker never extends into a longer run is not on the list: Go's `///`,
Lua's `---` and markdown's own `***` are all longer runs of a shipped marker that must still
match. Daneel's recommendation is 1: it is the smaller change to reason about — the rule is
one sentence on the group, and every test over the table stays a test over the table — and
a documented corner that silently swallows a span's tail is not the kind `sharp corners`
covers, since the corruption is silent.

Bold's escape hatch names the single backtick alone. Whether it should name all five runs is
decided when the part is built; `**LANDED (`abc`)**` needs one, and nothing measured needs more.

**Acceptance.** A committed fixture holding: a two-run span containing one backtick followed
by an entry-like line; a four-run fence enclosing a three-run fence followed by an entry-like
line; a three-run inside a two-run span. Expectations written by hand from CommonMark's rule;
the byte round-trip over the fixture. The fire alarms are the table reordered and the four
group removed, each of which must make the list-item count go red.

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
