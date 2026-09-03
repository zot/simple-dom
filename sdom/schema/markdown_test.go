package schema

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/zot/simple-dom/sdom"
)

func parseMarkdown(src string) (*sdom.Doc, *MarkdownParser) {
	p := NewMarkdownParser()
	return sdom.Parse(src, 0, p), p
}

func sample(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("testdata/trajectory-sample.md")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

type counts struct {
	headings, items, boxes, bolds int
	levels                        []int
}

func count(d *sdom.Doc) (c counts) {
	for _, n := range d.Nodes() {
		switch m := n.(type) {
		case *Heading:
			c.headings++
			c.levels = append(c.levels, m.Level())
		case *ListItem:
			c.items++
		case *Checkbox:
			c.boxes++
		case *sdom.Opener:
			if s, _ := m.Render(); s == "**" {
				c.bolds++
			}
		}
	}
	return
}

// CRC: crc-MarkdownParser.md | Seq: seq-markdown.md#1.3 | R226, R227, R228, R229, R230
func TestTheFixtureRoundTripsAndIsRecognized(t *testing.T) {
	src := sample(t)
	d, _ := parseMarkdown(src)
	if r, _ := d.Render(); r != src {
		t.Fatalf("render differs from source")
	}
	c := count(d)
	// Hand count: # Carve, ## Status, ### Notes, ## 8. → four headings, levels 1 2 3 2.
	if c.headings != 4 || !slices.Equal(c.levels, []int{1, 2, 3, 2}) {
		t.Errorf("headings %d levels %v, want 4 and [1 2 3 2]", c.headings, c.levels)
	}
	// Hand count: five list items in the status block (Item 1, Item 2, 2.1, 2.2, Item 4);
	// four checkboxes (Item 2 has none); the fence's `- [ ]` is not counted.
	if c.items != 5 || c.boxes != 4 {
		t.Errorf("items %d boxes %d, want 5 and 4", c.items, c.boxes)
	}
}

// CRC: crc-MarkdownParser.md | Seq: seq-markdown.md#1.3.2 | R230
func TestACheckboxOnlyAfterAListItem(t *testing.T) {
	d, _ := parseMarkdown("- [x] done\n- [ ] open\nsee [x] in prose\n[x] not an item\n")
	if c := count(d); c.boxes != 2 || c.items != 2 {
		t.Errorf("boxes %d items %d, want 2 and 2", c.boxes, c.items)
	}
}

// CRC: crc-MarkdownParser.md | Seq: seq-markdown.md#1.4 | R232
func TestCodeHidesStructure(t *testing.T) {
	src := "```\n- [ ] inside\n## not a heading\n**not bold**\n```\nand `a **code** span`\n"
	d, p := parseMarkdown(src)
	c := count(d)
	if c.items+c.boxes+c.headings+c.bolds != 0 {
		t.Errorf("structure recognized inside code: %+v", c)
	}
	ctx := p.Indent().Brackets().Context()
	for _, n := range d.Nodes() {
		o, ok := n.(*sdom.Opener)
		if !ok {
			continue
		}
		inner := ctx.InnerText(o)
		if !strings.Contains(inner, "\n") && !strings.Contains(inner, "**") {
			continue
		}
		// the interior must be ONE text node
		i := d.IndexOf(o)
		first, second := d.Nodes()[i+1], d.Nodes()[i+2]
		if _, isText := first.(*sdom.Text); !isText || second != sdom.Node(ctx.Closer(o)) {
			t.Errorf("a code interior is not a single text node")
		}
	}
}

// CRC: crc-MarkdownParser.md | R231
func TestNoLineHeadMarkerSharesAFirstByteWithAnOpener(t *testing.T) {
	for _, g := range LangMarkdown.Brackets {
		for _, o := range g.Open {
			if strings.ContainsAny(o[:1], "#-[") {
				t.Errorf("opener %q shares a first byte with a line-head marker", o)
			}
		}
	}
}

// CRC: crc-MarkdownParser.md | R233
func TestNodeTypeAgreesWithParse(t *testing.T) {
	src := "## H\n- [x] a\ntext **b**\n"
	d, _ := parseMarkdown(src)
	// Replay the parse and ask NodeType wherever a line-head marker sits.
	want := map[int]string{}
	for _, n := range d.Nodes() {
		switch n.(type) {
		case *Heading:
			want[n.Location().Offset()] = "heading"
		case *ListItem:
			want[n.Location().Offset()] = "list"
		case *Checkbox:
			want[n.Location().Offset()] = "checkbox"
		}
	}
	if len(want) != 3 {
		t.Fatalf("expected three markers, got %v", want)
	}
	rec := &nodeTypeRecorder{inner: NewMarkdownParser(), got: map[int]string{}, ok: map[int]bool{}}
	sdom.Parse(src, 0, rec)
	for off, k := range want {
		if rec.got[off] != k {
			t.Errorf("NodeType at %d = %q, want %q", off, rec.got[off], k)
		}
	}
	text := len("## H\n- [x] ") // the `a`, which is plain text
	if k := rec.got[text]; rec.ok[text] && k != "" {
		t.Errorf("NodeType at plain text = %q, want no node", k)
	}
}

// nodeTypeRecorder asks the wrapped parser's NodeType at every offer before
// delegating Parse to it.
type nodeTypeRecorder struct {
	inner *MarkdownParser
	got   map[int]string
	ok    map[int]bool
}

func (r *nodeTypeRecorder) Parse(st *sdom.ParserState) {
	k, ok := r.inner.NodeType(st)
	r.got[st.Pos()], r.ok[st.Pos()] = k, ok
	r.inner.Parse(st)
}

func (r *nodeTypeRecorder) NodeType(st *sdom.ParserState) (string, bool) { return r.inner.NodeType(st) }

func (r *nodeTypeRecorder) Done(d *sdom.Doc) { r.inner.Done(d) }
