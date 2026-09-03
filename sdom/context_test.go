// CRC: crc-BracketContext.md | R82, R83, R84, R85, R86, R87
package sdom

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

func shippedLangs() map[string]*BracketLang {
	return map[string]*BracketLang{
		"go": &LangGo, "shell": &LangShell, "pascal": &LangPascal, "js": &LangJavaScript,
	}
}

// CRC: crc-BracketContext.md | Seq: seq-pair.md#1.2 | R82, R83
func TestPairingIsRecordedBothWays(t *testing.T) {
	lang := &BracketLang{Brackets: []BracketGroup{
		{Open: []string{"{"}, Close: []string{"}"}},
		{Open: []string{"["}, Close: []string{"]"}},
	}}
	d, ctx := parse("a {b [c] d} e", 0, lang)
	var opens, closes []Node
	for _, n := range d.Nodes() {
		switch n.(type) {
		case *Opener:
			opens = append(opens, n)
		case *Closer:
			closes = append(closes, n)
		}
	}
	if len(opens) != 2 || len(closes) != 2 {
		t.Fatalf("expected two pairs, got %d openers and %d closers", len(opens), len(closes))
	}
	// {…} is outermost, so it pairs with the LAST closer; [ ] with the first.
	if ctx.Closer(opens[0]) != closes[1] || ctx.Opener(closes[1]) != opens[0] {
		t.Errorf("the outer pair does not agree in both directions")
	}
	if ctx.Closer(opens[1]) != closes[0] || ctx.Opener(closes[0]) != opens[1] {
		t.Errorf("the inner pair does not agree in both directions")
	}
}

// CRC: crc-BracketContext.md | Seq: seq-pair.md#1.3 | R84
func TestEveryNodeKnowsItsEnclosingOpener(t *testing.T) {
	d, ctx := parse("a {b {c} d} e", 0, codeLang())
	ns := d.Nodes()
	outer, inner := ns[1], ns[3] // the two openers
	want := map[int]Node{
		0: nil, 1: nil, 2: outer, 3: outer, 4: inner, 6: outer, 8: nil,
	}
	// Report WHICH node, not merely whether one exists: two different openers are
	// both non-nil, and a message that prints "got true, want true" on a failure
	// is worse than no message.
	name := func(n Node) string {
		if n == nil {
			return "none"
		}
		for j, c := range ns {
			if c == n {
				s, _ := c.Render()
				return fmt.Sprintf("node %d (%q)", j, s)
			}
		}
		return "a node from another document"
	}
	for i, w := range want {
		if got := ctx.Enclosing(ns[i]); got != w {
			s, _ := ns[i].Render()
			t.Errorf("node %d (%q): enclosed by %s, want %s", i, s, name(got), name(w))
		}
	}
}

// CRC: crc-BracketContext.md | Seq: seq-pair.md#2 | R87
//
// independentLinks derives every bracket link from the flat array ALONE — a stack
// of open openers, the source, and the table. No parse, no context, no index.
//
// This is the second derivation R87 promises a consumer can perform, and it lives
// HERE rather than in the library on purpose: an index checked by code that shares
// its author, its file and its helpers is checked by something liable to share its
// misconceptions too. What the library owes is that the answer is reproducible from
// the array; proving it is a consumer's job, and a test is a consumer.
func independentLinks(d *Doc, lang *BracketLang) (closerOf map[Node]*Closer, openerOf map[Node]*Opener, enclosing map[Node]Node, seps map[Node][]Node) {
	closerOf, openerOf, enclosing = map[Node]*Closer{}, map[Node]*Opener{}, map[Node]Node{}
	seps = map[Node][]Node{}
	var stack []*Opener
	top := func() *Opener {
		if len(stack) == 0 {
			return nil
		}
		return stack[len(stack)-1]
	}
	// closes resolves the group from the OPENER'S OWN BYTES, never from anything
	// recorded. Without it a stray closer — which the any-close fallback emits
	// unpaired — would be paired here and the two answers would differ on every
	// unbalanced file.
	closes := func(opener, closer Node) bool {
		ot, err := opener.Render()
		if err != nil {
			return false
		}
		g := lang.GroupFor(ot)
		if g == nil {
			return false
		}
		ct, err := closer.Render()
		return err == nil && slices.Contains(g.Close, ct)
	}
	for _, n := range d.Nodes() {
		switch m := n.(type) {
		case *Closer:
			// A closer records NO enclosing opener — it is paired with its own
			// instead. A stray one, which the any-close fallback emits unpaired,
			// records nothing at all and does not pop.
			if o := top(); o != nil && closes(o, m) {
				stack = stack[:len(stack)-1]
				closerOf[o], openerOf[m] = m, o
			}
		case *Opener:
			if e := top(); e != nil {
				enclosing[n] = e
			}
			stack = append(stack, m)
		case *Separator:
			if e := top(); e != nil {
				enclosing[n] = e
				seps[e] = append(seps[e], n)
			}
		default:
			if e := top(); e != nil {
				enclosing[n] = e
			}
		}
	}
	return
}

