// CRC: crc-Doc.md | R41, R43, R44, R45, R46, R49, R50, R51, R52, R53, R54, R55
package sdom

import (
	"errors"
	"strings"
	"testing"
)

func threeNodes() (*Doc, []*Text) {
	n := []*Text{
		NewText("aa\n", Source(0, 3)),
		NewText("bb\n", Source(3, 3)),
		NewText("cc", Source(6, 2)),
	}
	return New("aa\nbb\ncc", 0, n[0], n[1], n[2]), n
}

// CRC: crc-Doc.md | R41
func TestNavigationIsByNodeInDocumentOrder(t *testing.T) {
	d, n := threeNodes()
	if d.Next(n[0]) != n[1] || d.Next(n[1]) != n[2] {
		t.Fatalf("Next must walk the array in document order")
	}
	if d.Next(n[2]) != nil {
		t.Fatalf("Next past the last node must return nil")
	}
	if d.Prev(n[2]) != n[1] || d.Prev(n[1]) != n[0] {
		t.Fatalf("Prev must walk back in document order")
	}
	if d.Prev(n[0]) != nil {
		t.Fatalf("Prev before the first node must return nil")
	}
	if d.IndexOf(NewText("x", Synthetic(1))) != -1 {
		t.Fatalf("a node from another document must not be found")
	}
}

// CRC: crc-Doc.md | R43, R44
// The distinction the whole stamping protocol rests on.
func TestGenerationBumpsOnMembershipNotContent(t *testing.T) {
	d, n := threeNodes()
	before := d.Generation()

	if err := d.Mutate(func() error { n[0].SetText("AA\n"); return nil }); err != nil {
		t.Fatal(err)
	}
	if d.Generation() != before {
		t.Fatalf("a content edit must not bump the generation")
	}

	if err := d.Mutate(func() error { _, _, err := d.Split(n[1], 1); return err }); err != nil {
		t.Fatal(err)
	}
	if d.Generation() == before {
		t.Fatalf("a membership change must bump the generation")
	}
}

// fakeLayer is a layer above sdom holding derived state. Doc never learns it exists.
type fakeLayer struct {
	stamp    uint64
	built    bool
	rebuilds int
}

func (l *fakeLayer) refresh(d *Doc) {
	g := d.Generation()
	if !l.built || g != l.stamp {
		l.stamp, l.built = g, true
		l.rebuilds++
	}
}

// CRC: crc-Doc.md | Seq: seq-stamp.md | R45, R46
// The layer pulls; the document pushes nothing and registers nothing.
func TestStaleStampDrivesTheRebuild(t *testing.T) {
	d, n := threeNodes()
	l := &fakeLayer{}

	l.refresh(d)
	l.refresh(d)
	if l.rebuilds != 1 {
		t.Fatalf("a fresh stamp must not rebuild; rebuilds = %d", l.rebuilds)
	}

	if err := d.Mutate(func() error { return d.Remove(n[1]) }); err != nil {
		t.Fatal(err)
	}
	l.refresh(d)
	if l.rebuilds != 2 {
		t.Fatalf("a stale stamp must rebuild exactly once; rebuilds = %d", l.rebuilds)
	}
	l.refresh(d)
	if l.rebuilds != 2 {
		t.Fatalf("the re-stamped layer must be fresh again; rebuilds = %d", l.rebuilds)
	}
}

// CRC: crc-MutationWindow.md | Seq: seq-mutate.md#1.4.3 | R49, R50
// Every read that could observe a half-edited document refuses, as an error.
func TestGuardedReadsRefuseInsideTheWindow(t *testing.T) {
	reads := []struct {
		name string
		read func(*Doc, Node)
	}{
		{"IndexOf", func(d *Doc, n Node) { d.IndexOf(n) }},
		{"Next", func(d *Doc, n Node) { d.Next(n) }},
		{"Prev", func(d *Doc, n Node) { d.Prev(n) }},
		{"Line", func(d *Doc, _ Node) { d.Line(0) }},
		{"LineCount", func(d *Doc, _ Node) { d.LineCount() }},
		{"Generation", func(d *Doc, _ Node) { d.Generation() }},
	}
	for _, c := range reads {
		d, n := threeNodes()
		err := d.Mutate(func() error { c.read(d, n[0]); return nil })
		if err == nil {
			t.Errorf("%s inside a mutation window must refuse", c.name)
			continue
		}
		if !strings.Contains(err.Error(), "not available inside a mutation window") {
			t.Errorf("%s: refused with %v; want the typed sentinel as an error", c.name, err)
		}
	}
}

