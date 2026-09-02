package sdom

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

// parseIndent drives an IndentParser over src and hands back the document with the
// frame links it implies.
func parseIndent(src string, lang *IndentLang) (*Doc, *IndentContext) {
	ip := NewIndentParser(lang)
	return Parse(src, 0, ip), ip.Context()
}

// shape renders every Indent as its column, so a test states what it expects in one
// line: "0 2 4 2 4".
func shape(d *Doc, ic *IndentContext) string {
	var b []string
	for _, n := range d.Nodes() {
		if i, ok := n.(*Indent); ok {
			t, _ := i.Render()
			b = append(b, fmt.Sprint(ic.lang.column(t)))
		}
	}
	return strings.Join(b, " ")
}

func frames(d *Doc) []Node {
	var out []Node
	for _, n := range d.Nodes() {
		if _, ok := n.(*Indent); ok {
			out = append(out, n)
		}
	}
	return out
}

func mustRoundTrip(t *testing.T, d *Doc, src string) {
	t.Helper()
	got, err := d.Render()
	if err != nil || got != src {
		t.Fatalf("round trip: err=%v, got %q want %q", err, got, src)
	}
}

// CRC: crc-IndentParser.md | Seq: seq-indent.md#1.5 | R178
//
// A node at every change of level and none where it is unchanged. The second line
// at column 2 gets one because the level CHANGED; a line repeating a column with no
// change in between gets none, and its whitespace stays ordinary text.
func TestANodeAtEveryChangeAndNoneWhereUnchanged(t *testing.T) {
	src := "a\n  b\n    c\n  d\n    e\n"
	d, ic := parseIndent(src, &LangPython)
	if got := shape(d, ic); got != "0 2 4 2 4" {
		t.Fatalf("shape %q, want %q", got, "0 2 4 2 4")
	}
	mustRoundTrip(t, d, src)

	// Two lines at one column with no change between them: one node, not two.
	src = "a\n  b\n  c\n"
	d, ic = parseIndent(src, &LangPython)
	if got := shape(d, ic); got != "0 2" {
		t.Fatalf("repeated column: shape %q, want %q", got, "0 2")
	}
	mustRoundTrip(t, d, src)
}

// CRC: crc-IndentParser.md | Seq: seq-indent.md#1.1 | R179
//
// The root is emitted once, and an empty source produces no nodes at all — the walk
// never calls Parse, so nothing can emit a root.
func TestTheRootIsEmittedOnceAndNeverForAnEmptySource(t *testing.T) {
	d, _ := parseIndent("a\n", &LangPython)
	fs := frames(d)
	if len(fs) != 1 {
		t.Fatalf("one line: %d frames, want 1", len(fs))
	}
	if s, _ := fs[0].Render(); s != "" {
		t.Fatalf("the root must be zero-length, got %q", s)
	}
	if d.Nodes()[0] != fs[0] {
		t.Fatalf("the root must be the first node")
	}

	if d, _ := parseIndent("", &LangPython); len(d.Nodes()) != 0 {
		t.Fatalf("an empty source produced %d nodes, want none", len(d.Nodes()))
	}
}

// CRC: crc-IndentParser.md | Seq: seq-indent.md#1.6.2 | R180, R181
//
// A return to column 0 is one zero-length node however many levels it closes — the
// column is the level, so there is nothing for a run of dedent markers to add.
func TestAReturnToColumnZeroIsZeroLengthAndClosesSeveralLevels(t *testing.T) {
	src := "def a():\n  if x:\n    p\nb = 1\n"
	d, ic := parseIndent(src, &LangPython)
	if got := shape(d, ic); got != "0 2 4 0" {
		t.Fatalf("shape %q, want %q", got, "0 2 4 0")
	}
	fs := frames(d)
	if s, _ := fs[3].Render(); s != "" {
		t.Fatalf("the column-0 return must be zero-length, got %q", s)
	}
	if ic.Parent(fs[3]) != fs[0] {
		t.Fatalf("a column-0 return belongs to the root frame")
	}
	mustRoundTrip(t, d, src)
}

