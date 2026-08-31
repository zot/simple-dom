# Test Design: declaration schemas
**Source:** crc-DeclSchema.md, crc-GoSchema.md, crc-LuaSchema.md, crc-ShellSchema.md

## Test: a declaration under a comment is found
**Purpose:** R134, R145 — the shape that is the norm, not the edge
**Input:** `// CRC: crc-Doc.md | R1\nfunc Index(k string) int {`, where the comment's
closing newline is a `Closer` and the following text begins `func` with no
separator in front of it
**Expected:** one declaration, named `Index`
**Refs:** crc-DeclSchema.md, seq-declare.md#1.2.2
**Code:** sdom/schema/declaration_test.go
**Alarm:** 1
**Fire alarm:** Test the statement start against the immediately preceding sibling
without skipping comment groups. Red: **nearly every declaration in `sdom/`
disappears** — 183 of them sit directly under a `// CRC:` comment. The count is what
sees it; nothing about bytes or structure changes.
**Inject:** sdom/schema/schema.go:precededBySeparator
**Pulled:** 2026-08-31 — rang, and reproduced the real defect exactly. The injection
was this function's *own documented alternative* — skip fully backward over comments
and whitespace as `skipForward` does, then look at what you land on — rather than a
strawman. Red across the corpus, per file: **16 declarations found where column-0
keywords say 241**. That is the defect this code was written to fix, restored and
re-caught. ~~183 of them~~ — the prose said 183, counting only the anchored ones;
**225** vanish.

## Test: a comment before the keyword does not hide a declaration
**Purpose:** R145 — the backward skip, in its other position
**Input:** `/* c */ func Foo() {`
**Expected:** one declaration, named `Foo`
**Refs:** crc-DeclSchema.md, seq-declare.md#1.2.2
**Code:** sdom/schema/declaration_test.go
**Alarm:** 2
**Fire alarm:** Skip only whitespace backward, not comment groups. Red: the
preceding sibling is `*/`, no separator is found, and a real declaration is
rejected. Silent otherwise — the document parses and renders identically.
**Inject:** sdom/schema/schema.go:precededBySeparator
**Pulled:** 2026-08-31 — rang, alone and precisely: `/* c */ func Foo() {` yielded
nothing where `func[Foo]` was wanted. Nothing else objected, the document parsing and
rendering identically, as predicted.

## Test: comments and whitespace anywhere in a signature
**Purpose:** R147 — the skip runs before every decision, not once
**Input:** `func /* a */ (s *S) /* b */ Index /* c */ (x int) {` and
`func /* a */  /* b */\n/* c */ foo /* d */ (x) /* e */ {`
**Expected:** names `Index` and `foo`; the receiver group is jumped, not read as a
name
**Refs:** crc-DeclSchema.md, seq-declare.md#1.4
**Code:** sdom/schema/declaration_test.go
**Alarm:** 3
**Fire alarm:** Skip once after the keyword and then read the next node. Red: the
first case yields the receiver group's opener where a name was expected, and the
second yields nothing. A single-comment fixture would pass — which is why both
inputs are here.
**Inject:** sdom/schema/golang.go:goNameAfter
**Pulled:** 2026-08-31 — rang on **both** inputs, which is why both are here: skipping
once after the keyword yielded nothing for the method form *and* nothing for the
comments-across-a-line form. A single-comment fixture would have passed.

## Test: a declaration spanning lines is found
**Purpose:** R149 — skipped whitespace may contain separators
**Input:** Go's `func\nfoo(x int) {`, and Lua's `function\n\nfoo\n (x)\nreturn x\nend`
**Expected:** both yield one declaration named `foo`
**Refs:** crc-DeclSchema.md, seq-declare.md#1.4.3
**Code:** sdom/schema/declaration_test.go
**Alarm:** 4
**Fire alarm:** Stop the forward walk at a statement separator. Red: both fail —
which is the point of pairing them. The Go form is legal and vets clean; the Lua
form runs and returns a value. A rule that stopped at a separator would be wrong
for both languages at once.
**Inject:** sdom/schema/golang.go:goNameAfter
**Pulled:** 2026-08-31 — rang, **after the site was corrected, and the correction is
the finding.** The recorded site was `schema.go:isSpace`, and a delegate reported that
`isSpace` is invoked **zero times** while this test's Go case runs — so no change
confined to it could ever have reached this property. It was right to say so rather
than dress up a near miss.

Go's `func` ⏎ `foo(x int)` resolves **inside a single text node**, so the newline is
crossed by `goNameAfter`'s in-node prologue rather than by `skipForward`. Injecting
there — reject the prologue when the gap contains a newline — turns this test red on
its Go case, as designed.

*What the `isSpace` injection did ring on is worth keeping:* it failed
`TestCommentsAnywhereInASignature`'s second input, which is where `skipForward`
actually steps over newline-bearing whitespace. So `R149` has two halves living in two
places, and **this test only ever covered one of them** — weaker than it was designed
to be, and nothing but an injection would have said so.

