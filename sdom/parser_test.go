// CRC: crc-BracketParser.md | R57, R65, R71, R72, R73, R74, R75, R76, R77
package sdom

import (
	"fmt"
	"strings"
	"testing"
)

// parse drives a BracketParser over src and hands back the document with the
// context it filled — the pair every test here wants, which the API deliberately
// splits so that no caller type-asserts a context back out of a return value.
func parse(src string, base int, lang *BracketLang) (*Doc, *BracketContext) {
	bp := NewBracketParser(lang)
	return Parse(src, base, bp), bp.Context()
}

// stream renders a document's node array compactly for assertions: O/C/S for the
// marker kinds, T for text, each followed by its quoted bytes.
func stream(d *Doc) string { return nodeStream(d.Nodes()) }

// nodeStream is the same rendering for any node list, including the child lists
// that never become a document of their own.
func nodeStream(nodes []Node) string {
	var b strings.Builder
	for i, n := range nodes {
		if i > 0 {
			b.WriteByte(' ')
		}
		switch n.(type) {
		case *Opener:
			b.WriteByte('O')
		case *Closer:
			b.WriteByte('C')
		case *Separator:
			b.WriteByte('S')
		default:
			b.WriteByte('T')
		}
		s, _ := n.Render()
		fmt.Fprintf(&b, "%q", s)
	}
	return b.String()
}

func codeLang() *BracketLang {
	return &BracketLang{Brackets: []BracketGroup{
		{Open: []string{"{"}, Close: []string{"}"}},
	}}
}

// assertStream parses src with lang and checks the whole node stream against
// want, returning the document for any further assertions.
func assertStream(t *testing.T, lang *BracketLang, src, want string) *Doc {
	t.Helper()
	d, _ := parse(src, 0, lang)
	if got := stream(d); got != want {
		t.Errorf("%q:\n  got  %s\n  want %s", src, got, want)
	}
	return d
}

// checkCovers asserts the two properties a lossless parse always has: the
// document renders back to src byte for byte, and its nodes tile src in order
// with no gap and no overlap. what names the case in a failure.
func checkCovers(t *testing.T, d *Doc, src, what string) {
	t.Helper()
	if got, err := d.Render(); err != nil || got != src {
		t.Fatalf("%s: round-trip failed (err %v)", what, err)
	}
	next := 0
	for i, n := range d.Nodes() {
		l := n.Location()
		if l.Offset() != next {
			t.Fatalf("%s: node %d at %d, the previous node ended at %d", what, i, l.Offset(), next)
		}
		next = l.Offset() + l.Length()
	}
	if next != len(src) {
		t.Fatalf("%s: tiling ends at %d, the source is %d bytes", what, next, len(src))
	}
}

// CRC: crc-BracketParser.md | Seq: seq-parse.md#1.5 | R77
// The nesting lives on the call stack and is absent from the data.
func TestOpenerContentsAndCloserAreSiblings(t *testing.T) {
	d := assertStream(t, codeLang(), "a {b {c} d} e",
		`T"a " O"{" T"b " O"{" T"c" C"}" T" d" C"}" T" e"`)
	for i, n := range d.Nodes() {
		if len(n.Kids()) != 0 {
			t.Fatalf("node %d has %d children; the array is flat", i, len(n.Kids()))
		}
	}
}

// CRC: crc-BracketGroup.md | R71
// The rule that separates a word bracket from a substring.
func TestWordMarkersRespectBoundaries(t *testing.T) {
	words := &BracketLang{Brackets: []BracketGroup{
		{Open: []string{"do", "begin"}, Close: []string{"fi", "end"}},
	}}
	assertStream(t, words, "download do file fi begin_ end",
		`T"download " O"do" T" file " C"fi" T" begin_ " C"end"`)
}

// CRC: crc-BracketParser.md | Seq: seq-parse.md#1.3.3 | R72
func TestSeparatorsOnlyInsideTheirGroup(t *testing.T) {
	d, _ := parse("else if x then y else z fi", 0, &LangShell)
	got := stream(d)
	if strings.Count(got, `S"else"`) != 1 {
		t.Fatalf("expected exactly one separator else; got\n  %s", got)
	}
	if !strings.HasPrefix(got, `T"else "`) {
		t.Fatalf("the leading bare else must be text; got\n  %s", got)
	}
}

// CRC: crc-BracketParser.md | Seq: seq-parse.md#3.1 | R73
// The any-close fallback keeps an unbalanced file parsable.
func TestStrayCloserLandsAsABracket(t *testing.T) {
	assertStream(t, codeLang(), "a } b", `T"a " C"}" T" b"`)
}

// CRC: crc-BracketParser.md | Seq: seq-parse.md#3.2 | R74
// The guarantee that makes unknown input safe.
func TestTheParseNeverStalls(t *testing.T) {
	src := "$ \x00 \xff\xfe unknown ~`!@#%^&*"
	d, _ := parse(src, 0, codeLang())
	if got, err := d.Render(); err != nil || got != src {
		t.Fatalf("unrecognized input must still round-trip (err %v)", err)
	}
}

// CRC: crc-BracketParser.md | Seq: seq-parse.md#3.4 | R75
// An unbalanced file drops no bytes.
func TestUnclosedGroupClosesAtEndOfInput(t *testing.T) {
	for _, src := range []string{"func f() {", `s := "unterminated`, "// trailing", "`raw"} {
		d, _ := parse(src, 0, &LangGo)
		checkCovers(t, d, src, fmt.Sprintf("%q", src))
	}
}

// CRC: crc-BracketParser.md | R76
// A text run is everything between two markers, whitespace included.
func TestWhitespaceFoldsIntoText(t *testing.T) {
	assertStream(t, codeLang(), "{ a  b\n  c }", `O"{" T" a  b\n  c " C"}"`)
}