// CRC: crc-IndentParser.md | Seq: seq-collaborate.md#1.4.1 | R164, R180
//
// The case the two-signal no-change rule exists for. A zero-length Indent consumes
// nothing, so a walk checking only the position would read it as no match, advance,
// and eat the next byte as text — and after a dedent to column 0 that byte can open
// a string. Verified legal Python: a bare string, a parenthesized expression and a
// list display are all statements there.
func TestADedentToColumnZeroBeforeABracketOpenerKeepsTheOpener(t *testing.T) {
	for _, line := range []string{`"a string statement"`, `(1)`, `[x for x in ()]`} {
		src := "def f():\n    pass\n" + line + "\n"
		d, _ := parseIndent(src, &LangPython)
		ns := d.Nodes()
		var zero int
		for i, n := range ns {
			if ind, ok := n.(*Indent); ok {
				if s, _ := ind.Render(); s == "" && i > 0 {
					zero = i
				}
			}
		}
		if zero == 0 {
			t.Fatalf("%s: no column-0 return found", line)
		}
		if _, ok := ns[zero+1].(*Opener); !ok {
			s, _ := ns[zero+1].Render()
			t.Fatalf("%s: the byte after the dedent is %T(%q), want an Opener", line, ns[zero+1], s)
		}
		mustRoundTrip(t, d, src)
	}
}

// CRC: crc-IndentParser.md | Seq: seq-collaborate.md#2.4.2 | R174
//
// Indentation is inert inside brackets, and structurally so: the bracket parser
// takes the loop on an opener, so this parser is offered no position in there. CPython
// suppresses its indent tokens the same way, by counting parenthesis depth.
func TestIndentationIsInertInsideBrackets(t *testing.T) {
	src := "foo(1,\n            2,\n  3,\n\t\t4)\ndef g():\n    pass\n"
	d, ic := parseIndent(src, &LangPython)
	if got := shape(d, ic); got != "0 4" {
		t.Fatalf("shape %q, want %q — the continuation lines must produce nothing", got, "0 4")
	}
	mustRoundTrip(t, d, src)
}

// CRC: crc-IndentParser.md | Seq: seq-indent.md#1.4 | R183
//
// A blank line and a comment-only line do not change the level; a STRING-only line
// does. Both open a parse-restricted group, so nothing but the group's Kind
// separates them — which is why a lookahead reporting mere presence could not
// answer it. Measured against CPython, which indents on a docstring line and does
// not on a comment.
func TestBlankAndCommentOnlyLinesDoNotChangeTheLevel(t *testing.T) {
	for _, tc := range []struct{ name, src, want string }{
		{"blank line inside a suite", "def f():\n    a = 1\n\n    b = 2\n", "0 4"},
		{"comment-only line, deeper column", "def f():\n    pass\n        # deep\nx = 1\n", "0 4 0"},
		{"docstring line does indent", "def f():\n    \"\"\"doc\"\"\"\n", "0 4"},
	} {
		d, ic := parseIndent(tc.src, &LangPython)
		if got := shape(d, ic); got != tc.want {
			t.Errorf("%s: shape %q, want %q", tc.name, got, tc.want)
		}
		mustRoundTrip(t, d, tc.src)
	}
}

