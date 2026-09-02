# PythonSchema
**Requirements:** R189, R190, R191

How Python announces a declaration. The third worked schema, and the one that
proves the top-level predicate was never about indentation.

## Knows
- `LangPython`, and the context an indent parse of it produced
- its keywords: `def` and `class`

## Does
- matches a keyword at a **statement start** over top-level text nodes
- walks forward in the same text for the name — no receiver group intervenes, so
  this is Go's walk without the opener-skip step
- splits the name out of its text node and re-types both halves

## Constraints
- **A Python declaration is at bracket depth 0 at any indent depth.** A method sits
  inside its class body, which is an **indent frame and not a bracket group** — so
  it still has no bracket enclosing it, and Go's top-level predicate transfers
  untouched. This is the evidence that *top level* always meant bracket depth
- **Which depths a language's declarations live at is a language fact**, not a
  parameter: Go is depth 0 throughout, Java puts types at the top and members
  inside them, TypeScript does both, and JavaScript has no depth rule at all —
  where `Foo.prototype.bar = function …` even puts the **name before the keyword**.
  That is why recognition is a per-schema pass rather than a driver with a setting
- **Decorators are ordinary text** on their own lines above, so they neither hide
  the keyword nor get swallowed by the name walk
- **The structural statement-start rule is reached often here.** A comment's
  closing newline is a `Closer` rather than a byte of text, so a `def` after a
  comment line starts its text node with no separator in front of it

## Collaborators
- IndentParser: produced the parse this runs over
- BracketContext: the declaration links this fills
- Declaration: the two node kinds it produces

## Sequences
- seq-declare.md
