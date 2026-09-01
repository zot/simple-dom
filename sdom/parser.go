package sdom

import "strings"

// CRC: crc-BracketParser.md | Seq: seq-parse.md | R57, R77
//
// Parse parses src with lang into a flat, document-order array of nodes, and
// returns the document together with the context owning its pairing links.
//
// base is the document's own position within an outer document; node offsets are
// relative to src.
func Parse(src string, base int, lang *BracketLang) (*Doc, *BracketContext) {
	p := &parser{src: src, lang: lang, ctx: newBracketContext(lang)}
	p.parseBody(nil, nil)
	d := New(src, base, p.out...)
	p.ctx.attach(d)
	return d, p.ctx
}

// parser walks the source once, appending nodes in the order it meets them.
//
// The nesting lives on the CALL STACK — parseBody recurses — and is simply absent
// from the data afterwards, which is what makes the emitted array flat without
// anything having to flatten it. The stack field exists only so an emitted node
// can be told which opener contains it.
type parser struct {
	src       string
	pos       int
	textStart int
	lang      *BracketLang
	ctx       *BracketContext
	out       []Node
	stack     []Node
}

// CRC: crc-BracketParser.md | R89, R91
// at builds a location for a span of this parse, attributed to its origin.
func (p *parser) at(pos, length int) Loc {
	return Source(pos, length).In(p.ctx.origin)
}

// CRC: crc-BracketParser.md | Seq: seq-parse.md#1.5 | R76, R77
// flushText emits the bytes accumulated since the last marker. Whitespace is not
// a node of its own, so a text run is everything between two recognized markers.
func (p *parser) flushText() {
	if p.pos > p.textStart {
		p.emit(NewText(p.src[p.textStart:p.pos], p.at(p.textStart, p.pos-p.textStart)))
	}
	p.textStart = p.pos
}

// emit appends a node and records the opener enclosing it. A closer is paired
// with its opener instead, by the caller.
func (p *parser) emit(n Node) {
	if _, isCloser := n.(*Closer); !isCloser && len(p.stack) > 0 {
		p.ctx.enclose(n, p.stack[len(p.stack)-1])
	}
	p.out = append(p.out, n)
}

// take emits a marker, advances past it, and reopens the text run. The width
// comes from the marker's own location, so the two cannot disagree.
func (p *parser) take(n Node) {
	p.flushText()
	p.emit(n)
	p.pos += n.Location().Length()
	p.textStart = p.pos
}

// CRC: crc-BracketParser.md | Seq: seq-parse.md#1.4 | R82, R83
//
// closeGroup ends the open group when one of its closers is at pos, pairing the
// two nodes BOTH WAYS, and reports whether it did. Code mode and restricted mode
// differ in what they recognize but end a group identically, so both call this
// rather than each writing the pairing out.
//
// This is one of the two independent derivations of the links — from the
// recursion that produced them. The other is BracketContext.rebuild, which must
// never reuse anything recorded here.
func (p *parser) closeGroup(g *BracketGroup, opener Node) bool {
	m := matchAny(p.src, p.pos, g.Close)
	if m == "" {
		return false
	}
	c := NewCloser(m, p.at(p.pos, len(m)))
	p.take(c)
	p.ctx.pair(opener, c)
	return true
}

// CRC: crc-BracketParser.md | Seq: seq-parse.md#1 | R64, R65
// parseBody parses until enclosing's closer is found, or to end of input. enclosing
// is nil at top level.
func (p *parser) parseBody(enclosing *BracketGroup, opener Node) {
	if enclosing != nil && enclosing.Restricted() {
		p.parseRestricted(enclosing, opener)
		return
	}
	p.parseCode(enclosing, opener)
}

// CRC: crc-BracketParser.md | Seq: seq-parse.md#1.3 | R72, R73, R74, R75
//
// parseCode parses in code mode: openers of any group allowed here, then the open
// group's closers, then its separators, then the any-close fallback, then text.
func (p *parser) parseCode(enclosing *BracketGroup, opener Node) {
	for p.pos < len(p.src) {
		if g, m := p.matchOpen(enclosing); g != nil {
			p.open(g, m)
			continue
		}
		if enclosing != nil {
			if p.closeGroup(enclosing, opener) {
				return
			}
			if m := matchAny(p.src, p.pos, enclosing.Separators); m != "" {
				p.take(NewSeparator(m, p.at(p.pos, len(m))))
				continue
			}
		}
		// The any-close fallback: a stray closer lands as a bracket rather than
		// derailing the parse.
		if m := p.matchAnyClose(); m != "" {
			p.take(NewCloser(m, p.at(p.pos, len(m))))
			continue
		}
		// Nothing matched here, so this byte is text. Advancing unconditionally is
		// what guarantees the parse always consumes at least one byte.
		p.pos++
	}
	p.flushText() // a group left open at end of input closes there; no bytes drop
}

// CRC: crc-BracketParser.md | Seq: seq-parse.md#2 | R65
//
// parseRestricted parses inside a string or a comment: only this group's Close, its
// Escape, and the openers named in AllowedInner are recognized. Every other byte
// is literal — comments inside strings are not comments, and brackets inside
// comments are not brackets.
func (p *parser) parseRestricted(g *BracketGroup, opener Node) {
	for p.pos < len(p.src) {
		if p.closeGroup(g, opener) {
			return
		}
		if g.Escape != "" && strings.HasPrefix(p.src[p.pos:], g.Escape) {
			p.pos += len(g.Escape)
			if p.pos < len(p.src) {
				p.pos++ // the escaped byte is literal, whatever it is
			}
			continue
		}
		if inner, m := p.matchInner(g); inner != nil {
			p.open(inner, m)
			continue
		}
		p.pos++
	}
	p.flushText()
}

// open emits an opener for g and parses its body.
func (p *parser) open(g *BracketGroup, marker string) {
	o := NewOpener(marker, p.at(p.pos, len(marker)))
	p.take(o)
	p.stack = append(p.stack, o)
	p.parseBody(g, o)
	p.stack = p.stack[:len(p.stack)-1]
}

// CRC: crc-BracketParser.md | R66
// matchOpen finds the first group whose opener matches here and whose
// AllowedParent permits the enclosing group.
func (p *parser) matchOpen(enclosing *BracketGroup) (*BracketGroup, string) {
	for i := range p.lang.Brackets {
		g := &p.lang.Brackets[i]
		if !g.parentAllowed(enclosing) {
			continue
		}
		if m := matchAny(p.src, p.pos, g.Open); m != "" {
			return g, m
		}
	}
	return nil, ""
}

// CRC: crc-BracketParser.md | R65
// matchInner finds an opener named in g.AllowedInner, and the group owning it.
func (p *parser) matchInner(g *BracketGroup) (*BracketGroup, string) {
	for _, op := range g.AllowedInner {
		if !matchAt(p.src, p.pos, op) {
			continue
		}
		if owner := p.lang.groupFor(op); owner != nil {
			return owner, op
		}
	}
	return nil, ""
}

// CRC: crc-BracketParser.md | Seq: seq-parse.md#3.1 | R73
// matchAnyClose recognizes any code-mode group's closer, so depth stays
// consistent even when the document is unbalanced.
func (p *parser) matchAnyClose() string {
	for i := range p.lang.Brackets {
		g := &p.lang.Brackets[i]
		if g.Restricted() {
			continue
		}
		if m := matchAny(p.src, p.pos, g.Close); m != "" {
			return m
		}
	}
	return ""
}
