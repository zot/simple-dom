# Test Design: Pending
**Source:** crc-Pending.md

## Test: the fixture's entries read back
**Purpose:** R258, R259, R260, R264
**Input:** `minispecsdom/testdata/pending-sample.md`: a preamble and rule, three entries — one with a skill, one with a `Next:` line and a nested sub-item, one whose body holds a fenced `## 9.` — and a trailing `## Notes` heading
**Expected:** ids `8, 12, 3` in order; titles, skill, status, Source document and key, Next as written; `MaxID` 12; `Unread` lists `Notes`; the fenced `## 9.` is no entry; the render is byte-exact
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
**Pulled:** 2026-09-03 — rang: `position 0 accepted`, `position len+2 accepted`, `a refused Place changed the document`, only that test.

## Test: Remove drops exactly the run
**Purpose:** R263
**Input:** `Remove(12)` on the fixture, then `Remove(99)`
**Expected:** the render equals the fixture with entry 12's lines gone and its neighbours' bytes untouched; entries read `8, 3`; the unknown id errors
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
