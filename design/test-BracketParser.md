# Test Design: BracketParser
**Source:** crc-BracketParser.md, crc-BracketGroup.md

## Test: an opener, its contents and its closer are siblings
**Purpose:** R77 — the emitted stream is flat, whatever the nesting was
**Input:** `a {b {c} d} e` under a code-bracket language
**Expected:** a single flat run of nodes; no node has children; the two `{` are
`Opener`, the two `}` are `Closer`, and the depth appears nowhere in the array
**Refs:** crc-BracketParser.md, seq-parse.md#1.5
**Code:** sdom/parser_test.go

## Test: word markers respect boundaries
**Purpose:** R71 — the rule that separates a word bracket from a substring
**Input:** `download do file fi begin_ end` under a word-bracket language of two groups, `do`/`fi` and `begin`/`end`
**Expected:** `do`, `fi` and `end` are markers; the `do` inside `download`, the
`fi` inside `file` and the `begin` inside `begin_` are text
**Refs:** crc-BracketGroup.md
**Code:** sdom/parser_test.go
**Alarm:** 1
**Fire alarm:** Drop both word-boundary tests from `matchAt`, returning as soon as the bytes compare equal. Red: `do` fires inside `download`, `fi` inside `file`, `end` inside `begin_`. The round-trip and the tiling stay green — the bytes are all still there, just cut in the wrong places — so only an assertion about *which* nodes exist can see it.
**Inject:** sdom/bracket.go:boundaryOK
**Pulled:** 2026-09-05 — re-pulled after backtick-runs Item 1 touched the site; rang: this test, the recognition count (`shell: recognized 2 of "Sdo", expected 1`), and the new pattern-boundary test. The check moved from `matchAt` into `boundaryOK`, which the `Inject:` now names. Previously 2026-08-30 — rang. This test failed, and the recognition count with
it (`shell: 4 separators, expected 3` — `do` firing inside `download`). Both
corpus round-trips stayed green.

## Test: separators are recognized only inside their own group
**Purpose:** R72
**Input:** `else` at top level, and `if x then y else z fi`
**Expected:** the bare `else` is text; the one inside the `if` group is a
`Separator`
**Refs:** crc-BracketParser.md, seq-parse.md#1.3.3
**Code:** sdom/parser_test.go

## Test: a stray closer lands as a bracket
**Purpose:** R73 — the any-close fallback keeps an unbalanced file parsable
**Input:** `a } b` with nothing open
**Expected:** the `}` is a `Closer`, the parse continues, and `b` is text
**Refs:** crc-BracketParser.md, seq-parse.md#3.1
**Code:** sdom/parser_test.go
**Alarm:** 2
**Fire alarm:** Make `matchAnyClose` always return no match. Red: the unmatched `}` becomes text instead of a `Closer`, **and** `TestWordMarkersRespectBoundaries` fails too — its fixture ends with an `end` that only the fallback recognizes. No byte moves and the array still tiles, which is why a fallback that quietly stops firing needs its own assertion.
**Inject:** sdom/bracket_parser.go:BracketParser.matchAnyClose
**Pulled:** 2026-09-05 — re-pulled after backtick-runs Item 1 touched the site; rang: four tests — this one, the word-boundary fixture, `TestOpenerAndCloserAreTyped` (`a stray closer has an opener`) and the new close-is-open fallback test. Previously 2026-09-01 — re-pulled after the file split and rang again, on the same two tests. The anchor resolved under its new home, `sdom/bracket_parser.go:BracketParser.matchAnyClose`, which is what this pull was taken for. Previously 2026-08-31 — re-pulled after the parser rename and rang again, on this test and `TestWordMarkersRespectBoundaries`, as before. The rename moved no property; only symbols changed name. Originally 2026-08-30 — rang. This test failed, and so did
`TestWordMarkersRespectBoundaries`, whose fixture ends with an `end` that only
the fallback can recognize. **The recognition count did not catch it**, and
neither corpus round-trip did.

## Test: the parse never stalls
**Purpose:** R74 — the guarantee that makes unknown input safe
**Input:** bytes no table entry mentions, including a lone `$`, high-bit bytes
and a NUL
**Expected:** the parse terminates and the array tiles the input
**Refs:** crc-BracketParser.md, seq-parse.md#3.2
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
**Refs:** crc-BracketParser.md, seq-parse.md#3.4
**Code:** sdom/parser_test.go

