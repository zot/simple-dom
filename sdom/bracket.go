package sdom

import (
	"fmt"
	"regexp"
	"slices"
)

// CRC: crc-BracketLang.md | R58, R59, R60, R62, R68, R70
//
// BracketLang is a language's whole bracket table, and nothing else. Supporting a
// new language means adding an entry, not writing code.
//
// There is no comment configuration. A line comment is a group closing on a
// newline; a block comment is a group closing on its terminator. Both are
// parse-restricted, which is what makes a comment non-nesting with a literal
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

	// R207: how this language WRITES a comment. Distinct from recognition: the
	// group's opener is "//" but a written comment wants "// ", and the closer is a
	// structural "\n". Kind is what the written comment must parse back as, and
	// equals the recognizing group's Kind — guarded by a test per shipped language,
	// never by a runtime check. An empty Prefix means nothing can be constructed.
	Comment CommentStyle
}

// CRC: crc-BracketLang.md | R207, R209
//
// CommentStyle is the construction template for a comment. sdom compares Kind and
// never reads it: the value is the language's, the way IndentLang.Transparent is.
type CommentStyle struct {
	Prefix, Suffix, Kind string
}

// CRC: crc-BracketGroup.md | R61, R63, R64, R66, R67, R290, R291, R292, R293
//
// BracketGroup is one set of matching markers: a code bracket, a string, or a
// comment. Separators are mid-group markers, such as "else" between "if" and
// "fi".
//
// Open lists literal openers; OpenRegex is a pattern opener instead, anchored where
// the parse stands, and the two are exclusive. Close is the ONE closer — a language
// whose brackets close on different words is several groups. CloseIsOpen makes the
// closer the text that opened this instance, which is one word for a symmetric group
// and the only way to say it for a pattern group, since the closer's length is not
// known until the opener has matched. Lookahead is an anchored pattern the bytes after
// an opener or closer must satisfy, satisfied at end of input: with "[^`]" a run of
// backticks is a marker only where no further backtick follows, on either edge, in
// any table order. Markers stay byte comparisons; only the two pattern fields cost a
// regexp, compiled once when a parser is constructed.
//
// AllowedInner decides what is recognized inside the group:
//
//	nil            code mode — every group's openers are recognized inside
//	non-nil        parse-restricted — only this group's Close, its Escape, and the
//	               listed openers are recognized; every other byte is literal.
//	               An empty (but non-nil) slice is pure raw mode.
//
// AllowedParent is the dual, and it is not optional: with a flat table, code mode
// recognizes every group's openers, so without it "${" fires at top level where
// it is really a "$" followed by a "{".
//
//	nil            recognized in any context
//	non-nil        recognized only while parsing inside one of the listed openers
//
// A block comment nests only when its own opener appears in its AllowedInner.
// Nesting is not a field.
type BracketGroup struct {
	Open        []string
	OpenRegex   string
	Separators  []string
	Close       string
	CloseIsOpen bool
	Lookahead   string
	Escape      string

	AllowedInner  []string
	AllowedParent []string

	// R168: an uninterpreted label for a layer above; NEVER read here. Indent scope
	// needs to know which groups are transparent to the level, and no property of a
	// group's shape answers that — a comment and a string are the same shape with
	// different markers. Storing and ignoring it is not a mode: the parsing rules
	// are identical whatever it says, and a language that labels nothing parses the
	// same. Nothing in this package may branch on it, or "there is no comment
	// configuration" stops being true.
	Kind string
}

// CRC: crc-BracketGroup.md | R64, R67
// Restricted reports parse-restricted mode. nil means code mode; non-nil, even
// empty, means restricted — which is why this is a nil test and not a length one.
func (g *BracketGroup) Restricted() bool { return g.AllowedInner != nil }

// CRC: crc-BracketLang.md | R146, R295
// GroupFor resolves a marker's text back to the group that owns it, exported so a
// layer outside this package can read a group's Kind from a marker node it holds. A
// literal opener is found by lookup; a pattern group's marker is found by matching the
// text against the pattern in full, compiled here on demand — this is the consumer
// path, and the parser and context hold their patterns compiled once instead.
func (l *BracketLang) GroupFor(text string) *BracketGroup {
	if i := l.indexFor(text); i >= 0 {
		return &l.Brackets[i]
	}
	for i := range l.Brackets {
		g := &l.Brackets[i]
		if g.OpenRegex == "" {
			continue
		}
		if ok, _ := regexp.MatchString(`^(?:`+g.OpenRegex+`)$`, text); ok {
			return g
		}
	}
	return nil
}

