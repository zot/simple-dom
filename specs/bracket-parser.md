# The bracket parser

A **table-driven** parser that turns a source into a flat, document-order stream
of `sdom` nodes. Adding a language means adding a table entry, not code.

A string is not a special case: it is a bracket group with parsing turned off.
Neither is a comment. That is why interpolation, word brackets, shell
`if`/`then`/`fi` and `/* … */` all fall out of one mechanism.

## The table

```go
// BracketLang is the bracket table for one language.
type BracketLang struct {
    Brackets []BracketGroup
}

// BracketGroup is one set of matching markers — code, string, or comment.
type BracketGroup struct {
    Open       []string // openers: {"{"}, {"if","while"}, {"\""}, {"//"}
    Separators []string // mid-group markers: {"else","elif","then"}
    Close      []string // closers: {"}"}, {"end","done","fi"}, {"\n"}
    Escape     string   // escape sequence inside the group; "" for none

    AllowedInner  []string // nil = code mode; non-nil (even empty) = parse-restricted
    AllowedParent []string // nil = anywhere; non-nil = only inside these openers

    Kind string // an uninterpreted label for a layer above; never read here
}
```

**There is no comment configuration.** A line comment is a group opening `//` and
closing `\n`; a block comment opens `/*` and closes `*/`. Both are
parse-restricted, which is exactly what makes a comment non-nesting and its
interior literal. A language whose block comments *do* nest says so by listing its
own opener in `AllowedInner`.

**`Kind` does not change that, because this parser never reads it.** It is an
uninterpreted string carried through for a layer above — indent scope needs to know
which groups are transparent to the level, and no property of a group's *shape*
answers that, since a comment and a string are the same shape with different markers.
A label the parser stores and ignores is not a mode: the parsing rules below are
identical whatever it says, and a language that labels nothing parses the same.

### The two mode fields

**`AllowedInner` decides what is recognized inside the group.**

- **nil — code mode.** Full parsing: every other group's openers are recognized.
- **non-nil, including empty — parse-restricted.** Only three things are
  recognized: this group's `Close`, its `Escape`, and the openers listed. Every
  other byte is literal. An empty list is pure raw mode; a non-empty one names the
  escape hatches back into code.

**`AllowedParent` is its dual, and it is not optional.** With a flat table, code
mode recognizes every group's openers — so without it `${` fires at top level,
where it is really a `$` followed by a `{`. A group naming allowed parents is
recognized only inside them.

**nil and an empty slice mean different things** in both fields. nil is "no
restriction"; empty is "a restriction with an empty list." Be deliberate.

### The table is extended by embedding, never by flags

`BracketLang` carries **no** indent parameters and no switch that turns
indentation on. A language that needs indent scope is described by a type that
**embeds** `BracketLang` and adds what it needs, and **the type is the flag**. A
brace language then carries no indent parameters at all, rather than meaningless
zeroed ones.

## The languages the package ships

Tables are **Go values**, exported from the package:

- **`LangGo`** — code brackets, both comment forms, a symmetric string with an
  escape, and a raw string without one.
- **`LangShell`** — word brackets with separators: `if`/`then`/`else`/`fi`,
  `while`/`do`/`done`.
- **`LangPascal`** — `begin`/`end`, the other word-bracket shape.
- **`LangJavaScript`** — the only one exercising `AllowedInner` and
  `AllowedParent` together, through `` `text ${expr} more` ``.
- **`LangTypeScript`** — the same brackets as JavaScript; types add none.
- **`LangLua`** — `function`/`do`/`if` … `end`, `repeat`/`until`, `[[ ]]` long
  strings and `--[[ ]]` block comments.
- **`LangPython`** — the first `IndentLang`, and the only table modelling string
  **prefixes**. See below.

**The set has two jobs, and it used to have one.** It still covers the mechanism:
between them every field of `BracketGroup` is live, so no mode is dead code and the
recognition count has languages that actually recognize something. It now also
**serves the languages mini-spec reads** — Go, TypeScript, JavaScript, Lua and
Shell, with Python following the indent parser. `LangPascal` earns its place under
the first job alone. A consumer needing a language outside the set still constructs
its own `BracketLang`.

### Python's string prefixes, and why only some are modelled

Python writes a string as an optional prefix, then one of four quote forms —
`"""`, `'''`, `"`, `'`. The prefixes are `r`, `b`, `u`, `f` and the legal pairs of
`f` or `b` with `r`, in either order and either case.

**Only the `f` forms get groups**, because only they change how the text parses: an
f-string interpolates, so it is parse-restricted **with `{` as its one escape hatch**
— the same shape as a JavaScript template literal. Everything else is a plain
restricted group whatever its prefix, so `r`, `b` and `u` need nothing: the prefix
letter falls through as text and the quote that follows opens the ordinary group. The
string still parses correctly; only the prefix sits outside the literal's node.

**Raw does not mean unescaped.** `r"\""` compiles and `r"\"` does not — a backslash
still escapes the closing quote even in a raw string, which is a lexical rule rather
than a semantic one. So every string group carries the same `Escape`, and raw strings
need no group of their own.

**The `f` groups come first, longest quote form first**, so `f"""` is matched before
`f"`. The plain forms follow under the ordering they already need, `"""` before
`"`. Groups are tried in order and openers are matched at the position the parse has
reached, so a plain `"""` never competes with `f"""`: at the `f` it cannot match at
all.

**Interpolation needs no group of its own.** `AllowedInner` names `{`, which is
already Python's code-brace group — so the inside of `{…}` is full code mode, and a
dict display within an interpolation parses like any other.