## Test: whitespace folds into text
**Purpose:** R76 — a text run is everything between two markers
**Input:** `{ a  b\n  c }`
**Expected:** exactly one `Text` node between the `Opener` and the `Closer`,
holding the whitespace, the newline and the words together
**Refs:** crc-BracketParser.md
**Code:** sdom/parser_test.go

## Test: nothing inside a parse-restricted group is recognized
**Purpose:** R65 — the property that makes strings and comments the same case
**Input:** `"a { b // c"` and `// a "b" { c` and `/* a "b" // c */`
**Expected:** in each, one `Opener`, one literal `Text`, one `Closer`; the
brackets, quotes and comment markers inside are text
**Refs:** crc-BracketGroup.md, seq-parse.md#2.2
**Code:** sdom/parser_test.go
**Alarm:** 3
**Fire alarm:** Make `BracketGroup.Restricted` return false unconditionally. Red: every string and comment parses in code mode, so `{` inside a comment opens a group and `//` inside a string starts one. Bytes are preserved and the array still tiles, so the corpus round-trip stays green — this is the recognition-count failure class exactly.
**Inject:** sdom/bracket.go:BracketGroup.Restricted
**Pulled:** 2026-09-05 — re-pulled after the census attributed this day's struct edit to `Restricted` (the function itself did not change); rang in 15 tests. Previously 2026-09-04 — re-pulled at `ddcf09f` by delegation: rang, the named test's three cases and seven more across the parser, language, indent, escape and round-trip suites; restore clean. Previously 2026-08-30 — rang, and widest of the batch: six tests failed,
including `TestLangGo`, `TestLangJavaScript`, the escape test and the recognition
count. Both corpus round-trips stayed green even so — every string and comment
was parsing its interior as code.

## Test: an escape consumes itself and the byte after it
**Purpose:** R61 — and specifically that an escaped closer does not close
**Input:** `"a\"b"` with `Escape: "\\"`, and the same text with `Escape: ""`
**Expected:** one string group with escaping; two with it disabled — the raw-mode
case must differ, or the escape is doing nothing
**Refs:** crc-BracketGroup.md, seq-parse.md#2.2.2
**Code:** sdom/parser_test.go
**Alarm:** 4
**Fire alarm:** In `BracketParser.parseRestricted`, advance past the escape sequence without consuming the byte after it. Red: the escaped quote closes the string, so one string becomes two plus stray text. The round-trip survives intact, because every byte is still emitted somewhere.
**Inject:** sdom/bracket_parser.go:BracketParser.parseRestricted
**Pulled:** 2026-09-05 — pulled again after the simplifier restructured the site; rang. Earlier the same day: re-pulled by hand after Item 8 rewrote the site; rang: `TestEscapeConsumesItselfAndTheNextByte`, `TestLangGo` and the declaration pass — the restricted loop was reordered so the hatches precede the run rule. Previously 2026-09-05 — re-pulled after backtick-runs Item 1 touched the site; rang: `TestEscapeConsumesItselfAndTheNextByte`, `TestLangGo` and, across the package line, `TestADeclarationPassIsAdditive`. Previously 2026-09-03 — re-pulled after the self-advances became `Advance` and rang in three places: this test (`disabling the escape changed nothing`), `TestLangGo`, and the declaration count in `sdom/schema` (`found 3 declarations, column-0 keywords say 7`) — an unescaped quote splits a string and the pass reads the halves as code. Previously 2026-09-01 — re-pulled after the vocabulary pass and rang again. The anchor resolved correctly under the new name `parseRestricted`, which is what this pull was taken for. **Wider than the two earlier pulls:** this test, `TestLangGo`, and `TestADeclarationPassIsAdditive` in `sdom/schema` — 4 failures across 2 packages, the third of them not existing when this alarm was first written. The round-trip stayed green throughout, exactly as the prose says. Previously 2026-08-31 — re-pulled after the parser rename and rang again, on this test and `TestLangGo`, as before. The rename moved no property; only symbols changed name. Originally 2026-08-30 — rang, on this test and `TestLangGo`. Both of this
test's messages fired, including *disabling the escape changed nothing, so the
escape does nothing* — the second assertion earning its place. The recognition
count and both corpus round-trips stayed green.

