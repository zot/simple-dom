// CRC: crc-StencilBuilder.md | R98, R100, R101, R103, R105, R106, R108, R110, R117
package sdom

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

func parseTodo(t *testing.T, line string) *TodoItem {
	t.Helper()
	item := &TodoItem{}
	if _, err := item.Parse(line, Source(0, len(line))); err != nil {
		t.Fatalf("%q: %v", line, err)
	}
	return item
}

// mustBuild starts a stencil over the whole of text. Every case here feeds the
// pattern text it matches, so a non-match is a broken fixture rather than the
// thing under test.
func mustBuild(t *testing.T, re *regexp.Regexp, text string) *StencilBuilder {
	t.Helper()
	b, ok := NewStencilBuilder(re, text, Source(0, len(text)))
	if !ok {
		t.Fatalf("precondition: %v must match %q", re, text)
	}
	return b
}

// CRC: crc-StencilBuilder.md | Seq: seq-stencil.md#1.2.2 | R98
// The author writes what they bind; everything else becomes text from the gaps —
// "- [" and "] " are children though todoRe never mentions them.
func TestGlueIsComputedFromTheGaps(t *testing.T) {
	item := parseTodo(t, "- [x] write the spec")
	want := `T"- [" T"x" T"] " T"write the spec"`
	if got := nodeStream(item.Kids()); got != want {
		t.Fatalf("children =\n  %s\nwant\n  %s", got, want)
	}
}

// CRC: crc-StencilBuilder.md | R98, R110
func TestChildrenTileTheMatch(t *testing.T) {
	for _, line := range []string{
		"- [x] done", "- [ ] todo", "- [] no space", "- [    ] padded", "- [x] ",
	} {
		item := parseTodo(t, line)
		span := item.Location()
		next := span.Offset()
		for i, k := range item.Kids() {
			l := k.Location()
			if l.Offset() != next {
				t.Fatalf("%q: child %d begins at %d, previous ended at %d", line, i, l.Offset(), next)
			}
			next = l.Offset() + l.Length()
		}
		if next != span.Offset()+span.Length() {
			t.Fatalf("%q: children end at %d, the match ends at %d", line, next, span.Offset()+span.Length())
		}
		if got, _ := item.Render(); got != line {
			t.Fatalf("%q rendered as %q", line, got)
		}
	}
}

var twoGroups = regexp.MustCompile(`(?P<a>a)-(?P<b>b)`)

// CRC: crc-StencilBuilder.md | Seq: seq-stencil.md#1.4.1 | R99, R105
func TestUnfilledGroupPanics(t *testing.T) {
	b := mustBuild(t, twoGroups, "a-b")
	b.Put("a", NewText(b.Group("a")))
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("a group named and never filled must panic")
		}
		if !strings.Contains(fmt.Sprint(r), `"b"`) {
			t.Fatalf("the panic must name the group; got %v", r)
		}
	}()
	b.Done()
}

// CRC: crc-StencilBuilder.md | Seq: seq-stencil.md#1.4.2 | R106
// The only way a schema can break tiling from here.
func TestMisSpannedPlugPanics(t *testing.T) {
	b := mustBuild(t, twoGroups, "a-b")
	b.Put("a", NewText(b.Group("a")))
	sb, _ := b.Group("b")
	b.Put("b", NewText(sb, Source(99, 1))) // not the group's span
	defer func() {
		if recover() == nil {
			t.Fatal("a plugged node whose span is not its group's must panic")
		}
	}()
	b.Done()
}

// CRC: crc-StencilBuilder.md | Seq: seq-stencil.md#1.3.2 | R103, R117
// The falsifiable meaning of Omit, rather than an intention.
func TestOmittingAGroupIsTheSameAsNeverNamingIt(t *testing.T) {
	const text = "a-m-b"
	named := regexp.MustCompile(`(?P<a>a)-(?P<mid>m)-(?P<b>b)`)
	unnamed := regexp.MustCompile(`(?P<a>a)-m-(?P<b>b)`)

	build := func(re *regexp.Regexp, omit string) []Node {
		bl := mustBuild(t, re, text)
		for _, n := range []string{"a", "b"} {
			bl.Put(n, NewText(bl.Group(n)))
		}
		if omit != "" {
			bl.Omit(omit)
		}
		kids, _ := bl.Done()
		return kids
	}

	withOmit := build(named, "mid")
	without := build(unnamed, "")

	if len(withOmit) != len(without) {
		t.Fatalf("omitting produced %d children, never naming produced %d:\n  %s\n  %s",
			len(withOmit), len(without), nodeStream(withOmit), nodeStream(without))
	}
	for i := range withOmit {
		if !withOmit[i].Equals(without[i]) || withOmit[i].Location() != without[i].Location() {
			t.Fatalf("child %d differs:\n  %s\n  %s", i, nodeStream(withOmit), nodeStream(without))
		}
	}
}

var branches = regexp.MustCompile(`(?:(?P<x>x)(?P<y>y))|(?P<z>z)`)

