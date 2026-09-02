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
	{Open: []string{"//"}, Close: []string{"\n"}, AllowedInner: []string{}, Kind: "comment"},
	{Open: []string{"/*"}, Close: []string{"*/"}, AllowedInner: []string{}, Kind: "comment"},
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
	{Open: []string{"#"}, Close: []string{"\n"}, AllowedInner: []string{}, Kind: "comment"},
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
	{Open: []string{"(*"}, Close: []string{"*)"}, AllowedInner: []string{}, Kind: "comment"},
	{Open: []string{"{"}, Close: []string{"}"}, AllowedInner: []string{}, Kind: "comment"},
	{Open: []string{"'"}, Close: []string{"'"}, AllowedInner: []string{}},
	{Open: []string{"begin"}, Close: []string{"end"}},
	{Open: []string{"("}, Close: []string{")"}},
	{Open: []string{"["}, Close: []string{"]"}},
}}

// CRC: crc-BracketLang.md | R64, R66, R121
// LangJavaScript is the only table exercising AllowedInner and AllowedParent
// together: a template literal is parse-restricted with one escape hatch, and the
// interpolation that hatch opens is recognized nowhere else.
var LangJavaScript = BracketLang{Brackets: []BracketGroup{
	{Open: []string{"//"}, Close: []string{"\n"}, AllowedInner: []string{}, Kind: "comment"},
	{Open: []string{"/*"}, Close: []string{"*/"}, AllowedInner: []string{}, Kind: "comment"},
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
	{Open: []string{"--[["}, Close: []string{"]]"}, AllowedInner: []string{}, Kind: "comment"},
	{Open: []string{"--"}, Close: []string{"\n"}, AllowedInner: []string{}, Kind: "comment"},
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

// CRC: crc-IndentLang.md | R169, R170, R171, R172, R173
//
// LangPython is the first IndentLang, and the only table modelling string PREFIXES.
//
// Only the f forms get groups, because only they change how the text parses: an
// f-string interpolates, so it is restricted with "{" as its one escape hatch —
// naming Python's OWN code brace, so the inside of {…} is full code mode and a dict
// display parses like any other. r, b and u need nothing: the prefix letter falls
// through as text and the quote that follows opens the ordinary group.
//
// Raw does not mean unescaped. `r"\""` compiles and `r"\"` does not, so a backslash
// escapes the closing quote even in a raw string — a lexical rule rather than a
// semantic one — and every string group carries the same Escape.
//
// All ten f spellings are listed rather than the six strictly necessary: rf" would
// parse without one, since r falls through and f" matches at the next byte, but the
// prefix would then straddle two nodes and a literal is one thing.
//
// Order: f groups first, longest quote form first, so f""" is matched before f".
// A plain """ never competes — at the f it cannot match at all.
var LangPython = IndentLang{
	BracketLang: BracketLang{Brackets: []BracketGroup{
		{Open: []string{"#"}, Close: []string{"\n"}, AllowedInner: []string{}, Kind: "comment"},

		{Open: []string{`f"""`, `F"""`, `fr"""`, `fR"""`, `Fr"""`, `FR"""`, `rf"""`, `rF"""`, `Rf"""`, `RF"""`},
			Close: []string{`"""`}, Escape: `\`, AllowedInner: []string{"{"}},
		{Open: []string{"f'''", "F'''", "fr'''", "fR'''", "Fr'''", "FR'''", "rf'''", "rF'''", "Rf'''", "RF'''"},
			Close: []string{"'''"}, Escape: `\`, AllowedInner: []string{"{"}},
		{Open: []string{`f"`, `F"`, `fr"`, `fR"`, `Fr"`, `FR"`, `rf"`, `rF"`, `Rf"`, `RF"`},
			Close: []string{`"`}, Escape: `\`, AllowedInner: []string{"{"}},
		{Open: []string{"f'", "F'", "fr'", "fR'", "Fr'", "FR'", "rf'", "rF'", "Rf'", "RF'"},
			Close: []string{"'"}, Escape: `\`, AllowedInner: []string{"{"}},

		{Open: []string{`"""`}, Close: []string{`"""`}, Escape: `\`, AllowedInner: []string{}},
		{Open: []string{"'''"}, Close: []string{"'''"}, Escape: `\`, AllowedInner: []string{}},
		{Open: []string{`"`}, Close: []string{`"`}, Escape: `\`, AllowedInner: []string{}},
		{Open: []string{"'"}, Close: []string{"'"}, Escape: `\`, AllowedInner: []string{}},

		{Open: []string{"("}, Close: []string{")"}},
		{Open: []string{"["}, Close: []string{"]"}},
		{Open: []string{"{"}, Close: []string{"}"}},
	}},

	// CPython expands a tab to the next multiple of eight.
	Tab: 8,

	// The Kind this table marks its comments with. sdom compares this against a
	// group's label and never learns what the word means.
	Transparent: "comment",

	Continuation: `\`,
}
