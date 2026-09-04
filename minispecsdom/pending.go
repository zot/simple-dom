package minispecsdom

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/zot/simple-dom/sdom"
	"github.com/zot/simple-dom/sdom/schema"
)

// CRC: crc-Pending.md | R260
//
// The positions trajectory-format.md names, read from an entry's rendered bytes.
// A backtick is written \x60, which keeps every pattern a raw string.
var (
	entryHeadRe  = regexp.MustCompile(`^## (\d+)\. \*\*(.*?)\*\*(?: \(([^)]*)\))?\.?\s*(.*)$`)
	sourceLineRe = regexp.MustCompile(`^\s*Source:\s*\[[^\]]*\]\(([^)]*)\)(?:,\s*(part|gap)\s+\x60([^\x60]+)\x60)?`)
	nextLineRe   = regexp.MustCompile(`^\s*Next:\s*(.*)$`)
)

// entryOpenRe recognizes the heading text an entry opens with: `N.`.
var entryOpenRe = regexp.MustCompile(`^\d+\.`)

// ErrNoEntry reports an id no entry carries.
var ErrNoEntry = errors.New("minispecsdom: no entry with that id")

// CRC: crc-Pending.md | R288, R289
//
// gapIDRe is one gap ID — O136, R42, T7. A range, a list or a `#` is not a gap source,
// because an entry discharges one thing.
var gapIDRe = regexp.MustCompile(`^[A-Z]\d+$`)

// ErrBadGapSource reports a Place whose gap key is not one gap ID.
var ErrBadGapSource = errors.New("minispecsdom: a gap source names exactly one gap ID")

// CRC: crc-Pending.md | R287
//
// SourceKind says what a Source: line read as: a carve part, a gap, or neither.
type SourceKind int

const (
	SourceNone SourceKind = iota // no Source: line, or one that named neither form
	SourcePart
	SourceGap
)

// CRC: crc-Pending.md | R258
//
// Pending is the pending file schema: it embeds the markdown base, owns the document,
// and adds the queue entry as a VIEW over a heading region.
type Pending struct {
	doc    *sdom.Doc
	parser *schema.MarkdownParser
	ctx    *sdom.BracketContext

	entries []*Entry
	unread  []Unread
}

// CRC: crc-Pending.md | R284
//
// Unread is one thing a reader did not read as an entry: its 1-based line and its
// text, so the report is one somebody can act on. Shared with the done schema.
type Unread struct {
	Line int
	Text string
}

// CRC: crc-Pending.md | R260
//
// Entry is one queue entry: its derived values and the run of flat nodes it spans.
type Entry struct {
	ID                   int
	Title, Skill, Status string
	SourceDoc, SourceKey string
	Kind                 SourceKind
	Next                 string

	head *schema.Heading
	line int         // 1-based, at parse time
	run  []sdom.Node // head first, through the last node of the region
	tail *sdom.Text  // the run's last text when the region ends inside it
	cut  int         // where the region ends inside tail
}

// CRC: crc-Pending.md | R283
func (e *Entry) Line() int { return e.line }

// CRC: crc-Pending.md | R261, R289
// EntryText is what Place writes.
type EntryText struct {
	ID                                               int
	Title, Skill, Status, SourceDoc, SourceKey, Next string
	Kind                                             SourceKind // SourceGap writes the gap form; anything else the part form
}

// CRC: crc-Pending.md | Seq: seq-pending.md#1 | R258, R259, R260, R264
func ParsePending(src string) *Pending {
	p := &Pending{}
	p.parse(src)
	return p
}

// parse reads src with the markdown base and derives the view over the result.
func (p *Pending) parse(src string) {
	p.parser = schema.NewMarkdownParser()
	p.doc = sdom.Parse(src, 0, p.parser)
	p.ctx = p.parser.Indent().Brackets().Context()
	p.scan()
}

