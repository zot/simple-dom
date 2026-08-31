package schema

import (
	"regexp"

	"github.com/zot/simple-dom/sdom"
)

// CRC: crc-ShellSchema.md | Seq: seq-declare.md#1 | R136
//
// shellAssignRe announces a shell assignment. There is NO whitespace allowed
// around `=`, and that is not a stylistic choice: `NAME = value` is a command
// invocation, so a lenient `NAME[ \t]*=` would report a command as a declaration.
//
// Lua's rule is the lenient one. The two patterns differ by two characters and by
// what they mean, which is why the difference is written down rather than left to
// whoever copies one into the other.
var shellAssignRe = regexp.MustCompile(`(?:^|[\n;])[ \t]*(?P<name>[\pL_][\pL\pN_]*)=`)

// shellTailRe finds the identifier at the very end of a text node, which is where
// a function's name sits before its `()`.
var shellTailRe = regexp.MustCompile(`[\pL_][\pL\pN_]*$`)

// ShellLang is the Shell schema: no keyword for either shape it recognizes.
var ShellLang = Lang{Bracket: &sdom.LangShell, Comments: []string{"#"}}

const shellSeps = "\n;"

// CRC: crc-ShellSchema.md | Seq: seq-declare.md#2 | R122, R136
//
// Shell runs the shell declaration pass. Both shapes it recognizes are
// keyword-less, so every declaration's DeclarationType is zero-length — a marker
// for where a keyword would have been, and the key its names hang from.
//
// Note that shell's word brackets put much of a file below top level: `if`,
// `while`, `for` and `case` open groups, so assignments inside them are not sought
// by a top-level scan.
func Shell(d *sdom.Doc, ctx *sdom.BracketContext) error {
	links := map[sdom.Node][]sdom.Node{}
	done := map[int]bool{}
	for {
		hit := shellNext(d, ctx, done)
		if hit == nil {
			break
		}
		done[hit.abs] = true

		var kw sdom.Node
		var names []sdom.Node
		err := d.Mutate(func() error {
			kwNode, rest, err := carve(d, hit.node, hit.start, hit.start, newType)
			if err != nil {
				return err
			}
			kw = kwNode
			nm, _, err := carve(d, rest, 0, hit.end-hit.start, newName)
			if err != nil {
				return err
			}
			names = append(names, nm)
			return nil
		})
		if err != nil {
			return err
		}
		links[kw] = names
	}
	ctx.SetDeclarations(links)
	return nil
}

type shellHit struct {
	abs        int
	node       sdom.Node
	start, end int
}

// CRC: crc-ShellSchema.md | Seq: seq-declare.md#1 | R131, R136
//
// shellNext finds the first unhandled declaration of either shape.
//
// The function form is recognized from NODES rather than from a pattern: the parse
// has already turned `foo() {` into a text node, an opener, a closer and another
// opener, and that sequence is what this looks for. No string match over the text
// would see it, because the parens are not in the text any more.
func shellNext(d *sdom.Doc, ctx *sdom.BracketContext, done map[int]bool) *shellHit {
	ni := shellAssignRe.SubexpIndex("name")
	for _, n := range d.Nodes() {
		t, ok := n.(*sdom.Text)
		if !ok || ctx.Enclosing(n) != nil {
			continue
		}
		body, _ := t.Render()

		if m := shellTailRe.FindStringIndex(body); m != nil && shellIsFunc(d, ctx, n) {
			abs := n.Location().Offset() + m[0]
			if !done[abs] {
				return &shellHit{abs: abs, node: n, start: m[0], end: m[1]}
			}
		}
		for _, m := range shellAssignRe.FindAllStringSubmatchIndex(body, -1) {
			ns, ne := m[2*ni], m[2*ni+1]
			abs := n.Location().Offset() + ns
			if done[abs] {
				continue
			}
			if usedCaret(body, m[0], shellSeps) &&
				!ShellLang.precededBySeparator(d, ctx, n, shellSeps) {
				done[abs] = true
				continue
			}
			return &shellHit{abs: abs, node: n, start: ns, end: ne}
		}
	}
	return nil
}

// shellIsFunc reports whether n is followed by an empty paren group, which is what
// makes the identifier ending it a function name rather than a bare word.
func shellIsFunc(d *sdom.Doc, ctx *sdom.BracketContext, n sdom.Node) bool {
	o, ok := d.Next(n).(*sdom.Opener)
	if !ok {
		return false
	}
	if s, _ := o.Render(); s != "(" {
		return false
	}
	c, ok := d.Next(o).(*sdom.Closer)
	return ok && ctx.Opener(c) == sdom.Node(o)
}