// CRC: crc-BracketContext.md | Seq: seq-pair.md#2 | R87
//
// The check that makes the index a fact rather than an assertion. The context
// derives its links from the finished array; this derives them again, from the same
// array, with code the library does not share. Over the whole corpus under every
// shipped language — including the many combinations where the language is wrong
// for the file, which is exactly where a parser misbehaves.
//
// It reads the index through the PUBLIC accessors, because what R87 promises is
// what a consumer can see.
func TestIndexAgreesWithTheIndependentDerivation(t *testing.T) {
	langs := shippedLangs()
	for path, src := range corpus(t) {
		for name, lang := range langs {
			d, ctx := parse(src, 0, lang)
			closerOf, openerOf, enclosing, seps := independentLinks(d, lang)

			for _, n := range d.Nodes() {
				if got, want := ctx.Enclosing(n), enclosing[n]; got != want {
					t.Fatalf("%s under %s: the two derivations disagree on an enclosing opener", path, name)
				}
				switch n.(type) {
				case *Opener:
					if got, want := ctx.Closer(n), closerOf[n]; got != want {
						t.Fatalf("%s under %s: the two derivations disagree on a closer", path, name)
					}
					if got, want := ctx.Separators(n), seps[n]; !slices.Equal(got, want) {
						t.Fatalf("%s under %s: the two derivations disagree on separators", path, name)
					}
				case *Closer:
					if got, want := ctx.Opener(n), openerOf[n]; got != want {
						t.Fatalf("%s under %s: the two derivations disagree on an opener", path, name)
					}
				}
			}
		}
	}
}

// CRC: crc-BracketContext.md | Seq: seq-pair.md#2 | R87
//
// The same guarantee, checked by a genuinely DIFFERENT algorithm rather than a
// second stack walk: for each node, rescan from the start of the document counting
// depth, and take the innermost opener still unclosed when the node is reached.
//
// It is O(n) per node, so it runs on a fixture rather than the corpus — and that is
// the trade worth making. The corpus check above shares an idea with the library
// even though it shares no code; this one shares neither, so it is the one that
// would catch a mistake common to both.
func TestTheIndexAgreesWithARescanPerNode(t *testing.T) {
	d, ctx := parse("a {b [c] d} e (f) g", 0, codeLang())
	ns := d.Nodes()
	for i, n := range ns {
		if _, isCloser := n.(*Closer); isCloser {
			// A closer is paired with its opener rather than enclosed by one.
			if got := ctx.Enclosing(n); got != nil {
				t.Fatalf("node %d: a closer must record no enclosing opener", i)
			}
			continue
		}
		var stack []Node
		for _, m := range ns[:i] {
			switch m.(type) {
			case *Opener:
				stack = append(stack, m)
			case *Closer:
				if len(stack) > 0 {
					stack = stack[:len(stack)-1]
				}
			}
		}
		var want Node
		if len(stack) > 0 {
			want = stack[len(stack)-1]
		}
		if got := ctx.Enclosing(n); got != want {
			s, _ := n.Render()
			t.Fatalf("node %d (%q): index says %v, a rescan says %v", i, s, got, want)
		}
	}
}

