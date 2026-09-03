package sdom

import (
	"slices"
	"testing"
)

// --- test parsers -----------------------------------------------------------
//
// Each is the smallest thing that exercises one clause of the walk's contract.
// They are Parsers rather than fakes of one: the protocol is what is under test.

// recorder notes every position it is offered and never recognizes anything.
type recorder struct{ seen []int }

func (r *recorder) Parse(st *ParserState)                { r.seen = append(r.seen, st.Pos()) }
func (r *recorder) NodeType(*ParserState) (string, bool) { return "", false }
func (r *recorder) Done(*Doc)                            {}

// zeroEmitter emits ONE zero-length node at a chosen position — the shape an indent
// change back to column 0 has, and the case the position half of the no-change test
// cannot see.
type zeroEmitter struct {
	at   int
	done bool
	seen []int
}

func (z *zeroEmitter) Parse(st *ParserState) {
	z.seen = append(z.seen, st.Pos())
	if st.Pos() == z.at && !z.done {
		z.done = true
		st.Emit(NewText("", st.At(st.Pos(), 0)))
	}
}
func (z *zeroEmitter) NodeType(*ParserState) (string, bool) { return "", false }
func (z *zeroEmitter) Done(*Doc)                            {}

// advancer moves the position without emitting, the way an escape inside a
// restricted group does — the case the node-count half cannot see.
type advancer struct {
	at, by int
	done   bool
	seen   []int
}

func (a *advancer) Parse(st *ParserState) {
	a.seen = append(a.seen, st.Pos())
	if st.Pos() == a.at && !a.done {
		a.done = true
		st.Advance(a.by)
	}
}
func (a *advancer) NodeType(*ParserState) (string, bool) { return "", false }
func (a *advancer) Done(*Doc)                            {}

// lastRecorder notes Last() at every offer and emits a marker at one position.
type lastRecorder struct {
	at   int
	seen []string
}

func (r *lastRecorder) Parse(st *ParserState) {
	switch last := st.Last().(type) {
	case nil:
		r.seen = append(r.seen, "<nil>")
	case *Text:
		text, _ := last.Render()
		loc := last.Location()
		if loc.Offset()+loc.Length() != st.Pos() {
			text += "!lagging"
		}
		r.seen = append(r.seen, "T:"+text)
	default:
		text, _ := last.Render()
		r.seen = append(r.seen, "M:"+text)
	}
	if st.Pos() == r.at {
		st.Emit(NewSeparator("|", st.At(st.Pos(), 1)))
	}
}
func (r *lastRecorder) NodeType(*ParserState) (string, bool) { return "", false }
func (r *lastRecorder) Done(*Doc)                            {}

// peeker looks `by` bytes ahead with SetPos and moves back, emitting nothing.
type peeker struct{ at, by int }

func (p *peeker) Parse(st *ParserState) {
	if st.Pos() != p.at {
		return
	}
	st.SetPos(p.at + p.by)
	st.SetPos(p.at)
}
func (p *peeker) NodeType(*ParserState) (string, bool) { return "", false }
func (p *peeker) Done(*Doc)                            {}

// delegating records the positions it is offered and hands every one to a bracket
// parser, which takes the loop on an opener.
type delegating struct {
	inner *BracketParser
	seen  []int
}

func (p *delegating) Parse(st *ParserState) {
	p.seen = append(p.seen, st.Pos())
	p.inner.Parse(st)
}
func (p *delegating) NodeType(st *ParserState) (string, bool) { return p.inner.NodeType(st) }
func (p *delegating) Done(d *Doc)                             { p.inner.Done(d) }

func renders(d *Doc) []string {
	out := make([]string, 0, len(d.Nodes()))
	for _, n := range d.Nodes() {
		s, _ := n.Render()
		out = append(out, s)
	}
	return out
}

// --- the contract -----------------------------------------------------------

// CRC: crc-ParserState.md | Seq: seq-collaborate.md#1.4.1 | R164
//
// A zero-length node is not "nothing happened". Checking only the position would
// read the emission as a non-match, advance, and take the next byte as text — which
// after a dedent to column 0 can be a bracket opener rather than the line's content.
func TestAZeroLengthNodeIsNotReadAsNothingHappening(t *testing.T) {
	z := &zeroEmitter{at: 0}
	d := Parse("ab", 0, z)

	if got, want := renders(d), []string{"", "ab"}; !slices.Equal(got, want) {
		t.Fatalf("nodes %q, want %q", got, want)
	}
	// Position 0 was offered twice: once where the node was emitted, and again
	// because the walk did not advance over an emission.
	if got, want := z.seen, []int{0, 0, 1}; !slices.Equal(got, want) {
		t.Fatalf("offered %v, want %v", got, want)
	}
	if s, _ := d.Render(); s != "ab" {
		t.Fatalf("round trip %q", s)
	}
}

// CRC: crc-ParserState.md | Seq: seq-collaborate.md#1.4.2 | R164
//
// A parser that advances without emitting is not "nothing happened" either.
// Checking only the node count would ALSO take a byte here, so the walk would skip
// one — and the symptom is a missed marker, not lost bytes.
func TestAParserThatAdvancesWithoutEmittingIsNotReadAsAMatch(t *testing.T) {
	// From position 1, INSIDE a live run: extending the run changes no node count,
	// so this is the case the count term alone cannot see. At position 0 the run
	// would be created, the count would move, and the test would prove nothing.
	a := &advancer{at: 1, by: 2}
	d := Parse("abcd", 0, a)

	if got, want := a.seen, []int{0, 1, 3}; !slices.Equal(got, want) {
		t.Fatalf("offered %v, want %v — the walk must not advance over a parser that did", got, want)
	}
	if got, want := renders(d), []string{"abcd"}; !slices.Equal(got, want) {
		t.Fatalf("nodes %q, want %q", got, want)
	}
}