// CRC: crc-StencilBuilder.md | R100, R101
// The losing branch of an alternation has no bytes and owes nothing.
func TestNonParticipatingGroupOwesNothing(t *testing.T) {
	b := mustBuild(t, branches, "z")
	if _, l := b.Group("x"); l != (Loc{}) {
		t.Fatalf("a non-participating group must yield the zero Loc, got %+v", l)
	}
	s, l := b.Group("z")
	if l == (Loc{}) {
		t.Fatal("a participating group must yield a real location")
	}
	b.Put("z", NewText(s, l))
	kids, _ := b.Done() // must not panic though x and y were never filled
	if len(kids) != 1 {
		t.Fatalf("expected one child, got %s", nodeStream(kids))
	}
}

// CRC: crc-StencilBuilder.md | R108
// Alternation gives branches different group counts, so a fixed index would be
// right for one input and out of range for the other.
func TestBindingIsByNameNotPosition(t *testing.T) {
	for _, c := range []struct{ text, name string }{{"xy", "x"}, {"z", "z"}} {
		b := mustBuild(t, branches, c.text)
		s, l := b.Group(c.name)
		if s == "" || l == (Loc{}) {
			t.Fatalf("%q: group %q did not resolve", c.text, c.name)
		}
	}
}

// CRC: crc-Bool.md | R111, R112
// Nothing is stored, so nothing is normalised.
func TestCheckboxReadsWithoutNormalising(t *testing.T) {
	for _, c := range []struct {
		line string
		want bool
	}{
		{"- [ ] a", false}, {"- [] a", false}, {"- [    ] a", false},
		{"- [x] a", true}, {"- [X] a", true},
	} {
		item := parseTodo(t, c.line)
		if got := item.Checked().Value(); got != c.want {
			t.Errorf("%q read as %v, want %v", c.line, got, c.want)
		}
		if got, _ := item.Render(); got != c.line {
			t.Errorf("%q rendered back as %q", c.line, got)
		}
	}
}

// CRC: crc-Bool.md | Seq: seq-stencil.md#2 | R113, R114
func TestSettingACheckboxKeepsItsOffsetAndContracts(t *testing.T) {
	const line = "- [    ] task"
	item := parseTodo(t, line)
	box := item.Checked().Text()
	before := box.Location()
	if !before.Faithful() || before.Length() != 4 {
		t.Fatalf("precondition: a faithful four-byte checkbox, got %+v", before)
	}

	item.Checked().Set(true)

	after := box.Location()
	if after.Offset() != before.Offset() {
		t.Errorf("the checkbox must keep its offset: %d became %d", before.Offset(), after.Offset())
	}
	if !after.Altered() || after.Length() != 1 {
		t.Errorf("it must be altered and one byte long, got %+v", after)
	}
	if item.Location().Faithful() {
		t.Error("the stencil must be altered because a child is")
	}
	if item.Checked().Text() != box {
		t.Error("the Bool must still point at the same node in the child list")
	}
	if got, _ := item.Render(); got != "- [x] task" {
		t.Errorf("rendered %q, want %q", got, "- [x] task")
	}
}

// CRC: crc-Bool.md | R113
// An edit reformats what it touched — and setting a value the text already carries
// touches nothing.
func TestSettingAValueAlreadyHeldChangesNothing(t *testing.T) {
	item := parseTodo(t, "- [X] a")
	item.Checked().Set(true)
	if !item.Checked().Text().Location().Faithful() {
		t.Error("confirming a value must not rewrite the text")
	}
	if got, _ := item.Render(); got != "- [X] a" {
		t.Errorf("rendered %q; the X must survive", got)
	}
}

// CRC: crc-TodoItem.md | R96
// The whole mechanism over real markdown rather than a fixture.
func TestTodoItemRoundTripsAMarkdownList(t *testing.T) {
	lines := []string{
		"- [ ] plain",
		"- [x] done",
		"- [    ] padded",
		"- [] tight",
		"- [X] shouting",
		"not a todo at all",
	}
	for _, line := range lines[:5] {
		item := parseTodo(t, line)
		if got, _ := item.Render(); got != line {
			t.Errorf("%q rendered as %q", line, got)
		}
	}
	last := &TodoItem{}
	if _, err := last.Parse(lines[5], Source(0, len(lines[5]))); err == nil {
		t.Errorf("%q must not match", lines[5])
	}
}

// CRC: crc-StencilBuilder.md | R97
// A non-match is reported, and what it means is the schema's decision.
func TestNoMatchIsReported(t *testing.T) {
	if b, ok := NewStencilBuilder(twoGroups, "nothing here", Source(0, 12)); ok || b != nil {
		t.Fatalf("a non-matching regex must report false and no builder")
	}
	item := &TodoItem{}
	if _, err := item.Parse("not a todo", Source(0, 10)); err != ErrNoMatch {
		t.Fatalf("a schema decides what a non-match means; got %v", err)
	}
}

// CRC: crc-Doc.md | R118
// A synthesized node makes no claim about coordinates, so it cannot contradict one.
func TestASynthesizedNodeIsCompatibleWithAnyDocument(t *testing.T) {
	o := &Origin{Name: "scanned"}
	scanned := NewText("aa", Source(0, 2).In(o))
	synth := NewText("bb", Synthetic(2))
	d := New("aabb", 0, scanned, synth) // must not panic
	if len(d.Nodes()) != 2 {
		t.Fatalf("expected both nodes, got %d", len(d.Nodes()))
	}
}
