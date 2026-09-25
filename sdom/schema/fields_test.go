package schema

import (
	"reflect"
	"testing"

	"github.com/zot/simple-dom/sdom"
)

// CRC: crc-BracketLang.md | R121
//
// The tables are chosen to cover the mechanism; this is the assertion that keeps
// that true as they change. A field live in no table is dead code. It lives here
// rather than in sdom because the markdown base's table is the only user of the
// flanking, run and demotion fields, and sdom cannot import this package.
//
// The fields are read by reflection, so a field added to BracketGroup is covered the
// day it is added: CloseRegex went a day with no shipped table using it, and nothing
// said so. AllowedInner's three modes are checked by name, since nil is a mode.
func TestEveryFieldOfBracketGroupIsLiveSomewhere(t *testing.T) {
	tables := map[string]*sdom.BracketLang{
		"go": &sdom.LangGo, "shell": &sdom.LangShell, "pascal": &sdom.LangPascal,
		"js": &sdom.LangJavaScript, "typescript": &sdom.LangTypeScript, "lua": &sdom.LangLua,
		"python": &sdom.LangPython.BracketLang, "markdown": &LangMarkdown.BracketLang,
	}
	seen := map[string]bool{}
	for _, lang := range tables {
		for _, g := range lang.Brackets {
			v := reflect.ValueOf(g)
			for i := range v.NumField() {
				if !v.Field(i).IsZero() {
					seen[v.Type().Field(i).Name] = true
				}
			}
			switch {
			case g.AllowedInner == nil:
				seen["AllowedInner(nil)"] = true
			case len(g.AllowedInner) == 0:
				seen["AllowedInner(empty)"] = true
			default:
				seen["AllowedInner(named)"] = true
			}
		}
	}
	want := []string{"AllowedInner(nil)", "AllowedInner(empty)", "AllowedInner(named)"}
	for f := range reflect.TypeFor[sdom.BracketGroup]().Fields() {
		want = append(want, f.Name)
	}
	for _, field := range want {
		if !seen[field] {
			t.Errorf("%s is exercised by no shipped table, so it is dead code", field)
		}
	}
}
