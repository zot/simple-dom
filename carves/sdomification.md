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
2026-09-03 with their elaborations and decisions intact. Parts for the remaining document
kinds are not carved yet; they are added here as they are brainstormed, not listed ahead of
that.

## Status

- [ ] **Item 1 — say what could not be read.** **OPEN (not queued.)**
- [ ] **Item 2 — cut the tool over.** **OPEN (not queued.)**

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
