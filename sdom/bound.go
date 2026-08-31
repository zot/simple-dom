package sdom

import "strings"

// CRC: crc-Bool.md | R111, R114
//
// Bool is a typed view over a Text node — the first bound value, and the shape
// every other one follows.
//
// The TEXT IS THE STORAGE and the value is a view of it. Nothing is stored twice,
// so nothing can drift, and Equals needs no special case: a kind whose state is
// derived from its children compares nothing beyond them.
//
// It POINTS AT the node in the child list; it does not replace it. The schema puts
// a Text in the children and hands the same pointer here.
//
// The ancestor is TCL's Tcl_Obj, which cached a typed representation beside the
// string and invalidated one when the other changed. Pre-8.0 Tcl was pure strings,
// re-parsed at every reference — which is this design. The cache was added in 1997
// under a constraint we do not have, so its absence here is a decision rather than
// an oversight.
type Bool struct{ txt *Text }

// CRC: crc-Bool.md | R114
// NewBool returns a view over txt, which must already be in the child list.
func NewBool(txt *Text) *Bool { return &Bool{txt: txt} }

// CRC: crc-Bool.md | R114
// Text returns the node this views, which is the one in the child list.
func (b *Bool) Text() *Text { return b.txt }

// CRC: crc-Bool.md | R112
//
// Value derives the value from the text, on every read. Nothing is normalised on
// the way in, so "[ ]", "[]" and "[    ]" all read false and all render back
// byte-exact.
func (b *Bool) Value() bool {
	return strings.EqualFold(strings.TrimSpace(b.txt.text), "x")
}

// CRC: crc-Bool.md | R113
//
// Set writes the value through to the text, which keeps its offset and becomes
// altered — so the enclosing stencil reports altered because a child is, and a
// diagnostic still has the provenance.
//
// Setting a value the text already carries changes NOTHING. That is what keeps
// "[X]" from being rewritten as "[x]" by a tool that only meant to confirm it:
// an edit reformats what it touched, and this touched nothing.
func (b *Bool) Set(v bool) {
	if b.Value() == v {
		return
	}
	if v {
		b.txt.SetText("x")
	} else {
		b.txt.SetText(" ")
	}
}