// CRC: crc-BracketContext.md | Seq: seq-pair.md#1.5 | R86
func TestStaleStampRebuildsAndFreshDoesNot(t *testing.T) {
	d, ctx := parse("a {b} c", 0, codeLang())
	opener := d.Nodes()[1]

	if ctx.Enclosing(d.Nodes()[2]) != opener {
		t.Fatalf("precondition: the text inside the group is enclosed by its opener")
	}
	before := ctx.stamp

	if err := d.Mutate(func() error { return d.Remove(d.Nodes()[0]) }); err != nil {
		t.Fatal(err)
	}
	if d.Generation() == before {
		t.Fatalf("precondition: a membership change bumps the generation")
	}
	if ctx.Enclosing(d.Nodes()[1]) != opener {
		t.Fatalf("after a rebuild the links must still be right")
	}
	if ctx.stamp != d.Generation() {
		t.Fatalf("the context must re-stamp with the generation it read")
	}
}

// CRC: crc-BracketContext.md | Seq: seq-pair.md#1.5.1 | R86
// The context wrote no guard; it inherits one by reading the generation.
func TestContextInheritsTheMutationGuard(t *testing.T) {
	d, ctx := parse("a {b} c", 0, codeLang())
	n := d.Nodes()[2]
	err := d.Mutate(func() error {
		ctx.Enclosing(n) // reads the generation, which refuses inside the window
		return nil
	})
	if err == nil {
		t.Fatalf("a freshness check inside a mutation window must refuse")
	}
	if !strings.Contains(err.Error(), "not available inside a mutation window") {
		t.Fatalf("expected the typed sentinel as an error, got %v", err)
	}
}

// CRC: crc-BracketContext.md | R85
// The reason Doc does not own this.
func TestDocumentWithNoBracketsCarriesNoLinks(t *testing.T) {
	d, ctx := parse("# A markdown heading\n\nSome prose.\n", 0, &BracketLang{})
	if len(d.Nodes()) != 1 {
		t.Fatalf("an empty table should produce one Text node, got %d", len(d.Nodes()))
	}
	if len(ctx.info) != 0 {
		t.Fatalf("a document with no brackets must carry no links")
	}
}

// CRC: crc-BracketContext.md | R80
// The context carries the language it parsed with, and hands it back.
func TestContextCarriesItsLanguage(t *testing.T) {
	lang := codeLang()
	_, ctx := parse("a {b} c", 0, lang)
	if ctx.Language() != lang {
		t.Fatalf("the context must report the table it parsed with")
	}
}

// CRC: crc-BracketContext.md | Seq: seq-pair.md#1.4 | R152, R153
//
// An opener knows its separators, in document order, and each names it back. This
// forces a REBUILD before asking, because the hazard is not that the parse gets it
// wrong — it is that the parse is the only thing that records it, which nothing
// notices until a structural edit makes the stamp stale.
func TestAnOpenerKnowsItsSeparators(t *testing.T) {
	for _, tc := range []struct {
		name, src string
		opener    string
		want      []string
	}{
		{"a for loop", "for x in a b; do echo $x; done\n", "for", []string{"in", "do"}},
		{"an if chain", "if p; then q; elif r; then s; else t; fi\n", "if",
			[]string{"then", "elif", "then", "else"}},
		{"a group with none", "{ echo hi; }\n", "{", nil},
	} {
		d, ctx := parse(tc.src, 0, &LangShell)
		// Force the independent walk to be what answers.
		_ = d.Mutate(func() error { return nil })
		ctx.rebuild()

		var opener Node
		for _, n := range d.Nodes() {
			if o, ok := n.(*Opener); ok {
				if r, _ := o.Render(); r == tc.opener {
					opener = o
					break
				}
			}
		}
		if opener == nil {
			t.Fatalf("%s: no %q opener in the parse", tc.name, tc.opener)
		}
		var got []string
		for _, sep := range ctx.Separators(opener) {
			r, _ := sep.Render()
			got = append(got, r)
			if ctx.Opener(sep) != opener {
				t.Errorf("%s: separator %q does not name its opener back", tc.name, r)
			}
		}
		if !slices.Equal(got, tc.want) {
			t.Errorf("%s: separators %v, want %v", tc.name, got, tc.want)
		}
	}
}