## Test: AllowedInner reaches back into code mode
**Purpose:** R64, R65 — the escape hatch that makes interpolation work
**Input:** `` `text ${a + b} more` `` under `LangJavaScript`
**Expected:** the backtick group is restricted, `${` opens a code-mode group
inside it, `a + b` parses as code, and `}` closes back into the template
**Refs:** crc-BracketGroup.md, seq-parse.md#2.2.3
**Code:** sdom/parser_test.go

## Test: AllowedParent suppresses a marker outside its context
**Purpose:** R66 — the dual, and the reason it is not optional
**Input:** `${` at top level, and the same inside a backtick group
**Expected:** at top level it is a `$` followed by a `{` opener; inside the
template it is the interpolation opener
**Refs:** crc-BracketGroup.md
**Code:** sdom/parser_test.go
**Alarm:** 5
**Fire alarm:** Make `parentAllowed` return true unconditionally. Red: `${` opens an interpolation at top level, where it is really a `$` followed by a `{`. This is the failure the field exists to prevent, and it is invisible to everything except a test that parses `${` *outside* a template.
**Inject:** sdom/bracket.go:BracketGroup.parentAllowed
**Pulled:** 2026-09-05 — re-pulled after backtick-runs Item 1 touched the site; rang: `TestAllowedParentSuppressesOutsideItsContext` and `TestLangJavaScript` (`${ must not be an interpolation opener at top level`). The site now resolves the parent by `named`, pattern included. Previously 2026-08-30 — rang, on this test and `TestLangJavaScript`. `${x}` at
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
**Purpose:** R57 — parsing models more than `Text` did, and still loses nothing
**Input:** every corpus file, parsed with each shipped language
**Expected:** the document renders byte-identical to the source, and the array
tiles it
**Refs:** crc-BracketParser.md
**Code:** sdom/parser_test.go

## Test: a run closes only with a run of its own length
**Purpose:** R291, R292, R309 — CommonMark's rule from one pattern group: opener a run of backticks, `CloseIsOpen`; the closer is the pattern's match equal to the opener
**Input:** a two-run holding one backtick then text; a four-run enclosing a three-run; a three-run inside a two-run span; a run as the last bytes of the source
**Expected:** each span opens and closes on runs of equal length, the interior is one text node whatever shorter or longer runs it holds, and the last-byte closer pairs. The three-run inside the two-span is the leading-edge case: the parse must consume it whole rather than read its last two bytes as the closer
**Refs:** crc-BracketGroup.md, crc-BracketParser.md, seq-parse.md#1.4
**Code:** sdom/parser_test.go
**Fire alarm:** compare the closer by prefix — `matchAt(src, pos, opened)` alone, without the pattern equality. Red: the three-run inside the two-span closes it early — the parity inversion that lost 41 of 58 entries. (Until 2026-09-05 the injection was clearing a `Lookahead` field, since retired.)
**Inject:** sdom/bracket_parser.go:BracketParser.matchCloser
**Pulled:** 2026-09-05 — pulled again after the simplifier restructured the site; rang. Earlier the same day: re-pulled by hand after Item 8 rewrote the site; rang: the prefix comparison rang in four tests across both packages — this one, the reject test, the emphasis test and the carve's closes-nothing test. Previously 2026-09-05 — rang: `TestARunClosesOnlyWithARunOfItsOwnLength` on the three-run-inside-a-two-span case, only that test. **Past the list, same day:** disabling `skipOtherRun` whole rang here and in `TestBacktickRuns` (`items 3, want 4`; an opener with no closer); dropping `lookOK`'s end-of-input clause rang in three tests, the last-byte closer among them. Both are now named by this alarm's neighbours rather than by luck.

## Test: the closer is the opener's own bytes, not its length class
**Purpose:** R291 — `CloseIsOpen` compares text
**Input:** a symmetric literal group written with `CloseIsOpen` and no `Close`, with two openers in one group, over input using both
**Expected:** each instance closes on the marker that opened it and not on the other opener
**Refs:** crc-BracketGroup.md, seq-parse.md#1.4
**Code:** sdom/parser_test.go
**Fire alarm:** make `closeGroup` accept any of the group's openers as the closer. Red: the second opener closes the first.
**Inject:** sdom/bracket_parser.go:BracketParser.closeGroup
**Pulled:** 2026-09-05 — re-pulled by hand after Item 8 rewrote the site; rang: `TestTheCloserIsTheOpenersOwnBytes` alone. Previously 2026-09-05 — rang: `TestTheCloserIsTheOpenersOwnBytes`, only that test; the injection sits in `matchCloser`, which `closeGroup` calls.

