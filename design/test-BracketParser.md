# Test Design: BracketParser
**Source:** crc-BracketParser.md, crc-BracketGroup.md

## Test: an opener, its contents and its closer are siblings
**Purpose:** R77 — the emitted stream is flat, whatever the nesting was
**Input:** `a {b {c} d} e` under a code-bracket language
**Expected:** a single flat run of nodes; no node has children; the two `{` are
`Opener`, the two `}` are `Closer`, and the depth appears nowhere in the array
**Refs:** crc-BracketParser.md, seq-scan.md#1.5
**Code:** sdom/parser_test.go

## Test: word markers respect boundaries
**Purpose:** R71 — the rule that separates a word bracket from a substring
**Input:** `download do file fi begin_ end` under a word-bracket language
**Expected:** `do`, `fi` and `end` are markers; the `do` inside `download`, the
`fi` inside `file` and the `begin` inside `begin_` are text
**Refs:** crc-BracketGroup.md
**Code:** sdom/parser_test.go
**Alarm:** 1
**Fire alarm:** Drop both word-boundary tests from `matchAt`, returning as soon as the bytes compare equal. Red: `do` fires inside `download`, `fi` inside `file`, `end` inside `begin_`. The round-trip and the tiling stay green — the bytes are all still there, just cut in the wrong places — so only an assertion about *which* nodes exist can see it.
**Inject:** sdom/bracket.go:matchAt
**Pulled:** 2026-08-30 — rang. This test failed, and the recognition count with
it (`shell: 4 separators, expected 3` — `do` firing inside `download`). Both
corpus round-trips stayed green.

## Test: separators are recognized only inside their own group
**Purpose:** R72
**Input:** `else` at top level, and `if x then y else z fi`
**Expected:** the bare `else` is text; the one inside the `if` group is a
`Separator`
**Refs:** crc-BracketParser.md, seq-scan.md#1.3.3
**Code:** sdom/parser_test.go

## Test: a stray closer lands as a bracket
**Purpose:** R73 — the any-close fallback keeps an unbalanced file scannable
**Input:** `a } b` with nothing open
**Expected:** the `}` is a `Closer`, the scan continues, and `b` is text
**Refs:** crc-BracketParser.md, seq-scan.md#3.1
**Code:** sdom/parser_test.go
**Alarm:** 2
**Fire alarm:** Make `matchAnyClose` always return no match. Red: the unmatched `}` becomes text instead of a `Closer`. Nothing else objects — no byte moves and the array still tiles — which is why a fallback that quietly stops firing needs its own assertion.
**Inject:** sdom/parser.go:parser.matchAnyClose
**Pulled:** 2026-08-30 — rang. This test failed, and so did
`TestWordMarkersRespectBoundaries`, whose fixture ends with an `end` that only
the fallback can recognize. **The recognition count did not catch it**, and
neither corpus round-trip did.

## Test: the scan never stalls
**Purpose:** R74 — the guarantee that makes unknown input safe
**Input:** bytes no table entry mentions, including a lone `$`, high-bit bytes
and a NUL
**Expected:** the scan terminates and the array tiles the input
**Refs:** crc-BracketParser.md, seq-scan.md#3.2
**Code:** sdom/parser_test.go
**No fire alarm, deliberately.** The obvious injection — remove the unconditional
`lx.pos++` from the text branch — produces an **infinite loop**, which is the
absence of a test result rather than a red one, and reads in a terminal exactly
like a hang. The guarantee is structural instead of asserted: that branch is only
reached once no marker matched at this position, so its first advance always
fires. Recorded here so the missing alarm is a decision rather than an oversight.

## Test: an unclosed group closes at end of input
**Purpose:** R75 — no bytes are dropped by an unbalanced file
**Input:** `func f() {` and `"unterminated` and `// trailing comment` with no
newline
**Expected:** each round-trips byte-exact and the array tiles the source
**Refs:** crc-BracketParser.md, seq-scan.md#3.4
**Code:** sdom/parser_test.go

## Test: whitespace folds into text
**Purpose:** R76 — a text run is everything between two markers
**Input:** `{ a  b\n  c }`
**Expected:** exactly one `Text` node between the `Opener` and the `Closer`,
holding the whitespace, the newline and the words together
**Refs:** crc-BracketParser.md
**Code:** sdom/parser_test.go

