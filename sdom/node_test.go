// CRC: crc-Node.md | R7, R8, R10, R13, R15, R16, R28
package sdom

import "testing"

// CRC: crc-Node.md | R10
// Two nodes with identical content but different provenance are equal.
func TestEqualsIgnoresProvenance(t *testing.T) {
	a := NewText("hello", Source(0, 5))
	b := NewText("hello", Source(400, 5))
	if !a.Equals(b) || !b.Equals(a) {
		t.Fatalf("nodes with identical content and different offsets must be equal")
	}
	if !a.Equals(NewText("hello", Synthetic(5))) {
		t.Fatalf("provenance absence must not affect equality either")
	}
}

// CRC: crc-Node.md | R13
// The type check precedes any structural comparison.
func TestEqualsDistinguishesKinds(t *testing.T) {
	leaf := NewText("hello", Source(0, 5))
	comp := NewCompound(Source(0, 5), NewText("hello", Source(0, 5)))
	if leaf.Equals(comp) || comp.Equals(leaf) {
		t.Fatalf("a Text and a Compound rendering the same bytes must not be equal")
	}
}

// forgetful embeds Compound and deliberately declares no Equals of its own.
type forgetful struct{ Compound }

// CRC: crc-Node.md | R13
// A kind that forgets its own Equals fails closed, and the failure is visible.
func TestForgettingEqualsIsLoud(t *testing.T) {
	kids := []Node{NewText("ab", Source(0, 2))}
	a := &forgetful{Compound: *NewCompound(Source(0, 2), kids...)}
	b := &forgetful{Compound: *NewCompound(Source(0, 2), kids...)}
	if a.Equals(b) {
		t.Fatalf("the promoted Compound.Equals must fail its assertion against *forgetful, " +
			"so two identical instances compare unequal and any structural round-trip goes red")
	}
}

// CRC: crc-Text.md | R7, R8, R15
func TestLeafHasNoChildren(t *testing.T) {
	leaf := NewText("hello", Source(3, 5))
	if len(leaf.Kids()) != 0 {
		t.Fatalf("a leaf must have no children, got %d", len(leaf.Kids()))
	}
	s, err := leaf.Render()
	if err != nil || s != "hello" {
		t.Fatalf("Render() = %q, %v; want %q, nil", s, err, "hello")
	}
}

// CRC: crc-Compound.md | R16
func TestCompoundConcatenatesAndSums(t *testing.T) {
	c := NewCompound(Source(0, 6),
		NewText("aa", Source(0, 2)),
		NewText("bb", Source(2, 2)),
		NewText("cc", Source(4, 2)))
	s, err := c.Render()
	if err != nil || s != "aabbcc" {
		t.Fatalf("Render() = %q, %v; want %q, nil", s, err, "aabbcc")
	}
	loc := c.Location()
	if loc.Length() != 6 {
		t.Fatalf("Length() = %d; want the sum of the children, 6", loc.Length())
	}
	if !loc.Faithful() {
		t.Fatalf("a compound over faithful, contiguous children must be faithful")
	}
}

// CRC: crc-Compound.md | R28
// Alteration of any child is derived on read, with nothing propagated upward.
func TestCompoundDerivesAlteration(t *testing.T) {
	kids := []*Text{
		NewText("aa", Source(0, 2)),
		NewText("bb", Source(2, 2)),
		NewText("cc", Source(4, 2)),
	}
	c := NewCompound(Source(0, 6), kids[0], kids[1], kids[2])
	if !c.Location().Faithful() {
		t.Fatalf("precondition: the compound starts faithful")
	}
	kids[1].SetText("BBB")
	loc := c.Location()
	if !loc.Altered() || loc.Faithful() {
		t.Fatalf("altering any child must make the compound altered")
	}
	if loc.Length() != 7 {
		t.Fatalf("Length() = %d; want the new sum, 7", loc.Length())
	}
	if !kids[0].Location().Faithful() || !kids[2].Location().Faithful() {
		t.Fatalf("untouched siblings must keep their faithfulness")
	}
}