## Test: a pattern opener honours word boundaries and its edges
**Purpose:** R292, R309 — the boundary rule applies to the bytes a pattern matched; flanking is a table entry
**Input:** a word-run pattern such as `x+` over `xx xxa axx`; an emphasis group — a run of asterisks, `CloseIsOpen`, `AfterOpen` and `BeforeClose` non-whitespace, naming itself — over `**a **b** c** 2 * 3` and `**a *b* c**`
**Expected:** `xx` alone matches; `xxa` and `axx` are text; the emphasis nests — inner `**b**` and `*b*` each pair inside the outer — and `2 * 3` is text
**Refs:** crc-BracketGroup.md
**Code:** sdom/parser_test.go
**Fire alarm:** skip the boundary check for pattern matches. Red: `xx` fires inside `xxa`.
**Inject:** sdom/bracket_parser.go:BracketParser.matchPattern
**Pulled:** 2026-09-05 — rang: `TestAPatternOpenerHonoursWordBoundariesAndLookahead` — `xx` fired inside `xxa`, only that test.

## Test: AllowedInner names a group, and matching uses the group's opener
**Purpose:** R294, R295 — a hatch named by one opener admits the group's whole opener set, pattern included
**Input:** a restricted group whose `AllowedInner` names the run group by its pattern; inside it a two-run and a one-run
**Expected:** both open the run group with their own bytes and close on the same
**Refs:** crc-BracketParser.md, seq-parse.md#2.2.3
**Code:** sdom/parser_test.go
**Fire alarm:** match the naming string as a prefix, as `matchInner` did. Red: the two-run opens as a one-run and the interior parses wrong.
**Inject:** sdom/bracket_parser.go:BracketParser.matchInner
**Pulled:** 2026-09-05 — re-pulled by hand after Item 8 rewrote the site; rang: six carve and part-line tests — with the name prefix-matched, the `~~` group's hatch for emphasis never opens and every struck head misreads. Previously 2026-09-05 — rang: `TestAllowedInnerNamesAGroup` and, across the package line, `TestTheEscapeHatchesPair` (`"`": 0 paired groups, want 2`) — the pattern name never prefix-matches, so the hatch never opens.

## Test: a contradictory table panics at construction
**Purpose:** R296 — the three construction errors are seen by the table test, never a consumer
**Input:** a group with `Open` and `OpenRegex`; a group with `CloseIsOpen` and a `Close`; a group whose pattern does not compile
**Expected:** `NewBracketParser` panics for each, and the message names the group
**Refs:** crc-BracketLang.md
**Code:** sdom/parser_test.go
**Fire alarm:** drop the exclusivity check. Red: the first case constructs.
**Inject:** sdom/bracket.go:BracketLang.check
**Pulled:** 2026-09-05 — pulled again after the simplifier restructured the site; rang. Earlier the same day: rang: `Open and OpenRegex: constructed without panicking`, only that case of that test.

## Test: the any-close fallback ignores close-is-open groups
**Purpose:** R297
**Input:** a code-mode `CloseIsOpen` group beside a brace group, over a stray `}` and a lone symmetric marker
**Expected:** the stray `}` lands as a `Closer`; the lone marker opens its group and closes at end of input
**Refs:** crc-BracketParser.md, seq-parse.md#3.1
**Code:** sdom/parser_test.go

## Test: RejectLongerCloses ends the group on a longer run
**Purpose:** R309, R310 — a longer run inside is rejected, not content, and the context reports both halves
**Input:** the run group with `RejectLongerCloses`, over a four-run holding a one-run and a two-run, and over a two-run span holding a three-run
**Expected:** the shorter runs are content; the three-run is a `Closer` paired with nothing and ends the span, the following two-run opens afresh; `Unpaired` lists the three-run alone and `Unclosed` lists the ended span's opener and the trailing one
**Refs:** crc-BracketGroup.md, crc-BracketContext.md, seq-parse.md#1.4.1
**Code:** sdom/parser_test.go
**Fire alarm:** ignore the flag — treat the longer run as content. Red: the three-run is inside one text node and nothing is unpaired.
**Inject:** sdom/bracket_parser.go:BracketParser.otherRun
**Pulled:** 2026-09-05 — rang: this test, the emphasis test and the carve's closes-nothing test — the three-run became content and nothing was unpaired.
