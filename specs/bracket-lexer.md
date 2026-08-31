# The bracket lexer

A **table-driven** scanner that turns a source into a flat, document-order stream
of `sdom` nodes. Adding a language means adding a table entry, not code.

A string is not a special case: it is a bracket group with scanning turned off.
Neither is a comment. That is why interpolation, word brackets, shell
`if`/`then`/`fi` and `/* … */` all fall out of one mechanism.

## The table

```go
// BracketLang is the lexical table for one language.
type BracketLang struct {
    Brackets []BracketGroup
}

// BracketGroup is one set of matching markers — code, string, or comment.
type BracketGroup struct {
    Open       []string // openers: {"{"}, {"if","while"}, {"\""}, {"//"}
    Separators []string // mid-group markers: {"else","elif","then"}
    Close      []string // closers: {"}"}, {"end","done","fi"}, {"\n"}
    Escape     string   // escape sequence inside the group; "" for none

    AllowedInner  []string // nil = code mode; non-nil (even empty) = scan-restricted
    AllowedParent []string // nil = anywhere; non-nil = only inside these openers
}
```

**There is no comment configuration.** A line comment is a group opening `//` and
closing `\n`; a block comment opens `/*` and closes `*/`. Both are
scan-restricted, which is exactly what makes a comment non-nesting and its
interior literal. A language whose block comments *do* nest says so by listing its
own opener in `AllowedInner`.

### The two mode fields

**`AllowedInner` decides what is recognized inside the group.**

- **nil — code mode.** Full scanning: every other group's openers are recognized.
- **non-nil, including empty — scan-restricted.** Only three things are
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

They are chosen to **cover the mechanism, not to serve consumers**: between them
every field of `BracketGroup` is live, so no mode is dead code and the recognition
count has languages that actually recognize something. A consumer needing another
language constructs its own `BracketLang`.

**They are code, not a config-file format**, and `sdom` ships no loader for one.
The decisive reason is that **`nil` and an empty slice are semantically distinct
in both mode fields** — code mode versus pure raw mode — and that distinction is
exactly what a document format loses: an absent key and an empty list are the same
thing to most readers of most formats. Go source keeps it honest at compile time.
Beyond that, a format would mean a parser, a validator and error reporting for
data that changes rarely and is written by developers.

A consumer that needs *per-project* language settings manages its own
configuration. Mini-spec already does this, in `.minispec/config.yaml`.

## Scanning rules

- **Word-boundary matching for alphanumeric markers**, so `do` does not fire
  inside `download` nor `fi` inside `file`. A marker whose first byte is a word
  character must not be preceded or followed by one.
- **Separators** are matched against the group currently open, between its opener
  and its closer.
- **An any-close fallback**: when nothing else matches, any code-mode group's
  closer is recognized, so a stray `}` lands as a bracket rather than derailing
  the scan.
- **The scan always consumes at least one byte**, so nothing stalls on input it
  does not understand.
- **A group left open at end of input closes there.** The bytes are already
  accounted for; nothing is dropped.

**Whitespace is not a token.** It folds into text, so a text run is *everything
between two recognized markers* rather than a run of non-whitespace. A layer that
needs line or indent boundaries — Item 5 — scans a text node for them and splits
it only if it wants them addressable.

## Output

**Flat and document-order**, always. The scanner tracks nesting while it works and
the emitted stream carries none: an opener, everything between it and its closer,
and the closer are **siblings in the array**.

The kinds the lexer adds to `sdom`:

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
The lexer's carries the language while scanning, and **outlives the parse** to
carry the pairing links:

- an opener knows its **closer** and its **enclosing opener**
- a closer knows its **opener**
- any other node knows its **enclosing opener**

These links are owned by the context, **not by `Doc`** — not every document has
brackets, and a markdown DOM would carry two dead maps forever. They are a derived
index: the context stamps itself with the document's structural generation and
rebuilds when the stamp is stale.

**The index is checkable rather than merely believed.** A forward scan that skips
whole bracket pairs finds a node's enclosing opener independently, and must agree.
