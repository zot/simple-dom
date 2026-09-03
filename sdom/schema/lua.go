package schema

import (
	"regexp"

	"github.com/zot/simple-dom/sdom"
)

// CRC: crc-LuaSchema.md | Seq: seq-declare.md#1.3 | R131, R135
//
// luaDeclRe announces a Lua declaration two ways in ONE alternation: the keyword
// `local`, and a keyword-less global assignment. The branch that did not
// participate yields the zero location, which is how the schema tells which fired
// with no extra machinery — absence is the zero value, and this is the case the
// stencil builder's Group was built for.
//
// The name class admits `.` and `:` because `M.f = 2` and `M:m` are how a Lua
// module defines its members; \w+ would miss every one of them.
//
// Whitespace around `=` is PERMITTED here and forbidden in shell. The two look
// alike and are not: in shell, `NAME = value` is a command invocation.
var luaDeclRe = regexp.MustCompile(
	`(?:^|[\n;])[ \t]*(?:(?P<kw>local)\b|(?P<name>[\pL_][\pL\pN_.:]*)[ \t]*=)`)

// LuaLang is the Lua schema — the case that breaks anything written against Go,
// because `function` is a BRACKET OPENER rather than text and the name it
// introduces sits inside the group it opened.
var LuaLang = Lang{Bracket: &sdom.LangLua}

const luaSeps = "\n;"

// CRC: crc-LuaSchema.md | Seq: seq-declare.md#1.4 | R132
//
// luaNameAfter walks forward to a name. Unlike Go's walk it does NOT treat an
// opener as a receiver group to jump: Lua has no receivers, and the one opener it
// meets is `function` itself, which is part of the announcement rather than
// something between the announcement and the name. So it steps over that and takes
// the next text with content — which lies INSIDE the group the keyword opened,
// and needs no special case because the array is flat.
func luaNameAfter(d *sdom.Doc, ctx *sdom.BracketContext, from sdom.Node, after int) (sdom.Node, int, int) {
	if t, ok := from.(*sdom.Text); ok {
		if body, _ := t.Render(); after < len(body) {
			if m := luaIdentRe.FindStringIndex(body[after:]); m != nil {
				return from, after + m[0], after + m[1]
			}
		}
	}
	n := LuaLang.skipForward(d, ctx, d.Next(from))
	if o, ok := n.(*sdom.Opener); ok {
		if s, _ := o.Render(); s == "function" {
			n = LuaLang.skipForward(d, ctx, d.Next(o))
		}
	}
	t, ok := n.(*sdom.Text)
	if !ok {
		return nil, 0, 0
	}
	body, _ := t.Render()
	m := luaIdentRe.FindStringIndex(body)
	if m == nil {
		return nil, 0, 0
	}
	return n, m[0], m[1]
}

// luaIdentRe admits the dotted and colon forms a module uses.
var luaIdentRe = regexp.MustCompile(`[\pL_][\pL\pN_.:]*`)

// CRC: crc-LuaSchema.md | Seq: seq-declare.md#2 | R122, R131, R135
//
// Lua runs the Lua declaration pass. It recognizes three shapes: `local`, a bare
// `function` opener at a statement start, and a keyword-less assignment.
func Lua(d *sdom.Doc, ctx *sdom.BracketContext) error {
	links := map[*sdom.DeclarationType][]*sdom.DeclarationName{}
	done := map[int]bool{}
	for {
		hit := luaNext(d, ctx, done)
		if hit == nil {
			break
		}
		done[hit.abs] = true

		var kw *sdom.DeclarationType
		var names []*sdom.DeclarationName
		err := d.Mutate(func() error {
			var rest sdom.Node
			var err error
			if hit.kwIsNode {
				// `function` is already its own node: re-type it whole.
				body, _ := hit.kwNode.Render()
				kw = sdom.NewDeclarationType(body, hit.kwNode.Location())
				err = d.Replace(hit.kwNode, kw)
			} else {
				kw, rest, err = carve(d, hit.kwNode, hit.kwStart, hit.kwEnd, sdom.NewDeclarationType)
			}
			if err != nil {
				return err
			}
			target, ns, ne := hit.nameNode, hit.nameStart, hit.nameEnd
			if target == hit.kwNode && !hit.kwIsNode {
				target, ns, ne = rest, ns-hit.kwEnd, ne-hit.kwEnd
			}
			nm, _, err := carve(d, target, ns, ne, sdom.NewDeclarationName)
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

type luaHit struct {
	abs                int
	kwNode             sdom.Node
	kwIsNode           bool // the announcement IS a node, not a substring
	kwStart, kwEnd     int
	nameNode           sdom.Node
	nameStart, nameEnd int
}

// CRC: crc-LuaSchema.md | Seq: seq-declare.md#1.3 | R131
//
// luaNext finds the first unhandled declaration. It tests node KINDS as well as
// matching text, because Lua's `function` is an opener and no text pattern can
// ever see it — which is the whole reason recognition is per-schema.
func luaNext(d *sdom.Doc, ctx *sdom.BracketContext, done map[int]bool) *luaHit {
	kwi, ni := luaDeclRe.SubexpIndex("kw"), luaDeclRe.SubexpIndex("name")
	for _, n := range d.Nodes() {
		if ctx.Enclosing(n) != nil {
			continue
		}
		if o, ok := n.(*sdom.Opener); ok {
			s, _ := o.Render()
			abs := o.Location().Offset()
			if s != "function" || done[abs] {
				continue
			}
			nn, ns, ne := luaNameAfter(d, ctx, o, 0)
			if nn == nil {
				done[abs] = true
				continue
			}
			return &luaHit{abs: abs, kwNode: o, kwIsNode: true,
				nameNode: nn, nameStart: ns, nameEnd: ne}
		}
		t, ok := n.(*sdom.Text)
		if !ok {
			continue
		}
		body, _ := t.Render()
		for _, m := range luaDeclRe.FindAllStringSubmatchIndex(body, -1) {
			ks, ke := m[2*kwi], m[2*kwi+1]
			isKw := ks >= 0
			if !isKw {
				ks, ke = m[2*ni], m[2*ni+1]
			}
			abs := n.Location().Offset() + ks
			if done[abs] {
				continue
			}
			if usedCaret(body, m[0], luaSeps) &&
				!LuaLang.precededBySeparator(d, ctx, n, luaSeps) {
				done[abs] = true
				continue
			}
			if !isKw {
				// The keyword-less branch: the match already IS the name, and the
				// declaration has no keyword node of its own.
				return &luaHit{abs: abs, kwNode: n, kwStart: ks, kwEnd: ks,
					nameNode: n, nameStart: ks, nameEnd: ke}
			}
			nn, ns, ne := luaNameAfter(d, ctx, n, ke)
			if nn == nil {
				done[abs] = true
				continue
			}
			return &luaHit{abs: abs, kwNode: n, kwStart: ks, kwEnd: ke,
				nameNode: nn, nameStart: ns, nameEnd: ne}
		}
	}
	return nil
}