## Test: Lua announces two ways, and one is not text
**Purpose:** R131, R135 — a keyword that is a bracket marker, and the alternation
**Input:** `local function f(a) return a end`, `function M.f(a) end`, `x = 1`
**Expected:** three declarations — `f`, `M.f`, `x`; for the `function` forms the
name is found **inside** the group the keyword opened
**Refs:** crc-LuaSchema.md, seq-declare.md#1.3
**Code:** sdom/schema/declaration_test.go
**Alarm:** 5
**Fire alarm:** Match only over text nodes, dropping the node-kind test. Red: every
`function` declaration vanishes, because `function` is an `Opener` and no text
pattern can see it. `x = 1` still passes, so a Go-shaped fixture would report
success.
**Inject:** sdom/schema/lua.go:luaNext
**Pulled:** 2026-08-31 — rang. Dropping the node-kind branch made every `function`
declaration vanish, failing both the Lua span-lines case and the opener case, while
`x = 1` kept passing — so a Go-shaped fixture would indeed have reported success.

## Test: shell assignment is strict and a shell function is structural
**Purpose:** R136 — the two schemas that look alike and are not
**Input:** `NAME=value`, `NAME = value`, `foo() {\n  echo hi\n}`
**Expected:** `NAME=value` is a declaration; `NAME = value` is **not**, being a
command invocation; `foo` is a declaration recognized from nodes rather than a
pattern
**Refs:** crc-ShellSchema.md
**Code:** sdom/schema/declaration_test.go
**Alarm:** 6
**Fire alarm:** Give shell Lua's lenient `NAME[ \t]*=`. Red: `NAME = value` is
reported as a declaration. Nothing else objects, and the two patterns differ by two
characters — which is the whole reason this case is written down.
**Inject:** sdom/schema/shell.go:shellAssignRe
**Pulled:** 2026-08-31 — rang, alone: `NAME = value` was reported as `[NAME]` where
nothing was wanted. Two characters of regex separate a declaration from a command
invocation, and only this assertion sees it.

## Test: a Go func's name reaches its paren without a newline
**Purpose:** R151 — the one place Go's schema is stricter than the general walk,
and the rule that is easy to over-tighten into abutment
**Input:** legal — `func foo(x int) {`, `func foo (x int) {`,
`func bar /* c */ (x int) {`; illegal — `func\n\nfoo\n (x int)\n{` and
`func qux /*\n*/ (x int) {`
**Expected:** the three legal forms are declarations; the two illegal ones are not,
matching what the Go compiler accepts in each case
**Refs:** crc-GoSchema.md, seq-declare.md#1.4.4
**Code:** sdom/schema/declaration_test.go
**Alarm:** 7
**Fire alarm:** Require the name to **abut** the paren instead of merely reaching
it without a newline. Red: `func foo (x int) {` and the comment form are rejected,
though Go accepts both — and this is not hypothetical, it is what the first
implementation did. Injecting the opposite — dropping the check entirely — turns
the two illegal forms green instead, so both directions are worth trying.
**Inject:** sdom/schema/golang.go:goSameLine
**Pulled:** 2026-08-31 — rang in **both** directions, as this alarm asks.

*Tightening to abutment* rejected `func foo (x int) {` and `func bar /* c */ (x int)
{`, both legal Go — and took `TestCommentsAnywhereInASignature` and
`TestANameIsSlicedOutOfTheMiddle` down with them. That is not a hypothetical
regression: it is what the first implementation did, and this is the injection that
caught it.

*Dropping the check* accepted `func` ⏎⏎ `baz` ⏎ ` (x int)` and `func qux /*` ⏎ `*/ (x
int) {`, both of which Go rejects. Two directions, two different failures, one
correct rule between them.

## Test: a grouped declaration yields every name
**Purpose:** R139, R140 — one group, many names, and the capture that loses one
**Input:** `const (\n\tA = 1\n\tB, C = 2, 3\n)`
**Expected:** four declarations of `const`'s names: `A`, `B`, `C` — three names, one
keyword, linked one-to-many
**Refs:** crc-GoSchema.md, seq-declare.md#2.4
**Code:** sdom/schema/declaration_test.go
**Alarm:** 8
**Fire alarm:** Capture the name list with a repeated group and read its submatch.
Red: `C` is missing — a repeated capture reports only its last iteration, so
`B, C, D` yields `B` and `D`. A one-name-per-line fixture would pass.
**Inject:** sdom/schema/golang.go:goCarveGroup
**Pulled:** 2026-08-31 — rang: `const[A C]` where `const[A B C]` was wanted, and
`var[E]` where `var[D E]` was. ~~`B, C, D` yields `B` and `D`~~ — the prose predicted
the wrong casualty; on this fixture the repeated group keeps its **last** iteration and
`B` is what disappears. The mechanism is confirmed exactly; only which name it eats
depends on the input, which is itself the argument for a multi-name fixture.
