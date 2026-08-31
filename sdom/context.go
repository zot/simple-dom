package sdom

import "slices"

// CRC: crc-BracketContext.md | Seq: seq-pair.md | R12, R80, R81, R85
//
// BracketContext is the lexicon's parse context: a CONCRETE type, not an
// interface (R12). An interface here would exist only to let one signature serve
// heterogeneous node kinds, which nothing requires — each schema's context is
// shaped for its own needs, and generic core code that needs only the document
// takes *Doc. It carries the language through the scan and OUTLIVES the parse to
// own the bracket pairing links.
//
// Doc does not own these links. Not every document has brackets, and a markdown
// DOM would carry two dead maps forever — so the layer that needs the state owns
// it, and a document with no brackets carries none.
type BracketContext struct {
	lang *BracketLang
	doc  *Doc

	origin *Origin // minted once per parse, carried by every node it produces

	closerOf  map[Node]Node // opener  -> its closer
	openerOf  map[Node]Node // closer  -> its opener
	enclosing map[Node]Node // any other node -> the opener containing it

	stamp uint64
}

func newBracketContext(lang *BracketLang) *BracketContext {
	return &BracketContext{
		lang:      lang,
		origin:    &Origin{},
		closerOf:  map[Node]Node{},
		openerOf:  map[Node]Node{},
		enclosing: map[Node]Node{},
	}
}

// CRC: crc-BracketContext.md | R80
// Language returns the table this context scanned with.
func (bc *BracketContext) Language() *BracketLang { return bc.lang }

// CRC: crc-BracketContext.md | R91
//
// Origin returns the token identifying this parse, which every node the scan
// produced carries. It is minted before any node exists, which is why a location
// holds the context's token rather than a document reference. Its Name is the
// caller's to set.
func (bc *BracketContext) Origin() *Origin { return bc.origin }

// CRC: crc-BracketContext.md | Seq: seq-pair.md#1.2 | R82, R83
// Closer returns the closer paired with an opener, or nil when the group was
// left open at end of input.
func (bc *BracketContext) Closer(opener Node) Node {
	bc.refresh()
	return bc.closerOf[opener]
}

// CRC: crc-BracketContext.md | Seq: seq-pair.md#1.2 | R83
// Opener returns the opener paired with a closer, or nil for an unmatched one.
func (bc *BracketContext) Opener(closer Node) Node {
	bc.refresh()
	return bc.openerOf[closer]
}

// CRC: crc-BracketContext.md | Seq: seq-pair.md#1.3 | R84
// Enclosing returns the innermost opener containing n, or nil at top level.
func (bc *BracketContext) Enclosing(n Node) Node {
	bc.refresh()
	return bc.enclosing[n]
}

// CRC: crc-BracketContext.md | Seq: seq-pair.md#1.5 | R86
//
// refresh rebuilds the links when this context's stamp has gone stale.
//
// Reading the generation is the one call every stamped index makes, which is what
// extends the mutation-window guard to THIS index without the context having
// written a guard of its own: inside a window the read refuses.
func (bc *BracketContext) refresh() {
	if bc.doc == nil {
		return
	}
	if g := bc.doc.Generation(); g != bc.stamp {
		bc.rebuild()
		bc.stamp = g
	}
}

// CRC: crc-BracketContext.md | R81, R82, R83, R84
//
// rebuild derives every link a second way: by walking the finished flat array
// with a stack, rather than from the recursion that produced it. The scan records
// links from its own control flow; this recovers them from the resulting data.
// The two are independent derivations and must agree — which is what makes the
// index checkable rather than merely believed (R87).
func (bc *BracketContext) rebuild() {
	bc.closerOf = make(map[Node]Node, len(bc.closerOf))
	bc.openerOf = make(map[Node]Node, len(bc.openerOf))
	bc.enclosing = make(map[Node]Node, len(bc.enclosing))
	var stack []Node
	for _, n := range bc.doc.Nodes() {
		var open Node
		if len(stack) > 0 {
			open = stack[len(stack)-1]
		}
		switch n.(type) {
		case *Closer:
			// Pair only when this closer belongs to the open group. The group is
			// resolved from the OPENER'S TEXT through the table, not from anything
			// the scan recorded — a node holds no group pointer, and this
			// derivation must stay independent of the one it checks. Without it a
			// stray closer, which the any-close fallback emits unpaired, would be
			// paired here and the two derivations would disagree on every
			// unbalanced file.
			if open != nil && bc.closes(open, n) {
				stack = stack[:len(stack)-1]
				bc.closerOf[open] = n
				bc.openerOf[n] = open
			}
		case *Opener:
			if open != nil {
				bc.enclosing[n] = open
			}
			stack = append(stack, n)
		default:
			if open != nil {
				bc.enclosing[n] = open
			}
		}
	}
}

// closes reports whether closer ends the group that opener started, resolving the
// group from the opener's own bytes.
func (bc *BracketContext) closes(opener, closer Node) bool {
	ot, err := opener.Render()
	if err != nil {
		return false // a node that cannot render cannot be matched against a table
	}
	g := bc.lang.groupFor(ot)
	if g == nil {
		return false
	}
	ct, err := closer.Render()
	if err != nil {
		return false
	}
	return slices.Contains(g.Close, ct)
}

// attach binds the context to the document it was scanned from and stamps it
// with the generation the scan's own links describe, so the first read of a
// freshly scanned document rebuilds nothing.
func (bc *BracketContext) attach(d *Doc) {
	bc.doc = d
	bc.stamp = d.Generation()
}
