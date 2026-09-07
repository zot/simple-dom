# Carve: the design-document readers

> Opened 2026-09-06 to hold the readers over mini-spec's design documents that
> `carves/done/sdomification.md` said were "not carved yet": test designs, requirements,
> and the gaps section of `design.md`. The parts below are settled scope; the forks in
> their elaborations are open.

`carves/done/sdomification.md` carried the trajectory readers — carve, pending, done,
current — from `sdom` schemas into the mini-spec tool's readers, and closed with the
remaining document kinds uncarved. Mini-spec's reclaim (`~/work/mini-spec/carves/sdom-reclaim.md`
Items 5 and 7) now edits three of those kinds through readers that do not exist yet, and under
the carve decision of 2026-09-04 — simple-dom is the only reader, and mini-spec keeps thin
path-taking adapters — the readers belong here. This carve is those three readers.

**Provenance.** The three parts are the request mini-spec sent 2026-09-06, in the order it
asked for them; its numbering (1, 2, 3) is the request's and the parts below keep it. Our
acknowledgement is in `requests/`.
@ark-request-ref: /home/deck/work/mini-spec/requests/three-readers.md
`old-sdom` (`~/work/mini-spec`, branch `old-sdom`) had all three readers over its own DOM;
its requirements and code are cited in each part as prior art, not as a specification to
copy. The requirement numbers quoted are `old-sdom`'s.

## Status