// CRC: crc-MutationWindow.md | Seq: seq-mutate.md#3.2.2 | R51
// The guard must not swallow real bugs.
func TestForeignPanicIsReRaised(t *testing.T) {
	d, _ := threeNodes()
	boom := errors.New("something else entirely")
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("a foreign panic must escape Mutate rather than arrive as an error")
		}
		if r != any(boom) {
			t.Fatalf("recovered %v; want the original panic value %v", r, boom)
		}
	}()
	_ = d.Mutate(func() error { panic(boom) })
}

// CRC: crc-MutationWindow.md | R53
// A nested call is a pass-through: the window belongs to the outermost one.
func TestNestedMutateIsAPassThrough(t *testing.T) {
	d, n := threeNodes()
	before := d.Generation()
	err := d.Mutate(func() error {
		if _, _, err := d.Split(n[0], 1); err != nil {
			return err
		}
		return d.Mutate(func() error { _, _, err := d.Split(n[2], 1); return err })
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Nodes()) != 5 {
		t.Fatalf("both edits must land; got %d nodes, want 5", len(d.Nodes()))
	}
	if d.Generation() != before+1 {
		t.Fatalf("the window must close once, bumping once; generation went %d -> %d",
			before, d.Generation())
	}
}

// CRC: crc-MutationWindow.md | R53
//
// The window belongs to the OUTERMOST call, which is the half of the
// pass-through that node counts and generation bumps cannot see: an inner call
// that closed its own window would leave both of those correct while the guard
// silently lifted for the rest of the outer function.
func TestNestedMutateKeepsTheOuterWindowOpen(t *testing.T) {
	d, n := threeNodes()
	err := d.Mutate(func() error {
		if ierr := d.Mutate(func() error {
			_, _, e := d.Split(n[0], 1)
			return e
		}); ierr != nil {
			return ierr
		}
		d.Generation() // the outer window is still open, so this must refuse
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "not available inside a mutation window") {
		t.Fatalf("after a nested Mutate returns, the outer window must still be open; got %v", err)
	}
}

// CRC: crc-MutationWindow.md | Seq: seq-mutate.md#3.3 | R54, R55
// No rollback, and no reset.
func TestEscapingFailurePoisons(t *testing.T) {
	for _, how := range []string{"error", "panic"} {
		d, n := threeNodes()
		run := func() (err error) {
			defer func() { recover() }()
			return d.Mutate(func() error {
				if err := d.Remove(n[1]); err != nil {
					return err
				}
				if how == "panic" {
					panic("out")
				}
				return errors.New("out")
			})
		}
		_ = run()

		if len(d.Nodes()) != 2 {
			t.Errorf("%s: the edit must still be applied — there is no rollback; got %d nodes",
				how, len(d.Nodes()))
		}
		if _, err := d.Render(); !errors.Is(err, ErrPoisoned) {
			t.Errorf("%s: a poisoned document must refuse; Render gave %v", how, err)
		}
		if err := d.Mutate(func() error { return nil }); !errors.Is(err, ErrPoisoned) {
			t.Errorf("%s: a poisoned document offers no recovery; Mutate gave %v", how, err)
		}
	}
}

// CRC: crc-MutationWindow.md | R47
func TestStructuralEditOutsideAWindowIsRefused(t *testing.T) {
	d, n := threeNodes()
	if _, _, err := d.Split(n[0], 1); !errors.Is(err, ErrNotMutating) {
		t.Fatalf("Split outside a window gave %v; want ErrNotMutating", err)
	}
	if err := d.Remove(n[0]); !errors.Is(err, ErrNotMutating) {
		t.Fatalf("Remove outside a window gave %v; want ErrNotMutating", err)
	}
}

// CRC: crc-Doc.md | R39
// An offset outside the rendered document is outside it at both ends.
func TestLineRejectsOffsetsOutsideTheDocument(t *testing.T) {
	d, _ := threeNodes()
	rendered, err := d.Render()
	if err != nil {
		t.Fatal(err)
	}
	for _, off := range []int{-1, len(rendered), len(rendered) + 1, len(rendered) * 2} {
		if got := d.Line(off); got != 0 {
			t.Errorf("Line(%d) = %d; want 0 — the offset is outside a %d-byte document",
				off, got, len(rendered))
		}
	}
	if d.Line(len(rendered)-1) == 0 {
		t.Fatalf("the last byte is inside the document and must have a line")
	}
	empty := New("", 0)
	if got := empty.Line(0); got != 0 {
		t.Errorf("Line(0) on an empty document = %d; want 0", got)
	}
}

// CRC: crc-Doc.md | Seq: seq-mutate.md#1.6 | R52
// One rebuild at the exit, and it is a correct one.
func TestIndicesAreCorrectAfterTheWindowCloses(t *testing.T) {
	d, n := threeNodes()
	err := d.Mutate(func() error {
		l, r, err := d.Split(n[0], 1)
		if err != nil {
			return err
		}
		if _, err = d.Merge(l, r); err != nil {
			return err
		}
		n[2].SetText("cc\ndd")
		return d.Remove(n[1])
	})
	if err != nil {
		t.Fatal(err)
	}

	for want, node := range d.Nodes() {
		if got := d.IndexOf(node); got != want {
			t.Errorf("IndexOf returned %d for the node at %d", got, want)
		}
	}
	rendered, err := d.Render()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := d.LineCount(), strings.Count(rendered, "\n")+1; got != want {
		t.Fatalf("LineCount = %d; the rendered document has %d lines", got, want)
	}
	for off := range len(rendered) {
		// the line containing off is one more than the newlines strictly before it
		if got, want := d.Line(off), strings.Count(rendered[:off], "\n")+1; got != want {
			t.Fatalf("Line(%d) = %d; want %d", off, got, want)
		}
	}
}

// CRC: crc-MutationWindow.md | R43, R257
func TestInsertPlacesBeforeOrAtTheEnd(t *testing.T) {
	d, _ := parse("ab|cd", 0, &BracketLang{Brackets: []BracketGroup{{Open: []string{"|"}, Close: "\n"}}})
	g0 := d.Generation()
	var marker Node
	for _, n := range d.Nodes() {
		if _, ok := n.(*Opener); ok {
			marker = n
		}
	}
	if err := d.Mutate(func() error { return d.Insert(marker, NewText("X", Synthetic(1))) }); err != nil {
		t.Fatal(err)
	}
	g1 := d.Generation()
	if err := d.Mutate(func() error { return d.Insert(nil, NewText("Y", Synthetic(1))) }); err != nil {
		t.Fatal(err)
	}
	if r, _ := d.Render(); r != "abX|cdY" {
		t.Fatalf("render %q", r)
	}
	if g2 := d.Generation(); g1 <= g0 || g2 <= g1 {
		t.Errorf("generation did not advance twice: %d %d %d", g0, g1, g2)
	}
	// A refusal escaping the window poisons the document, as every escaping error does.
	if err := d.Mutate(func() error { return d.Insert(NewText("z", Synthetic(1)), NewText("W", Synthetic(1))) }); err == nil {
		t.Errorf("inserting before a foreign node was accepted")
	}
	if _, err := d.Render(); !errors.Is(err, ErrPoisoned) {
		t.Errorf("after a refused insert escaped the window: %v, want ErrPoisoned", err)
	}
}
