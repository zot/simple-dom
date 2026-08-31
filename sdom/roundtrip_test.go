// CRC: crc-Doc.md | R2, R3, R10, R25, R28
package sdom

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// corpus returns the project's own files, this package's source included. Not
// fixtures: a fixture contains only what its author thought to include, and what
// a lossy parse eats is precisely what nobody thought of.
func corpus(t *testing.T) map[string]string {
	t.Helper()
	var paths []string
	for _, pat := range []string{"*.go", "../*.md", "../specs/*.md", "../design/*.md", "../carves/*.md"} {
		m, err := filepath.Glob(pat)
		if err != nil {
			t.Fatal(err)
		}
		paths = append(paths, m...)
	}
	if len(paths) < 10 {
		t.Fatalf("corpus is suspiciously small (%d files) — the globs are probably wrong", len(paths))
	}
	out := make(map[string]string, len(paths))
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		out[p] = string(b)
	}
	return out
}

// wholeFile is the document a lexer-less parse produces: one Text over
// everything, which is the starting point every corpus property is checked from.
func wholeFile(src string) *Doc {
	return New(src, 0, NewText(src, Source(0, len(src))))
}

// shred splits the document's tail node every step bytes, so the flat-array
// properties are tested against many nodes rather than the single Text a
// lexer-less parse produces.
func shred(t *testing.T, d *Doc, step int) {
	t.Helper()
	if step < 1 {
		step = 1
	}
	err := d.Mutate(func() error {
		n, ok := d.Nodes()[len(d.Nodes())-1].(*Text)
		if !ok {
			return nil
		}
		for len(n.text) > step {
			_, right, err := d.Split(n, step)
			if err != nil {
				return err
			}
			n = right.(*Text)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// CRC: crc-Doc.md | R2
// Bytes the parse does not model come back unchanged.
func TestByteRoundTripOverTheCorpus(t *testing.T) {
	for path, src := range corpus(t) {
		d := wholeFile(src)
		if got, err := d.Render(); err != nil || got != src {
			t.Fatalf("%s: whole-file round-trip failed (err %v)", path, err)
		}
		shred(t, d, len(src)/23+1)
		if got, err := d.Render(); err != nil || got != src {
			t.Fatalf("%s: round-trip failed after splitting into %d nodes (err %v)",
				path, len(d.Nodes()), err)
		}
	}
}

// CRC: crc-Doc.md | R3
// Contiguous, half-open, first at 0 and last at the end of the source.
func TestTheArrayTilesTheDocument(t *testing.T) {
	for path, src := range corpus(t) {
		d := wholeFile(src)
		shred(t, d, len(src)/17+1)
		next := 0
		for i, n := range d.Nodes() {
			l := n.Location()
			if !l.Faithful() {
				t.Fatalf("%s: node %d lost faithfulness to a split, which moves no bytes", path, i)
			}
			if l.Offset() != next {
				t.Fatalf("%s: node %d begins at %d; the previous ended at %d", path, i, l.Offset(), next)
			}
			next = l.Offset() + l.Length()
		}
		if next != len(src) {
			t.Fatalf("%s: the array ends at %d; the source is %d bytes", path, next, len(src))
		}
	}
}

// CRC: crc-Loc.md | R25
func TestEveryFaithfulNodeRendersItsOwnSpan(t *testing.T) {
	for path, src := range corpus(t) {
		d := wholeFile(src)
		shred(t, d, len(src)/13+1)
		for i, n := range d.Nodes() {
			l := n.Location()
			if !l.Faithful() {
				continue
			}
			got, err := n.Render()
			if err != nil {
				t.Fatal(err)
			}
			if want := src[l.Offset() : l.Offset()+l.Length()]; got != want {
				t.Fatalf("%s: node %d rendered %q, its span holds %q", path, i, got, want)
			}
		}
	}
}

// nest builds the same structure over a document at the given base:
//
//	dom = [ outer ], outer = Compound{ inner, tail }, inner = Compound{ a, b }
func nest(base int) (*Doc, *Compound, *Compound, []*Text) {
	const src = "aabbcc"
	off := func(i int) int { return base + i }
	a := NewText("aa", Source(off(0), 2))
	b := NewText("bb", Source(off(2), 2))
	tail := NewText("cc", Source(off(4), 2))
	inner := NewCompound(Source(off(0), 4), a, b)
	outer := NewCompound(Source(off(0), 6), inner, tail)
	return New(src, base, outer), outer, inner, []*Text{a, b, tail}
}

// CRC: crc-Node.md | R10
// Passing also proves Equals ignores provenance: every offset differs.
func TestStructuralRoundTripAtADifferentBase(t *testing.T) {
	// In this item the comparison tree is reconstructed by the same construction
	// rather than re-parsed: nothing recovers a Compound from bytes until the
	// lexer lands in Item 2. The re-parse form of this test belongs there.
	_, here, _, hereLeaves := nest(0)
	_, there, _, thereLeaves := nest(500)

	if !here.Equals(there) {
		t.Fatalf("identical structures at different bases must compare equal")
	}
	hereLeaves[1].SetText("BB")
	if here.Equals(there) {
		t.Fatalf("mutating one tree must make them differ")
	}
	thereLeaves[1].SetText("BB")
	if !here.Equals(there) {
		t.Fatalf("the same mutation on both must restore equality despite every offset differing")
	}
	if here.Location().Offset() == there.Location().Offset() {
		t.Fatalf("precondition: the two trees must not share offsets")
	}
}

// CRC: crc-Node.md | R10, R57
//
// The structural round-trip in its real form: mutate a scanned document, render
// it, RE-PARSE that output at a different base, and require the two to compare
// equal node for node. Passing proves both that the parse is stable under its own
// output and that Equals ignores provenance — every offset differs, and the
// mutated node is altered on one side and freshly faithful on the other.
//
// Item 1 could only reconstruct the comparison tree, because nothing recovered a
// Compound from bytes until the lexer landed. This replaces that (gap O4).
func TestStructuralRoundTripThroughAReparse(t *testing.T) {
	const src = "func f(a int) {\n\t// note\n\treturn 1\n}\n"
	d, _ := Scan(src, 0, &LangGo)

	var target *Text
	for _, n := range d.Nodes() {
		if txt, ok := n.(*Text); ok && strings.Contains(txt.text, "return") {
			target = txt
			break
		}
	}
	if target == nil {
		t.Fatal("precondition: a text node holding the return statement")
	}
	if err := d.Mutate(func() error { target.SetText("\n\treturn 42 + 7\n"); return nil }); err != nil {
		t.Fatal(err)
	}
	if target.Location().Faithful() {
		t.Fatal("precondition: the edited node is no longer faithful")
	}

	out, err := d.Render()
	if err != nil {
		t.Fatal(err)
	}

	// Shift every offset by re-parsing the same bytes behind a prefix. A
	// different `base` would NOT do this: node offsets are relative to the
	// document's own source, and base is metadata about where that source sits in
	// an outer document — it never enters a location. Measured 2026-08-30, when an
	// alarm that should have rung did not.
	const prefix = "// shifted\n"
	reparsed, _ := Scan(prefix+out, 0, &LangGo)
	skip := 0
	for _, n := range reparsed.Nodes() {
		l := n.Location()
		if l.Offset()+l.Length() > len(prefix) {
			break
		}
		skip++
	}
	tail := reparsed.Nodes()[skip:]

	if len(tail) != len(d.Nodes()) {
		t.Fatalf("re-parsing produced %d nodes past the prefix; the mutated tree has %d",
			len(tail), len(d.Nodes()))
	}
	for i, want := range d.Nodes() {
		got := tail[i]
		if got.Location().Offset() == want.Location().Offset() {
			t.Fatalf("node %d shares an offset with the original; the prefix did not shift it", i)
		}
		if !got.Equals(want) {
			gs, _ := got.Render()
			ws, _ := want.Render()
			t.Fatalf("node %d differs after a re-parse: %q vs %q", i, gs, ws)
		}
	}
}

// CRC: crc-Compound.md | R28
// Exactly the edited node and its ancestors lose faithfulness.
func TestTheOneFieldDelta(t *testing.T) {
	_, outer, inner, leaves := nest(0)
	for _, n := range []Node{outer, inner, leaves[0], leaves[1], leaves[2]} {
		if !n.Location().Faithful() {
			t.Fatalf("precondition: everything starts faithful")
		}
	}
	leaves[0].SetText("AA")

	for _, n := range []Node{leaves[0], inner, outer} {
		if n.Location().Faithful() {
			t.Errorf("the edited node and its ancestors must lose faithfulness")
		}
	}
	for i, n := range []Node{leaves[1], leaves[2]} {
		if !n.Location().Faithful() {
			t.Errorf("untouched sibling %d must keep its faithfulness", i)
		}
	}
}

// CRC: crc-Doc.md | R2
//
// The alarm the corpus round-trip structurally cannot reach: a Render that hands
// back retained source leaves that test green over every document, no matter how
// little was modelled. Only requiring the output to LACK a removed node sees it.
func TestARemovedNodeLeavesTheOutput(t *testing.T) {
	const src = "alpha beta gamma"
	a := NewText("alpha ", Source(0, 6))
	b := NewText("beta ", Source(6, 5))
	c := NewText("gamma", Source(11, 5))
	d := New(src, 0, a, b, c)

	if err := d.Mutate(func() error { return d.Remove(b) }); err != nil {
		t.Fatal(err)
	}
	got, err := d.Render()
	if err != nil {
		t.Fatal(err)
	}
	if want := "alpha gamma"; got != want {
		t.Fatalf("Render() = %q; want %q — the removed node's bytes must be gone", got, want)
	}
	if strings.Contains(got, "beta") {
		t.Fatalf("Render() still contains the removed node's bytes: %q", got)
	}
}
