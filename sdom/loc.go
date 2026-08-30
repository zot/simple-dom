// Package sdom is a simple DOM: a parse that models only what a tool operates
// on, keeps every other byte where it was, and re-emits the source with nothing
// but the intended change in it.
package sdom

// CRC: crc-Loc.md | R20, R21, R22, R23, R24
//
// Loc is a node's location. It separates provenance — where the node was read
// from — from faithfulness — whether it still renders the bytes there. The two
// stop agreeing the moment anything is edited, which is why they are not one
// field.
//
// The offset is stored biased by one so that absence is the zero value: a Loc
// nobody set reports no provenance rather than offset 0, which would be the
// first byte of the file and a plausible wrong answer.
type Loc struct {
	offset  int // biased: 0 is no provenance, n is source offset n-1
	length  int
	altered bool
}

// CRC: crc-Loc.md | R20
// Source returns a faithful location for a node read from offset.
func Source(offset, length int) Loc {
	return Loc{offset: offset + 1, length: length}
}

// CRC: crc-Loc.md | R24
// Synthetic returns a location with no provenance, for a node that was not read
// from anywhere.
func Synthetic(length int) Loc {
	return Loc{length: length}
}

// CRC: crc-Loc.md | R21
// Offset is the source position the node was read from, or -1 when there is no
// provenance. The bias is what makes that -1 fall out without a branch.
func (l Loc) Offset() int { return l.offset - 1 }

// CRC: crc-Loc.md | R22
// Length is the node's current rendered extent, in bytes.
func (l Loc) Length() int { return l.length }

// CRC: crc-Loc.md | R23, R27
// Altered reports that the node was read from Offset and no longer renders the
// bytes there. An altered node keeps its offset: clearing it would discard
// provenance a diagnostic still wants.
func (l Loc) Altered() bool { return l.altered }

// CRC: crc-Loc.md | R24, R25
// Faithful reports that the node has provenance and still renders exactly the
// source span at its offset. This is the question callers should be asking:
// for an unfaithful node Offset is historical while Length is current, so the
// pair names a span the node never owned (R29).
func (l Loc) Faithful() bool { return l.offset != 0 && !l.altered }

// alter returns l marked as no longer rendering the bytes at its offset. The
// offset is kept: an altered node keeps its provenance.
func (l Loc) alter() Loc {
	l.altered = true
	return l
}

// CRC: crc-Loc.md | Seq: seq-mutate.md#2.3 | R34, R35
//
// mergeLocs is the location half of Merge. Two rules, and both are load-bearing:
//
// The result is faithful only if both operands are. "Unfaithful" covers both
// ways an operand can fail — altered, or having no provenance at all — and
// keying on faithfulness rather than on altered is what stops a merge with a
// synthesized node claiming to be faithful at an offset whose bytes it does not
// render.
//
// The result takes the first operand's offset when it has one and the second's
// otherwise: the leftmost provenance in the merged span. That makes the rule
// associative, so merging a run of nodes gives the same answer however the
// merges are grouped.
func mergeLocs(a, b Loc) Loc {
	l := Loc{offset: a.offset, length: a.length + b.length}
	if l.offset == 0 {
		l.offset = b.offset
	}
	l.altered = l.offset != 0 && !(a.Faithful() && b.Faithful())
	return l
}

// CRC: crc-Loc.md | R30
//
// splitLoc is the location half of Split: boundaries move, bytes and provenance
// do not. Each half keeps the provenance of the part of the source it now
// covers. For an altered operand the right half's offset is best-effort, which
// is no worse than the operand's own — an altered offset is historical either
// way, and R29 says to use it alone.
func splitLoc(l Loc, at int) (Loc, Loc) {
	left := Loc{offset: l.offset, length: at, altered: l.altered}
	right := Loc{length: l.length - at, altered: l.altered}
	if l.offset != 0 {
		right.offset = l.offset + at
	}
	return left, right
}
