package sdom

// CRC: crc-Marker.md | R78, R79
//
// Opener, Closer and Separator are the three leaf kinds the parser adds. Each
// holds the bytes it matched and nothing more — in particular NO pointer to its
// bracket group. Holding one would make Equals compare pointers, so two documents
// parsed with independently constructed languages would never be equal and the
// structural round-trip would fail on every bracketed document. The active group
// comes from the parse context instead.
//
// Three kinds rather than one carrying a role: a symmetric group's `"` is
// byte-identical opening and closing, so the role cannot be derived from the
// bytes, and a stored role would be state the children do not carry, which Equals
// would then have to compare. As separate types the discrimination is the type
// assertion every Equals already performs, and nothing extra is stored.
//
// Each embeds Text, so Kids, Location and Render are the leaf behaviour defined
// once. Each declares its own Equals, because a promoted one cannot see the outer
// type.
type Opener struct{ Text }

// Closer is the marker that ends a bracket group. See Opener.
type Closer struct{ Text }

// Separator is a mid-group marker, such as "else" between "if" and "fi".
type Separator struct{ Text }

// CRC: crc-Marker.md | R78
func NewOpener(text string, loc Loc) *Opener { return &Opener{Text{text: text, loc: loc}} }

// CRC: crc-Marker.md | R78
func NewCloser(text string, loc Loc) *Closer { return &Closer{Text{text: text, loc: loc}} }

// CRC: crc-Marker.md | R78
func NewSeparator(text string, loc Loc) *Separator {
	return &Separator{Text{text: text, loc: loc}}
}

// CRC: crc-Marker.md | R79
func (o *Opener) Equals(other Node) bool {
	x, ok := other.(*Opener)
	return ok && o.Text.Equals(&x.Text)
}

// CRC: crc-Marker.md | R79
func (c *Closer) Equals(other Node) bool {
	x, ok := other.(*Closer)
	return ok && c.Text.Equals(&x.Text)
}

// CRC: crc-Marker.md | R79
func (s *Separator) Equals(other Node) bool {
	x, ok := other.(*Separator)
	return ok && s.Text.Equals(&x.Text)
}
