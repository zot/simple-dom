package sdom

import (
	"errors"
	"slices"
)

// CRC: crc-BracketContext.md | Seq: seq-pair.md | R12, R80, R81, R85
//
// BracketContext is the schema's parse context: a CONCRETE type, not an
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

	// R154: ONE index rather than four. How the links are stored is not part of the
	// contract — what this owes is the answers — which is what made consolidating
	// them a free change.
	info map[Node]BracketInfo

	declStamp uint64
	declSet   bool

	stamp uint64
}

// CRC: crc-BracketContext.md | R152, R154
//
// BracketInfo is everything this context knows about one node. Which fields are
// set depends on what the node is: an opener has a closer, its separators and its
// enclosing opener; a closer has an opener; a separator has both; anything else has
// only an enclosing opener; and a declaration keyword has the names it declares.
//
// It is a FLAT VALUE STRUCT, and that won on allocations rather than on size. A
// boxed hierarchy — an interface with a small struct for plain nodes and a wide one
// for bracket participants — is the smaller index, because a plain node then carries
// 16 bytes instead of 96. But an interface value cannot hold a struct inline, so
// every entry becomes its own heap object. Measured over 15,747 nodes: 111
// allocations against 15,858, on an index that rebuild recreates at EVERY structural
// change. The flat struct also allocates less than the three maps it replaced.
type BracketInfo struct {
	opener, closer, enclosing Node
	separators                []Node

	// R126, R127: the names a declaration keyword declares — one for a plain
	// declaration, several for a group. FILLED IN by a schema, because filling it in
	// means knowing what announces a declaration, and wiped by any rebuild, because
	// this is the one part the context cannot re-derive.
	declaration []Node
}

func newBracketContext(lang *BracketLang) *BracketContext {
	return &BracketContext{lang: lang, origin: &Origin{}, info: map[Node]BracketInfo{}}
}

// with reads, modifies and writes one node's entry. Map values are structs, so this
// is the only way to touch a field.
func (bc *BracketContext) with(n Node, f func(*BracketInfo)) {
	i := bc.info[n]
	f(&i)
	bc.info[n] = i
}

// pair records a closed group, from either derivation.
func (bc *BracketContext) pair(opener, closer Node) {
	bc.with(opener, func(i *BracketInfo) { i.closer = closer })
	bc.with(closer, func(i *BracketInfo) { i.opener = opener })
}

// enclose records that n sits inside open. A separator also learns its opener,
// and the opener learns the separator — the direction R152 adds.
func (bc *BracketContext) enclose(n, open Node) {
	if _, isSep := n.(*Separator); !isSep {
		bc.with(n, func(i *BracketInfo) { i.enclosing = open })
		return
	}
	bc.with(n, func(i *BracketInfo) {
		i.enclosing = open
		i.opener = open
	})
	bc.with(open, func(i *BracketInfo) { i.separators = append(i.separators, n) })
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
	return bc.info[opener].closer
}

// CRC: crc-BracketContext.md | Seq: seq-pair.md#1.2 | R83
// Opener returns the opener paired with a closer, or nil for an unmatched one.
func (bc *BracketContext) Opener(closer Node) Node {
	bc.refresh()
	return bc.info[closer].opener
}

// CRC: crc-BracketContext.md | Seq: seq-pair.md#1.4 | R152
//
// Separators returns the separators belonging to opener's group, in document order.
//
// This is the direction the contract was missing. A separator could always be traced
// back through its enclosing opener; nothing could ask an opener which separators
// were its own without scanning forward for them. Consumer count did not decide it:
// a library answers the questions its own structure makes meaningful.
func (bc *BracketContext) Separators(opener Node) []Node {
	bc.refresh()
	return bc.info[opener].separators
}

// CRC: crc-BracketContext.md | Seq: seq-pair.md#1.3 | R84
// Enclosing returns the innermost opener containing n, or nil at top level.
func (bc *BracketContext) Enclosing(n Node) Node {
	bc.refresh()
	return bc.info[n].enclosing
}

// CRC: crc-BracketContext.md | R128
//
// ErrDeclarationsStale reports that the document changed structurally since a
// schema last filled these links in.
var ErrDeclarationsStale = errors.New(
	"sdom: declaration links are stale; re-run the schema's declaration pass")

// CRC: crc-BracketContext.md | Seq: seq-declare.md#2.5 | R126, R127, R128
//
// SetDeclarations replaces the declaration links and stamps them against the
// document's current generation. A schema calls it after its pass, OUTSIDE the
// mutation window the pass ran in — reading the generation refuses inside one.
func (bc *BracketContext) SetDeclarations(links map[Node][]Node) {
	// Refresh FIRST, and this is not an optimisation. rebuild recreates the whole
	// index, and the declaration links now live in it — so a rebuild pending at
	// this moment would wipe what we are about to write, and then set stamp to the
	// generation declStamp already holds, making the loss invisible to the
	// staleness check. Every schema pass reaches here in exactly that state: it
	// mutates inside a window and records afterwards. Doing the pending rebuild
	// before the write is what keeps the two stamps honest.
	bc.refresh()
	// Replace, as the name says: a keyword absent from links keeps nothing.
	for n := range bc.info {
		if bc.info[n].declaration != nil {
			bc.with(n, func(i *BracketInfo) { i.declaration = nil })
		}
	}
	for kw, names := range links {
		bc.with(kw, func(i *BracketInfo) { i.declaration = names })
	}
	bc.declSet = true
	if bc.doc != nil {
		bc.declStamp = bc.doc.Generation()
	}
}

// CRC: crc-BracketContext.md | Seq: seq-declare.md#2.5 | R127, R128
//
// Declarations returns the names a keyword node declares.
//
// This is the ONE index this context cannot rebuild. The pairing links are
// recoverable from the finished array by walking it; declarations are not, because
// sdom does not know what announces one in any language. So freshness is answered
// HERE, at the accessor, which is the one call every reader makes — and a stale
// accessor REFUSES.
//
// Returning the old map would answer from a document that has changed underneath
// it, and returning an empty one is worse: it reads identically to "this keyword
// declares nothing". That is the plausible wrong answer IndexOf already refuses on
// the same grounds, where -1 would be indistinguishable from end-of-document.
func (bc *BracketContext) Declarations(kw Node) ([]Node, error) {
	if !bc.declSet {
		return nil, ErrDeclarationsStale
	}
	if bc.doc != nil && bc.doc.Generation() != bc.declStamp {
		return nil, ErrDeclarationsStale
	}
	return bc.info[kw].declaration, nil
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

// CRC: crc-BracketContext.md | R81, R82, R83, R84, R152, R153
//
// rebuild derives every link a second way: by walking the finished flat array
// with a stack, rather than from the recursion that produced it. The scan records
// links from its own control flow; this recovers them from the resulting data.
// The two are independent derivations and must agree — which is what makes the
// index checkable rather than merely believed (R87).
func (bc *BracketContext) rebuild() {
	bc.info = make(map[Node]BracketInfo, len(bc.info))
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
				bc.pair(open, n)
			}
		case *Opener:
			if open != nil {
				bc.enclose(n, open)
			}
			stack = append(stack, n)
		default:
			// A Separator lands here, and enclose gives it BOTH directions: it
			// learns its opener, and its opener learns it. Recovering separators
			// from this walk rather than trusting the scan is what keeps them
			// inside R87's cross-derivation instead of riding along beside it.
			if open != nil {
				bc.enclose(n, open)
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
