package sdom

import (
	"slices"
	"strings"
)

// CRC: crc-Node.md | R4, R5, R6, R9, R11, R18
//
// Node is the protocol every kind satisfies. It is about document structure and
// nothing else: Parse is deliberately absent, because parsing constructs a
// concrete node and then asks it to parse, so the interface is never involved.
//
// Two methods take no context, and both omissions are load-bearing. Render is
// not given the document, which is what stops a node satisfying the byte
// round-trip with the source span it was read from. Equals never compares
// Location, so two nodes read from different files with identical content are
// equal — including provenance would make the structural round-trip false for
// every mutated node, which is the tree that test most needs to be true of.
//
// Every concrete kind declares its own Equals, with a type check that delegates.
// Go embedding promotes without dispatching, so a promoted Equals cannot see the
// outer type — but forgetting to declare one is loud rather than silent: the
// inherited assertion fails against the new kind, two identical nodes of it
// compare unequal, and the structural round-trip goes red on the first document
// containing one.
//
// A bracket group is never a node. Openers, contents and closers are siblings in
// the flat array; nesting is links owned by the layer that needs them (R4, R18).
type Node interface {
	Kids() []Node
	Location() Loc
	Render() (string, error)
	Equals(Node) bool
}

// CRC: crc-Text.md | R15
//
// Text is a leaf. It is the kind every unmodelled byte ends up in, and the
// reason a parse that models very little still round-trips.
type Text struct {
	text string
	loc  Loc
}

// CRC: crc-Text.md | R15
// NewText returns a leaf holding text, read from loc.
func NewText(text string, loc Loc) *Text { return &Text{text: text, loc: loc} }

// CRC: crc-Text.md | R7
// Kids returns nothing: a leaf has no children.
func (t *Text) Kids() []Node { return nil }

// CRC: crc-Text.md | R21, R22
// Location reports the node's provenance with its current extent. The length is
// recomputed from the bytes rather than stored, so the two cannot drift.
func (t *Text) Location() Loc {
	l := t.loc
	l.length = len(t.text)
	return l
}

// CRC: crc-Text.md | R8
// Render returns the bytes the node stands for now.
func (t *Text) Render() (string, error) { return t.text, nil }

// CRC: crc-Node.md | R10, R13
// Equals asserts the kind, then compares content. Location is never consulted.
func (t *Text) Equals(other Node) bool {
	o, ok := other.(*Text)
	return ok && t.text == o.text
}

// CRC: crc-Text.md | R27
//
// SetText rewrites the node's bytes, keeping its offset — an altered node keeps
// its provenance, because clearing it would discard what a diagnostic still
// wants.
//
// This is a content edit and belongs inside a mutation window. Nothing enforces
// that: a node holds no reference to its document, and adding one to police a
// rule this cheap to state would cost every node a back-pointer. Outside a
// window the document's line index is left stale with nothing to rebuild it.
func (t *Text) SetText(s string) {
	t.text = s
	t.loc = t.loc.alter()
}

// CRC: crc-Compound.md | R16, R17, R19
//
// Compound is a node whose children tile its span. It does no parsing: the
// layers above supply only how their own children are computed, and embed this
// for everything else — the tiling, the concatenation, the summed extent and the
// propagated alteration are defined once, here.
//
// Compounds exist only for stenciling, meaning a region a tool writes into.
// Never for bracket structure: modelling groups as compounds would make every
// span query a traversal and every edit a re-parent, and the array is flat
// precisely because searching and splicing are what this DOM exists for.
type Compound struct {
	kids []Node
	loc  Loc
}

// CRC: crc-Compound.md | R16
// NewCompound returns a compound over kids, which must tile loc's span.
func NewCompound(loc Loc, kids ...Node) *Compound {
	return &Compound{kids: kids, loc: loc}
}

// CRC: crc-Compound.md | R7
// Kids returns the children, in document order.
func (c *Compound) Kids() []Node { return c.kids }

// CRC: crc-Compound.md | R16, R28
//
// Location sums the children's extents and derives faithfulness from them in the
// same walk. Deriving on read rather than stamping at edit time removes the
// invalidation walk entirely — and that walk carried a defect, since a fold that
// short-circuits leaves later compounds claiming spans they can no longer
// honour. The read must visit every child to sum lengths anyway, so nothing is
// saved by stamping and nothing can be skipped.
//
// A compound is faithful when its children are faithful AND their spans run
// contiguously from its own offset. R28 states only the first half; the second
// is what stops a compound whose children no longer tile its source span from
// reporting Faithful. Tracked as a gap against R28.
func (c *Compound) Location() Loc {
	l := c.loc
	l.length = 0
	next := l.Offset()
	for _, k := range c.kids {
		kl := k.Location()
		l.length += kl.Length()
		if l.altered {
			continue // the lengths are still wanted; nothing more can be learned
		}
		if kl.Faithful() && (next < 0 || kl.Offset() == next) {
			next = kl.Offset() + kl.Length()
		} else {
			l.altered = true
		}
	}
	return l
}

// CRC: crc-Compound.md | R16
// Render concatenates the children's renders.
func (c *Compound) Render() (string, error) {
	var b strings.Builder
	for _, k := range c.kids {
		s, err := k.Render()
		if err != nil {
			return "", err
		}
		b.WriteString(s)
	}
	return b.String(), nil
}

// CRC: crc-Node.md | R10, R13, R14
//
// Equals asserts the kind, then compares children pairwise. An embedding kind
// declares its own Equals and delegates here:
//
//	func (d *Decl) Equals(other Node) bool {
//	    o, ok := other.(*Decl)
//	    return ok && d.Compound.Equals(&o.Compound)
//	}
//
// A kind holding state its children do not carry compares that state in the same
// expression; a kind whose state is derived from its children compares nothing
// beyond them, because the children already settle it.
func (c *Compound) Equals(other Node) bool {
	o, ok := other.(*Compound)
	return ok && slices.EqualFunc(c.kids, o.kids, func(a, b Node) bool { return a.Equals(b) })
}