## Test: nothing inside a scan-restricted group is recognized
**Purpose:** R65 — the property that makes strings and comments the same case
**Input:** `"a { b // c"` and `// a "b" { c` and `/* a "b" // c */`
**Expected:** in each, one `Opener`, one literal `Text`, one `Closer`; the
brackets, quotes and comment markers inside are text
**Refs:** crc-BracketGroup.md, seq-scan.md#2.2
**Code:** sdom/parser_test.go
**Alarm:** 3
**Fire alarm:** Make `BracketGroup.Restricted` return false unconditionally. Red: every string and comment scans in code mode, so `{` inside a comment opens a group and `//` inside a string starts one. Bytes are preserved and the array still tiles, so the corpus round-trip stays green — this is the recognition-count failure class exactly.
**Inject:** sdom/bracket.go:BracketGroup.Restricted
**Pulled:** 2026-08-30 — rang, and widest of the batch: six tests failed,
including `TestLangGo`, `TestLangJavaScript`, the escape test and the recognition
count. Both corpus round-trips stayed green even so — every string and comment
was parsing its interior as code.

## Test: an escape consumes itself and the byte after it
**Purpose:** R61 — and specifically that an escaped closer does not close
**Input:** `"a\"b"` with `Escape: "\\"`, and the same text with `Escape: ""`
**Expected:** one string group with escaping; two with it disabled — the raw-mode
case must differ, or the escape is doing nothing
**Refs:** crc-BracketGroup.md, seq-scan.md#2.2.2
**Code:** sdom/parser_test.go
**Alarm:** 4
**Fire alarm:** In `scanRestricted`, advance past the escape sequence without consuming the byte after it. Red: the escaped quote closes the string, so one string becomes two plus stray text. The round-trip survives intact, because every byte is still emitted somewhere.
**Inject:** sdom/parser.go:parser.scanRestricted
**Pulled:** 2026-08-30 — rang, on this test and `TestLangGo`. Both of this
test's messages fired, including *disabling the escape changed nothing, so the
escape does nothing* — the second assertion earning its place. The recognition
count and both corpus round-trips stayed green.

## Test: AllowedInner reaches back into code mode
**Purpose:** R64, R65 — the escape hatch that makes interpolation work
**Input:** `` `text ${a + b} more` `` under `LangJavaScript`
**Expected:** the backtick group is restricted, `${` opens a code-mode group
inside it, `a + b` scans as code, and `}` closes back into the template
**Refs:** crc-BracketGroup.md, seq-scan.md#2.2.3
**Code:** sdom/parser_test.go

## Test: AllowedParent suppresses a marker outside its context
**Purpose:** R66 — the dual, and the reason it is not optional
**Input:** `${` at top level, and the same inside a backtick group
**Expected:** at top level it is a `$` followed by a `{` opener; inside the
template it is the interpolation opener
**Refs:** crc-BracketGroup.md
**Code:** sdom/parser_test.go
**Alarm:** 5
**Fire alarm:** Make `parentAllowed` return true unconditionally. Red: `${` opens an interpolation at top level, where it is really a `$` followed by a `{`. This is the failure the field exists to prevent, and it is invisible to everything except a test that scans `${` *outside* a template.
**Inject:** sdom/bracket.go:BracketGroup.parentAllowed
**Pulled:** 2026-08-30 — rang, on this test and `TestLangJavaScript`. `${x}` at
top level became an interpolation. Nothing else in 56 tests objected.

## Test: a recognition count, per language
**Purpose:** the failure nothing else can see — a sub-schema that quietly stops
recognizing something falls back to opaque text, which loses no bytes and breaks
no round-trip
**Input:** a fixture per shipped language, with the number of openers, closers
and separators of each kind stated in the test
**Expected:** the counts match exactly. A drop is a regression even though every
byte survives
**Refs:** crc-BracketParser.md, crc-BracketLang.md
**Code:** sdom/parser_test.go

## Test: byte round-trip over the corpus, per language
**Purpose:** R57 — scanning models more than `Text` did, and still loses nothing
**Input:** every corpus file, scanned with each shipped language
**Expected:** the document renders byte-identical to the source, and the array
tiles it
**Refs:** crc-BracketParser.md
**Code:** sdom/parser_test.go