// CRC: crc-BracketLang.md | R58, R295
// groupFor resolves a NAME back to the group it names, which is how AllowedInner and
// AllowedParent reach a group: a string one of its Open lists, or its OpenRegex.
func (l *BracketLang) groupFor(name string) *BracketGroup {
	if i := l.indexFor(name); i >= 0 {
		return &l.Brackets[i]
	}
	return nil
}

// CRC: crc-BracketLang.md | R295
// indexFor is groupFor by index, for the callers that hold a group's compiled patterns
// in a slice parallel to the table; -1 when nothing is named.
func (l *BracketLang) indexFor(name string) int {
	for i := range l.Brackets {
		if l.Brackets[i].named(name) {
			return i
		}
	}
	return -1
}

// CRC: crc-BracketGroup.md | R295
// named reports whether name is one of this group's literal openers or its pattern.
func (g *BracketGroup) named(name string) bool {
	return slices.Contains(g.Open, name) || (g.OpenRegex != "" && g.OpenRegex == name)
}

// label names a group in a construction error: its first opener, its pattern, or
// its index.
func (l *BracketLang) label(i int) string {
	g := &l.Brackets[i]
	switch {
	case len(g.Open) > 0:
		return fmt.Sprintf("group %d (%q)", i, g.Open[0])
	case g.OpenRegex != "":
		return fmt.Sprintf("group %d (/%s/)", i, g.OpenRegex)
	}
	return fmt.Sprintf("group %d", i)
}

// CRC: crc-BracketLang.md | R296
//
// patterns holds a table's compiled OpenRegex and Lookahead, one slot per group and
// nil where a group has none, plus each pattern opener's literal prefix as a cheap
// gate before the regexp runs. A parser and its context each hold one.
type patterns struct {
	open, look []*regexp.Regexp
	prefix     []string
}

// CRC: crc-BracketLang.md | R296
//
// check compiles the table's patterns and reports the construction errors: a pattern
// that does not compile, an OpenRegex beside a non-empty Open, or CloseIsOpen beside a
// non-empty Close. NewBracketParser panics on the error; a test over every shipped
// table keeps that panic from a consumer.
func (l *BracketLang) check() (*patterns, error) {
	n := len(l.Brackets)
	p := &patterns{
		open:   make([]*regexp.Regexp, n),
		look:   make([]*regexp.Regexp, n),
		prefix: make([]string, n),
	}
	for i := range l.Brackets {
		g := &l.Brackets[i]
		if g.OpenRegex != "" && len(g.Open) > 0 {
			return nil, fmt.Errorf("sdom: %s: Open and OpenRegex are exclusive", l.label(i))
		}
		if g.CloseIsOpen && g.Close != "" {
			return nil, fmt.Errorf("sdom: %s: CloseIsOpen contradicts Close %q", l.label(i), g.Close)
		}
		if g.OpenRegex != "" {
			re, err := l.anchored(i, "OpenRegex", g.OpenRegex)
			if err != nil {
				return nil, err
			}
			p.open[i] = re
			// The anchored form reports no literal prefix, so the gate is read
			// off the bare pattern, which compiles because the anchored one did.
			bare, _ := regexp.Compile(g.OpenRegex)
			p.prefix[i], _ = bare.LiteralPrefix()
		}
		if g.Lookahead != "" {
			re, err := l.anchored(i, "Lookahead", g.Lookahead)
			if err != nil {
				return nil, err
			}
			p.look[i] = re
		}
	}
	return p, nil
}

// anchored compiles pat to match only where the parse stands — the form every table
// pattern takes — naming the group and the field it came from in the error.
func (l *BracketLang) anchored(i int, field, pat string) (*regexp.Regexp, error) {
	re, err := regexp.Compile(`^(?:` + pat + `)`)
	if err != nil {
		return nil, fmt.Errorf("sdom: %s: %s: %w", l.label(i), field, err)
	}
	return re, nil
}

// CRC: crc-BracketGroup.md | R293
// lookOK reports whether a group's Lookahead holds at end, the position just past a
// marker — trivially when the group has none, and at end of input, since a closer is
// often a file's last byte.
func lookOK(src string, end int, look *regexp.Regexp) bool {
	return look == nil || end >= len(src) || look.MatchString(src[end:])
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
		if enclosing.named(want) {
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
	return boundaryOK(src, pos, end)
}

// CRC: crc-BracketGroup.md | R71, R292
// boundaryOK is the word-boundary rule over the bytes src[pos:end], whether a literal
// marker or a pattern's match put them there.
func boundaryOK(src string, pos, end int) bool {
	if isWordChar(src[pos]) && pos > 0 && isWordChar(src[pos-1]) {
		return false
	}
	if isWordChar(src[end-1]) && end < len(src) && isWordChar(src[end]) {
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
