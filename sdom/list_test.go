package sdom

import (
	"slices"
	"testing"
)

// CRC: crc-List.md | R199, R200, R202
func TestListItemsDeriveAndTheLiteralIsPreserved(t *testing.T) {
	cases := map[string][]string{
		"a,b":                        {"a", "b"},
		" a ,  b ":                   {"a", "b"},
		"crc-Store.md, crc-Index.md": {"crc-Store.md", "crc-Index.md"},
	}
	for src, want := range cases {
		l, rest, ok := ParseList(src, Synthetic(0))
		if !ok || rest != "" {
			t.Fatalf("%q: ok=%v rest=%q", src, ok, rest)
		}
		if got := l.Items(); !slices.Equal(got, want) {
			t.Errorf("%q: items %q, want %q", src, got, want)
		}
		if r, _ := l.Render(); r != src {
			t.Errorf("%q rendered %q", src, r)
		}
		if len(l.Kids()) != 1 {
			t.Errorf("%q: %d children, want 1", src, len(l.Kids()))
		}
	}
	if _, rest, _ := ParseList("a, b | rest", Synthetic(0)); rest != "| rest" {
		t.Errorf("remainder %q, want %q", rest, "| rest")
	}
}

// CRC: crc-List.md | Seq: seq-anchor.md#3 | R201, R203
func TestSetItemsRewritesCanonicallyAndRefuses(t *testing.T) {
	l, _, _ := ParseList(" a ,  b ", Source(0, 8))
	if err := l.SetItems([]string{"x", "y", "z"}); err != nil {
		t.Fatal(err)
	}
	if r, _ := l.Render(); r != "x, y, z" || !l.Location().Altered() {
		t.Fatalf("after set: %q altered=%v", r, l.Location().Altered())
	}
	for _, bad := range [][]string{{"a,b", "c"}, {"p q"}} {
		if err := l.SetItems(bad); err == nil {
			t.Errorf("%q: accepted, and Items reads %q", bad, l.Items())
		}
	}
	if r, _ := l.Render(); r != "x, y, z" {
		t.Errorf("a refused write changed the literal to %q", r)
	}
}

// CRC: crc-RequirementList.md | R204, R206
func TestRequirementRangesExpand(t *testing.T) {
	l, rest, ok := ParseRequirementList("R4, R5-7, R10-R12, R9-R8", Synthetic(0))
	if !ok || rest != "" {
		t.Fatalf("ok=%v rest=%q", ok, rest)
	}
	if got, want := l.Items(), []int{4, 5, 6, 7, 10, 11, 12, 8}; !slices.Equal(got, want) {
		t.Errorf("items %v, want %v", got, want)
	}
	p, _, _ := ParseList("R5-7", Synthetic(0))
	if got := p.Items(); !slices.Equal(got, []string{"R5-7"}) {
		t.Errorf("the plain parser read %q; a range is not its business", got)
	}
}

// CRC: crc-RequirementList.md | R205
func TestRequirementSetItemsCondenses(t *testing.T) {
	l, _, _ := ParseRequirementList("R1", Synthetic(0))
	l.SetItems([]int{12, 7, 8, 9, 4, 4, 10, 11})
	if r, _ := l.Render(); r != "R4, R7-12" {
		t.Errorf("got %q, want %q", r, "R4, R7-12")
	}
	l.SetItems([]int{7, 8})
	if r, _ := l.Render(); r != "R7, R8" {
		t.Errorf("got %q, want %q", r, "R7, R8")
	}
}
