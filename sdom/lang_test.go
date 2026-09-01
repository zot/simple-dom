// CRC: crc-BracketLang.md | R59, R62, R64, R66, R121, R72
package sdom

import (
	"strings"
	"testing"
)

// assertContains checks that each want appears in got, a rendered node stream. A
// language test asserts what the table recognized rather than the whole stream,
// so that extending a table does not rewrite every assertion about it.
func assertContains(t *testing.T, got string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Errorf("expected %s in\n  %s", want, got)
		}
	}
}

// CRC: crc-BracketLang.md | R121
//
// The tables are chosen to cover the mechanism; this is the assertion that keeps
// that true as they change. A field live in no table is dead code.
func TestEveryFieldOfBracketGroupIsLiveSomewhere(t *testing.T) {
	seen := map[string]bool{}
	for _, lang := range shippedLangs() {
		for i := range lang.Brackets {
			g := &lang.Brackets[i]
			if len(g.Open) > 0 {
				seen["Open"] = true
			}
			if len(g.Separators) > 0 {
				seen["Separators"] = true
			}
			if len(g.Close) > 0 {
				seen["Close"] = true
			}
			if g.Escape != "" {
				seen["Escape"] = true
			}
			if g.AllowedInner == nil {
				seen["AllowedInner(nil)"] = true
			} else if len(g.AllowedInner) == 0 {
				seen["AllowedInner(empty)"] = true
			} else {
				seen["AllowedInner(named)"] = true
			}
			if g.AllowedParent != nil {
				seen["AllowedParent"] = true
			}
		}
	}
	for _, field := range []string{
		"Open", "Separators", "Close", "Escape",
		"AllowedInner(nil)", "AllowedInner(empty)", "AllowedInner(named)", "AllowedParent",
	} {
		if !seen[field] {
			t.Errorf("%s is exercised by no shipped table, so it is dead code", field)
		}
	}
}

// CRC: crc-BracketLang.md | R59, R62
// Comments and strings as groups, not special cases.
func TestLangGo(t *testing.T) {
	const src = "// c\n/* b */\ns := \"a\\\"b\"\nr := `raw \" { `\n"
	d, _ := Parse(src, 0, &LangGo)
	assertContains(t, stream(d),
		`O"//"`, `C"\n"`, // line comment is a group
		`O"/*"`, `C"*/"`, // block comment likewise
		`T"a\\\"b"`,        // the escaped quote did not close the string
		"T\"raw \\\" { \"", // nothing inside the raw string is recognized
	)
}

// CRC: crc-BracketLang.md | R72
// Word brackets with separators, which nothing else exercises.
func TestLangShell(t *testing.T) {
	d, _ := Parse("if a; then b; else c; fi\n", 0, &LangShell)
	assertContains(t, stream(d), `O"if"`, `S"then"`, `S"else"`, `C"fi"`)

	d2, _ := Parse("while x; do y; done\n", 0, &LangShell)
	got2 := stream(d2)
	assertContains(t, got2, `O"while"`, `S"do"`, `C"done"`)
	if strings.Contains(got2, `S"then"`) {
		t.Errorf("then is a separator of if, not of while:\n  %s", got2)
	}
}

// CRC: crc-BracketLang.md | R59
// The other word-bracket shape, and a language whose "{" is a comment.
func TestLangPascal(t *testing.T) {
	d, _ := Parse("begin { c } writeln('s'); (* o *) end", 0, &LangPascal)
	got := stream(d)
	assertContains(t, got, `O"begin"`, `C"end"`, `O"{"`, `T" c "`, `O"(*"`, `C"*)"`)
	// The brace comment must not act as a code bracket: its interior is literal.
	if strings.Contains(got, `O"("`) && !strings.Contains(got, `O"(*"`) {
		t.Errorf("(* must be matched before the bare ( :\n  %s", got)
	}
}

// CRC: crc-BracketLang.md | R64, R66
// The only table exercising both mode fields together, to full depth.
func TestLangJavaScript(t *testing.T) {
	assertStream(t, &LangJavaScript, "`a ${b + `c ${d}`} e`",
		"O\"`\" T\"a \" O\"${\" T\"b + \" O\"`\" T\"c \" O\"${\" T\"d\" C\"}\" C\"`\" C\"}\" T\" e\" C\"`\"")
	top, _ := Parse("${x}", 0, &LangJavaScript)
	if strings.Contains(stream(top), `O"${"`) {
		t.Errorf("${ must not be an interpolation opener at top level:\n  %s", stream(top))
	}
}
