// CRC: crc-Loc.md | R24, R25, R26, R29, R30, R33, R34, R35, R36
package sdom

import (
	"fmt"
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

// CRC: crc-Loc.md | R88
// A document's base is metadata about its source, not part of any location.
func TestOffsetsAreRelativeToTheDocumentsOwnSource(t *testing.T) {
	const src = "alpha beta"
	d, _ := parse(src, 500, &BracketLang{})
	if d.Base() != 500 {
		t.Fatalf("precondition: the document carries a base of 500")
	}
	next := 0
	for i, n := range d.Nodes() {
		l := n.Location()
		if l.Offset() != next {
			t.Fatalf("node %d begins at %d; offsets must start at 0 regardless of base", i, l.Offset())
		}
		if got, want := src[l.Offset():l.Offset()+l.Length()], mustRender(t, n); got != want {
			t.Fatalf("node %d: the source at its offset is %q, it renders %q", i, got, want)
		}
		next = l.Offset() + l.Length()
	}
	if next != len(src) {
		t.Fatalf("the array ends at %d; the source is %d bytes", next, len(src))
	}
}

func mustRender(t *testing.T, n Node) string {
	t.Helper()
	s, err := n.Render()
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// CRC: crc-Loc.md | R89, R92
func TestOriginIsSetByChaining(t *testing.T) {
	o := &Origin{Name: "somewhere.go"}
	plain := Source(3, 4)
	if plain.Origin() != nil {
		t.Fatalf("a location built without an origin must report none")
	}
	attributed := plain.In(o)
	if attributed.Origin() != o {
		t.Fatalf("In must attribute the location to the parse it is given")
	}
	if plain.Origin() != nil {
		t.Fatalf("In must not mutate the location it was chained onto")
	}
	if attributed.Offset() != plain.Offset() || attributed.Length() != plain.Length() {
		t.Fatalf("In must change nothing but the origin")
	}
}

// CRC: crc-Loc.md | R90
// The field is what makes Origin usable as an identity: Go may give every
// zero-size allocation the same address, so an empty marker would compare equal
// to an unrelated one.
func TestDistinctOriginsAreDistinct(t *testing.T) {
	// Mint them the way production does, through a context, so they ESCAPE to the
	// heap. Two `&Origin{}` locals do not escape; the compiler gives them distinct
	// stack slots, so they compare distinct even with no field at all — and this
	// test proved nothing. Measured 2026-08-30 by an injection that made Origin an
	// empty struct: four tests went red and this one stayed green.
	a := newBracketContext(codeLang()).Origin()
	b := newBracketContext(codeLang()).Origin()
	if a == b {
		t.Fatalf("two separately minted origins must not be the same identity")
	}
	if Source(0, 1).In(a).Origin() == Source(0, 1).In(b).Origin() {
		t.Fatalf("locations from different parses must be distinguishable")
	}
}

// CRC: crc-BracketContext.md | R91
// One origin per parse, shared by every node it produces.
func TestOneOriginPerParse(t *testing.T) {
	const src = "a {b} c"
	lang := codeLang()
	d1, c1 := parse(src, 0, lang)
	d2, c2 := parse(src, 0, lang)

	if c1.Origin() == c2.Origin() {
		t.Fatalf("two parses of the same source must be two origins")
	}
	carried := func(parse string, d *Doc, bc *BracketContext) {
		t.Helper()
		for i, n := range d.Nodes() {
			if n.Location().Origin() != bc.Origin() {
				t.Fatalf("%s parse: node %d does not carry its own parse's origin", parse, i)
			}
		}
	}
	carried("first", d1, c1)
	carried("second", d2, c2)
	// The whole point: same file, different parser, now answerable.
	if d1.Nodes()[0].Location().Origin() == d2.Nodes()[0].Location().Origin() {
		t.Fatalf("nodes from the same file but different parses must be distinguishable")
	}
}

// CRC: crc-Loc.md | R93
// A nil origin is unknown, not different — a synthesized node merges cleanly.
func TestNilOriginIsAbsentNotDifferent(t *testing.T) {
	o := &Origin{Name: "parsed"}
	parsed := Source(0, 2).In(o)
	cases := []struct {
		name string
		a, b Loc
		want *Origin
	}{
		{"synthesized on the right", parsed, Synthetic(3), o},
		{"synthesized on the left", Synthetic(3), parsed, o},
		{"neither known", Synthetic(1), Synthetic(2), nil},
	}
	for _, c := range cases {
		if got := mergeLocs(c.a, c.b).Origin(); got != c.want {
			t.Errorf("%s: merged origin = %v; want %v", c.name, got, c.want)
		}
	}
}

// CRC: crc-Doc.md | R118
// One document, one parse — checked once, where nothing else can break it.
func TestMixedParsesPanicAtConstruction(t *testing.T) {
	a := NewText("aa", Source(0, 2).In(&Origin{Name: "one"}))
	b := NewText("bb", Source(2, 2).In(&Origin{Name: "two"}))
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("building a document from two parses must panic")
		}
		if !strings.Contains(fmt.Sprint(r), "comes from") {
			t.Fatalf("the panic must name the offending node; got %v", r)
		}
	}()
	New("aabb", 0, a, b)
}
