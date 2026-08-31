package schema

import (
	"regexp"
	"strings"

	"github.com/zot/simple-dom/sdom"
)

// CRC: crc-GoSchema.md | Seq: seq-declare.md#1 | R137
//
// goDeclRe announces a Go declaration. The separator is CONSUMED by a
// non-capturing group and the tail is \b, and both are forced rather than chosen:
// Go's regexp supports neither lookbehind nor lookahead, so (?<=[\n;]\s*) and
// (?=\s) do not compile. \b is the better rule anyway — it rejects "constant"
// while admitting a keyword followed by punctuation.
var goDeclRe = regexp.MustCompile(`(?:^|[\n;])[ \t]*(?P<kw>func|var|type|const)\b`)

// goListRe captures a whole name list at the start of a line inside a group, for
// splitting afterwards. It is one group on purpose: a REPEATED capture reports only
// its last iteration, so "B, C, D" would yield B and D and lose C.
var goListRe = regexp.MustCompile(`(?:^|[\n;])[ \t]*(?P<names>[\pL_][\pL\pN_]*(?:[ \t]*,[ \t]*[\pL_][\pL\pN_]*)*)[ \t]*(?:=|[\pL_])`)

// GoLang is the Go schema. TypeScript and JavaScript follow the same shape; every
// Go declaration keyword is ordinary text, which is what makes it the easy case
// and what makes a schema written only against it fail to generalize.
var GoLang = Lang{Bracket: &sdom.LangGo, Comments: []string{"//", "/*"}}

const goSeps = "\n;"

type goHit struct {
	abs                int // document offset of the keyword, stable across carving
	kwNode             sdom.Node
	kwStart, kwEnd     int
	nameNode           sdom.Node // nil when the declaration is a group
	nameStart, nameEnd int
	group              sdom.Node // the single text node inside a grouped declaration
}

// CRC: crc-GoSchema.md | Seq: seq-declare.md#1 | R151
//
// goSameLine reports whether a name reaches its opening paren without crossing a
// newline. That is exactly what semicolon insertion makes required, and it lets
// this schema reject what Go rejects.
//
// Measured against the compiler: "func foo (x int) {" and "func bar /* c */ (x
// int) {" are both LEGAL, so the name need not abut the paren. "func baz\n(x int)"
// is not, and neither is a comment carrying a newline between them — Go treats a
// general comment containing one as a newline.
//
// So the test is over the SOURCE SPAN rather than over nodes: comment bytes are in
// it by construction, which is what makes the multi-line-comment case fall out
// instead of needing its own rule.
func goSameLine(d *sdom.Doc, ctx *sdom.BracketContext, nameNode sdom.Node, nameEnd int) bool {
	o, ok := GoLang.skipForward(d, ctx, d.Next(nameNode)).(*sdom.Opener)
	if !ok {
		// No paren came after the name, and that answer does not depend on
		// whether something else follows the name inside its own node.
		return false
	}
	if s, _ := o.Render(); s != "(" {
		return false
	}
	src := d.Source()
	start := nameNode.Location().Offset() + nameEnd
	end := o.Location().Offset()
	if start < 0 || end > len(src) || start > end {
		return false
	}
	return !strings.ContainsRune(src[start:end], '\n')
}

// CRC: crc-GoSchema.md | Seq: seq-declare.md#1 | R131, R134, R138, R141
//
// goNext finds the first declaration not already handled.
//
// R133: recognition runs over TOP-LEVEL text nodes only, so commented-out code
// cannot match — a comment is a bracket group and its interior is not top level,
// which costs nothing to get.
//
// R142: no indent node is produced anywhere here. Bracket depth already pinpoints
// a declaration, so indentation is irrelevant to recognition; a tool emitting a
// matching indent reads it from the preceding text.
//
// Hits are keyed by absolute document offset rather than by node, because carving
// replaces nodes while leaving every byte where it was.
func goNext(d *sdom.Doc, ctx *sdom.BracketContext, done map[int]bool) *goHit {
	kwi := goDeclRe.SubexpIndex("kw")
	for _, n := range d.Nodes() {
		t, ok := n.(*sdom.Text)
		if !ok || ctx.Enclosing(n) != nil {
			continue
		}
		body, _ := t.Render()
		for _, m := range goDeclRe.FindAllStringSubmatchIndex(body, -1) {
			ks, ke := m[2*kwi], m[2*kwi+1]
			abs := n.Location().Offset() + ks
			if done[abs] {
				continue
			}
			if usedCaret(body, m[0], goSeps) &&
				!GoLang.precededBySeparator(d, ctx, n, goSeps) {
				done[abs] = true
				continue
			}
			keyword := body[ks:ke]
			hit := &goHit{abs: abs, kwNode: n, kwStart: ks, kwEnd: ke}
			if keyword != "func" {
				if g := goGroupBody(d, ctx, n, ke); g != nil {
					hit.group = g
					return hit
				}
			}
			nn, ns, ne := goNameAfter(d, ctx, n, ke)
			if nn == nil {
				done[abs] = true
				continue
			}
			if keyword == "func" && !goSameLine(d, ctx, nn, ne) {
				done[abs] = true
				continue
			}
			hit.nameNode, hit.nameStart, hit.nameEnd = nn, ns, ne
			return hit
		}
	}
	return nil
}