// CRC: crc-BracketGroup.md | Seq: seq-parse.md#2.2 | R65
// The property that makes strings and comments the same case.
func TestNothingInsideARestrictedGroupIsRecognized(t *testing.T) {
	cases := []struct{ src, want string }{
		{`"a { b // c"`, `O"\"" T"a { b // c" C"\""`},
		{"// a \"b\" { c\n", `O"//" T" a \"b\" { c" C"\n"`},
		{`/* a "b" // c */`, `O"/*" T" a \"b\" // c " C"*/"`},
	}
	for _, c := range cases {
		assertStream(t, &LangGo, c.src, c.want)
	}
}

// CRC: crc-BracketGroup.md | Seq: seq-parse.md#2.2.2 | R61
// An escaped closer must not close — and disabling the escape must change that,
// or the escape is doing nothing.
func TestEscapeConsumesItselfAndTheNextByte(t *testing.T) {
	const src = `"a\"b"`
	// The two tables differ in the escape and in nothing else.
	stringLang := func(escape string) *BracketLang {
		return &BracketLang{Brackets: []BracketGroup{
			{Open: []string{`"`}, Close: []string{`"`}, Escape: escape, AllowedInner: []string{}},
		}}
	}
	withEsc := assertStream(t, stringLang(`\`), src, `O"\"" T"a\\\"b" C"\""`)
	withoutEsc, _ := parse(src, 0, stringLang(""))
	if stream(withEsc) == stream(withoutEsc) {
		t.Fatalf("disabling the escape changed nothing, so the escape does nothing")
	}
}

// CRC: crc-BracketGroup.md | Seq: seq-parse.md#2.2.3 | R64
// The escape hatch that makes interpolation work.
func TestAllowedInnerReachesBackIntoCodeMode(t *testing.T) {
	assertStream(t, &LangJavaScript, "`text ${a + b} more`",
		"O\"`\" T\"text \" O\"${\" T\"a + b\" C\"}\" T\" more\" C\"`\"")
}

// CRC: crc-BracketGroup.md | R66
// The dual, and the reason it is not optional.
func TestAllowedParentSuppressesOutsideItsContext(t *testing.T) {
	top, _ := parse("${x}", 0, &LangJavaScript)
	if got, want := stream(top), `T"$" O"{" T"x" C"}"`; got != want {
		t.Fatalf("at top level ${ must be a $ then a {:\n  got  %s\n  want %s", got, want)
	}
	inner, _ := parse("`${x}`", 0, &LangJavaScript)
	if !strings.Contains(stream(inner), `O"${"`) {
		t.Fatalf("inside a template ${ must open an interpolation; got\n  %s", stream(inner))
	}
}

// CRC: crc-BracketParser.md | R57
//
// The failure nothing else can see: a sub-schema that quietly stops recognizing
// something falls back to opaque text, which loses no bytes and breaks no
// round-trip. Only counting what you expected to find sees it.
//
// Counted PER MARKER, not per kind. A tally of "five openers" is blind to one
// marker being replaced by another — measured 2026-08-30, when reordering
// LangPascal stopped "(*" being recognized and the kind counts did not move,
// because a bare "(" took its place.
func TestRecognitionCountPerLanguage(t *testing.T) {
	cases := []struct {
		name string
		lang *BracketLang
		src  string
		want map[string]int // "O(*" -> 1, keyed by kind letter and marker text
	}{
		{"go", &LangGo, "func f(a int) {\n\t// c\n\ts := \"x\"\n\treturn `r`\n}\n",
			map[string]int{"O(": 1, "C)": 1, "O{": 1, "C}": 1, "O//": 1, "C\n": 1,
				`O"`: 1, `C"`: 1, "O`": 1, "C`": 1}},
		{"shell", &LangShell, "if a; then b; else c; fi\nwhile x; do download; done\n",
			map[string]int{"Oif": 1, "Sthen": 1, "Selse": 1, "Cfi": 1,
				"Owhile": 1, "Sdo": 1, "Cdone": 1}},
		{"pascal", &LangPascal, "begin { c } writeln('s'); (* o *) end",
			map[string]int{"Obegin": 1, "Cend": 1, "O{": 1, "C}": 1,
				"O'": 1, "C'": 1, "O(": 1, "C)": 1, "O(*": 1, "C*)": 1}},
		{"js", &LangJavaScript, "let x = `a ${b + `c ${d}`} e`; // ${no}\n",
			map[string]int{"O`": 2, "C`": 2, "O${": 2, "C}": 2, "O//": 1, "C\n": 1}},
	}
	for _, c := range cases {
		d, _ := parse(c.src, 0, c.lang)
		got := map[string]int{}
		for _, n := range d.Nodes() {
			var kind string
			switch n.(type) {
			case *Opener:
				kind = "O"
			case *Closer:
				kind = "C"
			case *Separator:
				kind = "S"
			default:
				continue
			}
			text, _ := n.Render()
			got[kind+text]++
		}
		for marker, want := range c.want {
			if got[marker] != want {
				t.Errorf("%s: recognized %d of %q, expected %d\n  %s",
					c.name, got[marker], marker, want, stream(d))
			}
		}
	}
}

// CRC: crc-BracketParser.md | R57, R77
// Parsing models far more than Text did, and still loses nothing.
func TestByteRoundTripPerLanguageOverTheCorpus(t *testing.T) {
	langs := shippedLangs()
	for path, src := range corpus(t) {
		for name, lang := range langs {
			d, _ := parse(src, 0, lang)
			checkCovers(t, d, src, fmt.Sprintf("%s under %s", path, name))
		}
	}
}
