package sdom

import (
	"errors"
	"fmt"
	"slices"
)

// CRC: crc-MutationWindow.md | R55
// ErrPoisoned is returned by every operation on a document whose mutation failed.
// There is no reset: recovery is to re-parse, because a poisoned document means a
// bug in the code that wrote to it.
var ErrPoisoned = errors.New("sdom: document is poisoned by a failed mutation; re-parse it")

// CRC: crc-MutationWindow.md | R47
// ErrNotMutating is returned by a structural edit attempted outside a mutation
// window, where nothing would rebuild the indices it invalidates.
var ErrNotMutating = errors.New("sdom: structural edits require an open mutation window")

// CRC: crc-MutationWindow.md | R49
// inMutation is the typed sentinel every guarded read panics with. Mutate
// converts it to an error; anything else panicking is re-raised untouched.
type inMutation struct{ op string }

func (e inMutation) Error() string {
	return "sdom: " + e.op + " is not available inside a mutation window"
}

// CRC: crc-MutationWindow.md | Seq: seq-mutate.md#1.4.3 | R49
func (d *Doc) guard(op string) {
	if d.mutating {
		panic(inMutation{op: op})
	}
}

// CRC: crc-MutationWindow.md | Seq: seq-mutate.md#1 | R47, R48, R50, R51, R52, R53, R54, R55, R56
//
// Mutate brackets a set of edits. Inside f, edits are direct: a content write
// changes the node, a structural change changes the array, and nothing is queued
// or deferred. What makes the document never observably half-edited is not
// deferral but the guard — every index-backed read refuses while the window is
// open.
//
// Resolve your targets before you enter. Navigation is legal outside, and a node
// reference survives whatever the mutation does; a position would not.
//
// A nested call is a pass-through, so the window belongs to the outermost call.
// The state is saved and restored rather than counted.
//
// There is no rollback. Edits landed as they were made, so there is nothing to
// restore to, and a partial restore would produce a plausible wrong state rather
// than an obvious one. An error or a panic escaping f poisons the document: both
// mean something the caller could not or did not handle got out, so what was
// applied by then is unknown. A caller that can handle a failure handles it
// inside f rather than propagating it.
//
// There is no operation log, no queued plan, no transaction and no undo — and
// none of them is prevented. This window is the seam a later layer would attach
// one to, and edits keyed by node identity rather than position are already the
// primitive such a system needs.
func (d *Doc) Mutate(f func() error) (err error) {
	if d.poisoned {
		return ErrPoisoned
	}
	if d.mutating {
		return f() // nested: a pass-through
	}
	d.mutating = true
	defer func() {
		d.mutating = false
		if r := recover(); r != nil {
			d.poisoned = true
			s, ok := r.(inMutation)
			if !ok {
				panic(r) // a foreign panic keeps its value and its stack
			}
			err = s
			return
		}
		if err != nil {
			d.poisoned = true
			return
		}
		if rerr := d.rebuild(); rerr != nil {
			d.poisoned = true
			err = rerr
		}
	}()
	return f()
}

// editable reports whether a structural edit may proceed.
func (d *Doc) editable() error {
	if d.poisoned {
		return ErrPoisoned
	}
	if !d.mutating {
		return ErrNotMutating
	}
	return nil
}

// CRC: crc-MutationWindow.md | Seq: seq-mutate.md#2 | R31, R32, R33
//
// Merge joins two adjacent nodes into one. It is a Doc method because it changes
// node membership.
//
// Adjacency is checked from the two locations when both operands are faithful —
// their offsets and lengths prove it at the call site, with no lookup. Otherwise
// it is not checked: an altered node's offset is historical while its length is
// current, so the arithmetic proves nothing about it, and carrying adjacency past
// that point would mean tracking it through the mutation. The requirement still
// holds there; only the check is absent.
//
// The merged location is faithful only if both operands are, and takes the first
// operand's offset when it has one and the second's otherwise.
func (d *Doc) Merge(a, b Node) (Node, error) {
	if err := d.editable(); err != nil {
		return nil, err
	}
	ta, oka := a.(*Text)
	tb, okb := b.(*Text)
	if !oka || !okb {
		return nil, fmt.Errorf("sdom: Merge is defined for *Text, got %T and %T", a, b)
	}
	la, lb := ta.Location(), tb.Location()
	if la.Faithful() && lb.Faithful() && la.Offset()+la.Length() != lb.Offset() {
		return nil, fmt.Errorf("sdom: Merge: %d+%d is not adjacent to %d",
			la.Offset(), la.Length(), lb.Offset())
	}
	i := d.find(a)
	if i < 0 {
		return nil, errors.New("sdom: Merge: first node is not in this document")
	}
	if i+1 >= len(d.dom) || d.dom[i+1] != b {
		return nil, errors.New("sdom: Merge: nodes are not adjacent in this document")
	}
	merged := &Text{text: ta.text + tb.text, loc: mergeLocs(la, lb)}
	d.dom = slices.Replace(d.dom, i, i+2, Node(merged))
	d.dirty = true
	return merged, nil
}

// CRC: crc-MutationWindow.md | R30, R32
//
// Split divides a node's span in two at at, an offset relative to the node's own
// start. Both halves keep the provenance of the part of the source they now
// cover; bytes and provenance do not move, only the boundary between them.
//
// A zero-length half is allowed: it is an insertion point a tool can later write
// into without moving its neighbours.
func (d *Doc) Split(n Node, at int) (Node, Node, error) {
	if err := d.editable(); err != nil {
		return nil, nil, err
	}
	t, ok := n.(*Text)
	if !ok {
		return nil, nil, fmt.Errorf("sdom: Split is defined for *Text, got %T", n)
	}
	if at < 0 || at > len(t.text) {
		return nil, nil, fmt.Errorf("sdom: Split at %d is outside a node of %d bytes", at, len(t.text))
	}
	i := d.find(n)
	if i < 0 {
		return nil, nil, errors.New("sdom: Split: node is not in this document")
	}
	ll, rl := splitLoc(t.Location(), at)
	left := &Text{text: t.text[:at], loc: ll}
	right := &Text{text: t.text[at:], loc: rl}
	d.dom = slices.Replace(d.dom, i, i+1, Node(left), Node(right))
	d.dirty = true
	return left, right, nil
}

// CRC: crc-MutationWindow.md | R32, R43
//
// Remove drops a node from the document. Its bytes leave the render with it,
// which is the one injection that reaches a Render handing back retained source:
// that failure loses no bytes and breaks no round-trip, so only requiring the
// output to lack the node can see it.
func (d *Doc) Remove(n Node) error {
	if err := d.editable(); err != nil {
		return err
	}
	i := d.find(n)
	if i < 0 {
		return errors.New("sdom: Remove: node is not in this document")
	}
	d.dom = slices.Delete(d.dom, i, i+1)
	d.dirty = true
	return nil
}
