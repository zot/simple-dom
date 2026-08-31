package sdom

// CRC: crc-Declaration.md | Seq: seq-declare.md#2 | R122, R123
//
// DeclarationType and DeclarationName are the two leaf kinds a declaration pass
// adds. A declaration is NOT a node: it is a keyword node and the name nodes it
// introduces, all ordinary siblings, with the structure between them — a receiver
// group, a parameter group — left exactly where the parse put it.
//
// One node covering both writable parts would drag those groups in as children,
// which the minimality rule forbids and the span rule would then oblige. Narrow
// siblings keep every byte with the owner the parse gave it.
//
// Each embeds Text, so Kids, Location and Render are the leaf behaviour defined
// once, and each declares its own Equals because a promoted one cannot see the
// outer type. Same shape as Opener, Closer and Separator, and for the same reasons.
type DeclarationType struct{ Text }

// DeclarationName is a name its DeclarationType introduces. A keyword may
// introduce several — a grouped const or var declares one per entry.
type DeclarationName struct{ Text }

// CRC: crc-Declaration.md | R123
func NewDeclarationType(text string, loc Loc) *DeclarationType {
	return &DeclarationType{Text{text: text, loc: loc}}
}

// CRC: crc-Declaration.md | R123
func NewDeclarationName(text string, loc Loc) *DeclarationName {
	return &DeclarationName{Text{text: text, loc: loc}}
}

// CRC: crc-Declaration.md | R122
//
// A DeclarationType holds NO reference to the names it declares. No node kind
// holds state its children do not carry, which is the rule that also keeps a
// marker from holding its bracket group; the link lives on the parse context.
func (d *DeclarationType) Equals(other Node) bool {
	x, ok := other.(*DeclarationType)
	return ok && d.Text.Equals(&x.Text)
}

// CRC: crc-Declaration.md | R122
func (d *DeclarationName) Equals(other Node) bool {
	x, ok := other.(*DeclarationName)
	return ok && d.Text.Equals(&x.Text)
}
