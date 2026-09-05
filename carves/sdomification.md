# Carve: sdom-ifying the mini-spec tool

> **DRAFT (2026-09-03).** Opened when `carves/done/trajectory.md` closed, to hold the two
> parts that carve did not finish and whatever the same direction spawns next. The parts
> below are settled scope; the forks in their elaborations are open.

The mini-spec tool at `~/work/mini-spec/tool` reads and writes its documents through an
older DOM of its own, plus regexes. The `sdom` project exists to replace that, and
`carves/done/simple-dom.md` and `carves/done/trajectory.md` built the machinery: the
protocol, the bracket and indent parsers, the traceability reader, the markdown base, and
a schema for each of the four trajectory files. This carve is the **continuation** — the
work that turns those readers into the tool's readers, and then carries the same treatment
to the document kinds the trajectory decision put behind the trajectory files:
requirements, design, CRC cards, sequences, and test designs.

**Provenance.** Items 1 and 2 were `carves/done/trajectory.md` Items 7 and 8, moved here
2026-09-03 with their elaborations and decisions intact. Items 3, 4 and 5 are the seven
requirements mini-spec sent 2026-09-04 after reading the readers against `old-sdom`, the
reference for what the tool's verbs consumed; the request is
`~/work/mini-spec/requests/trajectory-reader-requirements.md` and our acknowledgement is in
`requests/`. Their numbering below is the request's, not this carve's.
@ark-response-ref: requests/RESP-trajectory-reader-requirements.md
Items 6 to 10 are the seven defects mini-spec sent 2026-09-05 after porting its `pending`
verbs onto these readers, grouped by the code that repairs them; the request is
`~/work/mini-spec/requests/pending-reader-defects.md`, and the request's own numbering is
given in each part.
@ark-response-ref: requests/RESP-pending-reader-defects.md
Parts for the remaining document kinds are not carved yet; they are added here as they are
brainstormed, not listed ahead of that.

## Status

