// CRC: crc-MutationWindow.md | R120
package sdom

import (
	"testing"
	"unsafe"
)

// sharesBacking reports whether inner points into outer's bytes — that is,
// whether inner is a SLICE of outer rather than a copy with equal content.
//
// This needs unsafe because it is the only way to observe the difference: two
// strings with the same bytes are `==` whether one was sliced or freshly built.
// R120 says which path ran "is not observable in the result", and that is exactly
// what made the requirement unfalsifiable — the fast path was silently building
// the string it was supposed to avoid, and nothing could tell. This is the
// narrowest thing that can tell.
func sharesBacking(inner, outer string) bool {
	if len(inner) == 0 || len(outer) == 0 {
		return false
	}
	lo := uintptr(unsafe.Pointer(unsafe.StringData(outer)))
	at := uintptr(unsafe.Pointer(unsafe.StringData(inner)))
	return at >= lo && at+uintptr(len(inner)) <= lo+uintptr(len(outer))
}

// CRC: crc-MutationWindow.md | R120
//
// Merging two faithful nodes reuses the document's source; merging when either is
// altered cannot, because those bytes are not in the source at all.
func TestMergeSlicesTheSourceWhenItCan(t *testing.T) {
	build := func(alter bool) (*Doc, *Text, *Text) {
		const src = "alpha beta"
		a := NewText("alpha ", Source(0, 6))
		b := NewText("beta", Source(6, 4))
		d := New(src, 0, a, b)
		if alter {
			b.SetText("beta") // same bytes, but now altered
		}
		return d, a, b
	}

	d, a, b := build(false)
	if err := d.Mutate(func() error { _, err := d.Merge(a, b); return err }); err != nil {
		t.Fatal(err)
	}
	merged := d.Nodes()[0].(*Text)
	if !sharesBacking(merged.text, d.source) {
		t.Errorf("two faithful nodes must merge by slicing the source, not by building a new string")
	}

	d, a, b = build(true)
	if err := d.Mutate(func() error { _, err := d.Merge(a, b); return err }); err != nil {
		t.Fatal(err)
	}
	merged = d.Nodes()[0].(*Text)
	if sharesBacking(merged.text, d.source) {
		t.Errorf("an altered operand's bytes are not in the source, so the merge must build a new string")
	}
	if got, _ := d.Render(); got != "alpha beta" {
		t.Errorf("either path must produce the same bytes; got %q", got)
	}
}
