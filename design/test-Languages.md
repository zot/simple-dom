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
**Pulled:** 2026-09-05 — re-pulled after backtick-runs Item 1 touched the site; rang: this test, the recognition count and the comment-kind test, exactly as before; the table's `Close` is a string now and the ordering it protects is unchanged. Previously 2026-09-03 — re-pulled after `CommentStyle` landed on the table and rang in three tests, the new kind test among them (`pascal: opener "(" parses as kind …, want "comment"`). Previously 2026-09-01 — re-pulled after `LangPascal`'s comment groups were marked with a `Kind`, and rang again, on this test and `TestRecognitionCountPerLanguage`. **The prediction held exactly:** every failure was a recognition assertion — `(*` recognized 0 times and `(` twice — and the round-trip stayed green throughout. Marking the groups moved no property. Originally 2026-08-30 — rang, but **only on this test**, out of 56 — and that
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

## Test: every comment style constructs a comment of its own kind
**Purpose:** R207, R208, R209 — the agreement between how a language writes a comment and how it recognizes one, guarded here rather than at runtime
**Input:** for every shipped `BracketLang` (Python's through its `IndentLang`) with a non-empty `Comment.Prefix`: parse `Prefix + "x" + Suffix`
**Expected:** the first node is an `*Opener` whose group, resolved through `GroupFor`, has `Kind == Comment.Kind`; the interior is the single text `x`; a table with an empty `Prefix` is skipped and counted
**Refs:** crc-BracketLang.md
**Code:** sdom/lang_test.go
**Alarm:** 2
**Fire alarm:** set Lua's `Prefix` to `--[[ ` with `Suffix` `\n` — the opener is the block form, whose closer is `]]`, so the constructed comment does not close and the kind check still passes; make the test also require `ctx.Closer(opener)` to render `Suffix`, and then this injection is red.
**Inject:** sdom/lang.go:LangLua
**Pulled:** 2026-09-05 — re-pulled after backtick-runs Item 1 touched the site; rang: `lua: the constructed comment never closes`, only that case. Previously 2026-09-03 — rang: `lua: the constructed comment never closes`, only that case; the test already asserted the closer, so the injection's own caveat did not apply.

## Test: every shipped table constructs
**Purpose:** R296 — the construction check runs over every shipped `BracketLang` so a consumer never meets the panic
**Input:** each shipped table in `sdom`, Python's through its `IndentLang`; `LangMarkdown` is constructed by every markdown test and needs no case of its own
**Expected:** `NewBracketParser` returns without panicking for each
**Refs:** crc-BracketLang.md
**Code:** sdom/lang_test.go
