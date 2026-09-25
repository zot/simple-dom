# Carve: a closer that must match its opener

> **DRAFT (2026-09-25).** Opened the morning mini-spec asked for two things it needs to read
> code files through sdom with language tables loaded from a project's config. Scope and
> design were settled the same day; Items 4 and 5 joined from what the design turned up.

mini-spec is moving its code-file traceability reading onto sdom (its items #93/#94). A
project will define languages in `.minispec/config.toml`, the definition mirroring
`BracketLang` and `IndentLang` field for field. That changes who writes a table. Until now
every table was ours, compiled in, and `NewBracketParser` panicking on a malformed one was
right: a library invariant, kept from consumers by a test over every shipped table. A table
read from a user's file is caller input, and the caller needs the error, not the panic.

The second need is expressive. `CloseIsOpen` says a group closes on the text that opened it,
which is how a markdown backtick run closes on a run of the same length. Several string
syntaxes close on a closer that is *different* text from the opener but must agree with it
on one part: C++ `R"delim( … )delim"`, Lua `[==[ … ]==]`, Rust `r#"…"#`. None of them can be
written as a table today, so mini-spec's built-in C++ table leaves raw strings out, and our
own `LangLua` handles only level 0.

**Provenance.** Both parts are mini-spec's, 2026-09-25:
`~/work/mini-spec/requests/sdom-check-and-matching-close-groups.md`. Our acknowledgement is
in `requests/`.
@ark-request-ref: /home/deck/work/mini-spec/requests/sdom-check-and-matching-close-groups.md
@ark-response-ref: requests/RESP-sdom-check-and-matching-close-groups.md

**What the report got wrong, measured 2026-09-25.** The request says a `]]` inside a level-1
Lua string ends it early today. It does not, because there is no string: `LangLua` does not
recognize `[=[` at all. Probed on today's table:

- `x = [=[ a ]] b ]=] y` reads as two `[` index openers with the text `=` between them; the
  inner `]]` closes those two, and `]=]` leaves two unpaired `]` closers.
- `--[=[ a ]] b ]=] y` reads as a `--` line comment, so every later line of a multi-line
  level-1 comment is parsed as code.
- `[[ a ]=] b ]] y`, level 0, is already right: one string, closing at `]]`.

The consequence is worse than an early close. A keyword such as `end` inside a level-n string
or comment closes a real group. The repair is the one proposed.

## Status

- [x] ~~**Item 1 — `Check() error`, exported.**~~ **LANDED (2026-09-25 — `#40`.)**
- [ ] **Item 2 — a closer whose named groups must equal the opener's.** **OPEN (#43.)**
- [ ] **Item 3 — Lua long brackets at every level.** **OPEN (#44.)**
- [x] ~~**Item 4 — `rebuild()` ends a group wherever the parse does (R309).**~~ **LANDED (2026-09-25 — `#41`.)**
- [ ] **Item 5 — a closer that matches an enclosing group closes it, ending the groups in between.** **OPEN (#42.)**

## Decisions

**DECIDED (inherited from `carves/done/sdomification.md`, Bill, 2026-09-03): committed tests
rely only on code in this repository.** sdom ships no C++ or Rust table, and this carve does
not add them; the C++ raw-string and Rust raw-string cases are synthetic tables in the tests.
`LangLua` is the one shipped table that changes.

**DECIDED (Bill, 2026-09-25): no flag; named groups are the declaration.** A group with both
`OpenRegex` and `CloseRegex` closes only where the named groups the two patterns capture are
equal. Patterns naming no groups make a plain pattern closer. `Check` refuses a table whose
two patterns name different sets of groups, so a closer that misspells or forgets a name is
an error at load, not a group that never closes. `CloseRegex` and `Close` are exclusive.
mini-spec's `CloseGroupsMatchOpen` is not added.

**DECIDED (Bill, 2026-09-25): a closer that disagrees with its opener is text, unless it
closes a parent.** When `CloseRegex` matches but its groups disagree with the current
opener's, the parse tests it against the enclosing openers. If it would close one of them,
that is an error; otherwise it is content, and the group stays open. That is what the C++
case needs: `R"x( a )" b )x"` reads `)"` as string text. It differs from
`RejectLongerCloses` (R309), so the new rule is not a variant of that field.

Two parts of this are not yet settled:

**DECIDED (Bill, 2026-09-25): the stack of enclosing openers lives on `BracketParser`.**
It holds (group, opened text) for each open group, and it is scratch that is empty when the
parse ends, so the parse still records nothing. The parent test re-matches a parent's
opened text for its captures, as the current group's is re-matched.

**DECIDED (Bill, 2026-09-25): a closer that matches a parent ends the groups in between**,
as `RejectLongerCloses` ends its group. The ended groups are the error: `Unclosed()` lists
them.

**DECIDED (Bill, 2026-09-25): the parent-matching closer pairs with the parent.**
`{% block a %}{% block b %}{% endblock a %}` closes `a`, and `b` is unclosed.

**DECIDED (Bill, 2026-09-25): the parent rule covers every closer, not only `CloseRegex`
groups.** "Painful but more correct." It changes recovery in every shipped language: `( { )`
today emits `)` stray and leaves `{` open; after this carve `)` closes `(` and `{` is
unclosed, which is how compilers report it. A closer that matches no enclosing group is
still stray. The any-close fallback's wording (`specs/bracket-parser.md`, the bullet
beginning "An any-close fallback") and its tests (`parser_test.go` near line 120,
`context_test.go` near lines 97, 116 and 375) change with it. The rule is Item 5.

**DECIDED (Bill, 2026-09-25, on Daneel's recommendation): the groups in between end as they
would at end of input.** A `DemoteUnclosed` group demotes, and any other is unclosed, so
"ended without its closer" has one meaning. `parser_test.go` near line 437 already has a
demoting `<` beside a stray `>`.

**MEASURED (2026-09-25): `rebuild()` does not copy R309, and must copy both rules.** With a
rejecting backtick run and `(`…`)`, the parser reads a parenthesis holding a two-backtick
span that meets a three-backtick run as the span ended by the rejected run and `(` closed by
`)`. `rebuild()` pops only when `closes(top, closer)` holds, so the ended span stays on its
stack: `)` is reported unpaired, and both `(` and the span are unclosed. The R309 test
passes only because nothing follows its rejected run. Every way the parser ends a group must
be derivable in `rebuild()` from the node text. For a rejected run, the top group has
`RejectLongerCloses` and the closer is a longer match of its pattern, so `rebuild()` pops
without pairing. For a parent closer, it pops the groups in between and pairs with the
parent. The R309 repair is a defect in landed code and gets its own item.

The parent test applies in code mode only. In restricted mode, only the group's own closer
is recognized, and a parent's closer inside a string is literal. All three motivating cases
are restricted, so the test matters only for a code-mode group that nests, such as a
template's `{% block a %} … {% endblock a %}`.

**DECIDED (mini-spec amendment, 2026-09-25, Bill): `GroupFor` stays opener-only.** Closer
text is ambiguous (`LangLua`'s `function`/`for`/`while` group and its `if` group both close
on `end`), and a consumer holding a closer already has `ctx.Opener(closer)`, whose text
`GroupFor` resolves.

**DECIDED (Bill, 2026-09-25): `CloseIsOpen` stays its own field.** It is safer against typos:
the pattern is written once, not twice. `RejectLongerCloses` and the markdown base are
untouched.

## Item 1

Export `func (l *BracketLang) Check() error`, returning exactly the error `check()` returns
today. `IndentLang` embeds `BracketLang`, and `NewIndentParser` adds no checks of its own
(it constructs `NewBracketParser(&lang.BracketLang)`), so the one method is promoted to both.
`NewBracketParser` keeps panicking. The panic's documentation changes from "a library
invariant" to "call `Check` first for a table that is caller input".

Tests: every error `check()` can produce comes back from `Check()` with the same text, a
shipped table checks clean, and an `IndentLang` checks through the promoted method.

## Item 2

A group whose closer is a pattern that must agree with its opener on named groups:

- Nothing is stored at open (mini-spec amendment). Inside the group, the parser already
  carries the `opened` text; re-matching it against `OpenRegex` yields the captures.
- Inside the group, `CloseRegex` is tried anchored at each position. It closes when every
  named group agrees; otherwise it is content, unless Item 5's parent rule finds it closes
  an enclosing group.
- `BracketContext.closes(opener, closer)` compares the groups too (mini-spec amendment).
  `rebuild()` is the only derivation of pairing, and `closes()` decides from the two nodes'
  text, so it re-matches the opener's text against `OpenRegex`, the closer's against
  `CloseRegex`, and pairs only when the named groups agree. That covers a freshly parsed
  document and an edited one alike, and keeps an edit from pairing `R"x(` with `)y"`.
- `Check` refuses `Close` together with `CloseRegex`, and two patterns whose named-group
  sets differ. A literal `Open` names no groups, so a `CloseRegex` under it may name none.

Tests, over synthetic tables: round-trip and DOM equality for C++ raw strings, with an empty
delimiter, a non-empty one, the `u8`/`u`/`U`/`L` prefixes, and a mismatched-delimiter closer
inside the string (`R"x( a )" b )x"` closes only at `)x"`); Lua long brackets at levels 0, 1
and 2, with a closer of another level inside; and Rust `r#"…"#` with one and two hashes.

## Item 3

`LangLua`'s `[[` and `--[[` groups become pattern groups over every level:
`\[(?P<eq>=*)\[` closing on `\](?P<eq>=*)\]`, and the comment form with `--` in front. Order
still matters: the comment group precedes `--`, and the string group precedes `[`. The
description at `specs/bracket-parser.md:183` changes with it.

Tests: the three probes above, as fixtures with hand-written expectations, and a keyword
such as `end` inside a level-1 string and comment that closes nothing.

## Item 4

`rebuild()` pops the top opener, unpaired, at a closer that R309 rejected: the top group has
`RejectLongerCloses`, the closer is a match of its pattern, and it is longer than the
opener's text. Item 5's parent rule adds the second case. A test over the parse and the
derivation keeps the two in agreement.

Tests: a parenthesis holding a two-backtick span that meets a three-backtick run, over a
rejecting run table, pairs `(` with `)` and lists only the span as unclosed, and the same
shape in the markdown base.

## Item 5

In code mode, a closer that does not close the current group but closes one on the parser's
stack of enclosing groups ends every group in between and pairs with that group. This applies
to every closer: literal `Close`, and `CloseRegex` with its named groups compared against
that parent's opened text. A closer that closes no enclosing group is stray, as today.
Restricted mode is untouched: inside a string or comment, only the group's own closer is
recognized.

`rebuild()` copies the rule from the node text (Item 4's principle): at a closer that does not
close the top of its stack, it searches down the stack. If some opener there `closes()` it,
the openers above are popped unpaired and the closer pairs with that one. A group that
closes on the same text as another (`LangLua`'s `end`) resolves to the nearest, in the parse
and in `rebuild()` alike.

Order: after Item 4, whose repair is the same principle, and before Item 2, whose
disagreeing-closer rule depends on it.

Tests: `( { )` in a synthetic table and in each shipped code table; a Lua `end` closing an
`if` through an unclosed `(`; the template case for `CloseRegex`; the stray-closer tests
updated for a closer that now pairs; and a demoting group between a closer and its parent,
which demotes.
