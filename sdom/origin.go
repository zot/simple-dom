package sdom

import "fmt"

// CRC: crc-Loc.md | R89, R90, R91
//
// Origin identifies one PARSE, not one file. Two scans of the same source are two
// Origins, which is what makes "same file, different parser" answerable.
//
// It is the parse CONTEXT rather than the document: a context exists before any
// node does — a scan mints it, scans, and only then builds the document from what
// it emitted — so holding a document would need back-patching over every node.
//
// It is a concrete type rather than an interface, because each schema's context
// is its own concrete type and there is no single one for a location to hold. It
// doubles as a lookup key.
//
// It carries a field rather than being an empty marker: Go may give every
// zero-size allocation the same address, so an empty Origin would compare equal to
// an unrelated one — failing as an identity exactly where it was needed.
type Origin struct {
	Name string // a path, a URL, or whatever the caller finds useful
}

// String names the parse for a diagnostic. A parse nobody named still has to be
// distinguishable from another, so an unnamed one reports its identity.
func (o *Origin) String() string {
	if o == nil {
		return "unknown"
	}
	if o.Name == "" {
		return fmt.Sprintf("unnamed parse %p", o)
	}
	return o.Name
}
