package sdom

import "strings"

// CRC: crc-Lexer.md | Seq: seq-scan.md | R57, R77
//
// Scan tokenizes src with lang into a flat, document-order array of nodes, and
// returns the document together with the context owning its pairing links.
//
// base is the document's own position within an outer document; node offsets are
// relative to src.
func Scan(src string, base int, lang *BracketLang) (*Doc, *BracketContext) {
	lx := &lexer{src: src, lang: lang, ctx: newBracketContext(lang)}
	lx.scanBody(nil, nil)
	d := New(src, base, lx.out...)
	lx.ctx.attach(d)
	return d, lx.ctx
}

// lexer walks the source once, appending nodes in the order it meets them.
//
// The nesting lives on the CALL STACK — scanBody recurses — and is simply absent
// from the data afterwards, which is what makes the emitted array flat without
// anything having to flatten it. The stack field exists only so an emitted node
// can be told which opener contains it.
type lexer struct {
	src       string
	pos       int
	textStart int
	lang      *BracketLang
	ctx       *BracketContext
	out       []Node
	stack     []Node
}

// CRC: crc-Lexer.md | Seq: seq-scan.md#1.5 | R76, R77
// flushText emits the bytes accumulated since the last marker. Whitespace is not
// a token, so a text run is everything between two recognized markers.
func (lx *lexer) flushText() {
	if lx.pos > lx.textStart {
		lx.emit(NewText(lx.src[lx.textStart:lx.pos], Source(lx.textStart, lx.pos-lx.textStart)))
	}
	lx.textStart = lx.pos
}

// emit appends a node and records the opener enclosing it. A closer is paired
// with its opener instead, by the caller.
func (lx *lexer) emit(n Node) {
	if _, isCloser := n.(*Closer); !isCloser && len(lx.stack) > 0 {
		lx.ctx.enclosing[n] = lx.stack[len(lx.stack)-1]
	}
	lx.out = append(lx.out, n)
}

// take emits a marker, advances past it, and reopens the text run. The width
// comes from the marker's own location, so the two cannot disagree.
func (lx *lexer) take(n Node) {
	lx.flushText()
	lx.emit(n)
	lx.pos += n.Location().Length()
	lx.textStart = lx.pos
}

// closeGroup ends the open group when its closer is at pos, pairing the two
// nodes in both directions, and reports whether it did. Code mode and restricted
// mode differ in what they recognize but end a group identically, so both call
// this rather than each writing the pairing out.
// CRC: crc-Lexer.md | Seq: seq-scan.md#1.4 | R82, R83
//
// closeGroup ends the open group if one of its closers matches here, recording
// the pairing BOTH WAYS from the recursion that produced it. This is one of the
// two independent derivations of the links; the other is BracketContext.rebuild,
// which must never reuse anything recorded here.
func (lx *lexer) closeGroup(g *BracketGroup, opener Node) bool {
	m := matchAny(lx.src, lx.pos, g.Close)
	if m == "" {
		return false
	}
	c := NewCloser(m, Source(lx.pos, len(m)))
	lx.take(c)
	lx.ctx.closerOf[opener] = c
	lx.ctx.openerOf[c] = opener
	return true
}

// CRC: crc-Lexer.md | Seq: seq-scan.md#1 | R64, R65
// scanBody scans until enclosing's closer is found, or to end of input. enclosing
// is nil at top level.
func (lx *lexer) scanBody(enclosing *BracketGroup, opener Node) {
	if enclosing != nil && enclosing.Restricted() {
		lx.scanRestricted(enclosing, opener)
		return
	}
	lx.scanCode(enclosing, opener)
}

// CRC: crc-Lexer.md | Seq: seq-scan.md#1.3 | R72, R73, R74, R75
//
// scanCode scans in code mode: openers of any group allowed here, then the open
// group's closers, then its separators, then the any-close fallback, then text.
func (lx *lexer) scanCode(enclosing *BracketGroup, opener Node) {
	for lx.pos < len(lx.src) {
		if g, m := lx.matchOpen(enclosing); g != nil {
			lx.open(g, m)
			continue
		}
		if enclosing != nil {
			if lx.closeGroup(enclosing, opener) {
				return
			}
			if m := matchAny(lx.src, lx.pos, enclosing.Separators); m != "" {
				lx.take(NewSeparator(m, Source(lx.pos, len(m))))
				continue
			}
		}
		// The any-close fallback: a stray closer lands as a bracket rather than
		// derailing the scan.
		if m := lx.matchAnyClose(); m != "" {
			lx.take(NewCloser(m, Source(lx.pos, len(m))))
			continue
		}
		// Nothing matched here, so this byte is text. Advancing unconditionally is
		// what guarantees the scan always consumes at least one byte.
		lx.pos++
	}
	lx.flushText() // a group left open at end of input closes there; no bytes drop
}

// CRC: crc-Lexer.md | Seq: seq-scan.md#2 | R65
//
// scanRestricted scans inside a string or a comment: only this group's Close, its
// Escape, and the openers named in AllowedInner are recognized. Every other byte
// is literal — comments inside strings are not comments, and brackets inside
// comments are not brackets.
func (lx *lexer) scanRestricted(g *BracketGroup, opener Node) {
	for lx.pos < len(lx.src) {
		if lx.closeGroup(g, opener) {
			return
		}
		if g.Escape != "" && strings.HasPrefix(lx.src[lx.pos:], g.Escape) {
			lx.pos += len(g.Escape)
			if lx.pos < len(lx.src) {
				lx.pos++ // the escaped byte is literal, whatever it is
			}
			continue
		}
		if inner, m := lx.matchInner(g); inner != nil {
			lx.open(inner, m)
			continue
		}
		lx.pos++
	}
	lx.flushText()
}

// open emits an opener for g and scans its body.
func (lx *lexer) open(g *BracketGroup, marker string) {
	o := NewOpener(marker, Source(lx.pos, len(marker)))
	lx.take(o)
	lx.stack = append(lx.stack, o)
	lx.scanBody(g, o)
	lx.stack = lx.stack[:len(lx.stack)-1]
}

// CRC: crc-Lexer.md | R66
// matchOpen finds the first group whose opener matches here and whose
// AllowedParent permits the enclosing group.
func (lx *lexer) matchOpen(enclosing *BracketGroup) (*BracketGroup, string) {
	for i := range lx.lang.Brackets {
		g := &lx.lang.Brackets[i]
		if !g.parentAllowed(enclosing) {
			continue
		}
		if m := matchAny(lx.src, lx.pos, g.Open); m != "" {
			return g, m
		}
	}
	return nil, ""
}

// CRC: crc-Lexer.md | R65
// matchInner finds an opener named in g.AllowedInner, and the group owning it.
func (lx *lexer) matchInner(g *BracketGroup) (*BracketGroup, string) {
	for _, op := range g.AllowedInner {
		if !matchAt(lx.src, lx.pos, op) {
			continue
		}
		if owner := lx.lang.groupFor(op); owner != nil {
			return owner, op
		}
	}
	return nil, ""
}

// CRC: crc-Lexer.md | Seq: seq-scan.md#3.1 | R73
// matchAnyClose recognizes any code-mode group's closer, so depth stays
// consistent even when the document is unbalanced.
func (lx *lexer) matchAnyClose() string {
	for i := range lx.lang.Brackets {
		g := &lx.lang.Brackets[i]
		if g.Restricted() {
			continue
		}
		if m := matchAny(lx.src, lx.pos, g.Close); m != "" {
			return m
		}
	}
	return ""
}
