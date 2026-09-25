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

// CRC: crc-BracketLang.md | R59, R62
// Comments and strings as groups, not special cases.
func TestLangGo(t *testing.T) {
	const src = "// c\n/* b */\ns := \"a\\\"b\"\nr := `raw \" { `\n"
	d, _ := parse(src, 0, &LangGo)
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
	d, _ := parse("if a; then b; else c; fi\n", 0, &LangShell)
	assertContains(t, stream(d), `O"if"`, `S"then"`, `S"else"`, `C"fi"`)

	d2, _ := parse("while x; do y; done\n", 0, &LangShell)
	got2 := stream(d2)
	assertContains(t, got2, `O"while"`, `S"do"`, `C"done"`)
	if strings.Contains(got2, `S"then"`) {
		t.Errorf("then is a separator of if, not of while:\n  %s", got2)
	}
}

// CRC: crc-BracketLang.md | R59
// The other word-bracket shape, and a language whose "{" is a comment.
func TestLangPascal(t *testing.T) {
	d, _ := parse("begin { c } writeln('s'); (* o *) end", 0, &LangPascal)
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
	top, _ := parse("${x}", 0, &LangJavaScript)
	if strings.Contains(stream(top), `O"${"`) {
		t.Errorf("${ must not be an interpolation opener at top level:\n  %s", stream(top))
	}
}

// CRC: crc-BracketLang.md | R207, R208, R209
//
// The agreement between how a language WRITES a comment and how it recognizes
// one, guarded here rather than at runtime.
func TestEveryCommentStyleConstructsItsOwnKind(t *testing.T) {
	langs := shippedLangs()
	langs["lua"], langs["python"] = &LangLua, &LangPython.BracketLang
	for name, lang := range langs {
		cs := lang.Comment
		if cs.Prefix == "" {
			t.Logf("%s: no comment style", name)
			continue
		}
		d, ctx := parse(cs.Prefix+"x"+cs.Suffix, 0, lang)
		open, ok := d.Nodes()[0].(*Opener)
		if !ok {
			t.Errorf("%s: the constructed comment does not open with a marker", name)
			continue
		}
		s, _ := open.Render()
		if g := lang.GroupFor(s); g == nil || g.Kind != cs.Kind {
			t.Errorf("%s: opener %q parses as kind %v, want %q", name, s, g, cs.Kind)
		}
		if got := ctx.InnerText(open); strings.TrimSpace(got) != "x" {
			t.Errorf("%s: interior %q", name, got)
		}
		if cl := ctx.Closer(open); cl == nil {
			t.Errorf("%s: the constructed comment never closes", name)
		} else if c, _ := cl.Render(); !strings.HasSuffix(cs.Suffix, c) {
			t.Errorf("%s: closer %q is not the tail of Suffix %q", name, c, cs.Suffix)
		}
	}
}

// CRC: crc-BracketLang.md | R296
// Every shipped table constructs; the construction check panics for a table that
// cannot, and this is what keeps a consumer from ever seeing it.
func TestEveryShippedTableConstructs(t *testing.T) {
	langs := shippedLangs()
	langs["typescript"], langs["lua"] = &LangTypeScript, &LangLua
	langs["python"] = &LangPython.BracketLang
	for name, lang := range langs {
		if r := recovered(func() { NewBracketParser(lang) }); r != nil {
			t.Errorf("%s: %v", name, r)
		}
	}
}

// CRC: crc-BracketLang.md | R361
// Lua's long strings and block comments at every level: a closer of another level
// inside is content, and a keyword inside closes nothing. The first three are the
// probes that found level 1 unread (2026-09-25).
func TestLuaLongBracketsAtEveryLevel(t *testing.T) {
	cases := []struct{ name, src, want string }{
		{"a level-1 string holds a level-0 closer", "x = [=[ a ]] b ]=] y", "[=[]=] | |"},
		{"a level-0 string holds a level-1 closer", "x = [[ a ]=] b ]] y", "[[]] | |"},
		{"a level-1 comment holds a level-0 closer", "--[=[ a ]] b ]=] y", "--[=[]=] | |"},
		{"end inside a level-1 comment closes nothing", "function f()\n--[=[\nend\n]=]\nend", "functionend () --[=[]=] | |"},
		{"end inside a level-1 string closes nothing", "if x then s = [=[ end ]] ]=] end", "ifend [=[]=] | |"},
		{"index brackets are not long strings", "t[1] = t[ [[k]] ]", "[] [] [[]] | |"},
	}
	for _, c := range cases {
		d, ctx := parse(c.src, 0, &LangLua)
		if got := pairing(d, ctx); got != c.want {
			t.Errorf("%s: %q pairs as %q, want %q", c.name, c.src, got, c.want)
		}
		if r, _ := d.Render(); r != c.src {
			t.Errorf("%s: render %q, want the source", c.name, r)
		}
	}
}