**All ten `f` spellings are listed rather than the six that are strictly necessary.**
`rf"` would parse correctly without one, because `r` falls through as text and `f"`
matches at the next byte; but the prefix would then straddle two nodes, and a literal
is one thing.

**They are code, not a config-file format**, and `sdom` ships no loader for one.
The decisive reason is that **`nil` and an empty slice are semantically distinct
in both mode fields** — code mode versus pure raw mode — and that distinction is
exactly what a document format loses: an absent key and an empty list are the same
thing to most readers of most formats. Go source keeps it honest at compile time.
Beyond that, a format would mean a parser, a validator and error reporting for
data that changes rarely and is written by developers.

A consumer that needs *per-project* language settings manages its own
configuration. Mini-spec already does this, in `.minispec/config.yaml`.

## How it is driven

The bracket parser is one implementation of the **`Parser`** interface — see
[the parser protocol](parser-protocol.md), which owns the entry point, the walk, and
the rule for when a position was not recognized. What belongs here is only what this
parser recognizes.

It **takes the loop** on an opener, recursing until the matching closer, which is
what makes a restricted group's exclusivity structural rather than a flag — and what
lets a layer registered in the outermost loop, such as indent scope, be offered no
position inside a group at all.

## Parsing rules

- **Word-boundary matching for alphanumeric markers**, so `do` does not fire
  inside `download` nor `fi` inside `file`. A marker whose first byte is a word
  character must not be preceded or followed by one.
- **Separators** are matched against the group currently open, between its opener
  and its closer.
- **An any-close fallback**: when nothing else matches, any code-mode group's
  closer is recognized, so a stray `}` lands as a bracket rather than derailing
  the parse.
- **The parse always consumes at least one byte**, so nothing stalls on input it
  does not understand.
- **A group left open at end of input closes there.** The bytes are already
  accounted for; nothing is dropped.

**Whitespace is not a node of its own.** It folds into text, so a text run is *everything
between two recognized markers* rather than a run of non-whitespace. A layer that needs
line or indent boundaries **contributes a parser to the same pass** and emits them where
they occur — see [the parser protocol](parser-protocol.md). It does not split text
afterwards: a boundary is per line rather than per declaration, and re-carving them one
mutation window at a time is quadratic in a way the pass is not.

## Output

**Flat and document-order**, always. The parser tracks nesting while it works and
the emitted stream carries none: an opener, everything between it and its closer,
and the closer are **siblings in the array**.

The kinds the parser adds to `sdom`:

- **`Opener`**, **`Closer`**, **`Separator`** — marker nodes.
- Everything else is `Text`.

Three kinds rather than one with a role, because a symmetric group's `"` is
byte-identical opening and closing, so a role cannot be derived from the bytes. As
separate types the discrimination is the type assertion every `Equals` already
does, and nothing extra is stored or compared.

**A marker holds the bytes it matched and no pointer to its group.** Holding the
group would make `Equals` compare pointers, so two documents parsed with
independently constructed languages would never be equal and the structural
round-trip would fail on every bracketed document. The active group comes from the
parse context.

## The context, and the links it owns

Each schema has its own parse context, a concrete type rather than an interface.
The parser's carries the language during the parse, and **outlives** it to
carry the pairing links:

- an opener knows its **closer**, its **separators**, and its **enclosing opener**
- a closer knows its **opener**
- a separator knows its **opener**
- any other node knows its **enclosing opener**

**The separator direction completes an asymmetry, and it is a contract matter.** A
separator could always be traced back to its opener, because it is enclosed by one;
nothing could ask an opener which separators belong to it without scanning forward.
A library answers the questions its structure makes meaningful, and `for x in a b;
do … done` makes that one meaningful whether or not a consumer is asking yet.

These links are owned by the context, **not by `Doc`** — not every document has
brackets, and a markdown DOM would carry dead maps forever. They are a derived
index: the context stamps itself with the document's structural generation and
rebuilds when the stamp is stale.

**How they are stored is not part of this contract.** Whether the context keeps one
map or several is its own business; what it owes is the answers above.

**The accessors are typed.** `Opener(closer) *Opener` and `Closer(opener) *Closer`
return the marker kinds, not `Node`, so a consumer never asserts a type the context
already knew. Both return nil for an unmatched marker.

**And the context answers for a group's text**, mirroring `innerHTML` / `outerHTML`:

```go
// InnerText returns the bytes between a group's opener and closer.
func (bc *BracketContext) InnerText(n Node) string
// OuterText returns the bytes from a group's opener through its closer.
func (bc *BracketContext) OuterText(n Node) string
```

`n` names the group by being its opener or its closer. A group left open at end of
input runs to the end of the source. These exist because a reader that has found a
comment group wants its interior as one string, and slicing it from the node
locations by hand is the kind of thing a library owes rather than each consumer.

**Accessors that return a slice return the context's own.** `Separators` and
`DeclarationNames` hand back the stored slice, and a consumer does not write through
it. This is documented rather than guarded: every consumer of `sdom` builds a
document, reads or mutates it, and discards it within one operation, so there is no
lifetime in which aliasing becomes a hazard, and a copy per call would buy nothing.

**The index is checkable rather than merely believed.** Every link is derivable
from the flat array alone, so a consumer walking it with its own stack reaches the
same answers — the index is a convenience over structure the array already carries,
never a fact only the index holds. That is also why the parse records nothing: one
derivation, on demand, and nothing to fall out of step with.

