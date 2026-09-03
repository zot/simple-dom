package schema

import (
	"regexp"
	"strings"

	"github.com/zot/simple-dom/sdom"
)

// CRC: crc-MarkdownParser.md | R226, R227
//
// LangMarkdown is the markdown BASE: the subset mini-spec's own documents use, and
// only the structure a reader binds. The trajectory file schemas embed it; nothing
// parses a file with it alone. Depth is a list item's indentation, so it is an
// IndentLang.
//
// Match order matters twice: the fence before the code span, since ``` starts with `;
// and links are NOT groups — `[` open no group — because a line-head marker may share
// no first byte with an opener (R231). Both code forms carry one kind, so a consumer
// that skips code skips one label.
//
// Bold and strike are RESTRICTED with escape hatches — the template-literal shape —
// and not code-mode groups, because a symmetric marker in code mode reopens rather
// than closes: the parser tries openers before the enclosing closer there, and a
// second `**` would nest forever and swallow the file. A restricted group checks its
// closer first. Bold admits code spans; strike admits bold and code spans, which is
// what `~~**Item 1 — …**~~` and `**LANDED (`abc`)**` need and nothing more.
var LangMarkdown = sdom.IndentLang{
	BracketLang: sdom.BracketLang{Brackets: []sdom.BracketGroup{
		{Open: []string{"```"}, Close: []string{"```"}, AllowedInner: []string{}, Kind: "code"},
		{Open: []string{"`"}, Close: []string{"`"}, AllowedInner: []string{}, Kind: "code"},
		{Open: []string{"**"}, Close: []string{"**"}, AllowedInner: []string{"`"}},
		{Open: []string{"~~"}, Close: []string{"~~"}, AllowedInner: []string{"**", "`"}},
	}},
	Tab: 4,
}

// CRC: crc-MarkdownParser.md | R230, R234, R235
//
// Heading, ListItem and Checkbox are line-head markers, like Indent: each holds only
// the bytes it matched. Extents are a consumer's derivation from the array.
type Heading struct{ sdom.Text }

// ListItem holds the `- ` that opens an item.
type ListItem struct{ sdom.Text }

// Checkbox holds `[ ]` or `[x]`, recognized only immediately after a ListItem.
type Checkbox struct{ sdom.Text }

// CRC: crc-MarkdownParser.md | R230
// Level is derived from the bytes: the count of `#`.
func (h *Heading) Level() int {
	s, _ := h.Render()
	return strings.IndexByte(s, ' ')
}

// CRC: crc-Node.md | R10, R13, R235
func (h *Heading) Equals(o sdom.Node) bool {
	x, ok := o.(*Heading)
	return ok && h.Text.Equals(&x.Text)
}

// CRC: crc-Node.md | R10, R13, R235
func (l *ListItem) Equals(o sdom.Node) bool {
	x, ok := o.(*ListItem)
	return ok && l.Text.Equals(&x.Text)
}

// CRC: crc-Node.md | R10, R13, R235
func (c *Checkbox) Equals(o sdom.Node) bool {
	x, ok := o.(*Checkbox)
	return ok && c.Text.Equals(&x.Text)
}

var (
	headingRe  = regexp.MustCompile(`^#{1,6} `)
	listRe     = regexp.MustCompile(`^- `)
	checkboxRe = regexp.MustCompile(`^\[[ x]\]`)
)

// CRC: crc-MarkdownParser.md | R228
//
// MarkdownParser is one pass over markdown: it holds an IndentParser the way that one
// holds a BracketParser, delegates first, and emits a line-head marker where the
// delegate emitted nothing. No post-pass and no Replace.
type MarkdownParser struct {
	indent *sdom.IndentParser
}

// CRC: crc-MarkdownParser.md | R228
func NewMarkdownParser() *MarkdownParser {
	return &MarkdownParser{indent: sdom.NewIndentParser(&LangMarkdown)}
}

// CRC: crc-MarkdownParser.md | R228
// Indent returns the delegate, whose contexts a consumer reads.
func (p *MarkdownParser) Indent() *sdom.IndentParser { return p.indent }

// CRC: crc-MarkdownParser.md | Seq: seq-markdown.md#1 | R228, R229, R230, R231, R232
//
// Parse delegates, and checks for a line-head marker only when the delegate did
// nothing. Checking AFTER delegating is sound because `#`, `-` and `[` open no group
// (R231); and inside a fence or code span this parser is never offered a position at
// all, since the bracket parser takes the loop (R232).
func (p *MarkdownParser) Parse(st *sdom.ParserState) {
	pos, n := st.Pos(), st.NodeCount()
	p.indent.Parse(st)
	if st.Pos() != pos || st.NodeCount() != n {
		return
	}
	m, kind := p.head(st)
	if m == "" {
		return
	}
	text := *sdom.NewText(m, st.At(st.Pos(), len(m)))
	switch kind {
	case "heading":
		st.Emit(&Heading{text})
	case "list":
		st.Emit(&ListItem{text})
	case "checkbox":
		st.Emit(&Checkbox{text})
	}
}

// CRC: crc-MarkdownParser.md | Seq: seq-markdown.md#1.3 | R229, R230
//
// head reports the line-head marker at the position, if any: the bytes it matched and
// the kind NodeType answers with. Line-headness is read from Last() — which is what
// the live text run exists for.
func (p *MarkdownParser) head(st *sdom.ParserState) (marker, kind string) {
	rest := st.Src()[st.Pos():]
	if _, afterItem := st.Last().(*ListItem); afterItem {
		if m := checkboxRe.FindString(rest); m != "" {
			return m, "checkbox"
		}
		return "", ""
	}
	if !lineHead(st.Last()) {
		return "", ""
	}
	if m := headingRe.FindString(rest); m != "" {
		return m, "heading"
	}
	if m := listRe.FindString(rest); m != "" {
		return m, "list"
	}
	return "", ""
}

// CRC: crc-MarkdownParser.md | Seq: seq-markdown.md#1.3.1 | R229
//
// lineHead reports whether the node just emitted leaves the position at the head of a
// line: an Indent, or a Text ending in a newline followed only by spaces or tabs — an
// indented line at an UNCHANGED level keeps its leading spaces in the run.
func lineHead(last sdom.Node) bool {
	switch last := last.(type) {
	case *sdom.Indent:
		return true
	case *sdom.Text:
		s, _ := last.Render()
		return strings.HasSuffix(strings.TrimRight(s, " \t"), "\n")
	}
	return false
}

// CRC: crc-MarkdownParser.md | R233
// NodeType answers for the three line-head kinds and otherwise delegates.
func (p *MarkdownParser) NodeType(st *sdom.ParserState) (string, bool) {
	if k, ok := p.indent.NodeType(st); ok {
		return k, true
	}
	m, kind := p.head(st)
	return kind, m != ""
}

// CRC: crc-MarkdownParser.md | R233
func (p *MarkdownParser) Done(d *sdom.Doc) { p.indent.Done(d) }
