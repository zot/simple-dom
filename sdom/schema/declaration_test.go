// CRC: crc-DeclSchema.md | R129, R130
package schema

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/zot/simple-dom/sdom"
)

type pass func(*sdom.Doc, *sdom.BracketContext) error

// decls runs a schema and reports what it found as "kw[name name] ", so a test
// asserts on the whole outcome rather than on a count.
func decls(t *testing.T, src string, lang *sdom.BracketLang, p pass) string {
	t.Helper()
	d, ctx := sdom.Scan(src, 0, lang)
	if err := p(d, ctx); err != nil {
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

func check(t *testing.T, name, src string, lang *sdom.BracketLang, p pass, want string) {
	t.Helper()
	if got := decls(t, src, lang, p); got != want {
		t.Errorf("%s\n  src  %q\n  got  %q\n  want %q", name, src, got, want)
	}
}

// CRC: crc-DeclSchema.md | Seq: seq-declare.md#1.2.2 | R134, R145
//
// A declaration under a comment is the NORM, not an edge: the comment's closing
// newline is a Closer, so the declaration begins a text node with no separator in
// front of it. 183 of sdom's own declarations sit in exactly that position.
func TestDeclarationUnderACommentIsFound(t *testing.T) {
	check(t, "under a line comment",
		"// CRC: crc-Doc.md | R1\nfunc Index(k string) int {\n}\n",
		&sdom.LangGo, Go, "func[Index] ")
}

// CRC: crc-DeclSchema.md | Seq: seq-declare.md#1.2.2 | R145
//
// The backward skip in its other position: a comment BEFORE the keyword, whose
// closer would otherwise be read as the preceding content.
func TestCommentBeforeTheKeywordDoesNotHideIt(t *testing.T) {
	check(t, "comment first", "/* c */ func Foo() {\n}\n", &sdom.LangGo, Go, "func[Foo] ")
}

// CRC: crc-DeclSchema.md | Seq: seq-declare.md#1.4 | R145, R147
//
// The skip runs before EVERY decision. A single-comment fixture would pass with a
// skip-once implementation, which is why both inputs are here — the second
// interleaves comments with whitespace nodes across a newline.
func TestCommentsAnywhereInASignature(t *testing.T) {
	check(t, "method with comments",
		"func /* a */ (s *S) /* b */ Index /* c */ (x int) {\n}\n",
		&sdom.LangGo, Go, "func[Index] ")
	check(t, "comments and whitespace across a line",
		"func /* a */  /* b */   \n/* c */ foo /* d */ (x) {\n}\n",
		&sdom.LangGo, Go, "func[foo] ")
}

// CRC: crc-DeclSchema.md | Seq: seq-declare.md#1.4.3 | R149
//
// Skipped whitespace may contain a separator, so the walk crosses it. The two
// inputs are paired deliberately: the Go form is legal and vets clean, the Lua form
// runs and returns a value, so a rule stopping at a separator is wrong for both at
// once.
func TestADeclarationMaySpanLines(t *testing.T) {
	check(t, "go across a newline", "func\nfoo(x int) {\n}\n", &sdom.LangGo, Go, "func[foo] ")
	check(t, "lua across blank lines",
		"function\n\nfoo\n (x)\nreturn x\nend\n", &sdom.LangLua, Lua, "function[foo] ")
}

// CRC: crc-LuaSchema.md | Seq: seq-declare.md#1.3 | R131, R135
//
// Lua announces two ways and one is not text at all: `function` is an Opener, and
// the name it introduces sits inside the group it opened. A Go-shaped fixture would
// miss all of that and still report success on `x = 1`.
func TestLuaAnnouncesTwoWays(t *testing.T) {
	check(t, "local function", "local function f(a) return a end\n", &sdom.LangLua, Lua, "local[f] ")
	check(t, "function as an opener", "function M.f(a) end\n", &sdom.LangLua, Lua, "function[M.f] ")
	check(t, "keyword-less assignment", "x = 1\n", &sdom.LangLua, Lua, "[x] ")
}

// CRC: crc-ShellSchema.md | R136
//
// Shell's assignment forbids whitespace around `=`; `NAME = value` is a COMMAND
// INVOCATION. Lua's rule is the lenient one, and the two patterns differ by two
// characters.
func TestShellAssignmentIsStrict(t *testing.T) {
	check(t, "assignment", "NAME=value\n", &sdom.LangShell, Shell, "[NAME] ")
	check(t, "a command is not an assignment", "NAME = value\n", &sdom.LangShell, Shell, "")
	check(t, "a function is recognized structurally",
		"foo() {\n  echo hi\n}\n", &sdom.LangShell, Shell, "[foo] ")
}

// CRC: crc-GoSchema.md | Seq: seq-declare.md#1.4.4 | R151
//
// A func's name must reach its paren without crossing a NEWLINE — not abut it.
// Verified against the compiler: the first three are legal Go, the last two are
// not, the final one because Go treats a comment carrying a newline as a newline.
func TestGoFuncNameReachesItsParen(t *testing.T) {
	for _, tc := range []struct{ name, src, want string }{
		{"abutting", "func foo(x int) {\n}\n", "func[foo] "},
		{"a space between", "func foo (x int) {\n}\n", "func[foo] "},
		{"a comment between", "func bar /* c */ (x int) {\n}\n", "func[bar] "},
		{"a newline between", "func\n\nbaz\n (x int)\n{\n}\n", ""},
		{"a comment carrying a newline", "func qux /*\n*/ (x int) {\n}\n", ""},
	} {
		check(t, tc.name, tc.src, &sdom.LangGo, Go, tc.want)
	}
}

// CRC: crc-GoSchema.md | Seq: seq-declare.md#2.4 | R139, R140
//
// One group, many names — and `B, C = 2, 3` puts two on one line, so a repeated
// capture group would lose C. A one-name-per-line fixture would pass without
// noticing.
func TestGroupedDeclarationYieldsEveryName(t *testing.T) {
	check(t, "const group", "const (\n\tA = 1\n\tB, C = 2, 3\n)\n",
		&sdom.LangGo, Go, "const[A B C] ")
	check(t, "typed names", "var (\n\tD, E int\n)\n", &sdom.LangGo, Go, "var[D E] ")
}

// CRC: crc-Declaration.md | Seq: seq-declare.md#2.3 | R148
//
// A name is sliced out of the MIDDLE of its node. Re-typing the node whole renders
// identically and is wrong, so only an assertion about the node's own bytes sees it.
func TestANameIsSlicedOutOfTheMiddle(t *testing.T) {
	src := "func /* a */ foo /* b */ (x) {\n}\n"
	d, ctx := sdom.Scan(src, 0, &sdom.LangGo)
	if err := Go(d, ctx); err != nil {
		t.Fatalf("pass: %v", err)
	}
	for _, n := range d.Nodes() {
		if nm, ok := n.(*sdom.DeclarationName); ok {
			if got, _ := nm.Render(); got != "foo" {
				t.Fatalf("name renders %q, want %q — whitespace came with it", got, "foo")
			}
			return
		}
	}
	t.Fatal("no DeclarationName produced")
}

// CRC: crc-Declaration.md | Seq: seq-declare.md#2.6 | R124
//
// The additive property over the real corpus: a pass only splits and re-types, so
// no byte moves and no marker stops being recognized. Both halves are asserted
// because they fail independently — dropping whitespace changes bytes while leaving
// marker counts equal.
//
// The COUNT is checked against an independent derivation rather than against a
// number, in the same spirit as the bracket index's cross-check. Go files are
// gofmt'd, so a top-level declaration is exactly a keyword at column 0; that is a
// different route to the same answer, and it is what makes a silent under-count
// visible. Measured 2026-08-31: an earlier version of this test asserted only
// "more than zero" and passed while finding 16 of 225.
func TestADeclarationPassIsAdditive(t *testing.T) {
	files, _ := filepath.Glob("../*.go")
	if len(files) == 0 {
		t.Skip("no corpus")
	}
	topLevel := regexp.MustCompile(`(?m)^(func|var|type|const)\b`)
	total, expected := 0, 0
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		src := string(b)
		plain, _ := sdom.Scan(src, 0, &sdom.LangGo)
		d, ctx := sdom.Scan(src, 0, &sdom.LangGo)
		if err := Go(d, ctx); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		if got, _ := d.Render(); got != src {
			t.Errorf("%s: the pass changed bytes", f)
		}
		if a, b := markers(plain), markers(d); a != b {
			t.Errorf("%s: marker counts changed, %v -> %v", f, a, b)
		}
		found := 0
		for _, n := range d.Nodes() {
			if _, ok := n.(*sdom.DeclarationType); ok {
				found++
			}
		}
		want := len(topLevel.FindAllString(src, -1))
		if found != want {
			t.Errorf("%s: found %d declarations, column-0 keywords say %d", f, found, want)
		}
		total, expected = total+found, expected+want
	}
	t.Logf("%d declarations over %d files (independent count: %d)", total, len(files), expected)
}

func markers(d *sdom.Doc) [3]int {
	var c [3]int
	for _, n := range d.Nodes() {
		switch n.(type) {
		case *sdom.Opener:
			c[0]++
		case *sdom.Closer:
			c[1]++
		case *sdom.Separator:
			c[2]++
		}
	}
	return c
}
