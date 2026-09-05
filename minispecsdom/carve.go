package minispecsdom

import (
	"errors"
	"strings"

	"github.com/zot/simple-dom/sdom"
	"github.com/zot/simple-dom/sdom/schema"
)

// CRC: crc-Carve.md | R248
//
// Carve is the carve file schema: it embeds the markdown base, owns the document, and
// adds the status block. The first of the four trajectory file schemas.
type Carve struct {
	doc    *sdom.Doc
	parser *schema.MarkdownParser
	ctx    *sdom.BracketContext

	status    *schema.Heading
	parts     []*Part
	stateless []*PartLine
	unread    []Unread
}

// CRC: crc-Carve.md | R252
//
// Part is a status line with a checkbox, placed: its depth is the bullet's leading
// whitespace and its parent the nearest preceding part with a smaller depth.
type Part struct {
	*PartLine
	Depth  int
	Parent *Part
	line   int // 1-based, at parse time
}

// CRC: crc-Carve.md | R283
func (p *Part) Line() int { return p.line }

// ErrNoPart reports a write to a key no part carries.
var ErrNoPart = errors.New("minispecsdom: no part with that key")

// CRC: crc-Carve.md | R282
//
// ErrLanded reports a Land over a part whose checkbox is already checked. Refused rather
// than idempotent: a second landing with a different attribution would otherwise keep the
// first record silently.
var ErrLanded = errors.New("minispecsdom: the part is already landed")

// CRC: crc-PartLine.md | R281
//
// ErrReopen reports an OPEN written over a part whose checkbox is checked. A landed part
// is never returned to OPEN by a write, whatever the caller remembers.
var ErrReopen = errors.New("minispecsdom: the part is landed; a write may not reopen it")

// CRC: crc-PartLine.md | R280
//
// DeviationError refuses a write over a line carrying deviations, naming each rule and
// the shape it must take, so the caller can print the migration rather than write a
// canonical marker onto a non-conforming line.
type DeviationError struct {
	Key        string
	Deviations []Deviation
}

func (e *DeviationError) Error() string {
	var b strings.Builder
	b.WriteString("minispecsdom: ")
	if e.Key != "" {
		b.WriteString(e.Key + ": ")
	}
	b.WriteString("the line carries deviations, so it takes no write:")
	for _, d := range e.Deviations {
		b.WriteString("\n  " + d.Rule + ": " + d.Target)
	}
	return b.String()
}

// CRC: crc-Carve.md | Seq: seq-carve.md#1 | R248, R249, R250, R251, R252
//
// ParseCarve parses with the base, finds the status region, and turns the list items
// inside it — and only those — into part lines.
func ParseCarve(src string) *Carve {
	c := &Carve{parser: schema.NewMarkdownParser()}
	c.doc = sdom.Parse(src, 0, c.parser)
	c.ctx = c.parser.Indent().Brackets().Context()
	c.unread = unclosed(c.ctx) // R302

	from, to, ok := c.statusRegion()
	if !ok {
		return c
	}
	lines, _ := partLines(c.doc, c.ctx, func(it *schema.ListItem) bool {
		off := it.Location().Offset()
		return off >= from && off < to
	})
	var stack []*Part
	for _, l := range lines {
		if l.Checkbox() == nil {
			c.stateless = append(c.stateless, l)
			continue
		}
		p := &Part{PartLine: l, Depth: c.depth(l), line: c.doc.Line(l.item.Location().Offset())}
		for len(stack) > 0 && stack[len(stack)-1].Depth >= p.Depth {
			stack = stack[:len(stack)-1]
		}
		if len(stack) > 0 {
			p.Parent = stack[len(stack)-1]
		}
		stack = append(stack, p)
		c.parts = append(c.parts, p)
	}
	return c
}

// CRC: crc-Carve.md | Seq: seq-carve.md#1.2 | R249
//
// statusRegion is the byte range of the status block's body: from the level-2
// heading whose text is Status to the next heading of level 2 or higher. Matched on
// Heading NODES, so a fenced sample is invisible by construction.
func (c *Carve) statusRegion() (from, to int, ok bool) {
	nodes := c.doc.Nodes()
	start := -1
	for i, n := range nodes {
		h, isH := n.(*schema.Heading)
		if isH && h.Level() == 2 && i+1 < len(nodes) && headingText(nodes[i+1]) == "Status" {
			c.status, start = h, i
			break
		}
	}
	if start < 0 {
		return 0, 0, false
	}
	from = c.status.Location().Offset()
	for _, n := range nodes[start+1:] {
		if h, isH := n.(*schema.Heading); isH && h.Level() <= 2 {
			return from, h.Location().Offset(), true
		}
	}
	return from, len(c.doc.Source()), true
}

// headingText is the heading's title: the text after the marker, up to the newline.
func headingText(n sdom.Node) string {
	t, ok := n.(*sdom.Text)
	if !ok {
		return ""
	}
	s, _ := t.Render()
	s, _, _ = strings.Cut(s, "\n")
	return strings.TrimSpace(s)
}

// CRC: crc-Carve.md | R252
// depth is the bullet's leading whitespace, read from the source at the item's line.
func (c *Carve) depth(l *PartLine) int {
	off := l.item.Location().Offset()
	src := c.doc.Source()
	start := strings.LastIndexByte(src[:off], '\n') + 1
	return off - start
}

// CRC: crc-Carve.md | R248
func (c *Carve) Doc() *sdom.Doc { return c.doc }

// CRC: crc-Carve.md | R248
func (c *Carve) Render() (string, error) { return c.doc.Render() }

// CRC: crc-Carve.md | R249
func (c *Carve) HasStatus() bool { return c.status != nil }

// CRC: crc-Carve.md | R251
func (c *Carve) Parts() []*Part { return c.parts }

// CRC: crc-Carve.md | R251
func (c *Carve) Stateless() []*PartLine { return c.stateless }

// CRC: crc-Carve.md | R302
// Unread lists what the reader could not read: today, every group open at end of input.
func (c *Carve) Unread() []Unread { return c.unread }

// CRC: crc-Carve.md | R253
func (c *Carve) Part(key string) *Part {
	for _, p := range c.parts {
		if p.Key() == key {
			return p
		}
	}
	return nil
}

// CRC: crc-Carve.md | Seq: seq-carve.md#2.3 | R253, R254
func (c *Carve) SetMarker(key, verb, attribution string) error {
	p := c.Part(key)
	if p == nil {
		return ErrNoPart
	}
	return p.SetMarker(verb, attribution)
}

// CRC: crc-Carve.md | Seq: seq-carve.md#2 | R253, R255, R279, R282
//
// Land is the completion write, three markings in one act because the format requires
// them to agree: the box, the strike, and the LANDED record through the marker rule.
// Every refusal is decided before the first marking, so a refused line is byte-identical.
func (c *Carve) Land(key, attribution string) error {
	p := c.Part(key)
	if p == nil {
		return ErrNoPart
	}
	if err := p.refuse(); err != nil { // Seq: seq-carve.md#2.1.1
		return err
	}
	if p.Checkbox().Checked() { // Seq: seq-carve.md#2.1.2
		return ErrLanded
	}
	p.Checkbox().SetChecked(true)
	p.Strike(true)
	return p.SetMarker("LANDED", attribution)
}
