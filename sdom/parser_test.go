// CRC: crc-BracketParser.md | R57, R294, R71, R72, R73, R74, R75, R76, R77
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
		{Open: []string{"{"}, Close: "}"},
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
		{Open: []string{"do"}, Close: "fi"},
		{Open: []string{"begin"}, Close: "end"},
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

// CRC: crc-BracketGroup.md | Seq: seq-parse.md#2.2 | R294
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
			{Open: []string{`"`}, Close: `"`, Escape: escape, AllowedInner: []string{}},
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

// runLang is one pattern group: a run of backticks closing on a run of its own length;
// runs of other lengths inside are content.
func runLang() *BracketLang {
	return &BracketLang{Brackets: []BracketGroup{
		{OpenRegex: "`+", CloseIsOpen: true, AllowedInner: []string{}},
	}}
}

// rejectLang is runLang with longer runs rejected: a run longer than the opener is an
// unbalanced closer that ends the span, not content.
func rejectLang() *BracketLang {
	return &BracketLang{Brackets: []BracketGroup{
		{OpenRegex: "`+", CloseIsOpen: true, RejectLongerCloses: true, AllowedInner: []string{}},
	}}
}

// CRC: crc-BracketGroup.md | Seq: seq-parse.md#1.4 | R309, R310
func TestRejectLongerClosesEndsTheGroupOnALongerRun(t *testing.T) {
	// Shorter runs inside are content either way; a longer one is a closer that pairs
	// with nothing, and what follows opens afresh.
	assertStream(t, rejectLang(), "```` ` `` ````", "O\"````\" T\" ` `` \" C\"````\"")
	assertStream(t, rejectLang(), "``a```b`` c", "O\"``\" T\"a\" C\"```\" T\"b\" O\"``\" T\" c\"")
	_, ctx := parse("``a```b`` c", 0, rejectLang())
	if u := ctx.Unpaired(); len(u) != 1 {
		t.Errorf("%d unpaired closers, want the three-run alone", len(u))
	} else if s, _ := u[0].Render(); s != "```" {
		t.Errorf("unpaired %q, want the three-run", s)
	}
	// The rejected span's opener has no closer either — never closed — so two openers
	// are listed: the one the three-run ended, and the trailing one.
	if u := ctx.Unclosed(); len(u) != 2 {
		t.Errorf("%d unclosed openers, want the rejected span's and the trailing one", len(u))
	}
}

// CRC: crc-BracketGroup.md | Seq: seq-parse.md#1.4 | R291, R292, R309
func TestARunClosesOnlyWithARunOfItsOwnLength(t *testing.T) {
	assertStream(t, runLang(), "``a ` b`` x", "O\"``\" T\"a ` b\" C\"``\" T\" x\"")
	assertStream(t, runLang(), "```` ``` ````", "O\"````\" T\" ``` \" C\"````\"")
	assertStream(t, runLang(), "``a```b`` c", "O\"``\" T\"a```b\" C\"``\" T\" c\"")
	assertStream(t, runLang(), "a ``b``", "T\"a \" O\"``\" T\"b\" C\"``\"")
}

// CRC: crc-BracketGroup.md | Seq: seq-parse.md#1.4 | R291
func TestTheCloserIsTheOpenersOwnBytes(t *testing.T) {
	quotes := &BracketLang{Brackets: []BracketGroup{
		{Open: []string{`"`, "'"}, CloseIsOpen: true, AllowedInner: []string{}},
	}}
	assertStream(t, quotes, `"a'b" 'c"d'`, `O"\"" T"a'b" C"\"" T" " O"'" T"c\"d" C"'"`)
}

// CRC: crc-BracketGroup.md | R292, R309
func TestAPatternOpenerHonoursWordBoundariesAndItsEdges(t *testing.T) {
	xs := &BracketLang{Brackets: []BracketGroup{{OpenRegex: "x+", CloseIsOpen: true}}}
	assertStream(t, xs, "xx a xx xxa axx", `O"xx" T" a " C"xx" T" xxa axx"`)
	// Flanking as a table entry: a run of asterisks opens only before non-whitespace and
	// closes only after it, and the group names itself so emphasis nests.
	em := &BracketLang{Brackets: []BracketGroup{
		{OpenRegex: `\*+`, CloseIsOpen: true, AfterOpen: `\S`, BeforeClose: `\S`, AllowedInner: []string{`\*+`}},
	}}
	assertStream(t, em, "**a **b** c** 2 * 3", `O"**" T"a " O"**" T"b" C"**" T" c" C"**" T" 2 * 3"`)
	assertStream(t, em, "**a *b* c**", `O"**" T"a " O"*" T"b" C"*" T" c" C"**"`)
}

// CRC: crc-BracketParser.md | Seq: seq-parse.md#2.2.3 | R294, R295
func TestAllowedInnerNamesAGroup(t *testing.T) {
	lang := &BracketLang{Brackets: []BracketGroup{
		{Open: []string{"<"}, Close: ">", AllowedInner: []string{"`+"}},
		{OpenRegex: "`+", CloseIsOpen: true, AllowedInner: []string{}},
	}}
	assertStream(t, lang, "<``a`` `b`>", "O\"<\" O\"``\" T\"a\" C\"``\" T\" \" O\"`\" T\"b\" C\"`\" C\">\"")
}

// recovered runs f and hands back what it panicked with, or nil. A table that cannot
// be constructed panics, so the tests of that behaviour read the panic as a value
// instead of each wrapping a deferred recover of its own.
func recovered(f func()) (r any) {
	defer func() { r = recover() }()
	f()
	return
}

// CRC: crc-BracketLang.md | R296
func TestAContradictoryTablePanicsAtConstruction(t *testing.T) {
	cases := map[string]BracketGroup{
		"Open and OpenRegex":      {Open: []string{"a"}, OpenRegex: "a+", Close: "b"},
		"CloseIsOpen contradicts": {Open: []string{"a"}, Close: "b", CloseIsOpen: true},
		"OpenRegex":               {OpenRegex: "(", CloseIsOpen: true},
	}
	for want, g := range cases {
		r := recovered(func() { NewBracketParser(&BracketLang{Brackets: []BracketGroup{g}}) })
		if r == nil {
			t.Errorf("%s: constructed without panicking", want)
			continue
		}
		if msg := fmt.Sprint(r); !strings.Contains(msg, want) || !strings.Contains(msg, "group 0") {
			t.Errorf("%s: panic %q does not name the error and the group", want, msg)
		}
	}
}

// CRC: crc-BracketParser.md | Seq: seq-parse.md#3.1 | R297
func TestTheAnyCloseFallbackIgnoresCloseIsOpenGroups(t *testing.T) {
	lang := &BracketLang{Brackets: []BracketGroup{
		{Open: []string{"|"}, CloseIsOpen: true},
		{Open: []string{"{"}, Close: "}"},
	}}
	assertStream(t, lang, "a } |b", `T"a " C"}" T" " O"|" T"b"`)
}
