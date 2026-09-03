package sdom

import (
	"fmt"
	"slices"
	"sort"
	"strings"
)

// CRC: crc-Doc.md | Seq: seq-stamp.md | R1, R2, R3, R37, R38, R39, R40
//
// Doc is a parsed source: the bytes, the flat document-order node array over
// them, and two derived indices. It holds nothing schema-specific — anything
// else derived is owned and stamped by the layer that needs it, so a document
// type whose layers need no derived state carries none.
//
// The array is flat and tiles the source: contiguous, half-open, first at 0 and
// last at the end. That is what makes "every other byte stays where it was" a
// checkable property rather than an intention.
type Doc struct {
	source string
	base   int
	dom    []Node

	// Data is an open slot for whatever a consumer wants to hang here. Doc
	// never reads it.
	Data any

	nodeIndex   map[Node]int
	lineStart   []int
	renderedLen int

	generation uint64

	mutating bool
	dirty    bool
	poisoned bool
}

// CRC: crc-Doc.md | R37, R38
//
// New returns a document over source whose nodes are given in document order.
// base is the document's own position within an outer document, so the same text
// can be parsed as a sub-document without its nodes lying about where they came
// from.
//
// The nodes are expected to tile the source (R3). That is not checked here:
// stating the rule is enough, and the corpus tests are where it is proven.
func New(source string, base int, nodes ...Node) *Doc {
	oneParse(nodes)
	d := &Doc{source: source, base: base, dom: nodes}
	_ = d.rebuild() // no node kind in this package can fail to render
	return d
}

// CRC: crc-Doc.md | R118, R119
//
// oneParse requires every node to come from the same parse, and panics otherwise.
// Checked once here because nothing else can break it: Split inherits the origin,
// Merge joins only nodes already in the document, and Remove takes nothing in —
// so this is the ONLY check, and mergeLocs carries none. What it buys is
// that the whole array shares a coordinate system, which is what lets Merge trust
// a node's own claim about itself.
//
// What it does NOT catch, stated because nothing will: a document built over one
// source from nodes uniformly attributed to a different one. Verifying that would
// mean comparing the very bytes the check exists to avoid copying. So a document
// REQUIRES its nodes to describe its source and enforces only the cheap half —
// under a violation the array does not tile, faithful nodes do not render their
// spans, and the round-trip fails, so the failure is loud even unguarded.
func oneParse(nodes []Node) {
	if len(nodes) == 0 {
		return
	}
	var want *Origin
	for i, n := range nodes {
		got := n.Location().Origin()
		if got == nil {
			// A synthesized node makes no claim about coordinates, so it cannot
			// contradict one — nil is UNKNOWN rather than different, exactly as
			// mergeLocs treats it. Without this a hand-built document could not
			// mix parsed and synthesized nodes, though Merge joins them happily.
			continue
		}
		if want == nil {
			want = got
			continue
		}
		if got != want {
			panic(fmt.Sprintf("sdom: node %d comes from %s, but the document is %s",
				i, got, want))
		}
	}
}

// CRC: crc-Doc.md | R37
// Source returns the bytes the document was parsed from.
func (d *Doc) Source() string { return d.source }

// CRC: crc-Doc.md | R38
// Base returns the document's own position within an outer document.
func (d *Doc) Base() int { return d.base }

// CRC: crc-Doc.md | R1
// Nodes returns the document-order node array. It must not be modified: the
// document's own indices are derived from it, and Split, Merge and Remove are
// how membership changes.
func (d *Doc) Nodes() []Node { return d.dom }

// CRC: crc-Doc.md | Seq: seq-mutate.md#1.4.3 | R39, R49
//
// IndexOf returns n's position, or -1 when n is not in this document.
//
// It refuses inside a mutation window. Rebuilding the index mid-window would not
// rescue the caller: if the node they hold has been removed, this would return
// -1 and Next would hand back nothing, which is indistinguishable from
// end-of-document. Refusing says what happened instead.
func (d *Doc) IndexOf(n Node) int {
	d.guard("IndexOf")
	if i, ok := d.nodeIndex[n]; ok {
		return i
	}
	return -1
}

