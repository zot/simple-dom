package sdom

import (
	"errors"
	"strings"
)

// CRC: crc-BracketContext.md | Seq: seq-pair.md | R12, R80, R81, R85
//
// BracketContext is the schema's parse context: a CONCRETE type, not an
// interface (R12). An interface here would exist only to let one signature serve
// heterogeneous node kinds, which nothing requires — each schema's context is
// shaped for its own needs, and generic core code that needs only the document
// takes *Doc. It carries the language through the parse and OUTLIVES it to
// own the bracket pairing links.
//
// Doc does not own these links. Not every document has brackets, and a markdown
// DOM would carry two dead maps forever — so the layer that needs the state owns
// it, and a document with no brackets carries none.
type BracketContext struct {
	lang *BracketLang
	// the table's compiled patterns, for resolving a pattern marker; never nil,
	// because the only context is the one a parser makes after compiling them
	pats *patterns
	doc  *Doc

	origin *Origin // minted once per parse, carried by every node it produces

	// R154: ONE index rather than four. How the links are stored is not part of the
	// contract — what this owes is the answers — which is what made consolidating
	// them a free change.
	info map[Node]BracketInfo

	built bool // has rebuild ever run? see refresh

	// R349: every demotion the parse recorded, in document order. Recorded at the
	// rewind rather than derived, because the opener node no longer exists; anchored
	// to the run rather than a byte offset so a mutation elsewhere does not move it.
	demoted []DemotedOpener

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
	opener     *Opener
	closer     *Closer
	enclosing  Node
	separators []Node

	// R126, R127, R197: the names a declaration keyword declares — one for a plain
	// declaration, several for a group. FILLED IN by a schema, because filling it in
	// means knowing what announces a declaration, and wiped by any rebuild, because
	// this is the one part the context cannot re-derive. Typed, so no consumer
	// asserts a kind the context already knew.
	declaration []*DeclarationName
}

func newBracketContext(lang *BracketLang, pats *patterns) *BracketContext {
	return &BracketContext{lang: lang, pats: pats, origin: &Origin{}, info: map[Node]BracketInfo{}}
}

// with reads, modifies and writes one node's entry. Map values are structs, so this
// is the only way to touch a field.
func (bc *BracketContext) with(n Node, f func(*BracketInfo)) {
	i := bc.info[n]
	f(&i)
	bc.info[n] = i
}

// pair records a closed group, from either derivation.
func (bc *BracketContext) pair(opener *Opener, closer *Closer) {
	bc.with(opener, func(i *BracketInfo) { i.closer = closer })
	bc.with(closer, func(i *BracketInfo) { i.opener = opener })
}

