# Loc
**Requirements:** R20, R21, R22, R23, R24, R25, R26, R27, R29, R30, R34, R35, R36, R88, R89, R90, R92, R93

A node's location, separating **provenance** — where it was read from — from
**faithfulness** — whether it still renders those bytes. The two stop agreeing
the moment anything is edited, which is why they are not one field.

## Knows
- the source position the node was read from, or its absence — **relative to its
  own document's source**, never including that document's base
- the **parse** it came from, as an `Origin`, or nil when unknown
- the node's current rendered extent
- whether the node still renders the bytes at that position

## Does
- `Offset() int`: the source position, or −1 when there is no provenance
- `Length() int`: the current rendered extent in bytes
- `Altered() bool`: read from `Offset`, no longer rendering it
- `Faithful() bool`: has provenance and is not altered — the question callers
  should be asking
- `Origin() *Origin`: which parse produced this, or nil
- `In(*Origin) Loc`: the parser's chained setter
- supplies the location arithmetic for `Split` and `Merge`

## Constraints
- **The zero value means "no provenance", never "offset 0".** Offset 0 is the
  first byte of the file, a plausible wrong answer nothing would be forced to
  correct — so absence is what an unset location reads as
- **An altered node keeps its offset.** Clearing it discards provenance a
  diagnostic still wants
- **For an altered node the offset is historical while the length is current**, so
  the pair names a span the node never owned. A consumer uses `Offset` alone
  unless `Faithful()`
- **A faithful node renders exactly the source span at its offset**

## Origin
A concrete token identifying **one parse, not one file**, so two parses of the same
source are two origins and "same file, different parser" is answerable.

- It holds a `Name` — a path, a URL, whatever the caller finds useful. It carries a
  field rather than being an empty marker because Go may give every zero-size
  allocation the same address, and an empty one would fail as an identity exactly
  where it was needed. It doubles as a lookup key
- It is the **parse context**, not the document: the context exists before the
  nodes do, so a document reference would need back-patching over every node
- It is **concrete, not an interface** — each schema's context is its own concrete
  type, so there is no single one to hold
- **A nil origin is unknown, not different**, and is compatible with anything, the
  way no provenance already is
- Setting it changes no existing call site: the offset-and-length constructor is
  unchanged, and a parser chains `.In(o)` onto it
- `Equals` still never compares `Location`, which is what makes this safe where a
  node holding its bracket group is not

## Re-granulation
`Split` and `Merge` move boundaries; bytes and provenance do not move.

- **Split**: each half keeps the provenance of the part of the source it covers
- **Merge**: the result is **faithful only if both operands are** — "unfaithful"
  covering both altered and no-provenance — and takes **the first operand's
  offset when it has one, the second's otherwise**. Leftmost-provenance-wins makes
  the rule associative, so merging a run does not depend on how it is grouped
- **Two present, differing origins never meet.** The document refuses them at
  construction and `Merge` joins only its own nodes, so the merge arithmetic carries
  no check — the unreachable inner guard was removed 2026-09-02
- Split-then-merge of the same pair returns the original span, provenance intact

## Collaborators
- Node: every node reports one
- Doc: owns `Split` and `Merge`, since both change membership

## Sequences
- seq-mutate.md