// CRC: crc-Doc.md | R41
// Next returns the node after n, or nil at the end of the document. Navigation
// is by node: positions are never a currency the caller carries.
func (d *Doc) Next(n Node) Node {
	i := d.IndexOf(n)
	if i < 0 || i+1 >= len(d.dom) {
		return nil
	}
	return d.dom[i+1]
}

// CRC: crc-Doc.md | R41
// Prev returns the node before n, or nil at the start of the document.
func (d *Doc) Prev(n Node) Node {
	i := d.IndexOf(n)
	if i <= 0 {
		return nil
	}
	return d.dom[i-1]
}

// CRC: crc-Doc.md | R39, R49
// Line returns the 1-based line containing offset in the rendered document, or
// 0 when offset is outside it. It refuses inside a mutation window.
func (d *Doc) Line(offset int) int {
	d.guard("Line")
	if offset < 0 || offset >= d.renderedLen {
		return 0
	}
	return sort.Search(len(d.lineStart), func(i int) bool { return d.lineStart[i] > offset })
}

// CRC: crc-Doc.md | R39
// LineCount returns the number of lines in the rendered document.
func (d *Doc) LineCount() int {
	d.guard("LineCount")
	return len(d.lineStart)
}

// CRC: crc-Doc.md | Seq: seq-stamp.md#1.1 | R40, R42, R43, R44, R45, R46
//
// Generation returns the document's structural generation, bumped whenever node
// membership changes and never for a content edit — membership is unchanged
// there, and an index over structure survives one.
//
// A layer holding derived state stamps itself with this and rebuilds when the
// stamp is stale. Doc keeps no registry of such indices and issues no
// invalidation callbacks: the layers pull, and the document does not push.
//
// It refuses inside a mutation window, which is what extends the guard to a
// layer's index without that layer ever writing one, since checking freshness is
// the one call every stamped index makes. Poisoning the comparison instead —
// returning a value no stamp could match — would let the layer rebuild from a
// half-edited tree, which is a different wrong answer rather than a refusal.
func (d *Doc) Generation() uint64 {
	d.guard("Generation")
	return d.generation
}

// CRC: crc-Doc.md | R2
// Render returns the whole document. It is not guarded: under direct edits your
// own edit is readable, which is the point of them.
func (d *Doc) Render() (string, error) {
	if d.poisoned {
		return "", ErrPoisoned
	}
	var b strings.Builder
	b.Grow(len(d.source))
	for _, n := range d.dom {
		s, err := n.Render()
		if err != nil {
			return "", err
		}
		b.WriteString(s)
	}
	return b.String(), nil
}

// CRC: crc-Doc.md | Seq: seq-mutate.md#1.6 | R43, R52
//
// rebuild recomputes both derived indices and bumps the generation if membership
// changed. It runs once, at the exit of the outermost mutation window.
//
// Bumping once at the exit rather than once per membership change is
// unobservable: the read refuses inside the window, and a layer comparing stamps
// only cares that they differ.
func (d *Doc) rebuild() error {
	d.nodeIndex = make(map[Node]int, len(d.dom))
	for i, n := range d.dom {
		d.nodeIndex[n] = i
	}
	rendered, err := d.Render()
	if err != nil {
		return err
	}
	d.renderedLen = len(rendered)
	d.lineStart = append(d.lineStart[:0], 0)
	for i := range len(rendered) {
		if rendered[i] == '\n' {
			d.lineStart = append(d.lineStart, i+1)
		}
	}
	if d.dirty {
		d.generation++
		d.dirty = false
	}
	return nil
}

// find locates n by scanning, for use inside a mutation window where the derived
// index is stale and refuses. Splicing needs the position anyway, so the scan is
// not an extra cost beyond the edit itself.
func (d *Doc) find(n Node) int { return slices.Index(d.dom, n) }
