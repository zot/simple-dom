# Lists: a comma-separated field, and the numbered one

A **list** is a stenciled field whose value is a comma-separated sequence — the
`CRC:` cards, `Seq:` diagrams and `Test:` designs of a traceability comment, and
the requirement refs beside them. Sequence diagrams will want it too. It is
`sdom`'s because it knows nothing of CRC cards; it knows commas.

## One kind, one child

```go
type List struct{ Compound }          // exactly one child: the Text of the whole field

func ParseList(text string, loc Loc) (*List, string, bool)   // plain items
func (l *List) Items() []string
func (l *List) SetItems(items []string) error
```

`Items` **derives** the values from the literal on every call — split on commas,
whitespace trimmed — and stores nothing. `SetItems` **rewrites the whole literal**
canonically: items joined by `", "`. Nothing is normalised on the way in, so
`a,b` and `a ,  b` both read `[a b]` and both render back byte-exact until
something writes; after a write the field is canonical. An unedited *field* is
never touched by an edit to another field.

The grammar a list parses, and what it consumes:

```
list  ::= WS? item ( WS? "," WS? item )* WS?
item  ::= one run of bytes containing no whitespace and no comma
```

`ParseList` returns what it did not consume; whether leftover bytes are acceptable
is the caller's business, exactly as `NewStencilBuilder`'s `false` is.

**A write is guarded by re-parsing its own render.** `SetItems` renders the new
literal and parses it again, requiring the re-parse to consume all of it and yield
the same items. An item containing a comma or whitespace renders fine and loses no
bytes — a byte round-trip is blind to it — but reads back as two items, or stops
short; the write is **refused with an error and the literal is unchanged**. This is
the one place in `sdom` a write has a guard, because a node-level write changes
one literal and rollback is free. (An item containing the *enclosing* stencil's own
separator — `|` in a traceability comment — is legal to the list and is that
stencil's corner, not this one's.)

## The numbered list

```go
type RequirementList struct{ List }

func ParseRequirementList(text string, loc Loc) (*RequirementList, string, bool)
func (l *RequirementList) Items() []int
func (l *RequirementList) SetItems(items []int)
```

Items are requirement numbers, written `Rn`; a **range** `Rn-Rm` or `Rn-m` is one
item, and `Items` expands it. A reversed range contributes only its low ref.
`SetItems` sorts, de-duplicates, and emits **maximally condensed**: a run of three or
more consecutive numbers becomes `R7-12`, a pair stays `R7, R8`. It cannot be
refused — integers carry no separator — so it returns nothing.

**Only this parser accepts ranges**, and only requirements use it. A range is an
item shape the plain parser does not know, so `crc-a.md-crc-b.md` cannot come to
exist by construction. Which parser a field uses is the enclosing stencil's choice.

## Why a compound with one child

The field is a region a tool reads and writes as a value, so it is a stencil, and a stencil is a
compound. It has one child because its only write is whole-field replace: per-item
boundaries would imply an edit nobody makes. A tool that wants to add one item reads
`Items`, appends, and calls `SetItems`.
