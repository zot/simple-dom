package sdom

// CRC: crc-BracketLang.md | R69, R70
//
// The shipped tables are chosen to COVER THE MECHANISM rather than to serve
// consumers: between them every field of BracketGroup is live, so no mode is dead
// code and the recognition count has languages that recognize something. A
// consumer needing another language constructs its own BracketLang.
//
// Within a language, groups are listed so that a multi-character marker precedes
// any group whose marker is a prefix of it — "/*" before a bare "/", "(*" before
// "(" — because the first match wins.

// CRC: crc-BracketLang.md | R59, R62, R69
// LangGo exercises code brackets, both comment forms, a string with an escape,
// and a raw string without one.
var LangGo = BracketLang{Brackets: []BracketGroup{
	{Open: []string{"//"}, Close: []string{"\n"}, AllowedInner: []string{}},
	{Open: []string{"/*"}, Close: []string{"*/"}, AllowedInner: []string{}},
	{Open: []string{`"`}, Close: []string{`"`}, Escape: `\`, AllowedInner: []string{}},
	{Open: []string{"'"}, Close: []string{"'"}, Escape: `\`, AllowedInner: []string{}},
	{Open: []string{"`"}, Close: []string{"`"}, AllowedInner: []string{}},
	{Open: []string{"{"}, Close: []string{"}"}},
	{Open: []string{"("}, Close: []string{")"}},
	{Open: []string{"["}, Close: []string{"]"}},
}}

// CRC: crc-BracketLang.md | R69, R72
// LangShell exercises word brackets with separators, which nothing else does.
var LangShell = BracketLang{Brackets: []BracketGroup{
	{Open: []string{"#"}, Close: []string{"\n"}, AllowedInner: []string{}},
	{Open: []string{`"`}, Close: []string{`"`}, Escape: `\`, AllowedInner: []string{}},
	{Open: []string{"'"}, Close: []string{"'"}, AllowedInner: []string{}},
	{Open: []string{"if"}, Separators: []string{"then", "elif", "else"}, Close: []string{"fi"}},
	{Open: []string{"while", "until"}, Separators: []string{"do"}, Close: []string{"done"}},
	{Open: []string{"for"}, Separators: []string{"in", "do"}, Close: []string{"done"}},
	{Open: []string{"case"}, Separators: []string{"in"}, Close: []string{"esac"}},
	{Open: []string{"{"}, Close: []string{"}"}},
	{Open: []string{"("}, Close: []string{")"}},
}}

// CRC: crc-BracketLang.md | R69
// LangPascal exercises the other word-bracket shape, and a language whose "{" is
// a comment rather than a code bracket.
var LangPascal = BracketLang{Brackets: []BracketGroup{
	{Open: []string{"(*"}, Close: []string{"*)"}, AllowedInner: []string{}},
	{Open: []string{"{"}, Close: []string{"}"}, AllowedInner: []string{}},
	{Open: []string{"'"}, Close: []string{"'"}, AllowedInner: []string{}},
	{Open: []string{"begin"}, Close: []string{"end"}},
	{Open: []string{"("}, Close: []string{")"}},
	{Open: []string{"["}, Close: []string{"]"}},
}}

// CRC: crc-BracketLang.md | R64, R66, R69
// LangJavaScript is the only table exercising AllowedInner and AllowedParent
// together: a template literal is scan-restricted with one escape hatch, and the
// interpolation that hatch opens is recognized nowhere else.
var LangJavaScript = BracketLang{Brackets: []BracketGroup{
	{Open: []string{"//"}, Close: []string{"\n"}, AllowedInner: []string{}},
	{Open: []string{"/*"}, Close: []string{"*/"}, AllowedInner: []string{}},
	{Open: []string{"${"}, Close: []string{"}"}, AllowedParent: []string{"`"}},
	{Open: []string{"`"}, Close: []string{"`"}, Escape: `\`, AllowedInner: []string{"${"}},
	{Open: []string{`"`}, Close: []string{`"`}, Escape: `\`, AllowedInner: []string{}},
	{Open: []string{"'"}, Close: []string{"'"}, Escape: `\`, AllowedInner: []string{}},
	{Open: []string{"{"}, Close: []string{"}"}},
	{Open: []string{"("}, Close: []string{")"}},
	{Open: []string{"["}, Close: []string{"]"}},
}}
