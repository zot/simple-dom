package schema

import (
	"regexp"

	"github.com/zot/simple-dom/sdom"
)

// CRC: crc-PythonSchema.md | Seq: seq-declare.md#1 | R137, R189
//
// pyDeclRe announces a Python declaration. It is Go's shape with Python's keywords:
// the separator is consumed by a non-capturing group and the tail is \b, both forced
// because Go's regexp has neither lookbehind nor lookahead.
var pyDeclRe = regexp.MustCompile(`(?:^|[\n;])[ \t]*(?P<kw>def|class)\b`)

// CRC: crc-PythonSchema.md | R169
//
// PythonLang is the Python schema. It carries no comment markers of its own: the
// table marks its comment group with a Kind, and isComment reads it back.
var PythonLang = Lang{Bracket: &sdom.LangPython.BracketLang}

const pySeps = "\n;"

// pyHit is narrower than Go's. There is no receiver group to step over and no
// grouped form, so the name is always inside the keyword's own text node —
// `def` followed by a newline is a syntax error, verified against CPython, so a
// schema that insisted on the same node would be right to.
type pyHit struct {
	abs                int // document offset of the keyword, stable across carving
	kwNode             sdom.Node
	kwStart, kwEnd     int
	nameStart, nameEnd int
}

// CRC: crc-PythonSchema.md | Seq: seq-declare.md#1 | R134, R190, R191
//
// pyNext finds the first declaration not already handled.
//
// R190: the predicate is Enclosing(n) == nil — BRACKET depth 0 — and it is
// unchanged from Go's. A Python method sits inside its class body, which is an
// INDENT FRAME rather than a bracket group, so it still has no bracket enclosing it
// however deeply indented. That is the whole reason "top level" always meant bracket
// depth, and Python is the first language that shows it.
//
// R191: which depths a language's declarations live at is a fact about that
// language. Go is depth 0 throughout, Java puts types at the top and members inside
// them, JavaScript has no depth rule at all. That is why this is a per-schema pass
// and not a driver with a depth setting.
func pyNext(d *sdom.Doc, ctx *sdom.BracketContext, done map[int]bool) *pyHit {
	kwi := pyDeclRe.SubexpIndex("kw")
	for _, n := range d.Nodes() {
		t, ok := n.(*sdom.Text)
		if !ok || ctx.Enclosing(n) != nil {
			continue
		}
		body, _ := t.Render()
		for _, m := range pyDeclRe.FindAllStringSubmatchIndex(body, -1) {
			ks, ke := m[2*kwi], m[2*kwi+1]
			abs := n.Location().Offset() + ks
			if done[abs] {
				continue
			}
			// R134: a match at the head of a text node begins a statement only when
			// the preceding top-level content ends with a separator. Reached often
			// here, because a comment's closing newline is a Closer rather than a
			// byte of text — so `# c` then `def f` puts the keyword at the start of
			// its node with no separator in front of it.
			if usedCaret(body, m[0], pySeps) &&
				!PythonLang.precededBySeparator(d, ctx, n, pySeps) {
				done[abs] = true
				continue
			}
			ns, ne, ok := pyNameAfter(body, ke)
			if !ok {
				done[abs] = true
				continue
			}
			return &pyHit{abs: abs, kwNode: n, kwStart: ks, kwEnd: ke, nameStart: ns, nameEnd: ne}
		}
	}
	return nil
}

// CRC: crc-PythonSchema.md | Seq: seq-declare.md#1.4 | R132, R148, R189
//
// pyNameAfter finds the identifier following a keyword within the keyword's own
// node, and only there. Nothing may separate them but spaces and tabs: Python has no
// block comment, and a newline between `def` and its name does not compile.
//
// R148: the bounds are the identifier's alone, so slicing it out splits the node at
// both edges rather than retyping it whole with its surrounding whitespace.
func pyNameAfter(body string, after int) (start, end int, ok bool) {
	if after >= len(body) {
		return 0, 0, false
	}
	m := identRe.FindStringIndex(body[after:])
	if m == nil {
		return 0, 0, false
	}
	for _, c := range body[after : after+m[0]] {
		if c != ' ' && c != '\t' {
			return 0, 0, false
		}
	}
	return after + m[0], after + m[1], true
}

// CRC: crc-PythonSchema.md | Seq: seq-declare.md#2 | R122, R124, R125, R126, R127, R128, R189
//
// Python runs the Python declaration pass over an already-parsed document.
//
// One declaration per mutation window, for the reason Go's pass records: targets
// must be resolved before entering one, because navigation refuses inside, and
// carving invalidates the node references a later hit in the same node would need.
//
// The keyword and the name share a node here — always, unlike Go — so the carve of
// the name runs against what the keyword's carve left to its right.
func Python(d *sdom.Doc, ctx *sdom.BracketContext) error {
	links := map[sdom.Node][]sdom.Node{}
	done := map[int]bool{}
	for {
		hit := pyNext(d, ctx, done)
		if hit == nil {
			break
		}
		done[hit.abs] = true

		var kw, name sdom.Node
		err := d.Mutate(func() error {
			var rest sdom.Node
			var err error
			if kw, rest, err = carve(d, hit.kwNode, hit.kwStart, hit.kwEnd, newType); err != nil {
				return err
			}
			name, _, err = carve(d, rest, hit.nameStart-hit.kwEnd, hit.nameEnd-hit.kwEnd, newName)
			return err
		})
		if err != nil {
			return err
		}
		links[kw] = []sdom.Node{name}
	}
	ctx.SetDeclarations(links)
	return nil
}