// CRC: crc-IndentParser.md | Seq: seq-indent.md#1.3 | R184
//
// A continuation marker counts only at bracket depth 0, which needs no depth check:
// a marker counts exactly when it is still PENDING TEXT. One inside a comment was
// consumed by that group, whose closer is the newline. One inside a string needs no
// rule, the group being still open at the next line start.
//
// Measured against CPython: the comment case is an unexpected-indent error there,
// which is only reachable because the indent IS significant.
func TestAContinuationMarkerCountsOnlyAtBracketDepthZero(t *testing.T) {
	for _, tc := range []struct{ name, src, want string }{
		{"a real continuation", "a = 1 + \\\n    2\n", "0"},
		{"a backslash ending a comment", "a = 1\n# c \\\n    b = 2\n", "0 4"},
		{"a backslash inside a string", "a = \"x \\\n y\"\n", "0"},
	} {
		d, ic := parseIndent(tc.src, &LangPython)
		if got := shape(d, ic); got != tc.want {
			t.Errorf("%s: shape %q, want %q", tc.name, got, tc.want)
		}
		mustRoundTrip(t, d, tc.src)
	}
}

// CRC: crc-IndentContext.md | Seq: seq-indent.md#1.6 | R185
//
// A frame parents to the nearest preceding Indent with a strictly smaller column,
// so two frames at one column under one parent are siblings rather than the same
// frame — which is what a flat array expresses without a tree.
func TestAFrameParentsToTheNearestSmallerColumn(t *testing.T) {
	d, ic := parseIndent("a\n  b\n    c\n  d\n    e\n", &LangPython)
	f := frames(d) // columns 0 2 4 2 4
	for _, tc := range []struct {
		child, parent int
	}{{1, 0}, {2, 1}, {3, 0}, {4, 3}} {
		if got := ic.Parent(f[tc.child]); got != f[tc.parent] {
			t.Errorf("frame %d: parent is not frame %d", tc.child, tc.parent)
		}
	}
	if !slices.Equal(ic.Children(f[0]), []Node{f[1], f[3]}) {
		t.Errorf("the root's children must be exactly the two column-2 frames")
	}
	if !slices.Equal(ic.Children(f[1]), []Node{f[2]}) {
		t.Errorf("the first column-2 frame has one child")
	}
	if len(ic.Children(f[2])) != 0 {
		t.Errorf("a leaf frame has no children")
	}
}

// CRC: crc-IndentContext.md | Seq: seq-indent.md#1.6.3 | R188
//
// A dedent to a column matching no open level is a Python error and is not one
// here: it lands beside the nearest smaller column. sdom is no syntax checker.
func TestAnUnmatchedDedentParentsRatherThanRefusing(t *testing.T) {
	d, ic := parseIndent("a\n    b\n  c\n", &LangPython)
	f := frames(d) // 0 4 2
	if got := shape(d, ic); got != "0 4 2" {
		t.Fatalf("shape %q, want %q", got, "0 4 2")
	}
	if ic.Parent(f[2]) != f[0] {
		t.Fatalf("the unmatched column-2 frame must land inside the root")
	}
}

// CRC: crc-IndentParser.md | R182
//
// The column is derived from the node's text with tabs expanded, never stored — so
// a tab and eight spaces nest identically while each node still renders exactly its
// own whitespace.
func TestTheColumnIsDerivedWithTabsExpanded(t *testing.T) {
	src := "a\n\tb\n        c\n"
	d, ic := parseIndent(src, &LangPython)
	if got := shape(d, ic); got != "0 8" {
		t.Fatalf("shape %q, want %q — a tab reaches column 8, and the next line matches it", got, "0 8")
	}
	if s, _ := frames(d)[1].Render(); s != "\t" {
		t.Fatalf("the node must render its own whitespace, got %q", s)
	}
	mustRoundTrip(t, d, src)
}