// enclose records that n sits inside open. A separator also learns its opener,
// and the opener learns the separator — the direction R152 adds.
//
// A separator therefore stores the same node in BOTH enclosing and opener, and that
// redundancy is deliberate: it is what lets Opener() and Enclosing() answer for a
// separator without either growing a case for it. Collapsing the two fields looks
// like an easy 16 bytes and would break Opener(separator) silently, since nothing
// else asks that question yet.
func (bc *BracketContext) enclose(n Node, open *Opener) {
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
// Language returns the table this context parsed with.
func (bc *BracketContext) Language() *BracketLang { return bc.lang }

// CRC: crc-BracketContext.md | R222
// Doc returns the document this context is bound to, or nil before the parse's Done.
func (bc *BracketContext) Doc() *Doc { return bc.doc }

// CRC: crc-BracketContext.md | R91
//
// Origin returns the token identifying this parse, which every node the parse
// produced carries. It is minted before any node exists, which is why a location
// holds the context's token rather than a document reference. Its Name is the
// caller's to set.
func (bc *BracketContext) Origin() *Origin { return bc.origin }

// CRC: crc-BracketContext.md | Seq: seq-pair.md#1.2 | R82, R83, R193
// Closer returns the closer paired with an opener, or nil when the group was
// left open at end of input.
func (bc *BracketContext) Closer(opener Node) *Closer {
	bc.refresh()
	return bc.info[opener].closer
}

// CRC: crc-BracketContext.md | Seq: seq-pair.md#1.2 | R83, R193
// Opener returns the opener paired with a closer, or nil for an unmatched one.
func (bc *BracketContext) Opener(closer Node) *Opener {
	bc.refresh()
	return bc.info[closer].opener
}

// CRC: crc-BracketContext.md | R194
//
// InnerText returns the bytes between a group's opener and its closer — innerHTML.
// n names the group by being either marker; anything else yields "". A group left
// open at end of input runs to the end of the document.
func (bc *BracketContext) InnerText(n Node) string {
	open, cl := bc.group(n)
	if open == nil {
		return ""
	}
	return bc.text(bc.doc.IndexOf(open)+1, bc.end(cl))
}

// CRC: crc-BracketContext.md | R195
//
// OuterText returns the bytes from a group's opener through its closer — outerHTML.
// Same naming and end-of-input rules as InnerText.
func (bc *BracketContext) OuterText(n Node) string {
	open, cl := bc.group(n)
	if open == nil {
		return ""
	}
	to := bc.end(cl)
	if cl != nil {
		to++ // through the closer; an open group already runs to the end
	}
	return bc.text(bc.doc.IndexOf(open), to)
}

// group resolves n to its opener and closer; the closer is nil for an open group.
func (bc *BracketContext) group(n Node) (*Opener, *Closer) {
	switch m := n.(type) {
	case *Opener:
		return m, bc.Closer(m)
	case *Closer:
		return bc.Opener(m), m
	}
	return nil, nil
}

// end is the index of the closer, or the document's length for an open group.
func (bc *BracketContext) end(cl *Closer) int {
	if cl == nil {
		return len(bc.doc.Nodes())
	}
	return bc.doc.IndexOf(cl)
}

// text renders the nodes in [from, to). A node that cannot render contributes
// nothing; the only such node is one in a poisoned document, which is already lost.
func (bc *BracketContext) text(from, to int) string {
	var b strings.Builder
	for _, n := range bc.doc.Nodes()[from:to] {
		s, _ := n.Render()
		b.WriteString(s)
	}
	return b.String()
}

// CRC: crc-BracketContext.md | Seq: seq-pair.md#1.4 | R152, R196
//
// Separators returns the separators belonging to opener's group, in document order.
// It is the context's own slice (R196).
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

// CRC: crc-BracketContext.md | Seq: seq-declare.md#2.5 | R126, R127, R128, R197
//
// SetDeclarations replaces the declaration links and stamps them against the
// document's current generation. A schema calls it after its pass, OUTSIDE the
// mutation window the pass ran in — reading the generation refuses inside one.
func (bc *BracketContext) SetDeclarations(links map[*DeclarationType][]*DeclarationName) {
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

// CRC: crc-BracketContext.md | Seq: seq-declare.md#2.5 | R127, R128, R196, R198
//
// DeclarationNames returns the names a keyword declares — the document's OWN nodes,
// so a consumer can match one against a node it skimmed, and the context's own
// slice, which the consumer does not write through (R196: documented, not guarded;
// every consumer discards its document within one operation).
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
func (bc *BracketContext) DeclarationNames(kw *DeclarationType) ([]*DeclarationName, error) {
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
	// R87: built, not just the stamp. The parse records no links, so a freshly
	// attached context is EMPTY — and its stamp would match the document's on the
	// first read, reporting a current index that knows nothing. The flag is what
	// makes "never built" different from "built and still fresh".
	if g := bc.doc.Generation(); !bc.built || g != bc.stamp {
		bc.rebuild()
		bc.stamp = g
		bc.built = true
	}
}

// CRC: crc-BracketContext.md | R81, R82, R83, R84, R152, R153
//
// rebuild derives every link by walking the finished flat array with a stack. It
// is the ONLY derivation the library performs: the parse records nothing, because
// every link is already implied by the array and a second copy is a thing that can
// disagree.
//
// R87 is what makes that safe rather than merely cheap — a consumer can walk the
// same array and reach the same answers, so the index is checkable from outside
// rather than an assertion only this package can make.
func (bc *BracketContext) rebuild() {
	bc.info = make(map[Node]BracketInfo, len(bc.info))
	var stack []*Opener
	for _, n := range bc.doc.Nodes() {
		var open *Opener
		if len(stack) > 0 {
			open = stack[len(stack)-1]
		}
		switch m := n.(type) {
		case *Closer:
			// Pair only when this closer belongs to the open group. The group is
			// resolved from the OPENER'S TEXT through the table, not from anything
			// the parse recorded — a node holds no group pointer, and this
			// derivation must stay independent of the one it checks. Without it a
			// stray closer, which the any-close fallback emits unpaired, would be
			// paired here and the two derivations would disagree on every
			// unbalanced file.
			if open != nil && bc.closes(open, m) {
				stack = stack[:len(stack)-1]
				bc.pair(open, m)
			}
		case *Opener:
			if open != nil {
				bc.enclose(m, open)
			}
			stack = append(stack, m)
		default:
			// A Separator lands here, and enclose gives it BOTH directions: it
			// learns its opener, and its opener learns it. Recovering separators
			// from this walk rather than trusting the parse is what keeps them
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
	g := bc.groupOf(ot)
	if g == nil {
		return false
	}
	ct, err := closer.Render()
	if err != nil {
		return false
	}
	if g.CloseIsOpen {
		return ct == ot // R291
	}
	return ct == g.Close
}

// CRC: crc-BracketContext.md | R295
// groupOf resolves an opener's bytes to its group: a literal opener by lookup, a
// pattern group's marker by matching the whole text against its compiled pattern.
func (bc *BracketContext) groupOf(text string) *BracketGroup {
	if g := bc.lang.groupFor(text); g != nil {
		return g
	}
	for i, re := range bc.pats.open {
		if re == nil {
			continue
		}
		if loc := re.FindStringIndex(text); loc != nil && loc[1] == len(text) {
			return &bc.lang.Brackets[i]
		}
	}
	return nil
}

// CRC: crc-BracketContext.md | R299
// Unclosed returns the openers whose group ran to end of input rather than to a
// closer, in document order — derived from the pairing like every other answer here.
// A fence that runs to end of file takes every later heading with it, and this is where
// that is visible.
func (bc *BracketContext) Unclosed() []*Opener {
	bc.refresh()
	var out []*Opener
	for _, n := range bc.doc.Nodes() {
		if o, ok := n.(*Opener); ok && bc.info[o].closer == nil {
			out = append(out, o)
		}
	}
	return out
}

// CRC: crc-BracketContext.md | R310
// Unpaired returns the closers that pair with no opener, in document order: a stray
// closer the any-close fallback emitted, or a run a group rejected as longer than its
// opener. The mirror of Unclosed, and reported the same way.
func (bc *BracketContext) Unpaired() []*Closer {
	bc.refresh()
	var out []*Closer
	for _, n := range bc.doc.Nodes() {
		if c, ok := n.(*Closer); ok && bc.info[c].opener == nil {
			out = append(out, c)
		}
	}
	return out
}

// CRC: crc-BracketContext.md | R349
//
// DemotedOpener is one opener the parse demoted to text: the bytes that opened, the
// text run they were folded into, and their offset within that run. The line is the
// run's start plus the offset, asked of the document when wanted.
type DemotedOpener struct {
	Marker string
	Run    *Text
	Offset int
}

// CRC: crc-BracketContext.md | R349
// Demoted returns the openers the parse demoted to text, in document order — the
// context's own slice, like the other accessors.
func (bc *BracketContext) Demoted() []DemotedOpener { return bc.demoted }

// attach binds the context to the document it was parsed from and stamps it
// with the generation the parse's own links describe, so the first read of a
// freshly parsed document rebuilds nothing.
func (bc *BracketContext) attach(d *Doc) { bc.doc = d }
