package minispecsdom

import (
	"errors"
	"slices"
	"strings"

	"github.com/zot/simple-dom/sdom"
	"github.com/zot/simple-dom/sdom/schema"
)

// Placeholder is what the Active region holds when nothing is active.
const Placeholder = "_No active item._"

// ErrOccupied reports a SetActive over a held item.
var ErrOccupied = errors.New("minispecsdom: `## Active` already holds an item; park it in the pending file first")

// CRC: crc-Current.md | R273
//
// Current is the current file schema: it embeds the markdown base, owns the document,
// and adds the `## Active` region — the only region a tool may write.
type Current struct {
	doc    *sdom.Doc
	parser *schema.MarkdownParser
	ctx    *sdom.BracketContext

	active   *schema.Heading
	from, to int // node indices: the heading, and one past the region's last node
	standing []string
}

// CRC: crc-Current.md | Seq: seq-current.md#1 | R273, R274, R275
//
// ParseCurrent refuses a document with no `## Active` or more than one — a region that
// cannot be told apart is refused, never picked. Matched on Heading nodes, so a fenced
// `## Active` counts for nothing.
func ParseCurrent(src string) (*Current, error) {
	c := &Current{}
	if err := c.parse(src); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Current) parse(src string) error {
	c.parser = schema.NewMarkdownParser()
	c.doc = sdom.Parse(src, 0, c.parser)
	c.ctx = c.parser.Indent().Brackets().Context()
	c.active, c.standing = nil, nil
	nodes := c.doc.Nodes()
	for i, n := range nodes {
		h, ok := n.(*schema.Heading)
		if !ok || h.Level() != 2 || i+1 >= len(nodes) {
			continue
		}
		title := headingText(nodes[i+1])
		if !strings.EqualFold(title, "Active") {
			c.standing = append(c.standing, title)
			continue
		}
		if c.active != nil {
			return errors.New("minispecsdom: more than one `## Active` heading; the region to clear is ambiguous")
		}
		c.active, c.from = h, i
	}
	if c.active == nil {
		return errors.New("minispecsdom: no `## Active` heading; the active item cannot be told from the standing context")
	}
	c.to = c.regionEnd(c.from)
	return nil
}

// CRC: crc-Current.md | Seq: seq-current.md#1.3 | R274
// regionEnd is one past the last node of the region opening at the heading at start:
// it ends at the next heading of level 2 or higher, so a standing section is never
// inside it.
func (c *Current) regionEnd(start int) int {
	nodes := c.doc.Nodes()
	for i := start + 1; i < len(nodes); i++ {
		if h, ok := nodes[i].(*schema.Heading); ok && h.Level() <= 2 {
			return i
		}
	}
	return len(nodes)
}

// body renders the region after the heading's title line.
func (c *Current) body() string {
	var b strings.Builder
	for _, n := range c.doc.Nodes()[c.from+1 : c.to] {
		s, _ := n.Render()
		b.WriteString(s)
	}
	_, rest, _ := strings.Cut(b.String(), "\n")
	return rest
}

// CRC: crc-Current.md | R273
func (c *Current) Doc() *sdom.Doc { return c.doc }

// CRC: crc-Current.md | R273
func (c *Current) Render() (string, error) { return c.doc.Render() }

// CRC: crc-Current.md | R275
// Active is the region's body, trimmed; "" when it holds only the placeholder.
func (c *Current) Active() string {
	s := strings.TrimSpace(c.body())
	if s == Placeholder {
		return ""
	}
	return s
}

// CRC: crc-Current.md | R275
func (c *Current) Occupied() bool { return c.Active() != "" }

// CRC: crc-Current.md | R275
func (c *Current) Standing() []string { return c.standing }

// CRC: crc-Current.md | Seq: seq-current.md#2 | R276, R277, R278
// SetActive replaces the region's body, refusing over a held item.
func (c *Current) SetActive(body string) error {
	if c.Occupied() {
		return ErrOccupied
	}
	return c.write(body)
}

// CRC: crc-Current.md | Seq: seq-current.md#2 | R276, R278
// Reset writes the placeholder.
func (c *Current) Reset() error { return c.write(Placeholder) }

// CRC: crc-Current.md | Seq: seq-current.md#2.2 | R276
//
// write replaces the region's body as one unit: split the heading's following text
// after its title line, remove every node from the right half to the region's end,
// and insert one synthetic text before the next heading or at the end. No node outside
// the region is addressed, so every byte outside it is unchanged by construction.
func (c *Current) write(body string) error {
	nodes := c.doc.Nodes()
	title, ok := nodes[c.from+1].(*sdom.Text)
	if !ok {
		return errors.New("minispecsdom: the Active heading has no title text")
	}
	titleText, _ := title.Render()
	cut := strings.IndexByte(titleText, '\n') + 1
	var next sdom.Node
	if c.to < len(nodes) {
		next = nodes[c.to]
	}
	text := "\n" + strings.TrimSuffix(body, "\n") + "\n"
	if next != nil {
		text += "\n" // the blank line before the next heading; none at the end of the file
	}
	err := c.doc.Mutate(func() error {
		// Cloned before the split: a sub-slice of the document's own array shifts
		// under a mutation.
		run := slices.Clone(nodes[c.from+2 : c.to])
		if cut < len(titleText) {
			_, right, err := c.doc.Split(title, cut)
			if err != nil {
				return err
			}
			run = append([]sdom.Node{right}, run...)
		}
		for _, n := range run {
			if err := c.doc.Remove(n); err != nil {
				return err
			}
		}
		return c.doc.Insert(next, sdom.NewText(text, sdom.Synthetic(len(text))))
	})
	if err != nil {
		return err
	}
	src, _ := c.doc.Render()
	return c.parse(src)
}
