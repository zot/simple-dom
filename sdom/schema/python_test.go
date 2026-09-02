package schema

import (
	"fmt"
	"strings"
	"testing"

	"github.com/zot/simple-dom/sdom"
)

// pyDecls parses src as an INDENT-parsed Python document, runs the schema, and
// renders what it found as "kw[name] " pairs. It also checks the pass changed no
// bytes, which is the additive property in its cheapest form.
func pyDecls(t *testing.T, src string) string {
	t.Helper()
	ip := sdom.NewIndentParser(&sdom.LangPython)
	d := sdom.Parse(src, 0, ip)
	ctx := ip.Brackets().Context()
	if err := Python(d, ctx); err != nil {
		t.Fatalf("pass: %v", err)
	}
	if got, _ := d.Render(); got != src {
		t.Fatalf("the pass changed bytes:\n got %q\nwant %q", got, src)
	}
	var b strings.Builder
	for _, n := range d.Nodes() {
		ty, ok := n.(*sdom.DeclarationType)
		if !ok {
			continue
		}
		kw, _ := ty.Render()
		names, err := ctx.Declarations(ty)
		if err != nil {
			t.Fatalf("Declarations: %v", err)
		}
		b.WriteString(kw + "[")
		for i, nm := range names {
			if i > 0 {
				b.WriteString(" ")
			}
			v, _ := nm.Render()
			b.WriteString(v)
		}
		b.WriteString("] ")
	}
	return b.String()
}

func checkPy(t *testing.T, name, src, want string) {
	t.Helper()
	if got := pyDecls(t, src); got != want {
		t.Errorf("%s\n  src  %q\n  got  %q\n  want %q", name, src, got, want)
	}
}

// CRC: crc-PythonSchema.md | Seq: seq-declare.md#1 | R190
//
// The test this schema exists to pass. A method sits inside its class body, which
// in an indent-parsed document is inside an INDENT FRAME — and an indent frame is
// not a bracket group, so the method's text node still has no bracket enclosing it.
// Go's top-level predicate transfers untouched, at any depth.
func TestAMethodInsideAClassIsFound(t *testing.T) {
	checkPy(t, "class and method",
		"class Widget:\n    def draw(self):\n        pass\n",
		"class[Widget] def[draw] ")
	checkPy(t, "three levels deep",
		"class A:\n    class B:\n        def deep(self):\n            pass\n",
		"class[A] class[B] def[deep] ")
}

// CRC: crc-PythonSchema.md | R133
//
// A def inside a string is not a declaration, because a string is a bracket group
// and its interior is not a top-level text node. Costs nothing to get.
func TestADefInsideADocstringIsNotADeclaration(t *testing.T) {
	checkPy(t, "module docstring",
		"\"\"\"a doc\ndef notreal():\n    pass\n\"\"\"\ndef real():\n    pass\n",
		"def[real] ")
	checkPy(t, "commented out",
		"# def notreal():\ndef real():\n    pass\n",
		"def[real] ")
}

// CRC: crc-PythonSchema.md | R189
//
// A decorator is ordinary text on its own line above, so it neither hides the
// keyword nor is mistaken for one.
func TestADecoratedDeclarationIsFound(t *testing.T) {
	checkPy(t, "decorated", "@cache\ndef f():\n    pass\n", "def[f] ")
	checkPy(t, "decorator with arguments", "@dec(1, 2)\nclass C:\n    pass\n", "class[C] ")
}

// CRC: crc-PythonSchema.md | Seq: seq-declare.md#1.2.2 | R134
//
// The structural half of the statement-start rule, which Python reaches constantly:
// a comment's closing newline is a Closer rather than a byte of text, so the
// declaration begins its text node with no separator in front of it.
func TestADefAfterACommentLineBeginsAStatement(t *testing.T) {
	checkPy(t, "after a comment", "# CRC: crc-Doc.md\ndef f():\n    pass\n", "def[f] ")
}

// CRC: crc-PythonSchema.md | R189
//
// The name must follow the keyword in its own node with nothing but spaces between:
// `def` then a newline does not compile, so a schema that insists is right to.
func TestTheNameFollowsTheKeywordInTheSameNode(t *testing.T) {
	checkPy(t, "space before the paren", "def foo ():\n    pass\n", "def[foo] ")
	checkPy(t, "parameters spanning lines", "def foo(\n  x):\n    pass\n", "def[foo] ")
	checkPy(t, "a keyword with no name", "def\n", "")
	checkPy(t, "constant is not const", "defer = 1\nclassy = 2\n", "")
}

// CRC: crc-PythonSchema.md | R124
//
// The pass is additive: it splits text nodes and re-types halves, so the same bytes
// come back and the node count grows by exactly two per declaration — never
// consuming a marker the parse already produced.
func TestThePythonPassIsAdditive(t *testing.T) {
	src := "class A:\n    def f(self):\n        return \"x\"\n\n# c\ndef g():\n    pass\n"

	plainParser := sdom.NewIndentParser(&sdom.LangPython)
	plain := sdom.Parse(src, 0, plainParser)
	before := len(plain.Nodes())

	ip := sdom.NewIndentParser(&sdom.LangPython)
	d := sdom.Parse(src, 0, ip)
	ctx := ip.Brackets().Context()
	if err := Python(d, ctx); err != nil {
		t.Fatalf("pass: %v", err)
	}
	if got, _ := d.Render(); got != src {
		t.Fatalf("the pass changed bytes")
	}
	var decls int
	for _, n := range d.Nodes() {
		if _, ok := n.(*sdom.DeclarationType); ok {
			decls++
		}
	}
	if decls != 3 {
		t.Fatalf("found %d declarations, want 3", decls)
	}
	// The property is not a node count — carve yields one node or two depending on
	// whether the keyword starts its node — it is that NO MARKER stops being
	// recognized. The pass splits text and re-types halves, so every Opener, Closer,
	// Separator and Indent must survive it unchanged, in order and in text.
	if got, want := markerShape(d), markerShape(plain); got != want {
		t.Fatalf("the marker stream changed:\n got %s\nwant %s", got, want)
	}
	if before >= len(d.Nodes()) {
		t.Fatalf("the pass added no nodes at all")
	}
}

// markerShape renders every non-text node as kind and bytes, which is what the
// additive property is about: a sub-schema that quietly stops recognizing something
// falls back to opaque text, loses no bytes and breaks no round-trip.
func markerShape(d *sdom.Doc) string {
	var b strings.Builder
	for _, n := range d.Nodes() {
		switch n.(type) {
		case *sdom.Opener, *sdom.Closer, *sdom.Separator, *sdom.Indent:
			s, _ := n.Render()
			fmt.Fprintf(&b, "%T%q ", n, s)
		}
	}
	return b.String()
}

// CRC: crc-PythonSchema.md | R189
//
// Only spaces and tabs may separate a keyword from its name. Python has no block
// comment and a newline there does not compile, so anything else between them means
// this is not the declaration it looks like.
//
// Found by injecting past the alarm list: deleting the guard entirely left the whole
// suite green, so nothing was asking this question.
func TestOnlySpacesMaySeparateTheKeywordFromTheName(t *testing.T) {
	checkPy(t, "a tab is fine", "def\tfoo():\n    pass\n", "def[foo] ")
	checkPy(t, "several spaces are fine", "def    foo():\n    pass\n", "def[foo] ")
	checkPy(t, "anything else is not a declaration", "def -foo():\n    pass\n", "")
	checkPy(t, "nor is punctuation", "class *A:\n    pass\n", "")
}
