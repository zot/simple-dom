# simple-dom

[![Go Reference](https://pkg.go.dev/badge/github.com/zot/simple-dom.svg)](https://pkg.go.dev/github.com/zot/simple-dom/sdom)

A lossless, minimal DOM for source files, written in Go.

Most parsers model everything and then struggle to put the file back the way they
found it. `simple-dom` does the opposite: **it models only what a tool operates on,
keeps every other byte where it was, and re-emits the source with nothing but the
intended change in it.** When you rename a function, what changes is the name.
Comments, blank lines, odd spacing and the brace in a string literal all stay as
they were.

```go
import (
    "github.com/zot/simple-dom/sdom"
    "github.com/zot/simple-dom/sdom/schema"
)
```

```sh
go get github.com/zot/simple-dom
```

## A taste

Parse some Go, find the declaration the Go schema recognizes, rename it, and render:

```go
const src = `// Index finds k.
func Index(k string) int {
	return len(k) // not { a bracket }
}
`

bp := sdom.NewBracketParser(&sdom.LangGo)
doc := sdom.Parse(src, 0, bp)
ctx := bp.Context()

if err := schema.Go(doc, ctx); err != nil { // mark declarations
	panic(err)
}

// Resolve targets outside the mutation window...
var name *sdom.DeclarationName
for _, n := range doc.Nodes() {
	if kw, ok := n.(*sdom.DeclarationType); ok {
		names, _ := ctx.DeclarationNames(kw)
		name = names[0]
	}
}

// ...then edit inside it.
err := doc.Mutate(func() error {
	name.SetText("Lookup")
	return nil
})

out, _ := doc.Render()
fmt.Print(out)
```

```
// Index finds k.
func Lookup(k string) int {
	return len(k) // not { a bracket }
}
```

## The ideas

**The array is flat.** A `Doc` is the source bytes plus a flat, document-order
array of nodes that *tiles* the source: contiguous, half-open, starting at 0 and
ending at the end. So "every other byte stays where it was" is a property you can
check, not just a goal. Nesting is never node children. A bracket group is not a
node. Its opener, contents and closer are siblings, and the pairing links belong
to whichever layer needs them.

**Provenance is not faithfulness.** Every node's `Loc` answers two questions
separately: where it was read from, and whether it still renders those bytes. An
edited node keeps its provenance for diagnostics and reports that it has been
altered.

**One source, one pass.** Parsers work together over a single walk. Each one
recognizes what it knows at the head of the input, and whatever nobody recognizes
builds up as text. A language composes parsers by nesting them. The walk never
arbitrates between them.

**Derived state is stamped, not registered.** The document has a structural
generation that goes up whenever node membership changes. Any index built over
the document stamps itself with that generation and rebuilds when the stamp is
stale. The document knows nothing about what sits above it.

**Mutation happens in a window.** `Doc.Mutate` brackets a set of edits. Inside it,
edits are direct and nothing is queued, but navigation (`Prev`, `Next`, position
and line lookups, and reading the generation) is refused, because no correct answer
exists until the edits finish. Resolve your targets first, carry node references
in, and the indices rebuild once on the way out. `Replace`, `Insert`, `Remove`,
`Split` and `Merge` are the structural verbs.

## What's in the box

### `sdom`: the core

| | |
|---|---|
| `Node`, `Text`, `Compound` | the node protocol: `Kids`, `Location`, `Render`, `Equals` (never compares location) |
| `Loc` | provenance and faithfulness; the zero value means *no provenance*, never offset 0 |
| `Doc` | the source, the flat array, navigation, the generation, the mutation window |
| `Parser`, `ParserState` | the one-pass walk and the interface a parser implements |
| `BracketParser`, `BracketLang` | a table-driven bracket parser with restricted (string/comment) modes; ships `LangGo`, `LangJavaScript`/`LangTypeScript`, `LangLua`, `LangShell`, `LangPascal` |
| `BracketContext` | pairing links built after the parse: `Opener`, `Closer`, `Enclosing`, `Separators`, `InnerText`, plus the unclosed and unpaired brackets |
| `IndentParser`, `IndentLang` | indentation as scope, significant only at bracket depth 0 (Python's own rule); ships `LangPython` |
| `DeclarationType`, `DeclarationName` | typed nodes for a declaration's keyword and the names it introduces |
| `StencilBuilder` | compounds that parse themselves by regex, where each named group becomes a bound field |
| `Bool`, `List`, `RequirementList` | ready-made stencil values: a checkbox, a comma-separated field, a numbered list that accepts ranges |

### `sdom/schema`: bundled language schemas

Declaration passes for **Go, Lua, Shell and Python**, each run over a parsed
document and its `BracketContext`. Also a narrow **markdown** base
(`LangMarkdown`, `MarkdownParser`) covering headings, list items, checkboxes,
fences, code spans, emphasis and strikethrough. It has only the structure a tool
binds to, and it is meant to be embedded by more specific file schemas.

## Status

Pre-1.0 and in active use. The bracket parser, the document core, stencils, lists
and the Go/Lua/Shell/markdown schemas are complete and tested. The indent parser
and the Python schema work but are still being finished. The API may still move.

## How it's built

The project is developed with [mini-spec](https://github.com/zot/mini-spec). Human
specs in [`specs/`](specs/index.md) lead to a design in [`design/`](design/design.md)
(CRC cards, sequence diagrams, test designs), and the design leads to code. Source
comments carry traceability links such as `// CRC: crc-Doc.md | R1, R2` back to the
design and requirements. Start with [`specs/index.md`](specs/index.md) if you want
the reasoning behind a decision.

## License

[MIT](LICENSE)
