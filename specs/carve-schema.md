# The carve schema

The first of the four file schemas: it embeds the markdown base, **owns a carve file's
DOM**, and adds what a carve has that markdown does not — the `## Status` block and the
part lines inside it. Nothing parses a carve with the base alone.

```go
type Carve struct { /* the document, its contexts, the parts */ }

func ParseCarve(src string) *Carve
func (c *Carve) Doc() *sdom.Doc
func (c *Carve) Render() (string, error)

func (c *Carve) HasStatus() bool
func (c *Carve) Parts() []*Part          // every part line with a checkbox, in order
func (c *Carve) Stateless() []*PartLine  // status lines with no checkbox: split parents, standing constraints
func (c *Carve) Part(key string) *Part   // nil when no part keys so

func (c *Carve) SetMarker(key, verb, attribution string) error
func (c *Carve) Land(key, attribution string) error

type Part struct {
    *PartLine
    Depth  int    // the bullet's leading whitespace; 0 is top level
    Parent *Part  // the nearest preceding part with a smaller depth, or nil
}
```

## What it reads

**The status block is a region**: from the level-2 heading whose text is `Status` to the
next heading of level 2 or higher, or the end of the file. A carve with no such heading has
no status block, and `HasStatus` says so rather than the reader guessing.

**Only list items inside that region become part lines.** A carve's body carries other
checkbox lists — open questions, breakdowns inside an elaboration — that track different
things, and they are left as the base parsed them. Inside a fence the base emits no list
item, so a fenced sample of a status block is invisible here by construction.

**A line with a checkbox is a part; one without is stateless.** No checkbox means
something — a split parent, a standing constraint — so it is recorded rather than dropped,
and never given a state the document declined to state.

**Depth is the bullet's leading whitespace, and a subpart is an indented sibling.** The
flat array has no children; a part's `Parent` is derived as the nearest preceding part
with a smaller depth, the rule the indent frames already use.

Every part carries its line's deviations, so the shape a line must take travels with it.

## What it writes

**`SetMarker(key, verb, attribution)`** applies the tool's rule to the keyed line: replace
the first *transient* marker — one whose verb is `OPEN` — with the new one, remove any
other transient, and append a marker when the line carries none. It selects by the kind of
what it replaces, never by what it writes, so setting a record over a line that also
carries `NOT VERIFIED` leaves that standing.

**`Land(key, attribution)`** is the completion write, three markings in one act because the
format requires them to agree: the checkbox becomes `[x]`, the head is struck, and
`LANDED (attribution)` goes on through the marker rule.

Both edit the node's own children — the flat array is untouched — and a write to a key
no part carries is an error, never a silent no-op.