- [ ] **Item 1 — say what could not be read.** **OPEN (not queued.)**
- [ ] **Item 2 — cut the tool over.** **OPEN (not queued.)**
- [x] ~~**Item 3 — write paths refuse.**~~ **LANDED (`44a8955`, 2026-09-04 — `#23`.)**
- [x] ~~**Item 4 — lines, and flexible input.**~~ **LANDED (`ddcf09f`, 2026-09-04 — `#24`.)**
- [x] ~~**Item 5 — gap sources.**~~ **LANDED (`85fef32`, 2026-09-04 — `#25`.)**
- [ ] **Item 6 — pending writes end at the rule and round-trip.** **OPEN (#29.)**
- [ ] **Item 7 — markers replace every transient and precede prose.** **OPEN (#30.)**
- [ ] **Item 8 — the title read follows emphasis to its own close.** **OPEN (#31.)**
- [x] ~~**Item 9 — the current file's refusals are sentinels.**~~ **LANDED (`0c606db`, 2026-09-05 — `#28`.)**
- [ ] **Item 10 — `Mutate` proves the tree describes the bytes.** **OPEN (#32.)**
- [ ] **Item 11 — one key form: the fragment.** **OPEN (#33.)**

## Decisions

**DECIDED (inherited from trajectory-format.md, via `carves/done/trajectory.md`): the
reader reports what it could not read**, always — a count of entry-like lines it did not
recognize and of references it could not reach — because a shape-based check is blind to
whatever is outside the shape and reports clean over it. Item 1 is this decision made
schedulable.

**DECIDED (Bill, 2026-09-03, inherited from `carves/done/trajectory.md`): committed tests
rely only on code in this repository; the tool is a development instrument, and we wean off
it before the switch.** The mini-spec tool is being upgraded to replace its current
techniques with `sdom`, so it will not remain a reliable oracle — after the switch the two
would be tested against each other. While a part is being built, comparing the new reader
against the tool's answers over live files is a fine way to find what the reader misses;
none of that becomes a test. Acceptance for every part is: fixtures **committed here** (the
trajectory files are private in a live project, so samples are copied in, not read in
place), expectations written by hand from `trajectory-format.md`'s table, and the byte
round-trip over that corpus. No committed test shells out to `minispec`, and none reads
`~/work/mini-spec`.

## Item 1

The report: what was found and what could not be read, printed even when nothing else
is. Per file, the count of shape-like lines not recognized and of references not reached.
This is a reader obligation across the four trajectory schemas in `minispecsdom` — carve,
pending, done, current — rather than a feature; it is a part so that it is scheduled
rather than assumed. The done schema already produces the first input: entry-like lines
(`- ` at column 0) that did not read as entries.

## Item 2

Cutting the tool over: `parser/trajectory.go`, `parser/carve.go` and `partline` in
`~/work/mini-spec/tool` re-pointed at the new readers, and the tool's own older `sdom`
retired from those paths. This is the consumer's work and lives in that repository; it is
listed here so the carve says where the trajectory work ends.

**DECIDED (Bill, 2026-09-03): early comparisons, then wean, then one cut.** The old
reader is a useful instrument while the readers are being built and stops being one the
moment the tool is switched, so every comparison is throwaway and the acceptance behind
the cut is this repository's own tests. The cutover is then a consumer change with those
tests already green behind it.

## Item 3

Write paths refuse. `Deviation`'s own comment states the rule — *a read path lists it and a
write path refuses on it* — and `Carve.SetMarker` and `Land` do not honor it: they select the
part by key and write whatever its deviations. Three refusals, one part, because each is the
same principle applied at a different guard (request items 2, 5, 6):

- **Over a line carrying deviations**, either write returns an error naming every
  deviation's rule and target — `Deviations()` already carries both — and the line is
  unchanged. The one refusal that exists today, `ErrMarkerRefused`, guards the marker's own
  canon, not the line.
- **`OPEN` over a checked part.** From mini-spec's backup slot, the 2026-08-18 measured
  incident: a write that would return a part to `OPEN` refuses when its checkbox is `[x]`,
  independently of anything else. A second guard, not the caller's remembering.
- **`Land` over a checked part.** Today it writes a second `LANDED` beside the first — the
  two-contradictory-markers failure the marker rule exists to prevent. The request allowed
  refuse-or-idempotent; see the decision.

**DECIDED (Bill, 2026-09-04, on Daneel's proposal): `Land` refuses.** Idempotent
would keep the first record silently when a second landing carried a different commit and
date — a lie in whichever direction the caller meant. Refusal makes a double `finish` visible
where it happens, which is the composite verb's problem to handle, not the reader's to hide.

A refusal is decided before any of the three markings, so a refused `Land` leaves the line
byte-identical: the box, the strike and the marker agree because none of them moved.

Landed `44a8955`: the decisions are R279–R282 and `specs/carve-schema.md`'s "What it writes".

## Item 4

Two reader obligations the tool's diagnostics need (request items 3, 4):

- **Every part and entry reports its line**, 1-based: `Line()` on `Part`, `Entry` and
  `DoneEntry`, and the unread lists carrying a line each — `Pending.Unread()` is `[]string`
  today and `Done.Unread()` a bare count. A count without a line is a report nobody can
  act on; the tool prints `file:line`.
- **Flexible on input, rigid on output — one more case.** Bill's 2026-08-18 ruling: the
  reader accepts case, spacing and an optional full stop; the tool writes one canonical form.
  The verb already reads that way; the `OPEN` attribution does not — `not queued` without the
  stop, or `Not queued.`, reports as a deviation, and the format's own verb table writes
  `OPEN (not queued)`. The boundary that stays: a superseded *scheme* — bare `#N`, `**OPEN,
  not queued.**` — is a deviation.

Item 4 widens what Item 3 will accept, so their order matters only in the interim.

**DECIDED (Bill, 2026-09-04): a mis-cased verb stays a deviation.** The ruling's flexibility is
for the attribution; the verb is the shape itself.

Landed `ddcf09f`: R283–R286 and the four specs' API blocks. The comma form, silent before, is now
the `marker scheme` deviation.

## Item 5

**A gap ID is a valid `Source:`, and the reader must say which kind it read** (request item
1). Since 2026-08-26 a pending entry may be sourced from a gap rather than a carve part; the
line the tool writes is

    Source: [<doc>](<path>), gap `O136`.

— the gap ID after the word `gap`, no `#`. Today `sourceLineRe` reads the link and `PartKey`
only after `part`, so a gap-sourced entry parses with an empty key and is not listed by
`Unread()` either — the silent case. Needed: `Entry` carries the key for both shapes and a
`Kind` that says which; `EntryText.Text()` writes the gap form when asked. **DECIDED (Bill,
2026-09-04, at mini-spec): the gap-source idea is kept**; the shape is being added to
`trajectory-format.md`'s pending-entry section, which never carried it.

**A door kept open, not built** (request item 7): an entry may discharge several parts —
`Source:` names one, a done entry's body may name more. The old surface was
`Parts() []PartRef`. Nothing here should make that a breaking change later.

Landed `85fef32`: R287–R289 and `specs/pending-schema.md`. `PartKey` became `SourceKey`, a
consumer-visible rename taken while mini-spec's adapters were still being written.

## Item 6

Two defects in `Pending.Place` and `Pending.Remove` with one repair region (request items 1
and 2), both measured by test on mini-spec's side 2026-09-05:

- **`Place` at the last position lands after the rule.** `Place(e, n+1)` inserts at
  `len(nodes)`, so a pending file whose entries are followed by `---` and commentary — the
  shape `trajectory-format.md` describes and the fixture both trees test with — gets its new
  entry inside the commentary. `regionEnd` already returns `cut`, where the last entry stops
  at the rule; the last position is that point.
- **`Place` then `Remove` is not the identity.** `Place` at the end prepends a newline when the
  source does not end in two; `Remove` drops the entry's run and leaves that newline in the
  preceding text, so `add-item --last` then `finish` grows the file by one blank line every
  round. Old R480 on `old-sdom` said it: *`add-item` and `finish` are inverses or they are not*,
  and the byte-identical round trip is the test that says which. mini-spec's copy of that
  test skips by name until this lands.

Acceptance is that round trip over a fixture with and without trailing commentary.

## Item 7

Two defects in `partline.go`'s `SetMarker` (request items 5 and 6), one measured smoke test:
after `pending revert` then `pending replay` a part line read
`… **REVERTED (#70.)** Needs Item 1. **OPEN (#70.)**` — both states on one line, and the
marker after the prose.

- **`REVERTED` is a transient.** `isOpen` and the reopen guard at the same site replace a
  transient only when its verb is `OPEN`. mini-spec's `backup.md` (its R230) names `REVERTED`
  a transient too: it is what a revert writes over `OPEN (#N.)` and what a replay writes
  `OPEN (#N.)` back over. *mini-spec's fault first:* `trajectory-format.md`'s marker table never
  listed it, so the lexicon here had nothing to read it from; the table was fixed 2026-09-05.
- **An appended marker lands after trailing prose.** With no transient to replace, `SetMarker`
  appends at the end of the line. The grammar is `Head Marker* Text?` — annotations first,
  prose last, prose the terminator — so the line written is one this reader would list as
  non-conforming if it read the marker as one. Insert before the trailing prose.

## Item 8

**The title read is lazy** (request item 3). `entryHeadRe` matches the first emphasis run
non-greedily, so a title with emphasis inside it — `**A title mentioning **OPEN** here**` —
reads back as `A title mentioning `. `old-sdom` (its R389, 2026-08-24) read the first
emphasis run to *its own* close, with code spans suppressed and openers told from closers by
CommonMark's flanking rule; `**Say (**loudly**) now**` was the case a whitespace-only test got
wrong. Measured on mini-spec's live queue: 0 of 11 titles carry interior emphasis, so nothing
misreads today, which is why this is a part and not an incident. The write side refuses a
*wrapped* title and nothing else; whether it should refuse interior emphasis, or the read
should follow it, is decided when the part is built.

## Item 9

**`ParseCurrent`'s two refusals have no sentinels** (request item 4). The no-heading and
many-headings cases are `errors.New` at `current.go`, and mini-spec tells them apart by
`strings.Contains` on the message to name the repair — *add the heading beneath the rule,
holding `_No active item._`*. `ErrNoActive` and `ErrManyActive`, beside the existing
`ErrOccupied`, make that a type switch. Small, and the smallest part here.

Landed `0c606db`: R304.

## Item 10

**A parse check at the end of `Mutate`** — proposed by Bill, 2026-09-05, and the one part here
that guards the *next* class of defect rather than one already found. After a mutation,
`Parse(Render())` must yield the same node kinds, offsets and texts as the tree `Mutate` left
in memory. Every reader already pays the re-parse in `reload()`, so the comparison is the only
new cost, and writes are rare.

**DECIDED (Bill, 2026-09-05, at mini-spec): on a mismatch, panic.** It is a library invariant,
not caller input, and must not become an error a caller can swallow. The message is a crank
handle: the first diverging node — its kind, its line, and a short excerpt of the held text
against the re-parsed text. mini-spec's `editFile` recovers the panic into a refusal naming
the file, and the file stays untouched because nothing is written until render returns.

Stated plainly, as the request states it: this proves the tree describes the bytes. It does
not prove the edit addressed the right region, which the August re-parse check claimed and
could not do.

## Item 11

**A part has three key shapes, and `validate trajectory` is red here because of it.** Found
2026-09-05 running mini-spec's validator over this repository: every sdomification landing —
`DONE.md` entries for `#23`, `#24` and `#25` — is reported an orphan, *claims part N, which
that carve's status block does not hold*, while `query carves` reads the same block as three
landed parts. Probed with our own readers the same morning: `ParseCarve` keys the parts
`Item 3`, `Item 4`, `Item 5`; `ParseDone` reads the pointers as `3`, `4`, `5`; mini-spec's
check is an exact comparison of the two. Not a backtick problem — the block reads clean.

The shapes: `trajectory-format.md` says *there is one key form, `Item N`*, and in the same
breath that the fragment carries the bare number — `carves/x.md#9.1`. `PartLine.Key()`
(`specs/part-line.md`) returns `Item 1` for a part and `2.2` for a subpart, so the display
word is part of a part's key and absent from a subpart's. The pending writer emits
the word `part` and then the key behind a hash in a code span, using that `Key()`, so a
`Source:` line says `#Item 4`; mini-spec's `finish`
writes the done pointer in the format's fragment form, `#4`. The landing write succeeded
because it was keyed from the `Source:` line; the validator fails because it is keyed from the
pointer. Both are reading the format correctly, which is how a format with one-and-a-half key
forms fails: at the seam between two consumers.

**Needed: one form, and it is the fragment.** `Key()` returns what `#<key>` carries — `4`, `6.2` —
for parts and subparts alike; `Part(key)`, `SetMarker` and `Land` resolve that; the pending writer
emits `#4` after the word `part`; the display head `Item 4 — ` is rendered from the number, not
stored as the key. The alternative — `Part(key)` accepting either shape — keeps two forms alive to
be compared by the next consumer, and two forms is the defect. Consumer-visible, like the
`SourceKey` rename in Item 5: mini-spec's `Source:` lines and its `SetPartLanded` calls change
shape with it, and `trajectory-format.md`'s *one key form* sentence is rewritten to say which one.
Daneel's recommendation; Bill decides.

**Not a reader defect, recorded here so it is not re-investigated:** the same run reports
`#4 #5 #6` as item numbers in no readable entry. They are absent from both trajectory files;
`#7` is known only because a done header names it in its identifier slot. Those are August
items that were never recorded, and whether they get tombstone entries is a records decision.
