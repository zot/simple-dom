package sdom

import (
	"errors"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// CRC: crc-List.md | R202
//
// listRe consumes a comma-separated list of non-space, non-comma items, with
// whitespace as glue anywhere. Anchored at the head: what follows is the caller's.
var listRe = regexp.MustCompile(`^\s*[^\s,]+(?:\s*,\s*[^\s,]+)*\s*`)

// CRC: crc-RequirementList.md | R204, R206
//
// reqListRe is the ONLY pattern that accepts a range. Items are Rn, Rn-Rm or Rn-m.
var reqListRe = regexp.MustCompile(`^\s*R\d+(?:-R?\d+)?(?:\s*,\s*R\d+(?:-R?\d+)?)*\s*`)

// ErrListRefused reports a SetItems whose render would not read back as the items
// that were set — an item carrying a comma or whitespace. The literal is unchanged.
var ErrListRefused = errors.New("sdom: an item would not read back; the list is unchanged")

// CRC: crc-List.md | R199, R200, R201
//
// List is a comma-separated field as a stencil: a compound with exactly ONE child,
// the Text of the whole field. Its only write is whole-field replace, so per-item
// boundaries would imply an edit nobody makes.
type List struct {
	Compound
	txt *Text
}

// CRC: crc-List.md | R202
//
// ParseList consumes a list from the head of text and returns the node, what it
// did not consume, and whether anything matched. Whether leftover bytes are
// acceptable is the caller's business.
func ParseList(text string, loc Loc) (*List, string, bool) {
	l := &List{}
	rest, ok := l.parse(listRe, text, loc)
	return l, rest, ok
}

func (l *List) parse(re *regexp.Regexp, text string, loc Loc) (string, bool) {
	m := re.FindStringIndex(text)
	if m == nil {
		return text, false
	}
	span := Synthetic(m[1]).In(loc.Origin())
	if loc.Offset() >= 0 {
		span = Source(loc.Offset(), m[1]).In(loc.Origin())
	}
	l.txt = NewText(text[:m[1]], span)
	l.Compound = Compound{kids: []Node{l.txt}, loc: span}
	return text[m[1]:], true
}

// CRC: crc-List.md | R200
// Items derives the values from the literal on every call and stores nothing.
func (l *List) Items() []string {
	var out []string
	for _, it := range strings.Split(l.txt.text, ",") {
		if it = strings.TrimSpace(it); it != "" {
			out = append(out, it)
		}
	}
	return out
}

// CRC: crc-List.md | Seq: seq-anchor.md#3 | R201, R203
//
// SetItems rewrites the whole literal canonically — and is GUARDED by re-parsing
// its own render. An item carrying a comma or whitespace renders fine and loses no
// bytes, so a byte round-trip is blind to it; the re-parse reads it back as two
// items or stops short, and the write is refused with the literal untouched. This
// is the one guarded write in sdom, because a node-level write changes one literal
// and rollback is free.
func (l *List) SetItems(items []string) error {
	literal := strings.Join(items, ", ")
	probe := &List{}
	rest, ok := probe.parse(listRe, literal, Synthetic(0))
	if !ok || rest != "" || !slices.Equal(probe.Items(), items) {
		return ErrListRefused
	}
	l.txt.SetText(literal)
	return nil
}

// CRC: crc-Node.md | R10, R13
func (l *List) Equals(other Node) bool {
	x, ok := other.(*List)
	return ok && l.Compound.Equals(&x.Compound)
}

// CRC: crc-RequirementList.md | R204, R205, R206
//
// RequirementList is a List whose items are requirement refs, where a range is one
// item and Items expands it. A range exists only through ParseRequirementList, so
// a plain list cannot contain one by construction.
type RequirementList struct {
	List
}

// CRC: crc-RequirementList.md | R204, R206
func ParseRequirementList(text string, loc Loc) (*RequirementList, string, bool) {
	l := &RequirementList{}
	rest, ok := l.parse(reqListRe, text, loc)
	return l, rest, ok
}

// CRC: crc-RequirementList.md | R204
//
// Items expands every range; a reversed range contributes only its low ref.
func (l *RequirementList) Items() []int {
	var out []int
	for _, it := range l.List.Items() {
		lo, hi, _ := strings.Cut(it, "-")
		a, _ := strconv.Atoi(strings.TrimPrefix(lo, "R"))
		b := a
		if hi != "" {
			b, _ = strconv.Atoi(strings.TrimPrefix(hi, "R"))
		}
		if b < a {
			a = b
		}
		for n := a; n <= b; n++ {
			out = append(out, n)
		}
	}
	return out
}

// CRC: crc-RequirementList.md | R205
//
// SetItems sorts, de-duplicates and emits maximally condensed: a run of three or
// more as R7-12, a pair as R7, R8. It cannot be refused — integers carry no
// separator — so it returns nothing.
func (l *RequirementList) SetItems(items []int) {
	l.txt.SetText(RequirementText(items))
}

// CRC: crc-RequirementList.md | R205
//
// RequirementText is the canonical literal for a set of refs — what SetItems
// writes, exported so a constructor can assemble the same bytes a write would.
func RequirementText(items []int) string {
	ns := slices.Clone(items)
	slices.Sort(ns)
	ns = slices.Compact(ns)
	var parts []string
	for i := 0; i < len(ns); {
		j := i
		for j+1 < len(ns) && ns[j+1] == ns[j]+1 {
			j++
		}
		if j-i >= 2 {
			parts = append(parts, "R"+strconv.Itoa(ns[i])+"-"+strconv.Itoa(ns[j]))
			i = j + 1
			continue
		}
		parts = append(parts, "R"+strconv.Itoa(ns[i]))
		i++
	}
	return strings.Join(parts, ", ")
}

// CRC: crc-Node.md | R10, R13
func (l *RequirementList) Equals(other Node) bool {
	x, ok := other.(*RequirementList)
	return ok && l.Compound.Equals(&x.Compound)
}