// reload re-reads the document from its bytes. A placed entry is one synthetic text
// until it is parsed, so after a write the array and the view are rebuilt together.
func (p *Pending) reload() {
	src, _ := p.doc.Render()
	p.parse(src)
}

// scan re-derives the entries and the unread headings from the document.
func (p *Pending) scan() {
	p.entries, p.unread = nil, nil
	nodes := p.doc.Nodes()
	for i, n := range nodes {
		h, ok := n.(*schema.Heading)
		if !ok || h.Level() != 2 {
			continue
		}
		var title string
		if i+1 < len(nodes) {
			title = headingText(nodes[i+1])
		}
		line := p.doc.Line(h.Location().Offset())
		if !entryOpenRe.MatchString(title) {
			p.unread = append(p.unread, Unread{line, title})
			continue
		}
		end, cut := p.regionEnd(i)
		// Cloned: a sub-slice of the document's own array shifts under a Remove.
		e := &Entry{head: h, line: line, run: slices.Clone(nodes[i:end])}
		if cut >= 0 {
			e.tail, e.cut = nodes[end-1].(*sdom.Text), cut
		}
		if bad := e.derive(); bad != nil {
			p.unread = append(p.unread, *bad)
		}
		p.entries = append(p.entries, e)
	}
}

// CRC: crc-Pending.md | Seq: seq-pending.md#1.3 | R259
//
// regionEnd returns the index one past the region's last node, and where the region
// ends inside that last node when a `---` line cuts it (-1 otherwise). A region ends
// at the next heading of level 2 or higher; a fenced heading is no heading at all to
// the base, so it cannot end one.
func (p *Pending) regionEnd(start int) (end, cut int) {
	nodes := p.doc.Nodes()
	for i := start + 1; i < len(nodes); i++ {
		if h, ok := nodes[i].(*schema.Heading); ok && h.Level() <= 2 {
			return i, -1
		}
		if t, ok := nodes[i].(*sdom.Text); ok {
			s, _ := t.Render()
			if at := ruleAt(s); at >= 0 {
				return i + 1, at
			}
		}
	}
	return len(nodes), -1
}

// ruleAt is the byte offset of the first line that is exactly `---`, or -1.
// SplitAfter keeps each newline with the line it ends, so every element begins one.
func ruleAt(s string) int {
	off := 0
	for _, line := range strings.SplitAfter(s, "\n") {
		if strings.TrimRight(line, " \t\n") == "---" {
			return off
		}
		off += len(line)
	}
	return -1
}

// derive reads the values from the run's rendered bytes, and returns the `Source:` line
// it could not read as either form, or nil.
func (e *Entry) derive() *Unread {
	var b strings.Builder
	for i, n := range e.run {
		s, _ := n.Render()
		if e.tail != nil && i == len(e.run)-1 {
			s = s[:e.cut]
		}
		b.WriteString(s)
	}
	lines := strings.Split(b.String(), "\n")
	if m := entryHeadRe.FindStringSubmatch(lines[0]); m != nil {
		e.ID, _ = strconv.Atoi(m[1])
		e.Title, e.Skill, e.Status = m[2], m[3], strings.TrimSuffix(m[4], ".")
	}
	var bad *Unread
	for i, l := range lines[1:] {
		if m := sourceLineRe.FindStringSubmatch(l); m != nil && e.SourceDoc == "" {
			e.SourceDoc = m[1]
			word, key := m[2], m[3]
			switch { // Seq: seq-pending.md#1.4.1
			case word == "part":
				e.Kind, e.SourceKey = SourcePart, strings.TrimPrefix(key, "#")
			case word == "gap" && gapIDRe.MatchString(key):
				e.Kind, e.SourceKey = SourceGap, key
			default: // Seq: seq-pending.md#1.4.2
				bad = &Unread{e.line + i + 1, l}
			}
		} else if m := nextLineRe.FindStringSubmatch(l); m != nil && e.Next == "" {
			e.Next = m[1]
		}
	}
	return bad
}

