# Test Design: Carve
**Source:** crc-Carve.md

## Test: the fixture's status block reads as parts
**Purpose:** R248, R249, R250, R251, R252, R256
**Input:** `sdom/schema/testdata/trajectory-sample.md`
**Expected:** `HasStatus`; four parts keyed `Item 1`, `2.1`, `2.2`, `Item 4` with depths `0, 2, 2, 0` and parents nil, `Item 1`, `Item 1`, nil — the subparts follow `Item 1` because the split parent between them has no checkbox; one stateless line keyed `Item 2`; the body's `- [ ] inside a fence` is not a part; the render is byte-exact
**Refs:** crc-Carve.md, seq-carve.md#1
**Code:** minispecsdom/carve_test.go
**Alarm:** 1
**Fire alarm:** drop the region bound so every list item in the file is tried. Red: nothing here, since the fixture's other bullets are fenced — so the test also parses a source with a bullet under a later `## Notes`, which becomes a part under the injection; a `### Sub` inside the block does not end it.
**Inject:** minispecsdom/carve.go:ParseCarve

## Test: no status block, and a fenced one
**Purpose:** R249, R250
**Input:** a document with no `## Status`; a document whose only `## Status` is inside a fence
**Expected:** `HasStatus` false and no parts for both
**Refs:** crc-Carve.md, seq-carve.md#1.2
**Code:** minispecsdom/carve_test.go
**Alarm:** 2
**Fire alarm:** match the status heading by text alone rather than a `Heading` node. Red: the fenced document reports a status block.
**Inject:** minispecsdom/carve.go:Carve.statusRegion

## Test: SetMarker follows the tool's rule
**Purpose:** R253, R254
**Input:** `Item 4` (`OPEN (not queued.)`) → `SetMarker("OPEN", "#9.")`; a line carrying `**NOT VERIFIED.** **OPEN (#8.)**` → `SetMarker("LANDED", "x")`; a line with no marker → `SetMarker("DEFERRED", "Bill, 2026-09-03")`; a line carrying `**OPEN (#4.)** **OPEN (#5.)**` → `SetMarker("OPEN", "#6.")`; an unknown key
**Expected:** `OPEN (#9.)` replaces in place; `NOT VERIFIED` stands and the `OPEN` becomes `LANDED (x)`; the bare line gains ` **DEFERRED (Bill, 2026-09-03)**`; the two-transient line becomes `**OPEN (#6.)**` alone; the unknown key errors and nothing changes
**Refs:** crc-Carve.md, seq-carve.md#2.3
**Code:** minispecsdom/carve_test.go
**Alarm:** 3
**Fire alarm:** replace the first marker regardless of verb. Red: `NOT VERIFIED` is overwritten. A second injection: stop dropping the other transients (`drop` never appended). Red: `**OPEN (#6.)** **OPEN (#5.)**` — this case was added 2026-09-03 after a past-the-list probe; that probe's green was a mis-applied injection (a `sed` that matched nothing), so whether the earlier suite guarded the rule was never established — no earlier test carried two transients, which is the inference the case rests on. Re-run with the injection verified applied, it rang here.
**Inject:** minispecsdom/partline.go:PartLine.SetMarker

## Test: Land is three markings in one act
**Purpose:** R255
**Input:** `2.2` → `Land("` + "`abc`" + `, 2026-09-03 — ` + "`#8`" + `.")`
**Expected:** the line renders `  - [x] ~~**2.2 — the config move.**~~ **LANDED (`abc`, 2026-09-03 — `#8`.)**`
**Refs:** crc-Carve.md, seq-carve.md#2
**Code:** minispecsdom/carve_test.go
**Alarm:** 4
**Fire alarm:** skip the strike in `Land`. Red: the render keeps the head unstruck while the box and marker changed — the three no longer agree.
**Inject:** minispecsdom/carve.go:Carve.Land
