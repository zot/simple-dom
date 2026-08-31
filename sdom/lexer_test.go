// CRC: crc-Lexer.md | R57, R65, R71, R72, R73, R74, R75, R76, R77
package sdom

import (
	"fmt"
	"strings"
	"testing"
)

// stream renders the node array compactly for assertions: O/C/S for the marker
// kinds, T for text, each followed by its quoted bytes.
func stream(d *Doc) string {
	var b strings.Builder
	for i, n := range d.Nodes() {
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

// assertStream scans src with lang and checks the whole node stream against
// want, returning the document for any further assertions.
func assertStream(t *testing.T, lang *BracketLang, src, want string) *Doc {
	t.Helper()
	d, _ := Scan(src, 0, lang)
	if got := stream(d); got != want {
		t.Errorf("%q:\n  got  %s\n  want %s", src, got, want)
	}
	return d
}

// checkCovers asserts the two properties a lossless scan always has: the
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

// CRC: crc-Lexer.md | Seq: seq-scan.md#1.5 | R77
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

// CRC: crc-Lexer.md | Seq: seq-scan.md#1.3.3 | R72
func TestSeparatorsOnlyInsideTheirGroup(t *testing.T) {
	d, _ := Scan("else if x then y else z fi", 0, &LangShell)
	got := stream(d)
	if strings.Count(got, `S"else"`) != 1 {
		t.Fatalf("expected exactly one separator else; got\n  %s", got)
	}
	if !strings.HasPrefix(got, `T"else "`) {
		t.Fatalf("the leading bare else must be text; got\n  %s", got)
	}
}

// CRC: crc-Lexer.md | Seq: seq-scan.md#3.1 | R73
// The any-close fallback keeps an unbalanced file scannable.
func TestStrayCloserLandsAsABracket(t *testing.T) {
	assertStream(t, codeLang(), "a } b", `T"a " C"}" T" b"`)
}

// CRC: crc-Lexer.md | Seq: seq-scan.md#3.2 | R74
// The guarantee that makes unknown input safe.
func TestTheScanNeverStalls(t *testing.T) {
	src := "$ \x00 \xff\xfe unknown ~`!@#%^&*"
	d, _ := Scan(src, 0, codeLang())
	if got, err := d.Render(); err != nil || got != src {
		t.Fatalf("unrecognized input must still round-trip (err %v)", err)
	}
}

// CRC: crc-Lexer.md | Seq: seq-scan.md#3.4 | R75
// An unbalanced file drops no bytes.
func TestUnclosedGroupClosesAtEndOfInput(t *testing.T) {
	for _, src := range []string{"func f() {", `s := "unterminated`, "// trailing", "`raw"} {
		d, _ := Scan(src, 0, &LangGo)
		checkCovers(t, d, src, fmt.Sprintf("%q", src))
	}
}

// CRC: crc-Lexer.md | R76
// A text run is everything between two markers, whitespace included.
func TestWhitespaceFoldsIntoText(t *testing.T) {
	assertStream(t, codeLang(), "{ a  b\n  c }", `O"{" T" a  b\n  c " C"}"`)
}

// CRC: crc-BracketGroup.md | Seq: seq-scan.md#2.2 | R65
// The property that makes strings and comments the same case.
func TestNothingInsideARestrictedGroupIsTokenized(t *testing.T) {
	cases := []struct{ src, want string }{
		{`"a { b // c"`, `O"\"" T"a { b // c" C"\""`},
		{"// a \"b\" { c\n", `O"//" T" a \"b\" { c" C"\n"`},
		{`/* a "b" // c */`, `O"/*" T" a \"b\" // c " C"*/"`},
	}
	for _, c := range cases {
		assertStream(t, &LangGo, c.src, c.want)
	}
}

// CRC: crc-BracketGroup.md | Seq: seq-scan.md#2.2.2 | R61
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
	withoutEsc, _ := Scan(src, 0, stringLang(""))
	if stream(withEsc) == stream(withoutEsc) {
		t.Fatalf("disabling the escape changed nothing, so the escape does nothing")
	}
}

// CRC: crc-BracketGroup.md | Seq: seq-scan.md#2.2.3 | R64
// The escape hatch that makes interpolation work.
func TestAllowedInnerReachesBackIntoCodeMode(t *testing.T) {
	assertStream(t, &LangJavaScript, "`text ${a + b} more`",
		"O\"`\" T\"text \" O\"${\" T\"a + b\" C\"}\" T\" more\" C\"`\"")
}

// CRC: crc-BracketGroup.md | R66
// The dual, and the reason it is not optional.
func TestAllowedParentSuppressesOutsideItsContext(t *testing.T) {
	top, _ := Scan("${x}", 0, &LangJavaScript)
	if got, want := stream(top), `T"$" O"{" T"x" C"}"`; got != want {
		t.Fatalf("at top level ${ must be a $ then a {:\n  got  %s\n  want %s", got, want)
	}
	inner, _ := Scan("`${x}`", 0, &LangJavaScript)
	if !strings.Contains(stream(inner), `O"${"`) {
		t.Fatalf("inside a template ${ must open an interpolation; got\n  %s", stream(inner))
	}
}

// CRC: crc-Lexer.md | R57
// The failure nothing else can see: a sub-lexicon that quietly stops recognizing
// something falls back to opaque text, which loses no bytes and breaks no
// round-trip. Only counting what you expected to find sees it.
func TestRecognitionCountPerLanguage(t *testing.T) {
	cases := []struct {
		name             string
		lang             *BracketLang
		src              string
		open, close, sep int
	}{
		{"go", &LangGo, "func f(a int) {\n\t// c\n\ts := \"x\"\n\treturn `r`\n}\n", 5, 5, 0},
		{"shell", &LangShell, "if a; then b; else c; fi\nwhile x; do download; done\n", 2, 2, 3},
		{"pascal", &LangPascal, "begin { c } writeln('s'); (* o *) end", 5, 5, 0},
		{"js", &LangJavaScript, "let x = `a ${b + `c ${d}`} e`; // ${no}\n", 5, 5, 0},
	}
	for _, c := range cases {
		d, _ := Scan(c.src, 0, c.lang)
		var o, cl, s int
		for _, n := range d.Nodes() {
			switch n.(type) {
			case *Opener:
				o++
			case *Closer:
				cl++
			case *Separator:
				s++
			}
		}
		if o != c.open || cl != c.close || s != c.sep {
			t.Errorf("%s: recognized %d/%d/%d (open/close/sep), expected %d/%d/%d\n  %s",
				c.name, o, cl, s, c.open, c.close, c.sep, stream(d))
		}
	}
}

// CRC: crc-Lexer.md | R57, R77
// Scanning models far more than Text did, and still loses nothing.
func TestByteRoundTripPerLanguageOverTheCorpus(t *testing.T) {
	langs := shippedLangs()
	for path, src := range corpus(t) {
		for name, lang := range langs {
			d, _ := Scan(src, 0, lang)
			checkCovers(t, d, src, fmt.Sprintf("%s under %s", path, name))
		}
	}
}