// CRC: crc-ParserState.md | Seq: seq-collaborate.md#1.5 | R163, R167
//
// Every position is offered exactly once when nothing is recognized, and the walk
// terminates.
func TestEveryPositionIsOfferedExactlyOnce(t *testing.T) {
	r := &recorder{}
	d := Parse("abc", 0, r)

	if got, want := r.seen, []int{0, 1, 2}; !slices.Equal(got, want) {
		t.Fatalf("offered %v, want %v", got, want)
	}
	if got, want := renders(d), []string{"abc"}; !slices.Equal(got, want) {
		t.Fatalf("nodes %q, want %q — everything unrecognized is one text run", got, want)
	}
}

// CRC: crc-ParserState.md | R157
//
// One parse, one Origin — the property that makes merging a location from one
// parser with a location from another legal. A per-context origin would mint two
// for a single document, and New refuses a mixed one.
func TestOneParseHasOneOrigin(t *testing.T) {
	ip := NewIndentParser(&LangPython)
	d := Parse("a\n  (b)\n", 0, ip)

	var indent, opener Node
	for _, n := range d.Nodes() {
		switch n.(type) {
		case *Indent:
			if indent == nil {
				indent = n
			}
		case *Opener:
			if opener == nil {
				opener = n
			}
		}
	}
	if indent == nil || opener == nil {
		t.Fatalf("expected both an Indent and an Opener, got %q", renders(d))
	}
	if indent.Location().origin != opener.Location().origin {
		t.Fatalf("two parsers in one pass produced two origins")
	}
	if ip.Brackets().Context().Origin() != indent.Location().origin {
		t.Fatalf("the context's origin is not the pass's")
	}
}

// CRC: crc-ParserState.md | Seq: seq-collaborate.md#1.4 | R156, R224, R225
//
// The array is the only parse state, and its last node is current at every offer:
// the live text run while inside one, the marker just after one. A lookahead via
// SetPos consumes nothing.
func TestTheLastNodeIsTheLiveText(t *testing.T) {
	rec := &lastRecorder{at: 2}
	d := Parse("ab|cd", 0, rec)

	if got, want := rec.seen, []string{"<nil>", "T:a", "T:ab", "M:|", "T:c"}; !slices.Equal(got, want) {
		t.Fatalf("Last at each offer: %q, want %q", got, want)
	}
	if got, want := renders(d), []string{"ab", "|", "cd"}; !slices.Equal(got, want) {
		t.Fatalf("nodes %q, want %q", got, want)
	}
	ns := d.Nodes()
	if a, b := ns[0].Location(), ns[1].Location(); a.Offset()+a.Length() != b.Offset() || !a.Faithful() {
		t.Fatalf("the text run must end exactly where the marker begins, and stay faithful")
	}

	d = Parse("abcd", 0, &peeker{at: 1, by: 2})
	if got, want := renders(d), []string{"abcd"}; !slices.Equal(got, want) {
		t.Fatalf("a lookahead consumed bytes: %q", got)
	}
}

// CRC: crc-Parser.md | Seq: seq-collaborate.md#3 | R160
//
// NodeType reports two different things with two values: a node whose group carries
// no kind, and no node at all. One string would conflate them — and the blank/comment
// rule turns on exactly that difference.
func TestNodeTypeDistinguishesNoNodeFromAnUnlabelledNode(t *testing.T) {
	lang := &BracketLang{Brackets: []BracketGroup{
		{Open: []string{"#"}, Close: []string{"\n"}, AllowedInner: []string{}, Kind: "comment"},
		{Open: []string{`"`}, Close: []string{`"`}, AllowedInner: []string{}},
	}}
	bp := NewBracketParser(lang)
	st := &ParserState{src: `x"y#z`, origin: &Origin{}, parser: bp}

	for _, tc := range []struct {
		pos  int
		kind string
		ok   bool
	}{
		{0, "", false},       // plain text: no node here
		{1, "", true},        // a string opens: a node, carrying no kind
		{3, "comment", true}, // a comment opens: a node, carrying one
	} {
		st.SetPos(tc.pos)
		kind, ok := bp.NodeType(st)
		if kind != tc.kind || ok != tc.ok {
			t.Errorf("at %d: (%q, %v), want (%q, %v)", tc.pos, kind, ok, tc.kind, tc.ok)
		}
	}
	// Looking ahead moves nothing and emits nothing.
	if st.Pos() != 3 || st.NodeCount() != 0 {
		t.Fatalf("NodeType must not consume: pos %d, %d nodes", st.Pos(), st.NodeCount())
	}
}

// CRC: crc-Parser.md | Seq: seq-collaborate.md#2.4 | R165, R166
//
// A delegate that TAKES THE LOOP hides every position inside its group from the
// outer parser. That is what makes suppression structural: "indentation is
// significant only at bracket depth 0" needs no depth check anywhere, because the
// indent parser is simply never asked.
func TestADelegateTakingTheLoopHidesPositionsFromTheOuterParser(t *testing.T) {
	p := &delegating{inner: NewBracketParser(codeLang())}
	d := Parse("a{b}c", 0, p)

	if got, want := p.seen, []int{0, 1, 4}; !slices.Equal(got, want) {
		t.Fatalf("offered %v, want %v — positions 2 and 3 are inside the group", got, want)
	}
	if got, want := renders(d), []string{"a", "{", "b", "}", "c"}; !slices.Equal(got, want) {
		t.Fatalf("nodes %q, want %q", got, want)
	}
}
