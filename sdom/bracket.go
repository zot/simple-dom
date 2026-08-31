package sdom

import "slices"

// CRC: crc-BracketLang.md | R58, R59, R60, R62, R68, R70
//
// BracketLang is a language's whole lexical table, and nothing else. Supporting a
// new language means adding an entry, not writing code.
//
// There is no comment configuration. A line comment is a group closing on a
// newline; a block comment is a group closing on its terminator. Both are
// scan-restricted, which is what makes a comment non-nesting with a literal
// interior — a string is the same shape with different markers.
//
// There are no indent parameters and no flag enabling indentation. A language
// needing indent scope is described by a type that EMBEDS BracketLang, and the
// type is the flag, so a brace language carries no indent parameters at all
// rather than meaningless zeroed ones.
//
// Tables are Go values and no loader ships for a file format. The deciding reason
// is that nil and an empty slice are semantically distinct in both mode fields
// below — code mode versus pure raw mode — and that distinction is precisely what
// a document format loses, since an absent key and an empty list are the same
// thing to most readers of most formats.
type BracketLang struct {
	Brackets []BracketGroup
}

// CRC: crc-BracketGroup.md | R61, R63, R64, R65, R66, R67
//
// BracketGroup is one set of matching markers: a code bracket, a string, or a
// comment. Separators are mid-group markers, such as "else" between "if" and
// "fi".
//
// AllowedInner decides what is recognized inside the group:
//
//	nil            code mode — every group's openers are recognized inside
//	non-nil        scan-restricted — only this group's Close, its Escape, and the
//	               listed openers are recognized; every other byte is literal.
//	               An empty (but non-nil) slice is pure raw mode.
//
// AllowedParent is the dual, and it is not optional: with a flat table, code mode
// recognizes every group's openers, so without it "${" fires at top level where
// it is really a "$" followed by a "{".
//
//	nil            recognized in any context
//	non-nil        recognized only while scanning inside one of the listed openers
//
// A block comment nests only when its own opener appears in its AllowedInner.
// Nesting is not a field.
type BracketGroup struct {
	Open       []string
	Separators []string
	Close      []string
	Escape     string

	AllowedInner  []string
	AllowedParent []string
}

// CRC: crc-BracketGroup.md | R64, R65, R67
// Restricted reports scan-restricted mode. nil means code mode; non-nil, even
// empty, means restricted — which is why this is a nil test and not a length one.
func (g *BracketGroup) Restricted() bool { return g.AllowedInner != nil }

// CRC: crc-BracketLang.md | R58
// groupFor resolves an opener string back to the group that owns it, which is how
// AllowedInner reaches a nested group.
func (l *BracketLang) groupFor(open string) *BracketGroup {
	for i := range l.Brackets {
		if slices.Contains(l.Brackets[i].Open, open) {
			return &l.Brackets[i]
		}
	}
	return nil
}

// CRC: crc-BracketGroup.md | R66
// parentAllowed reports whether g may be recognized while enclosing is open.
// enclosing is nil at top level.
func (g *BracketGroup) parentAllowed(enclosing *BracketGroup) bool {
	if g.AllowedParent == nil {
		return true
	}
	if enclosing == nil {
		return false
	}
	for _, want := range g.AllowedParent {
		if slices.Contains(enclosing.Open, want) {
			return true
		}
	}
	return false
}

// CRC: crc-BracketGroup.md | R71
//
// matchAt reports whether marker occurs at pos, honouring word boundaries so that
// "do" does not fire inside "download" nor "fi" inside "file".
//
// Each edge is tested only when that edge of the marker is itself a word
// character: "end" is checked on both sides, "{" on neither, and "${" only on its
// trailing edge.
func matchAt(src string, pos int, marker string) bool {
	end := pos + len(marker)
	if marker == "" || end > len(src) || src[pos:end] != marker {
		return false
	}
	if isWordChar(marker[0]) && pos > 0 && isWordChar(src[pos-1]) {
		return false
	}
	if isWordChar(marker[len(marker)-1]) && end < len(src) && isWordChar(src[end]) {
		return false
	}
	return true
}

// matchAny returns the first of markers occurring at pos, or "" when none does.
// The empty string is a safe "no match" answer because matchAt never matches an
// empty marker.
func matchAny(src string, pos int, markers []string) string {
	for _, m := range markers {
		if matchAt(src, pos, m) {
			return m
		}
	}
	return ""
}

func isWordChar(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' || b == '_'
}
