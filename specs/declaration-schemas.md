# Declaration schemas

A **schema** says how one language announces a declaration. The machinery it
drives — the two node kinds, `Doc.Replace`, and the link map on `BracketContext` —
is `sdom`'s and is specified in [declarations.md](declarations.md). Nothing here
is.

**The schema does the linking.** `sdom` provides the map; only a schema can fill it
in, because filling it in means knowing what a declaration looks like.

Bundled: **Go, TypeScript, JavaScript, Lua and Shell**, with **Python** following
the indent parser. A consumer needing another language writes its own; these are
here because they are the languages mini-spec reads.

## Recognition is the schema's own business

**There is no shared recognition rule, and there should not be one.** A schema
parses its own language's declarations, using whatever the parse gives it. What
follows is what the bundled schemas do — patterns to copy, not a contract the
machinery imposes.

The three shapes among them, which differ more than they look:

- **A keyword in text.** Go, TypeScript and JavaScript announce a declaration with
  a keyword at the start of a statement, in an ordinary top-level text node, with
  `\n` and `;` as separators.
- **A keyword that is a bracket marker.** Lua's `function` *opens a group closing
  with `end`*, so it is an `Opener` node and no text pattern will ever see it — and
  the name it introduces sits **inside** that group rather than at top level.
- **A shape rather than a word.** Shell's `foo() { … }` has no keyword at all: the
  name is text, followed by an empty paren group, followed by a brace. Recognizing
  it means looking at nodes, not matching a string.

Go hides all of this, because none of its declaration keywords are brackets. A
schema written only against Go will not generalize, and is not meant to.

**Commented-out code cannot match, and this costs nothing.** A comment is a
bracket group, so its interior is not a top-level text node.

**Statement separation is partly structural.** A comment closes on `\n`, so that
newline is a `Closer` node rather than a byte in the following text — and a
declaration written under a comment therefore begins a text node with no separator
in front of it. A match at the start of a top-level text node begins a statement
when the **preceding top-level content ends with a separator**, and *preceding
content* means what is left after skipping backward. Within the text, the pattern's
own separator branch decides.

This is not an edge case: in `sdom` itself, nearly every one of the 183
declarations carrying a `// CRC:` comment sits in exactly that position.

### Comments and whitespace are skipped, in both directions

**A comment can appear anywhere a space can**, so a schema walking the node array
must step over **whole comment groups and whitespace-only text**, in any number and
in any interleaving. This applies **backward**, when testing whether a match begins
a statement, and **forward**, when looking for the name.

Both directions have a case that fails without it:

```
/* c */ func Foo() {
[ Opener "/*", Text " c ", Closer "*/", Text " func Foo", … ]
```

The pattern matches `func` happily. The backward test then finds `*/` as the
preceding sibling, sees no separator, and **rejects a real declaration** — unless
the comment group is skipped, at which point the document start is what precedes it.

```
func /* hi */ Index(a) int {
[ Text "func ", Opener "/*", Text " hi ", Closer "*/", Text " Index", … ]
```

The name is three nodes past the keyword, and `func /* a */ /* b */ Index` puts it
arbitrarily further.

**Skipping a comment group backward is one hop**: a `Closer` names its `Opener`,
which is a link the parse already owns.

**How a schema recognizes a comment group is the schema's business**, as recognition
always is. A comment and a string are both *parse-restricted* groups and the table
does not distinguish them — deliberately, since there is no comment configuration.
A schema knows its own comment markers (`//` and `/*`, or `--`, or `#`) and matches
the opener's text against them.

## The patterns the bundled schemas use

**By keyword in text** — Go, TypeScript, JavaScript, and Lua's `local`:

```go
`(?:^|[\n;])[ \t]*(?P<kw>func|var|type|const)\b`
```

**By shape, with no keyword at all** — Lua's global assignment and Shell's:

```go
lua   `(?:^|[\n;])[ \t]*(?P<name>[\w.:]+)[ \t]*=`   // M.f = … and M:m = … count
shell `(?:^|[\n;])[ \t]*(?P<name>\w+)=`             // no whitespace, see below
```

**Shell and Lua do not share a rule, and the difference is not cosmetic.** Shell
assignment forbids whitespace around `=`; `NAME = value` is a *command
invocation*. A lenient `NAME[ \t]*=` matches it and would report a command as a
declaration, so Shell takes the strict form and Lua the lenient one.

**A language may use both forms in one pattern**, as an alternation with a `kw`
branch and a `name` branch. The branch that did not participate yields the **zero
`Loc`**, which is how a schema tells them apart with no extra machinery — absence
is the zero value.

**Two regex constraints, both forced by RE2** rather than chosen. Go supports
neither lookbehind nor lookahead, so the separator is consumed by a non-capturing
group and the trailing boundary is `\b` — which is also the better rule, rejecting
`constant` while admitting a keyword followed by punctuation.

`import` is deliberately absent from every keyword set: an import is not a
declaration a tool anchors.

## From the keyword to the name

The name is found by walking **forward in document order** from whatever announced
the declaration. The walk is a small loop, and **the skip runs before every
decision in it, not once at the start**:

```
skip*                            comments and whitespace-only nodes
if the next node is an Opener:   it is a receiver group — jump to its Closer
skip*                            again
the next text with content:      the name is the identifier inside it
```