- [x] ~~**Item 1 — the test-document reader.**~~ **LANDED (`478875e`, 2026-09-06 — `#35`.)**
- [ ] **Item 2 — the requirements reader.** **OPEN (#36.)**
- [x] ~~**Item 3 — the gaps reader.**~~ **LANDED (`50f104d`, 2026-09-07 — `#34`.)**

## Decisions

**DECIDED (Bill, 2026-09-06): the three readers live in `minispecsdom`, not as schemas in
`sdom/schema`.** They read mini-spec-specific file formats — a test design's alarm fields, a
requirements list, a gaps section — not general markdown shapes. Mini-spec's request left the
choice to us and said either shape serves its adapters.

**DECIDED (inherited from `carves/done/sdomification.md`, Bill, 2026-09-03): committed tests
rely only on code in this repository.** Fixtures are copied in and committed here,
expectations are written by hand from the format, and every part's acceptance includes the
byte round-trip over its corpus. No committed test shells out to `minispec` and none reads
`~/work/mini-spec`. Comparing a reader against `old-sdom`'s answers over live documents while
a part is built is a fine way to find what it misses; none of that becomes a test.

**DECIDED (inherited from `carves/done/sdomification.md`, Item 1): every reader reports what it
could not read.** `Unread()` with a line on each entry, and `Line()` on every entry, exactly as
the four trajectory readers do. A doubled field, an unkeyed entry, a shape-like line that did
not read: listed by the read path, refused by the write path, never guessed at.

**DECIDED (inherited from `carves/done/sdomification.md`, Items 3 and 10): writes go through
node references and refuse over deviations, and `Mutate` proves the tree describes the bytes.**
A writer never consults a stored line number; a write over an entry carrying deviations returns
an error naming them and leaves the bytes unchanged; every write reads itself back.

**The ordering is mini-spec's: 1, then 3, then 2.** Its Item 5 is queued behind Item 1 here;
its Item 7 needs Items 2 and 3. Nothing here blocks mini-spec, which works its Item 8 (the
skill) meanwhile. The status block above keeps the request's numbering; the queue carries the
order.

## Item 1

`minispecsdom.TestDoc`, for `design/test-*.md`.

**What it reads.** Each `## Test:` heading is an entry; its fields are read inside that
entry's own nodes and nowhere else. The fields, each `**Name:**` at the head of a line:

- `**Fire alarm:**` — prose, folded across continuation lines to the next field or heading.
- `**Inject:**` — a comma-separated list of `file:symbol` sites. The symbol is what the
  injection *edits*, which is a rule for the writer of the field and not for the reader.
- `**Pulled:**` — a leading `YYYY-MM-DD` date, then prose. The census reads the leading date.
  Written by the reader that runs the injection, never by the delegate — that rule lives in
  the skill; the reader only carries the line. **Never a commit hash** (Bill, 2026-09-04):
  `old-sdom` read an optional `@ <commit>` after the date, and no live test document carries
  one (measured 2026-09-06, 0 of the lines in `~/work/mini-spec/tool/design/test-*.md`), so
  the form is not read here.
- `**Code:**` — the test file or files the alarm vouches for.
- `**Alarm:**` — an integer identifier, local to the document, naming the alarm as
  `<doc>#<n>`. Absent means unmigrated, reported as an absence rather than numbered by position.

A field quoted inside a fence is body, not a field (`old-sdom` R458 — the DOM emits body inside a
fence, so a fenced `## Test:` cannot be returned as an entry and a fenced `**Alarm:** 1` cannot
be an alarm). A **doubled field** in one entry is a deviation, not a guess (`old-sdom` R426).
Fields other than these five under a `## Test:` — `**Purpose:**`, `**Input:**`, `**Expected:**`,
`**Refs:**` — are the test design's own and are read as body here, since no verb consumes them.

**What it writes**, through node references and never an indexed line (`old-sdom` R460):

- **Set or replace the `**Pulled:**` line** for one alarm: a new date and body, the previous
  line folded after it as history rather than overwritten (`old-sdom` R383). The verb
  `update pulled` sits on it; the date comes from the caller.
- **Replace the `**Inject:**` line** (`old-sdom` R384), and on the caller's say-so demote the
  `**Pulled:**` line out of field shape in the same write, naming the sites it was earned at.
  Whether the sites really moved is the verb's judgment — it resolves symbols and the reader
  does not — but only the writer holds the old sites at the moment of the write, so the
  demotion is a flag on the replace rather than a second write (Daneel, 2026-09-06).
- **Insert an `**Alarm:**` field** where one is absent: append-only, never renumbering, the
  next free number per file (`old-sdom` R379). The verb `update number-alarms` sits on it.

Where a field is inserted when the entry has none — after the last existing alarm field, at
the end of the entry's content — is a design choice the part settles against the live corpus,
not this carve.

`Unread()` and `Line()` on the entry, as the other readers have.

**Prior art.** `old-sdom` `tool/internal/parser/testdoc.go`, `tool/internal/alarm/alarm.go`,
`tool/design/crc-Alarm.md`, `tool/design/test-Alarm.md`; requirements R375–R384, R425, R426,
R458, R460.

## Item 2

A requirements reader, for `design/requirements.md`.

**What it reads.** Headings at any level, each with its own content region running to the
next heading of *any* level. `- **Rn:**` lines, including the retired form
`**~~Rn:~~** (Retired Tn — see Rm)`, which keeps its number and its original text behind the
marker. The `**Source:**` line per feature. A requirement body folds across continuation
lines like an alarm field does.

**What it writes.**

- **Append a requirement line** at the end of a heading's own content, before its first
  sub-heading — the `old-sdom` add-req rule. `update add-req` here mints the number
  (`query next-id req`, which counts retired ones) and calls the append.
- **The retire rewrite** of one line: the strikethrough, the `(Retired Tn — see Rm)` marker,
  the original text kept.

**Prior art.** `old-sdom` `tool/internal/parser/requirements.go`; requirements R77, R80,
R193, R454, R455.

## Item 3

A gaps reader, for the `## Gaps` section of `design/design.md`.

**What it reads.** Gap items `- [ ] O81: …` — the checkbox, the type letter, the number, the
text — and their nested sub-items. The permanent forms `A` and `T` carry no checkbox; a
permanent form written with one is a deviation the tool already names (`permanent gaps with
checkbox`). The section runs to the next heading of the same level or higher, and a `## `
quoted in a fence cannot end it early.

**What it writes.**

- **Add an item**: checkbox for S/R/D/C/I/O, none for A/T, the number minted by the caller.
- **Check one.**
- **Convert a checkbox item to a permanent one** — `approve-gap`, which rewrites the line
  under a new `A` number.

`query gaps [RANGE] [--open] [--closed]` and `update add-gap`/`resolve-gap`/`approve-gap` sit
on it in mini-spec; today those run on mini-spec's pre-sdom line readers, the second owner
the carve decision retires.

**Prior art.** `old-sdom` `tool/internal/query/gaps.go`, `tool/design/test-GapSelection.md`;
requirements R8, R22, R23, R62, R73, R82, R83, R446, R449.
