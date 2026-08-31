package sdom

import (
	"errors"
	"regexp"
)

// CRC: crc-TodoItem.md | R96
//
// todoRe captures ONLY what TodoItem binds. "- [", "] " and the line's end are not
// in the pattern at all: they become Text from the gaps between named groups. The
// author writes what they bind, and a pattern cannot eat bytes.
var todoRe = regexp.MustCompile(`- \[(?P<checked>[^\]]*)\] ?(?P<label>.*)`)

// ErrNoMatch reports that a schema's regex did not match. What a non-match means
// is the caller's decision, so this is returned rather than acted on.
var ErrNoMatch = errors.New("sdom: the text does not match this stencil")

// CRC: crc-TodoItem.md | R96, R115, R116
//
// TodoItem is a markdown todo line — "- [ ] label" — and the worked example of
// what a schema writes. Real syntax rather than an invented fixture, and the case
// Bool exists for.
//
// Whether label deserves to be a bound field is an EDITABILITY question, not a
// parsing one: a tool that only ever toggles the checkbox would leave it as glue.
// It is bound here because a fixture needs two groups to prove the gap BETWEEN
// them is computed, which is a test-shape reason and is recorded as one.
type TodoItem struct {
	Compound
	checked *Bool
	label   *Text
}

// CRC: crc-TodoItem.md | R114
func (t *TodoItem) Checked() *Bool { return t.checked }

// CRC: crc-TodoItem.md | R115
func (t *TodoItem) Label() *Text { return t.label }

// CRC: crc-Node.md | R10, R13
func (t *TodoItem) Equals(other Node) bool {
	x, ok := other.(*TodoItem)
	return ok && t.Compound.Equals(&x.Compound)
}

// CRC: crc-TodoItem.md | Seq: seq-stencil.md#1 | R96, R102, R104
//
// Parse builds what each group becomes, keeps its own references, and takes the
// children. It is NOT on the Node interface: parsing constructs a concrete node and
// then sends it parse, so the interface is never involved at this moment.
func (t *TodoItem) Parse(text string, loc Loc) (string, error) {
	b, ok := NewStencilBuilder(todoRe, text, loc)
	if !ok {
		return text, ErrNoMatch
	}

	// A Bool is not patched in — the Text goes in the children and the Bool points
	// at it, so there is one copy of the bytes and nothing to keep in step.
	cs, cl := b.Group("checked")
	checked := NewText(cs, cl)
	t.checked = NewBool(checked)
	b.Put("checked", checked)

	ls, ll := b.Group("label")
	t.label = NewText(ls, ll)
	b.Put("label", t.label)

	kids, remain := b.Done()
	t.Compound = Compound{kids: kids, loc: b.Span()}
	return remain, nil
}
