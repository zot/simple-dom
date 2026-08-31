package sdom

import (
	"fmt"
	"regexp"
)

// CRC: crc-StencilBuilder.md | Seq: seq-stencil.md | R96, R97, R99, R107
//
// StencilBuilder turns a regex match into a tiled child list. It is a BUILDER,
// never a Node — "stencil" alone names the region a tool writes into, and this is
// the tool that cuts it.
//
// A schema drives it: the machinery lays out the bytes, and the schema decides
// what each named group becomes. That division is the point. A helper that
// returned, say, a Text per group would impose a shape on every schema forever,
// and a group that should become a nested compound or a declaration could not.
//
// It returns no name-to-node map. A schema assigns every node to its own field as
// it builds it, so handing the same references back would be a second copy of
// something that only flows one way.
type StencilBuilder struct {
	text  string
	loc   Loc
	match []int
	names []string

	// A participating named group starts with NO entry here rather than defaulting
	// to a Text: a group exists because a schema binds it, so one named and never
	// filled should not have been named, and Done makes that omission loud where a
	// default would swallow it (R99).
	slot map[string]Node
	omit map[string]bool
}

// CRC: crc-StencilBuilder.md | R97
//
// NewStencilBuilder matches re against text and prepares to build. It reports
// false when the regex does not match; what that means is the schema's decision,
// not the builder's.
//
// loc is where text sits — its offset within the document and the parse it came
// from — so every location this builds is in the document's coordinate system.
func NewStencilBuilder(re *regexp.Regexp, text string, loc Loc) (*StencilBuilder, bool) {
	m := re.FindStringSubmatchIndex(text)
	if m == nil {
		return nil, false
	}
	return &StencilBuilder{
		text:  text,
		loc:   loc,
		match: m,
		names: re.SubexpNames(),
		slot:  map[string]Node{},
		omit:  map[string]bool{},
	}, true
}

// CRC: crc-StencilBuilder.md | R110
// span returns the location of text[start:end], in the document's coordinates.
func (b *StencilBuilder) span(start, end int) Loc {
	if b.loc.Offset() < 0 {
		return Synthetic(end - start).In(b.loc.Origin())
	}
	return Source(b.loc.Offset()+start, end-start).In(b.loc.Origin())
}

// CRC: crc-StencilBuilder.md | R110
// Span is the location of the whole match — what the stencil node itself covers.
func (b *StencilBuilder) Span() Loc { return b.span(b.match[0], b.match[1]) }

// mustIndex returns a named group's submatch index. A name the regex does not
// define is a programming error in the schema rather than a data condition, so it
// is refused here — at the call that used the wrong name.
func (b *StencilBuilder) mustIndex(name string) int {
	for i, n := range b.names {
		if i > 0 && n == name {
			return i
		}
	}
	panic(fmt.Sprintf("sdom: no group named %q in this stencil", name))
}

// CRC: crc-StencilBuilder.md | R101, R108
//
// Group returns a named group's matched text and its provenance. Binding is BY
// NAME, never by position: with alternation the branches have different group
// counts, so a fixed index is right for one input and out of range for another.
//
// A group that did not participate in the match yields the ZERO Loc — no
// provenance — while a participating but empty group yields a real one at a real
// offset. Absence is the zero value, so the two are distinguishable without a
// third return.
func (b *StencilBuilder) Group(name string) (string, Loc) {
	i := b.mustIndex(name)
	s, e := b.match[2*i], b.match[2*i+1]
	if s < 0 {
		return "", Loc{}
	}
	return b.text[s:e], b.span(s, e)
}

// CRC: crc-StencilBuilder.md | R102
// Put patches a node the schema built into its group's slot.
func (b *StencilBuilder) Put(name string, n Node) {
	b.mustIndex(name)
	b.slot[name] = n
}

// CRC: crc-StencilBuilder.md | R103, R117
//
// Omit marks a participating group as glue after all. Its bytes exist and must go
// somewhere, and without this a schema would have to Put a Text it does not care
// about purely to satisfy the nil check — which would make the child list claim a
// field where there is none.
//
// Done folds the span into the surrounding glue rather than leaving it as a
// separate node, so omitting a group produces the IDENTICAL child list to a regex
// that never named it.
func (b *StencilBuilder) Omit(name string) {
	b.mustIndex(name)
	b.omit[name] = true
}

// bound is a named group that becomes a child: participating, not omitted, and
// not nested inside an earlier one.
type bound struct {
	name       string
	start, end int
}

// CRC: crc-StencilBuilder.md | R100, R109
//
// bounds returns the groups that break the match into children, in order. A group
// that did not participate has no bytes and is skipped; an omitted one is skipped
// so its span falls into the glue; and a NESTED group is skipped because it cannot
// double-count and would produce nothing.
func (b *StencilBuilder) bounds() []bound {
	var out []bound
	last := b.match[0]
	for i, name := range b.names {
		if i == 0 || name == "" || b.omit[name] {
			continue
		}
		s, e := b.match[2*i], b.match[2*i+1]
		if s < 0 || s < last {
			continue // absent, or nested inside a group already taken
		}
		out = append(out, bound{name, s, e})
		last = e
	}
	return out
}

// CRC: crc-StencilBuilder.md | Seq: seq-stencil.md#1.4 | R98, R104, R105, R106, R117
//
// Done assembles the children and reports what the match did not consume.
//
// The glue is COMPUTED, never required: a Text is made for the head of the match,
// for every gap between bound groups, and for the tail. A regex therefore names
// only the groups its schema binds and cannot silently eat bytes — the failure a
// tiling check would have caught is unrepresentable rather than detected. Empty
// gaps produce no node, so no two adjacent children are ever both plain glue.
//
// It panics on either failure it can see: a group left unfilled, and a plugged
// node whose span is not its group's. Both are programming errors, and the second
// is the only way a schema can break tiling from here — it would otherwise surface
// as the compound reporting itself altered, a plausible wrong answer rather than a
// refusal.
func (b *StencilBuilder) Done() ([]Node, string) {
	var kids []Node
	glue := func(s, e int) {
		if e > s {
			kids = append(kids, NewText(b.text[s:e], b.span(s, e)))
		}
	}
	at := b.match[0]
	for _, g := range b.bounds() {
		glue(at, g.start)
		n, ok := b.slot[g.name]
		if !ok {
			panic(fmt.Sprintf("sdom: group %q was named but never filled; Put it or Omit it", g.name))
		}
		if want := b.span(g.start, g.end); n.Location() != want {
			panic(fmt.Sprintf("sdom: the node put for group %q spans %d+%d, but the group is %d+%d",
				g.name, n.Location().Offset(), n.Location().Length(), want.Offset(), want.Length()))
		}
		kids = append(kids, n)
		at = g.end
	}
	glue(at, b.match[1])
	return kids, b.text[b.match[1]:]
}
