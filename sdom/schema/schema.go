// Package schema holds the bundled declaration schemas. Each knows how ONE
// language announces a declaration and drives sdom's machinery with that
// knowledge; nothing here knows what a CRC card is.
//
// It is a separate package from sdom on purpose: driving the machinery from
// outside is what finds the accessors and constructors sdom's export surface is
// missing, which building the schemas inside it would have hidden.
package schema

import (
	"regexp"
	"slices"
	"strings"

	"github.com/zot/simple-dom/sdom"
)

// CRC: crc-DeclSchema.md | R129, R130
//
// Lang is what one language contributes.
//
// R144: there is deliberately no shared recognition rule and no driver interface.
// Each schema carries its own pass, and the duplication between them is the
// evidence a later part reads; lifting the common half into a reusable tool waits
// until three schemas exist to generalize from, because extracting from one is
// guessing and the guess would be built into an export surface.
//
// R150: sdom neither validates text nor requires a schema to be lax about it.
// Nothing here is a syntax checker, and how strict a schema's own walk is, is that
// schema's choice — Go's rejects what the Go compiler rejects, Lua's accepts the
// same layout because Lua does.
//
// R143: which declarations OUGHT to carry a traceability comment is not decided
// here and is not decidable here. That is a reader's policy, and mini-spec's
// approaches it from unanchored requirements rather than by filtering these.
type Lang struct {
	Bracket  *sdom.BracketLang
	Comments []string // this language's comment openers — see below
}

// CRC: crc-DeclSchema.md | R146
//
// isComment reports whether n opens a comment. The language table cannot answer
// this: a comment and a string are both parse-restricted, and there is deliberately
// no comment configuration. So a schema matches the opener against the markers it
// knows, which is language knowledge and belongs here.
func (l Lang) isComment(n sdom.Node) bool {
	o, ok := n.(*sdom.Opener)
	if !ok {
		return false
	}
	s, err := o.Render()
	if err != nil {
		return false
	}
	return slices.Contains(l.Comments, s)
}

// isSpace reports whether n is a text node holding only whitespace. Such a node is
// stepped over — note it may CONTAIN a statement separator, which the forward walk
// crosses (R149).
func isSpace(n sdom.Node) bool {
	t, ok := n.(*sdom.Text)
	if !ok {
		return false
	}
	s, err := t.Render()
	return err == nil && strings.TrimSpace(s) == ""
}

// CRC: crc-DeclSchema.md | Seq: seq-declare.md#1.4 | R145, R147
//
// skipForward advances past whole comment groups and whitespace-only nodes. A
// comment goes anywhere a space goes, so this runs before EVERY decision in a
// walk, not once at its start.
func (l Lang) skipForward(d *sdom.Doc, ctx *sdom.BracketContext, n sdom.Node) sdom.Node {
	for n != nil {
		switch {
		case isSpace(n):
			n = d.Next(n)
		case l.isComment(n):
			if c := ctx.Closer(n); c != nil {
				n = d.Next(c)
			} else {
				return nil // unterminated comment: nothing follows it
			}
		default:
			return n
		}
	}
	return nil
}

// CRC: crc-DeclSchema.md | Seq: seq-declare.md#1.2.2 | R134, R145
//
// precededBySeparator reports whether a statement ended before n.
//
// There is deliberately no skipBack to mirror skipForward. One was written and
// turned out to be the wrong shape: skipping backward and THEN looking is what
// hides the answer, since the whitespace stepped over is often the separator
// itself. What replaced it examines each node as it passes.
//
// It walks backward over comments and whitespace — a declaration written under a
// comment begins a text node with no separator in front of it, because the
// comment's closing newline is a CLOSER rather than a byte in the following text.
//
// But it does NOT simply skip and then look: skipped whitespace may CONTAIN the
// separator, and stepping over it destroys the evidence being sought. `}` ⏎⏎
// `// c` ⏎ `func New(` is the ordinary shape of a Go file, and skipping to the `}`
// finds no separator there while the blank line between them was the answer. So
// each node is examined as it is passed, and a comment's own closer counts too,
// since a line comment closes on a newline.
//
// Nothing before it means the document began, which counts.
func (l Lang) precededBySeparator(d *sdom.Doc, ctx *sdom.BracketContext, n sdom.Node, seps string) bool {
	for p := d.Prev(n); p != nil; {
		if isSpace(p) {
			body, _ := p.Render()
			if strings.ContainsAny(body, seps) {
				return true
			}
			p = d.Prev(p)
			continue
		}
		if c, ok := p.(*sdom.Closer); ok {
			if o := ctx.Opener(c); o != nil && l.isComment(o) {
				if body, _ := c.Render(); strings.ContainsAny(body, seps) {
					return true
				}
				p = d.Prev(o)
				continue
			}
		}
		body, err := p.Render()
		return err == nil && body != "" &&
			strings.ContainsRune(seps, rune(body[len(body)-1]))
	}
	return true // the document began
}

// usedCaret reports whether a match at m0 in body leaned on the pattern's ^ branch
// rather than consuming a separator — which is exactly when the backward test is
// needed.
func usedCaret(body string, m0 int, seps string) bool {
	return m0 == 0 && (body == "" || !strings.ContainsRune(seps, rune(body[0])))
}

// CRC: crc-DeclSchema.md | Seq: seq-declare.md#2.3 | R148
//
// carve splits [start,end) out of a text node and re-types that middle, returning
// the new node and whatever remains to its right.
//
// The middle is what matters: a name arrives inside a node carrying whitespace on
// both sides — Text " Index " — so BOTH edges are split. Re-typing the node whole
// would put the spaces inside the DeclarationName, which renders identically and is
// wrong. Returning the right remainder is what lets a caller carve a second thing
// out of the same node, which happens whenever a keyword and its name share one.
//
// It must run inside a mutation window.
func carve(d *sdom.Doc, n sdom.Node, start, end int,
	mk func(string, sdom.Loc) sdom.Node) (sdom.Node, sdom.Node, error) {

	mid := n
	var err error
	if start > 0 {
		if _, mid, err = d.Split(n, start); err != nil {
			return nil, nil, err
		}
	}
	var right sdom.Node
	if t, ok := mid.(*sdom.Text); ok {
		if body, _ := t.Render(); end-start < len(body) {
			if mid, right, err = d.Split(mid, end-start); err != nil {
				return nil, nil, err
			}
		}
	}
	body, _ := mid.Render()
	out := mk(body, mid.Location())
	if err = d.Replace(mid, out); err != nil {
		return nil, nil, err
	}
	return out, right, nil
}

func newType(s string, l sdom.Loc) sdom.Node { return sdom.NewDeclarationType(s, l) }
func newName(s string, l sdom.Loc) sdom.Node { return sdom.NewDeclarationName(s, l) }

// identRe finds an identifier. \pL rather than \w so a name is not ASCII-only.
var identRe = regexp.MustCompile(`[\pL_][\pL\pN_]*`)
