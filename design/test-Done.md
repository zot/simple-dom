# Test Design: Done
**Source:** crc-Done.md

## Test: the fixture's entries read back
**Purpose:** R266, R267, R268, R269, R270, R272
**Input:** `minispecsdom/testdata/done-sample.md`: a preamble and rule; an entry with `#8 / R189–R198` in the slot, a commit and a part pointer in the header, and an indented bullet in its body; an entry with an `O201` slot and a `#4` quoted in its body; an entry with no slot; a body holding a fenced quotation of an entry; an entry-like `- not an entry` bullet
**Expected:** three entries, the first's region keeping its indented bullet; IDs `[8]`, none, none; `HasSlot` true, true, false; the body's `#4` contributes nothing; part pointers as written, the second found in the body; `MaxID` 8; `Unread` 1; the render is byte-exact
**Refs:** crc-Done.md, seq-done.md#1
**Code:** minispecsdom/done_test.go
**Alarm:** 1
**Fire alarm:** read every `#N` in the entry rather than only the slot. Red: the second entry gains ID 4 and `MaxID` still reads 8 — so the test asserts the IDs, not only the max. A second injection, in `regionEnd`: let any list item end a region, not only one at column 0. Red: entry 0 loses its indented bullet — added 2026-09-03 after a past-the-list probe found the column-0 condition unguarded.
**Inject:** minispecsdom/done.go:DoneEntry.derive, minispecsdom/done.go:Done.regionEnd

## Test: Prepend lands after the rule
**Purpose:** R271
**Input:** `Prepend` on the fixture; `Prepend` on a ledger with a rule and no entries
**Expected:** the new entry is the first, separated from the rule and from the old first entry by blank lines; on the empty ledger it follows the rule; `Entries` reflects it
**Refs:** crc-Done.md, seq-done.md#2
**Code:** minispecsdom/done_test.go
**Alarm:** 2
**Fire alarm:** insert at the end instead of before the first entry. Red: the new entry is last.
**Inject:** minispecsdom/done.go:Done.Prepend
