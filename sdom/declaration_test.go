// CRC: crc-Declaration.md | R122, R123
package sdom

import (
	"errors"
	"testing"
)

// CRC: crc-MutationWindow.md | Seq: seq-declare.md#2.2 | R125
//
// Replace keeps a node's position and bumps the generation. The generation half is
// what matters: without it a stamped index keeps a stale stamp and never rebuilds,
// and nothing about the bytes would say so.
func TestReplaceKeepsPositionAndBumpsGeneration(t *testing.T) {
	src := "func Index(a int) int {\n}\n"
	d, _ := Scan(src, 0, &LangGo)
	before := d.Generation()
	n := d.Nodes()[0]
	want := len(d.Nodes())

	repl := NewDeclarationType("func Index", n.Location())
	if err := d.Mutate(func() error { return d.Replace(n, repl) }); err != nil {
		t.Fatalf("Mutate: %v", err)
	}
	if got, _ := d.Render(); got != src {
		t.Errorf("render changed: %q", got)
	}
	if len(d.Nodes()) != want {
		t.Errorf("array length %d, want %d", len(d.Nodes()), want)
	}
	if d.Nodes()[0] != Node(repl) {
		t.Error("replacement is not at the old node's index")
	}
	if d.Generation() == before {
		t.Error("generation did not advance; a stamped index would never rebuild")
	}
}

// CRC: crc-MutationWindow.md | R125
// Replace refuses outside a mutation window, like every other structural edit.
func TestReplaceNeedsAWindow(t *testing.T) {
	d, _ := Scan("a\n", 0, &LangGo)
	n := d.Nodes()[0]
	if err := d.Replace(n, NewDeclarationType("a\n", n.Location())); !errors.Is(err, ErrNotMutating) {
		t.Errorf("got %v, want ErrNotMutating", err)
	}
}

// CRC: crc-BracketContext.md | Seq: seq-declare.md#2.5 | R126, R127, R128
//
// A stale declaration accessor REFUSES. It cannot rebuild — sdom does not know
// what announces a declaration — and an empty answer would read identically to
// "this keyword declares nothing", which is the plausible wrong answer IndexOf
// already refuses on the same grounds.
func TestStaleDeclarationsRefuse(t *testing.T) {
	d, ctx := Scan("func Index(a int) int {\n}\n", 0, &LangGo)
	kw := NewDeclarationType("x", Synthetic(1))
	one := NewDeclarationName("y", Synthetic(1))
	two := NewDeclarationName("z", Synthetic(1))

	ctx.SetDeclarations(map[Node][]Node{kw: {one, two}})
	got, err := ctx.Declarations(kw)
	if err != nil {
		t.Fatalf("fresh links refused: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("got %d names, want 2 — the link is one-to-many", len(got))
	}

	// Any structural change invalidates them.
	if err := d.Mutate(func() error { _, _, e := d.Split(d.Nodes()[0], 2); return e }); err != nil {
		t.Fatalf("Mutate: %v", err)
	}
	if _, err := ctx.Declarations(kw); !errors.Is(err, ErrDeclarationsStale) {
		t.Errorf("got %v, want ErrDeclarationsStale", err)
	}
}

// CRC: crc-Declaration.md | R122, R123
// Each kind declares its own Equals, so two kinds with identical bytes are not
// equal. A promoted Equals could not see the outer type.
func TestDeclarationKindsAreDistinct(t *testing.T) {
	l := Synthetic(3)
	ty, nm := NewDeclarationType("foo", l), NewDeclarationName("foo", l)
	if ty.Equals(nm) || nm.Equals(ty) {
		t.Error("a DeclarationType compared equal to a DeclarationName")
	}
	if !ty.Equals(NewDeclarationType("foo", l)) {
		t.Error("two identical DeclarationTypes are not equal")
	}
	if ty.Equals(NewText("foo", l)) {
		t.Error("a DeclarationType compared equal to a Text")
	}
}
