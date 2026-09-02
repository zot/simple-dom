# Indent scope

Indentation is significant **only at bracket depth 0**. One rule rather than two
mechanisms selected per language: Python needs it because a continuation line inside
`foo(1,` carries no scope, YAML needs it because flow style *is* JSON, and a brace
language is the degenerate case where the rule never fires.

It is also Python's own rule. CPython's tokenizer tracks a parenthesis depth and
suppresses line and indent tokens entirely while it is above zero.

## The language

```go
// IndentLang describes a language whose indentation carries scope.
type IndentLang struct {
    BracketLang           // the type is the flag; there is no boolean

    Tab          int      // a tab advances to the next multiple of this
    Transparent  string   // groups of this Kind do not affect the level
    Continuation string   // a line after this marker is never indented
}
```

**`Transparent` names a `Kind`, not a syntax.** `BracketGroup` carries an
uninterpreted `Kind` label; a language marks its comment groups with some value and
names that value here. So this package never learns what a comment *is* — it compares
two configured strings — and the bracket parser never reads `Kind` at all.

A language needing that distinction has to have it: a comment-only line does not
change the level and a **string-only line does**, measured against CPython, and both
open a restricted group, so no property of the group's shape separates them.

## The nodes

A single new kind, **`Indent`**, holding the leading whitespace of the line whose
level it announces.

**One node at every change of level, and none where the level is unchanged.**
Consecutive lines at one column share the frame opened by the last change, and their
leading whitespace stays ordinary text. The same bytes are therefore a node on one
line and text on the next — which is what the reference implementation does, and what
keeps a node meaning *something happened here*.

**Two of them are zero-length**, and half-open indices already make that free:

- **The root.** Every parse of a non-empty source opens with an `Indent("")`, so
  top-level frames have somewhere to link and their list lives with every other
  frame's children rather than in a collection beside it. An empty source produces no
  nodes at all, as it does today.
- **A return to column 0**, which has no whitespace to own.

**A dedent is not a node of its own.** The column *is* the level, so closing several
levels at once is one node with a smaller column rather than a run of empty markers —
less machinery than a token stream needs, because an index can hold the pairing.

**The column is derived from the node's text, never stored**, expanding tabs by
`Tab`. Deriving costs a walk over a short string; storing costs an invalidation
protocol and a second thing that can be wrong.

## What does not change the level

- **A blank line**, and a line holding only groups of the `Transparent` kind.
- **A line after a `Continuation` marker** — and this needs no rule of its own,
  because a marker counts only at **bracket depth 0**, exactly as indentation does.
  A backslash ending a *comment* sits inside the comment group, so it does not
  continue; measured against CPython, which treats the following indent as
  significant and rejects it. A backslash inside a string needs no rule either: the
  string group is still open at the next line start, so depth-0 suppression already
  covers it.

## The links the context owns

```go
func (ic *IndentContext) Parent(indent Node) Node    // the frame containing this one
func (ic *IndentContext) Children(indent Node) []Node // the frames directly inside it
```

**A frame's parent is the nearest preceding `Indent` with a strictly smaller
column.** So a dedent past several levels re-parents to the level it lands in, not to
the one it left, and two frames at one column under one parent are siblings rather
than the same frame — which is what the flat array can express without a tree.

The links live in the **indent context**, which owns one index. Brackets cannot
contain an indent region, so the indent context is the outer one; a brace language
constructs none of this and pays nothing for it.

**The index is checkable rather than merely believed.** Every frame link is derivable
from the flat array alone — an `Indent` node carries its own whitespace, so a consumer
walking with a stack of open columns reaches the same parent and children the context
reports, needing no language knowledge to do it. The pass itself records nothing: one
derivation, on demand, and nothing to fall out of step with. This is the same guarantee
the bracket links carry, and it is what a representation change is most likely to lose.

## What this does not do

`sdom` is not a syntax checker. A dedent to a column matching no open level is a
Python error and is not one here: the frame parents to the nearest smaller column and
a schema may object if it wants to. Scope for a brace language's *locals* would nest
by indentation rather than by brace, which is wrong — and unreachable, since a brace
language has no indent parser.
