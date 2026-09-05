package sdom

import (
	"fmt"
	"regexp"
	"slices"
	"unicode/utf8"
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

// CRC: crc-BracketGroup.md | R61, R63, R64, R66, R67, R290, R291, R292, R309
//
// BracketGroup is one set of matching markers: a code bracket, a string, or a
// comment. Separators are mid-group markers, such as "else" between "if" and
// "fi".
//
// Open lists literal openers; OpenRegex is a pattern opener instead, anchored where
// the parse stands, and the two are exclusive. Close is the ONE closer — a language
// whose brackets close on different words is several groups. CloseIsOpen makes the
// closer the text that opened this instance: for a pattern group that is the pattern's
// match here EQUAL to the opened text, so a greedy run match settles both edges of a run
// by itself. AfterOpen and BeforeClose are patterns with one role each — the first must
// match after an opener, the second against the rune before a closer, both satisfied at
// the edge of the input — which is how flanking becomes a table entry rather than a
// parser rule. RejectLongerCloses makes a pattern match longer than the opened text,
// inside a close-is-open group, an unbalanced closer that ends the group instead of
// content; shorter matches are content either way. Markers stay byte comparisons; only
// the patterns cost a regexp, compiled once when a parser is constructed.
//
// AllowedInner decides what is recognized inside the group: nil is code mode, where
// every group's openers are recognized; non-nil is parse-restricted, where only this
// group's Close, its Escape and the openers it lists are, and every other byte is
// literal — an empty (but non-nil) slice is pure raw mode. AllowedParent is the dual,
// and it is not optional: with a flat table, code mode recognizes every group's
// openers, so without it "${" fires at top level where it is really a "$" followed by
// a "{". nil is recognized in any context; non-nil, only while parsing inside one of
// the listed openers. A block comment nests only when its own opener appears in its
// AllowedInner. Nesting is not a field.
type BracketGroup struct {
	Open        []string
	OpenRegex   string
	Separators  []string
	Close       string
	CloseIsOpen bool
	AfterOpen   string
	BeforeClose string
	Escape      string

	// R309: inside a close-is-open pattern group, a match longer than the opened text
	// is rejected as content — an unbalanced closer that ends the group — rather than
	// literal, which is CommonMark's reading and the default.
	RejectLongerCloses bool

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
// patterns holds a table's compiled OpenRegex, AfterOpen and BeforeClose, one slot per
// group and nil where a group has none, plus each pattern opener's literal prefix as a
// cheap gate before the regexp runs. A parser and its context each hold one.
type patterns struct {
	open, after, before []*regexp.Regexp
	prefix              []string
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
		after:  make([]*regexp.Regexp, n),
		before: make([]*regexp.Regexp, n),
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
		if g.AfterOpen != "" {
			re, err := l.anchored(i, "AfterOpen", g.AfterOpen)
			if err != nil {
				return nil, err
			}
			p.after[i] = re
		}
		if g.BeforeClose != "" {
			re, err := l.anchoredEnd(i, "BeforeClose", g.BeforeClose)
			if err != nil {
				return nil, err
			}
			p.before[i] = re
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

// anchoredEnd compiles pat to match only at the END of the text handed to it — the form
// BeforeClose takes, since it reads the rune before a marker rather than the bytes after
// one — naming the group and the field it came from in the error.
func (l *BracketLang) anchoredEnd(i int, field, pat string) (*regexp.Regexp, error) {
	re, err := regexp.Compile(`(?:` + pat + `)$`)
	if err != nil {
		return nil, fmt.Errorf("sdom: %s: %s: %w", l.label(i), field, err)
	}
	return re, nil
}

// CRC: crc-BracketGroup.md | R309
// afterOK reports whether a group's AfterOpen holds at end, the position just past an
// opener — trivially when the group has none, and at end of input.
func afterOK(src string, end int, after *regexp.Regexp) bool {
	return after == nil || end >= len(src) || after.MatchString(src[end:])
}

// CRC: crc-BracketGroup.md | R309
// beforeOK reports whether a group's BeforeClose holds against the one rune before pos —
// trivially when the group has none, and at the start of input. One rune is all a
// flanking rule needs, so this is a bounded check rather than a regexp over the prefix.
func beforeOK(src string, pos int, before *regexp.Regexp) bool {
	if before == nil || pos == 0 {
		return true
	}
	_, w := utf8.DecodeLastRuneInString(src[:pos])
	return before.MatchString(src[pos-w : pos])
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