// CRC: crc-Pending.md | R258
func (p *Pending) Doc() *sdom.Doc { return p.doc }

// CRC: crc-Pending.md | R258
func (p *Pending) Render() (string, error) { return p.doc.Render() }

// CRC: crc-Pending.md | R260
func (p *Pending) Entries() []*Entry { return p.entries }

// CRC: crc-Pending.md | R264
func (p *Pending) Unread() []Unread { return p.unread }

// CRC: crc-Pending.md | R260
func (p *Pending) Entry(id int) *Entry {
	for _, e := range p.entries {
		if e.ID == id {
			return e
		}
	}
	return nil
}

// CRC: crc-Pending.md | R264
func (p *Pending) MaxID() int {
	m := 0
	for _, e := range p.entries {
		m = max(m, e.ID)
	}
	return m
}

// CRC: crc-Pending.md | R262
func (p *Pending) After(id int) (int, error) {
	for i, e := range p.entries {
		if e.ID == id {
			return i + 2, nil
		}
	}
	return 0, fmt.Errorf("%w: --after %d", ErrNoEntry, id)
}

// CRC: crc-Pending.md | Seq: seq-pending.md#2.2 | R261, R289
// Text renders the canonical entry: heading line, Source line in the part or gap form by
// Kind, Next line, blank line.
func (e EntryText) Text() string {
	var b strings.Builder
	fmt.Fprintf(&b, "## %d. **%s**", e.ID, e.Title)
	if e.Skill != "" {
		fmt.Fprintf(&b, " (%s)", e.Skill)
	}
	word, key := "part", "#"+e.SourceKey
	if e.Kind == SourceGap {
		word, key = "gap", e.SourceKey
	}
	fmt.Fprintf(&b, ". %s\n   Source: [%s](%s), %s `%s`.\n", e.Status, e.SourceDoc, e.SourceDoc, word, key)
	if e.Next != "" {
		fmt.Fprintf(&b, "   Next: %s\n", e.Next)
	}
	b.WriteString("\n")
	return b.String()
}

// CRC: crc-Pending.md | Seq: seq-pending.md#2 | R261, R265
//
// Place inserts the canonical entry as ONE synthetic text before the entry at pos, or
// at the end when pos is one past the last. Refused rather than clamped outside that
// range: a clamp silently reinterprets an instruction the caller was specific about.
func (p *Pending) Place(e EntryText, pos int) error {
	n := len(p.entries)
	if pos < 1 || pos > n+1 {
		return fmt.Errorf("minispecsdom: position %d is outside 1 … %d; refused rather than clamped", pos, n+1)
	}
	if e.Kind == SourceGap && !gapIDRe.MatchString(e.SourceKey) { // Seq: seq-pending.md#2.1.1
		return ErrBadGapSource
	}
	text := e.Text()
	var before sdom.Node
	if pos <= n {
		before = p.entries[pos-1].head
	} else if !strings.HasSuffix(p.doc.Source(), "\n\n") {
		text = "\n" + text
	}
	if err := p.doc.Mutate(func() error {
		return p.doc.Insert(before, sdom.NewText(text, sdom.Synthetic(len(text))))
	}); err != nil {
		return err
	}
	p.reload()
	return nil
}

// CRC: crc-Pending.md | Seq: seq-pending.md#3 | R263, R265
//
// Remove drops the entry's run inside one window, splitting the shared tail text at
// the region's end so the next entry's bytes stay.
func (p *Pending) Remove(id int) error {
	e := p.Entry(id)
	if e == nil {
		return ErrNoEntry
	}
	if err := p.doc.Mutate(func() error {
		run := slices.Clone(e.run)
		if e.tail != nil {
			left, _, err := p.doc.Split(e.tail, e.cut)
			if err != nil {
				return err
			}
			run[len(run)-1] = left // the right half is the next entry's
		}
		for _, n := range run {
			if err := p.doc.Remove(n); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	p.reload()
	return nil
}
