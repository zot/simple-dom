# Test Design: Pending
**Source:** crc-Pending.md

## Test: the fixture's entries read back
**Purpose:** R258, R259, R260, R264
**Input:** `minispecsdom/testdata/pending-sample.md`: a preamble and rule, five entries — one with a skill, one with a `Next:` line and a nested sub-item, one whose body holds a fenced `## 9.`, a gap-sourced one, one whose gap source is a range — and a `## Notes` heading between
**Expected:** ids `8, 12, 3, 14, 15` in order; titles, skill, status, Source document and key, Next as written; `MaxID` 15; `Unread` lists `Notes` and the range-sourced `Source:` line, each with its line; the fenced `## 9.` is no entry; the render is byte-exact
**Refs:** crc-Pending.md, seq-pending.md#1
**Code:** minispecsdom/pending_test.go
**Alarm:** 1
**Fire alarm:** end a region at any heading regardless of level. Red: the entry with the nested `###` sub-item loses its Next line to the sub-item's region.
**Inject:** minispecsdom/pending.go:Pending.regionEnd
**Pulled:** 2026-09-03 — rang: `the sub-item is not inside entry 12's region`, and `Remove(12)` left the sub-item behind — two tests.

## Test: Place by position, refused not clamped
**Purpose:** R261, R262, R265
**Input:** `Place` at 1, at `len+1`, and after id 12; then at 0 and at `len+2`
**Expected:** the renders show the new entry before the first, at the end, and after 12's region, each separated by a blank line; `Entries` reflects each write; the two bad positions error and the render is unchanged
**Refs:** crc-Pending.md, seq-pending.md#2
**Code:** minispecsdom/pending_test.go
**Alarm:** 2
**Fire alarm:** clamp an out-of-range position to the nearest valid one. Red: `Place` at 0 succeeds and the entry lands first.
**Inject:** minispecsdom/pending.go:Pending.Place
**Pulled:** 2026-09-05 — re-pulled by a delegated puller (stale since Item 5 moved `Place` on 2026-09-04); rang: `position 0 accepted`, `position len+2 accepted`, `a refused Place changed the document`, only that test. Previously 2026-09-03 — rang: `position 0 accepted`, `position len+2 accepted`, `a refused Place changed the document`, only that test.

## Test: Remove drops exactly the run
**Purpose:** R263
**Input:** `Remove(12)` on the fixture, then `Remove(99)`
**Expected:** the render equals the fixture with entry 12's lines gone and its neighbours' bytes untouched; entries read `8, 3, 14, 15`; the unknown id errors
**Refs:** crc-Pending.md, seq-pending.md#3
**Code:** minispecsdom/pending_test.go
**Alarm:** 3
**Fire alarm:** skip the tail split so the next entry's leading bytes go with the removed run. Red: the render loses the following entry's heading.
**Inject:** minispecsdom/pending.go:Pending.Remove
**Pulled:** 2026-09-03 — rang: the rule-ends-region test, `after Remove: "# Pending\n\n---\n\n"` — the rule and the prose went with the run.

## Test: a rule ends a region
**Purpose:** R259, R263 — found by injecting past the alarm list, not by design
**Input:** one entry followed by `---` and trailing prose
**Expected:** the entry's region stops before the rule; `Remove` leaves the rule and the prose exactly as they were
**Refs:** crc-Pending.md, seq-pending.md#1.3
**Code:** minispecsdom/pending_test.go
**Alarm:** 4
**Fire alarm:** drop the `---` branch in `regionEnd`. Red: the region runs to the end of the file and `Remove` takes the rule and the prose with it. Every other test stayed green under this injection on 2026-09-03, since the fixture's only rule precedes its entries.
**Inject:** minispecsdom/pending.go:Pending.regionEnd
**Pulled:** 2026-09-03 — rang: `the region ran past the rule`, only this test; injection verified applied, restored by reverse edit.

## Test: entries and unread headings carry their line
**Purpose:** R283, R284
**Input:** the fixture (folded into "the fixture's entries read back")
**Expected:** entries 8, 12, 3 at lines 8, 12, 19; `Unread` is `{25, "Notes"}` then the range-sourced line at 33
**Refs:** crc-Pending.md, seq-pending.md#1
**Code:** minispecsdom/pending_test.go
**Alarm:** 5
**Fire alarm:** record every line as 0 in `scan` — entries and unread alike. Red: the line assertions and the unread comparison both fail.
**Inject:** minispecsdom/pending.go:Pending.scan
**Pulled:** 2026-09-05 — re-pulled by a delegated puller after the reader gained the unclosed report; rang: `unread [{Line:0 Text:Notes} …]` and `lines 0 0 0`, only the fixture read-back. Previously 2026-09-04 — rang: `unread [{Line:0 Text:Notes}]` and `lines 0 0 0`; only that test. A first attempt did not build: the probe comment swallowed the closing parenthesis on the same line. Re-pulled the same day after the simplification pass hoisted the line into a local: rang again.

## Test: a source is a part or a gap
**Purpose:** R287, R288, R289
**Input:** the fixture's entries 8 (`part `#7``), 14 (`gap `O136``), 15 (`gap `O1-O3``, a range) and 3 (no `Source:` line); `EntryText{Kind: SourceGap, SourceKey: "O7"}.Text()`; `Place` of a gap entry keyed `O1, O2`; `Place` of the `O7` entry at the end
**Expected:** 8 is `SourcePart` key `7`; 14 is `SourceGap` key `O136`; 15 is `SourceNone` with an empty key and its document still read; 3 is `SourceNone` and not unread; the text renders `gap `O7``; the list is refused with `ErrBadGapSource`; the placed entry reads back as a gap source
**Refs:** crc-Pending.md, seq-pending.md#1.4.1, seq-pending.md#2.1.1
**Code:** minispecsdom/pending_test.go
**Alarm:** 6
**Fire alarm:** drop the `gapIDRe` check from `derive` so any backquoted gap key reads as `SourceGap`. Red: entry 15 reads as a gap keyed `O1-O3` and leaves `Unread`. A second injection: drop the refusal from `Place`. Red: the list is placed, and reads back as `SourceNone`.
**Inject:** minispecsdom/pending.go:Entry.derive, minispecsdom/pending.go:Pending.Place
**Pulled:** 2026-09-04 — both rang: without the ID check entry 15 read as `Kind:2` keyed `O1-O3` and `Unread` shrank to `Notes` alone (the fixture test objected too); without the refusal `a list as a gap source: <nil>`; only those tests; restore clean. The `derive` injection re-pulled the same day after the simplification pass named the groups: rang again.

## Test: a group open at end of input is unread
**Purpose:** R301
**Input:** a pending file whose entry is followed by a two-backtick span that never closes
**Expected:** `Unread` holds one item at the span's line naming the marker; the entry still reads
**Refs:** crc-Pending.md
**Code:** minispecsdom/pending_test.go