`func /* a */ (s *S) /* b */ Index /* c */ (x int) {` exercises every step: skip,
meet `(`, jump it, skip, read `Index`.

**Two different whitespace problems, and only one of them is skipping.** A
whitespace-only text node is stepped over. But the name arrives inside a node that
carries whitespace on **both sides** — `Text " Index "` — so it is *sliced out of
the middle*, splitting that node at both edges. Taking the node whole would put the
spaces inside the `DeclarationName`.

**Skipped whitespace may contain statement separators, and the walk crosses them.**

```
func /* a */  /* b */
/* c */ foo /* d */ (x) /* e */ {

… Closer "*/", Text "   \n", Opener "/*", …
                     ^^^^^^ whitespace-only, and a newline
```

A declaration may span lines, so the forward walk cannot stop at a separator. This
does **not** contradict the backward statement-start test: that asks *did a
statement begin here*, this asks *where is the name of the declaration already
announced*. Two questions, one of which happens to be about the same byte.

**The consequence, stated because nothing enforces it:** the forward walk has no
terminator. On valid input it cannot over-run, because a keyword is always followed
by its name; on truncated or malformed input it would walk to the end of the
document. A schema wanting a bound imposes its own.

**Newline-insensitivity is required, not merely tolerated.** The same layout is
illegal in one bundled language and legal in another:

```
func              function
                  
foo               foo
 (x int)           (x)
{                 return x
                  end
```

*Verified 2026-08-31.* Go rejects the left one — `expected '(', found newline`,
because a semicolon is inserted after the identifier `foo` at end of line — while
Lua runs the right one and returns its value. Even in Go the near variant
`func` ⏎ `foo(x int) {` is legal and vets clean, so a newline between a keyword and
its name is ordinary there too.

A shared rule that stopped the walk at a separator would therefore be **wrong for
both**: it would break Lua outright and break legal Go besides.

## How strict a walk is, is the schema's choice

`sdom` models text and does not validate it — and it does not require a schema to
be lax either. **Each schema does its own walk**, so each is as strict as its own
language warrants.

**Go's is strict exactly where semicolon insertion makes it so:** a `func`'s name
must reach its **opening parenthesis without crossing a newline**. That is what the
compiler enforces, and it lets the Go schema reject `func` ⏎⏎ `foo` ⏎ `(x)`, which
Go rejects too.

**Not abutment**, which is the stricter rule it is easy to mistake this for.
*Verified against the compiler:* `func foo (x int) {` is legal, `func bar /* c */
(x int) {` is legal, `func baz` ⏎ `(x int)` is not — and neither is
`func qux /*` ⏎ `*/ (x int)`, because Go treats a comment containing a newline as a
newline. So the test runs over the **source span** between the name and the paren,
where comment bytes are included by construction and the last case needs no rule of
its own. It applies to `func` alone.

**Lua's schema imposes no such rule**, because Lua has none: the same layout runs.

So the general walk described above is the *loosest* a schema may be, not what one
must do. What `sdom` guarantees is only that it will never refuse text on a
schema's behalf: a DOM that rejected malformed input would be useless for the
editing this library exists to support, and every other layer already keeps
provenance for text no compiler would accept. **Nothing in `sdom` is a syntax
checker**; whether a given schema is, is that schema's business.

The same walk covers the cases that look unrelated:

- **A plain Go declaration** — the name is in the next text with content, which is
  the same node as the keyword only when no comment intervenes.
- **A Go method** — the receiver group is the `Opener` the walk jumps.
- **A Lua function** — the name is a forward node *inside* the group the keyword
  opened, which the walk reaches without caring that a group was entered.
- **A keyword-less declaration** — the match already *is* the name, and no walk
  runs.

## Grouped declarations

`const`, `var` and `type` may take a group, and one group declares several names:

```go
const (
    A = 1
    B, C = 2, 3
)
```

The parse puts the group's whole interior in **one text node inside the parens**,
so names come from a second pass over it: an identifier list at the start of a line
within the group. Each name becomes its own `DeclarationName`; the text between
them stays as ordinary siblings.

**A name list is captured whole and split afterwards, never matched with a repeated
group.** A repeated capture reports only its last iteration — `B, C, D` yields `B`
and `D`, losing `C` — so the list is one group and the identifiers come out of it.
The same shape covers `A = 1`, `B, C = 2, 3` and `D, E int`.

## Scope

**Top-level text nodes.** Go's declarations are at bracket depth 0. A language
whose declarations nest — a Java method inside its class body — wants the same
scan at another depth: the same code against a different node set, with no `if` or
`for` following it in, those not being keywords in any table.

**Indent is not part of a declaration.** Bracket groups already pinpoint them, so
indentation is irrelevant to recognition in any bracket-parsed document. A tool
emitting a matching indent reads it from the preceding text; it is not a field and
gets no node. In an *indent-parsed* document indentation is the scope mechanism
itself, which is a separate concern.

**Which declarations *ought* to carry a comment is a policy, and it is not here.**
A schema makes them addressable; a reader decides which ones matter.

## The duplication is deliberate

Each schema carries its own scan-match-slice-link pass, and the repetition between
them is expected. Generalizing it into a reusable `sdom` tool waits until Go, an
indent language and Python have all been written, because three worked schemas is
when it is knowable what should generalize — and a tool guessed from one would be
built into `sdom`'s export surface, where withdrawing it costs every consumer.
