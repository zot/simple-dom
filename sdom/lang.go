package sdom

// CRC: crc-BracketLang.md | R121, R70
//
// The shipped tables have TWO jobs. They cover the mechanism — between them every
// field of BracketGroup is live, so no mode is dead code and the recognition count
// has languages that recognize something — and they serve the languages mini-spec
// reads: Go, TypeScript, JavaScript, Lua and Shell, with Python following the
// indent parser. LangPascal earns its place under the first job alone. A consumer
// needing a language outside the set still constructs its own BracketLang.
//
// Within a language, groups are listed so that a multi-character marker precedes
// any group whose marker is a prefix of it — "/*" before a bare "/", "(*" before
// "(" — because the first match wins.

// CRC: crc-BracketLang.md | R59, R62, R121
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

// CRC: crc-BracketLang.md | R121, R72
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

// CRC: crc-BracketLang.md | R121
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

// CRC: crc-BracketLang.md | R64, R66, R121
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

// CRC: crc-BracketLang.md | R121
// LangTypeScript is JavaScript's table: types add no brackets of their own, and a
// consumer asking for TypeScript should not have to know that.
var LangTypeScript = LangJavaScript

// CRC: crc-BracketLang.md | R121, R71
//
// LangLua is the table with no prior art — microfts2 configures eight languages
// and Lua is not among them. It is also the one whose declaration keyword is a
// BRACKET: `function` opens a group closing with `end`, so no text pattern can see
// it, which is what the schema layer has to cope with.
//
// `do` is a SEPARATOR of the loop group, never an opener. That is what Separators
// are for, and it is how microfts2 shapes shell's `while`/`for`..`do`..`done`.
// Lua's standalone `do ... end` block is therefore not modelled: with `do` a
// separator, an inner `do ... end` would read as separator-then-close and end the
// enclosing group early, and making it an opener breaks the loops instead.
//
// Order matters twice here: `--[[` precedes `--` and `[[` precedes `[`, because
// the first match wins; and `elseif` precedes `else` among the separators, since
// `else` is a prefix of it.
var LangLua = BracketLang{Brackets: []BracketGroup{
	{Open: []string{"--[["}, Close: []string{"]]"}, AllowedInner: []string{}},
	{Open: []string{"--"}, Close: []string{"\n"}, AllowedInner: []string{}},
	{Open: []string{"[["}, Close: []string{"]]"}, AllowedInner: []string{}},
	{Open: []string{`"`}, Close: []string{`"`}, Escape: `\`, AllowedInner: []string{}},
	{Open: []string{"'"}, Close: []string{"'"}, Escape: `\`, AllowedInner: []string{}},
	{Open: []string{"function", "for", "while"}, Separators: []string{"do"}, Close: []string{"end"}},
	{Open: []string{"if"}, Separators: []string{"elseif", "else", "then"}, Close: []string{"end"}},
	{Open: []string{"repeat"}, Close: []string{"until"}},
	{Open: []string{"("}, Close: []string{")"}},
	{Open: []string{"{"}, Close: []string{"}"}},
	{Open: []string{"["}, Close: []string{"]"}},
}}
