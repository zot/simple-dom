# Test Design: the shipped language tables
**Source:** crc-BracketLang.md

## Test: every field of BracketGroup is live somewhere
**Purpose:** R121 — the tables cover the mechanism as well as serving the
languages mini-spec reads, and this is the assertion that keeps the first half
true as they change
**Input:** the six shipped tables
**Expected:** across them, each of `Open`, `Separators`, `Close`, `Escape`,
`AllowedInner` (both nil and non-nil) and `AllowedParent` is exercised by at
least one group. A field live in no table is dead code
**Refs:** crc-BracketLang.md
**Code:** sdom/lang_test.go

## Test: Go
**Purpose:** R59, R62 — comments and strings as groups, not special cases
**Input:** a Go fixture with `//` and `/* */` comments, a `"..."` string with an
escaped quote, and a backtick raw string containing a quote and a brace
**Expected:** both comment forms parse as restricted groups; the escaped quote does
not close the string; nothing inside the raw string is recognized
**Refs:** crc-BracketLang.md
**Code:** sdom/lang_test.go

## Test: Shell
**Purpose:** R72 — word brackets with separators, which nothing else exercises
**Input:** `if a; then b; else c; fi` and `while x; do y; done`
**Expected:** `if`/`fi` and `while`/`done` pair; `then` and `else` are separators
of the `if` group and not of the `while` group
**Refs:** crc-BracketLang.md
**Code:** sdom/lang_test.go

## Test: Pascal
**Purpose:** R59 — the other word-bracket shape
**Input:** nested `begin` … `end` with a `{ comment }` and a `'string'`
**Expected:** the word brackets nest correctly, and Pascal's brace-comment does
not act as a code bracket
**Refs:** crc-BracketLang.md
**Code:** sdom/lang_test.go
**Alarm:** 1
**Fire alarm:** Move the `(*` group after the bare `(` group in `LangPascal`. Red: `(*` never fires, because `(` matches first and wins — Pascal's block comments stop being recognized and their interiors parse as code. The round-trip stays green and no byte moves; only the recognition assertions see it. This is what the ordering comment in the table is protecting.
**Inject:** sdom/lang.go:LangPascal
**Pulled:** 2026-08-30 — rang, but **only on this test**, out of 56 — and that
corrected the alarm's own prediction. `TestRecognitionCountPerLanguage` stayed
GREEN, because the reorder swapped `(*` for a bare `(` and the *kind* counts did
not move: five openers before, five after. The count was measuring how many
rather than what. It has since been rewritten to tally **per marker**, and now
reports `0 of "O(*", expected 1`; re-verified against this same injection on
2026-08-30. Both corpus round-trips stayed green throughout.

## Test: JavaScript
**Purpose:** R64, R66 — the only table exercising both mode fields together
**Input:** `` `a ${b + `c ${d}`} e` `` — an interpolation containing a nested
template containing another interpolation
**Expected:** the modes alternate correctly to full depth, and `${` at top level
is not an interpolation opener
**Refs:** crc-BracketLang.md
**Code:** sdom/lang_test.go