// CRC: crc-GoSchema.md | Seq: seq-declare.md#2.4 | R139
//
// goGroupBody returns the single text node inside a grouped declaration's parens,
// or nil when the keyword introduces no group. The whole group's entries live in
// one node, so its names come from a second pass over it.
func goGroupBody(d *sdom.Doc, ctx *sdom.BracketContext, kwNode sdom.Node, after int) sdom.Node {
	if body, _ := kwNode.Render(); after < len(body) && strings.TrimSpace(body[after:]) != "" {
		return nil // something other than a group follows the keyword
	}
	o, ok := GoLang.skipForward(d, ctx, d.Next(kwNode)).(*sdom.Opener)
	if !ok {
		return nil
	}
	if s, _ := o.Render(); s != "(" {
		return nil
	}
	inner := d.Next(o)
	if _, ok := inner.(*sdom.Text); !ok {
		return nil
	}
	return inner
}

// CRC: crc-GoSchema.md | Seq: seq-declare.md#2 | R122, R124, R125, R126, R127, R128, R140, R148
//
// Go runs the Go declaration pass over an already-scanned document.
//
// One declaration per mutation window, because targets must be resolved BEFORE
// entering one — navigation refuses inside — and carving invalidates the node
// references a later hit in the same node would have needed. Re-scanning per
// declaration is O(n^2) over a document; there is no consumer to measure yet, and
// the alternative threads remainders through hits that may span four nodes.
//
// The links are stamped AFTER the last window closes, since reading the generation
// refuses inside one.
func Go(d *sdom.Doc, ctx *sdom.BracketContext) error {
	links := map[sdom.Node][]sdom.Node{}
	done := map[int]bool{}
	for {
		hit := goNext(d, ctx, done)
		if hit == nil {
			break
		}
		done[hit.abs] = true

		var kw sdom.Node
		var names []sdom.Node
		err := d.Mutate(func() error {
			var rest sdom.Node
			var err error
			if kw, rest, err = carve(d, hit.kwNode, hit.kwStart, hit.kwEnd, newType); err != nil {
				return err
			}
			if hit.group != nil {
				names, err = goCarveGroup(d, hit.group)
				return err
			}
			target, ns, ne := hit.nameNode, hit.nameStart, hit.nameEnd
			if target == hit.kwNode {
				// The name shared the keyword's node, so it now lives in what
				// remained to the right of it.
				target, ns, ne = rest, ns-hit.kwEnd, ne-hit.kwEnd
			}
			nm, _, err := carve(d, target, ns, ne, newName)
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

// CRC: crc-GoSchema.md | Seq: seq-declare.md#2.4 | R139, R140
//
// goCarveGroup carves every name out of a grouped declaration's single interior
// text node, threading the remainder so each carve starts from what the last one
// left. A name list is captured WHOLE and split here, because a repeated capture
// group reports only its last iteration.
func goCarveGroup(d *sdom.Doc, group sdom.Node) ([]sdom.Node, error) {
	body, _ := group.Render()
	li := goListRe.SubexpIndex("names")
	node, base := group, 0
	var names []sdom.Node
	for _, m := range goListRe.FindAllStringSubmatchIndex(body, -1) {
		listStart, listEnd := m[2*li], m[2*li+1]
		list := body[listStart:listEnd]
		for _, im := range identRe.FindAllStringIndex(list, -1) {
			s, e := listStart+im[0], listStart+im[1]
			if node == nil || s < base {
				continue
			}
			nm, rest, err := carve(d, node, s-base, e-base, newName)
			if err != nil {
				return nil, err
			}
			names = append(names, nm)
			node, base = rest, e
		}
	}
	return names, nil
}

// CRC: crc-GoSchema.md | Seq: seq-declare.md#1.4 | R132, R147, R149
//
// goNameAfter walks forward from a keyword to the name it announces, and the skip
// runs before EVERY decision rather than once at the start.
//
// rest is what remains of the keyword's own node after it, which is where a plain
// Go declaration's name lives. An opener met after skipping is a receiver group and
// is jumped to its closer. Skipped whitespace may contain a separator and the walk
// crosses it, because a declaration may span lines.
//
// It returns the node holding the name and the identifier's bounds within it.
func goNameAfter(d *sdom.Doc, ctx *sdom.BracketContext,
	kwNode sdom.Node, after int) (sdom.Node, int, int) {

	if t, ok := kwNode.(*sdom.Text); ok {
		if body, _ := t.Render(); after < len(body) {
			if m := identRe.FindStringIndex(body[after:]); m != nil &&
				strings.TrimSpace(body[after:after+m[0]]) == "" {
				return kwNode, after + m[0], after + m[1]
			}
		}
	}
	n := GoLang.skipForward(d, ctx, d.Next(kwNode))
	if o, ok := n.(*sdom.Opener); ok && !GoLang.isComment(o) {
		c := ctx.Closer(o)
		if c == nil {
			return nil, 0, 0
		}
		n = GoLang.skipForward(d, ctx, d.Next(c))
	}
	t, ok := n.(*sdom.Text)
	if !ok {
		return nil, 0, 0
	}
	body, _ := t.Render()
	m := identRe.FindStringIndex(body)
	if m == nil {
		return nil, 0, 0
	}
	return n, m[0], m[1]
}
