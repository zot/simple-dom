// CRC: crc-Loc.md | R24, R25, R26, R29, R30, R33, R34, R35, R36
package sdom

import (
	"strings"
	"testing"
)

// CRC: crc-Loc.md | R24, R26
// Absence is the zero value, and offset 0 stays distinguishable from it.
func TestZeroLocHasNoProvenance(t *testing.T) {
	var unset Loc
	if unset.Offset() != -1 {
		t.Fatalf("the zero Loc must report no provenance, got offset %d", unset.Offset())
	}
	if unset.Faithful() {
		t.Fatalf("a location with no provenance can never be faithful")
	}
	first := Source(0, 3)
	if first.Offset() != 0 {
		t.Fatalf("a location at the first byte must report offset 0, got %d", first.Offset())
	}
	if !first.Faithful() {
		t.Fatalf("a location at offset 0 must be faithful, not read as absent")
	}
}

// CRC: crc-Loc.md | R25
func TestFaithfulNodeRendersItsSourceSpan(t *testing.T) {
	src := "alpha beta gamma"
	d := New(src, 0,
		NewText("alpha ", Source(0, 6)),
		NewText("beta ", Source(6, 5)),
		NewText("gamma", Source(11, 5)))
	for _, n := range d.Nodes() {
		l := n.Location()
		if !l.Faithful() {
			t.Fatalf("precondition: every node starts faithful")
		}
		got, err := n.Render()
		if err != nil {
			t.Fatal(err)
		}
		if want := src[l.Offset() : l.Offset()+l.Length()]; got != want {
			t.Fatalf("faithful node rendered %q; its source span is %q", got, want)
		}
	}
}

// CRC: crc-Loc.md | R27, R29
// Provenance survives the edit that costs faithfulness, and length moves on.
func TestAlteredNodeKeepsItsOffset(t *testing.T) {
	n := NewText("beta ", Source(6, 5))
	n.SetText("BETA GAMMA ")
	l := n.Location()
	if l.Offset() != 6 {
		t.Fatalf("an altered node must keep its offset, got %d", l.Offset())
	}
	if !l.Altered() || l.Faithful() {
		t.Fatalf("a rewritten node must be altered and unfaithful")
	}
	if l.Length() != len("BETA GAMMA ") {
		t.Fatalf("Length() = %d; want the current extent %d", l.Length(), len("BETA GAMMA "))
	}
}

// locKind is one of the three ways a location can arrive at a merge.
type locKind struct {
	name string
	loc  Loc
}

func locKinds() []locKind {
	return []locKind{
		{"faithful", Source(10, 4)},
		{"altered", Source(40, 4).alter()},
		{"none", Synthetic(4)},
	}
}

// CRC: crc-Loc.md | R34
// "Unfaithful" covers altered and no-provenance alike: only faithful+faithful survives.
func TestMergedFaithfulnessOverTheWholeTable(t *testing.T) {
	for _, a := range locKinds() {
		for _, b := range locKinds() {
			got := mergeLocs(a.loc, b.loc).Faithful()
			want := a.name == "faithful" && b.name == "faithful"
			if got != want {
				t.Errorf("merge(%s, %s).Faithful() = %v; want %v", a.name, b.name, got, want)
			}
		}
	}
}

// CRC: crc-Loc.md | R35
func TestMergedOffsetTakesLeftmostProvenance(t *testing.T) {
	with, without := Source(10, 4), Synthetic(4)
	cases := []struct {
		name string
		a, b Loc
		want int
	}{
		{"both", with, Source(14, 4), 10},
		{"first only", with, without, 10},
		{"second only", without, Source(14, 4), 14},
		{"neither", without, without, -1},
	}
	for _, c := range cases {
		if got := mergeLocs(c.a, c.b).Offset(); got != c.want {
			t.Errorf("%s: merged offset = %d; want %d", c.name, got, c.want)
		}
	}
}

// CRC: crc-Loc.md | R35
// Leftmost-provenance-wins is associative, which is what makes it safe to fold.
func TestMergingARunIsAssociative(t *testing.T) {
	for pattern := 0; pattern < 8; pattern++ {
		var l [3]Loc
		for i := range l {
			if pattern&(1<<i) != 0 {
				l[i] = Source(10+4*i, 4)
			} else {
				l[i] = Synthetic(4)
			}
		}
		left := mergeLocs(mergeLocs(l[0], l[1]), l[2])
		right := mergeLocs(l[0], mergeLocs(l[1], l[2]))
		if left.Offset() != right.Offset() || left.Faithful() != right.Faithful() {
			t.Errorf("pattern %03b: left-to-right gave (%d, %v), right-to-left gave (%d, %v)",
				pattern, left.Offset(), left.Faithful(), right.Offset(), right.Faithful())
		}
	}
}

// CRC: crc-Loc.md | Seq: seq-mutate.md#2.2 | R33
// The check fires exactly where the locations can prove something.
func TestAdjacencyIsCheckedExactlyWhereItCan(t *testing.T) {
	newPair := func() (*Doc, *Text, *Text) {
		a := NewText("aa", Source(0, 2))
		b := NewText("bb", Source(90, 2)) // array-adjacent, source-distant
		return New("aabb", 0, a, b), a, b
	}
	merge := func(d *Doc, a, b Node) error {
		return d.Mutate(func() error {
			_, err := d.Merge(a, b)
			return err
		})
	}

	d, a, b := newPair()
	err := merge(d, a, b)
	if err == nil || !strings.Contains(err.Error(), "not adjacent") {
		t.Fatalf("a non-adjacent faithful pair must be refused at the call; got %v", err)
	}

	d, a, b = newPair()
	a.SetText("aa") // same bytes, but now altered — the arithmetic proves nothing
	if err := merge(d, a, b); err != nil {
		t.Fatalf("an unfaithful operand must not be adjacency-checked; got %v", err)
	}
}

// CRC: crc-Loc.md | R30, R36
// Re-granulation moves boundaries and nothing else.
func TestSplitThenMergeIsTheIdentity(t *testing.T) {
	const src = "abcdefgh"
	for at := 0; at <= len(src); at++ {
		n := NewText(src, Source(0, len(src)))
		d := New(src, 0, n)
		before := n.Location()
		if err := d.Mutate(func() error {
			l, r, err := d.Split(n, at)
			if err != nil {
				return err
			}
			_, err = d.Merge(l, r)
			return err
		}); err != nil {
			t.Fatalf("at %d: %v", at, err)
		}
		if len(d.Nodes()) != 1 {
			t.Fatalf("at %d: expected one node back, got %d", at, len(d.Nodes()))
		}
		after := d.Nodes()[0].Location()
		if after.Offset() != before.Offset() || after.Length() != before.Length() ||
			after.Faithful() != before.Faithful() {
			t.Fatalf("at %d: got (%d,%d,%v); want (%d,%d,%v)", at,
				after.Offset(), after.Length(), after.Faithful(),
				before.Offset(), before.Length(), before.Faithful())
		}
		if got, _ := d.Render(); got != src {
			t.Fatalf("at %d: rendered %q; want %q", at, got, src)
		}
	}
}