// CRC: crc-BracketContext.md | Seq: seq-pair.md#1.5 | R86, R152
//
// Separators refreshes like every other accessor. Found by injection rather than by
// design: TestAnOpenerKnowsItsSeparators calls rebuild itself, so removing the
// refresh from Separators left the whole suite green.
//
// A missing refresh only gives a WRONG answer once the document genuinely differs,
// which is why this removes a separator rather than making some unrelated edit: the
// parse's record still lists it, and only the rebuild the accessor is supposed to
// trigger can notice it is gone.
func TestSeparatorsRefreshesLikeEveryOtherAccessor(t *testing.T) {
	d, ctx := parse("for x in a b; do echo $x; done\n", 0, &LangShell)
	var opener, sep Node
	for _, n := range d.Nodes() {
		if o, ok := n.(*Opener); ok && opener == nil {
			opener = o
		}
		if s, ok := n.(*Separator); ok && sep == nil {
			sep = s
		}
	}
	if opener == nil || sep == nil {
		t.Fatal("expected an opener and a separator in this parse")
	}
	if got := len(ctx.Separators(opener)); got != 2 {
		t.Fatalf("before the edit: %d separators, want 2", got)
	}

	if err := d.Mutate(func() error { return d.Remove(sep) }); err != nil {
		t.Fatalf("Mutate: %v", err)
	}
	// No explicit rebuild: the accessor's own refresh is what must notice.
	if got := len(ctx.Separators(opener)); got != 1 {
		t.Errorf("after removing one separator: %d, want 1 — the accessor answered "+
			"from an index the document has moved past", got)
	}
}

// CRC: crc-BracketContext.md | R193
//
// The accessors are TYPED: a consumer never asserts a kind the context already
// knew. The stray `}` is the case the typed nil has to get right.
func TestOpenerAndCloserAreTyped(t *testing.T) {
	d, ctx := parse("a(b)c}", 0, &LangGo)
	var open *Opener
	var stray *Closer
	for _, n := range d.Nodes() {
		switch m := n.(type) {
		case *Opener:
			open = m
		case *Closer:
			stray = m // the last closer seen is the stray `}`
		}
	}
	close := ctx.Closer(open)
	if close == nil || ctx.Opener(close) != open {
		t.Fatalf("the ( ) pair does not agree through the typed accessors")
	}
	if s, _ := close.Render(); s != ")" {
		t.Errorf("closer renders %q, want %q", s, ")")
	}
	if ctx.Opener(stray) != nil {
		t.Errorf("a stray closer has an opener")
	}
}

// CRC: crc-BracketContext.md | R194, R195
//
// innerHTML / outerHTML for a group, named from either end, with the open-at-EOF
// rule. The comment group's closer IS end of input under LangGo, so both texts run
// to the end.
func TestInnerAndOuterText(t *testing.T) {
	d, ctx := parse("x = (a, [b]) // tail", 0, &LangGo)
	var paren, comment *Opener
	for _, n := range d.Nodes() {
		if o, ok := n.(*Opener); ok {
			switch s, _ := o.Render(); s {
			case "(":
				paren = o
			case "//":
				comment = o
			}
		}
	}
	cases := []struct{ name, got, want string }{
		{"inner from opener", ctx.InnerText(paren), "a, [b]"},
		{"inner from closer", ctx.InnerText(ctx.Closer(paren)), "a, [b]"},
		{"outer", ctx.OuterText(paren), "(a, [b])"},
		{"comment inner runs to EOF", ctx.InnerText(comment), " tail"},
		{"comment outer runs to EOF", ctx.OuterText(comment), "// tail"},
		{"not a marker", ctx.InnerText(d.Nodes()[0]), ""},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, c.got, c.want)
		}
	}
	d, ctx = parse("f(a", 0, &LangGo)
	open := d.Nodes()[1].(*Opener)
	if got := ctx.InnerText(open); got != "a" {
		t.Errorf("open group inner: got %q, want %q", got, "a")
	}
	if got := ctx.OuterText(open); got != "(a" {
		t.Errorf("open group outer: got %q, want %q", got, "(a")
	}
}
