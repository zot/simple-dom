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
	d, _ := parse(src, 0, &LangGo)
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
	d, _ := parse("a\n", 0, &LangGo)
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
	d, ctx := parse("func Index(a int) int {\n}\n", 0, &LangGo)
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

// CRC: crc-BracketContext.md | Seq: seq-declare.md#2.5 | R126, R128
//
// Declarations survive a rebuild that happens AFTER they were recorded — and the
// order here is the one every schema pass uses: mutate inside a window, record at
// the end, and let some later accessor trigger the refresh.
//
// This is the regression consolidating the maps introduced. rebuild recreates the
// whole index, the declaration links now live in it, and rebuild then sets stamp to
// the generation declStamp already held — so the loss was invisible to the staleness
// check and the accessor returned (nil, nil). That empty answer is exactly the one
// its own doc comment promises to refuse, because it reads as "declares nothing".
//
// TestStaleDeclarationsRefuse cannot see this: it edits AFTER recording, so the
// stamps differ and the refusal is correct. The hole is a rebuild firing while they
// agree.
func TestDeclarationsSurviveALaterRebuild(t *testing.T) {
	d, ctx := parse("func Index(a int) int {\n}\n", 0, &LangGo)
	kw := NewDeclarationType("func", Synthetic(4))
	nm := NewDeclarationName("Index", Synthetic(5))

	if err := d.Mutate(func() error { _, _, e := d.Split(d.Nodes()[0], 2); return e }); err != nil {
		t.Fatalf("Mutate: %v", err)
	}
	ctx.SetDeclarations(map[Node][]Node{kw: {nm}})

	ctx.Enclosing(d.Nodes()[0]) // any accessor that refreshes

	got, err := ctx.Declarations(kw)
	if err != nil {
		t.Fatalf("refused after a rebuild it should have survived: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("got %d names after a rebuild, want 1 — a silent wipe reads as "+
			"'this keyword declares nothing'", len(got))
	}
}

// CRC: crc-BracketContext.md | R126
//
// SetDeclarations REPLACES, as its name says: a keyword absent from the new map
// keeps nothing. Consolidating the maps turned it into a merge, because writing
// entry-by-entry leaves untouched entries alone where assigning a whole map did not.
func TestSetDeclarationsReplaces(t *testing.T) {
	_, ctx := parse("a\n", 0, &LangGo)
	first := NewDeclarationType("first", Synthetic(5))
	second := NewDeclarationType("second", Synthetic(6))
	nm := NewDeclarationName("n", Synthetic(1))

	ctx.SetDeclarations(map[Node][]Node{first: {nm}})
	ctx.SetDeclarations(map[Node][]Node{second: {nm}})

	if got, _ := ctx.Declarations(first); len(got) != 0 {
		t.Errorf("a keyword absent from the second call kept %d names; "+
			"SetDeclarations replaces, it does not merge", len(got))
	}
	if got, _ := ctx.Declarations(second); len(got) != 1 {
		t.Errorf("the second call's own keyword got %d names, want 1", len(got))
	}
}