// CRC: crc-IndentContext.md | Seq: seq-indent.md#2 | R187
//
// independentFrames derives every frame link from the flat array alone — the Indent
// nodes and their own whitespace, nothing else. Written here rather than in the
// library for the reason the bracket derivation is: an index checked by code sharing
// its author and its helpers is checked by something liable to share its
// misconceptions.
func independentFrames(d *Doc, tab int) (parent map[Node]Node, children map[Node][]Node) {
	parent, children = map[Node]Node{}, map[Node][]Node{}
	col := func(ws string) int {
		c := 0
		for _, r := range ws {
			if r == '\t' && tab > 0 {
				c += tab - c%tab
			} else {
				c++
			}
		}
		return c
	}
	type fr struct {
		n Node
		c int
	}
	var stack []fr
	for _, n := range d.Nodes() {
		i, ok := n.(*Indent)
		if !ok {
			continue
		}
		s, _ := i.Render()
		c := col(s)
		for len(stack) > 1 && stack[len(stack)-1].c >= c {
			stack = stack[:len(stack)-1]
		}
		if len(stack) > 0 {
			p := stack[len(stack)-1].n
			parent[n] = p
			children[p] = append(children[p], n)
		}
		stack = append(stack, fr{n, c})
	}
	return
}

// CRC: crc-IndentContext.md | Seq: seq-indent.md#2 | R187
//
// The check that makes the frame index a fact rather than an assertion, over the
// corpus — including files that are not Python, which is exactly where a parser
// misbehaves.
func TestTheFrameIndexAgreesWithAnIndependentWalk(t *testing.T) {
	for path, src := range corpus(t) {
		d, ic := parseIndent(src, &LangPython)
		parent, children := independentFrames(d, LangPython.Tab)
		for _, n := range d.Nodes() {
			if _, ok := n.(*Indent); !ok {
				continue
			}
			if ic.Parent(n) != parent[n] {
				t.Fatalf("%s: the two derivations disagree on a frame's parent", path)
			}
			if !slices.Equal(ic.Children(n), children[n]) {
				t.Fatalf("%s: the two derivations disagree on a frame's children", path)
			}
		}
		mustRoundTrip(t, d, src)
	}
}

// CRC: crc-IndentParser.md | Seq: seq-collaborate.md#3 | R160
//
// IndentParser.NodeType, which nothing in the library calls: an IndentParser is
// always the root parser, and the walk only ever calls Parse on that. It is here
// because the interface requires it, and a library prices its API by contract
// rather than by demand — so it is tested for the case that makes it real, an
// IndentParser nested inside another parser that looks ahead before delegating.
//
// Found by injecting past the alarm list: a panic as its first statement left the
// whole suite green.
func TestIndentParserReportsWhatItWouldEmit(t *testing.T) {
	ip := NewIndentParser(&LangPython)
	src := "a\n  b\n  # c\n"
	st := &ParserState{src: src, origin: &Origin{}, parser: ip}

	// Nothing emitted yet: the root is what Parse would produce.
	if kind, ok := ip.NodeType(st); !ok || kind != "" {
		t.Fatalf("at the start: (%q, %v), want (\"\", true) — the root carries no kind", kind, ok)
	}
	// Drive the walk, then ask at a line start where the level changes.
	d := Parse(src, 0, ip)
	if len(d.Nodes()) == 0 {
		t.Fatalf("precondition: the parse produced nothing")
	}

	// A fresh state mid-source: at "  b" the level changes, so an Indent would come.
	ip2 := NewIndentParser(&LangPython)
	st2 := &ParserState{src: src, origin: &Origin{}, parser: ip2}
	ip2.Parse(st2) // emits the root and seeds the stack
	st2.SetPos(2)  // the start of "  b"
	if kind, ok := ip2.NodeType(st2); !ok || kind != "" {
		t.Errorf("at a level change: (%q, %v), want (\"\", true)", kind, ok)
	}
	// Where it would not match, the answer is the delegate's: a comment opener
	// reports its Kind, so a nesting parser can tell a comment from a string.
	st2.SetPos(strings.Index(src, "#"))
	if kind, ok := ip2.NodeType(st2); !ok || kind != "comment" {
		t.Errorf("at a comment: (%q, %v), want (\"comment\", true)", kind, ok)
	}
	// And plain text is no node at all.
	st2.SetPos(strings.Index(src, "b"))
	if kind, ok := ip2.NodeType(st2); ok {
		t.Errorf("at plain text: (%q, %v), want no node", kind, ok)
	}
}
