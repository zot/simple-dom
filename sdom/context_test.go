// CRC: crc-BracketContext.md | R82, R83, R84, R85, R86, R87
package sdom

import (
	"fmt"
	"maps"
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
	d, ctx := Scan("a {b [c] d} e", 0, lang)
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
	d, ctx := Scan("a {b {c} d} e", 0, codeLang())
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
// The check that makes the index a fact rather than an assertion. The scan
// records links from its own recursion; rebuild derives them again from the
// finished flat array. Two independent derivations, over the whole corpus, under
// every shipped language — including the many combinations where the language is
// wrong for the file, which is exactly where a parser misbehaves.
func TestIndexAgreesWithTheIndependentDerivation(t *testing.T) {
	langs := shippedLangs()
	for path, src := range corpus(t) {
		for name, lang := range langs {
			_, ctx := Scan(src, 0, lang)
			fromScan := maps.Clone(ctx.enclosing)
			pairsFromScan := maps.Clone(ctx.closerOf)

			ctx.rebuild() // the second derivation, from the data rather than the recursion

			if len(fromScan) != len(ctx.enclosing) || len(pairsFromScan) != len(ctx.closerOf) {
				t.Fatalf("%s under %s: scan recorded %d enclosings / %d pairs; "+
					"the independent walk found %d / %d",
					path, name, len(fromScan), len(pairsFromScan),
					len(ctx.enclosing), len(ctx.closerOf))
			}
			for n, want := range fromScan {
				if got := ctx.enclosing[n]; got != want {
					t.Fatalf("%s under %s: the two derivations disagree on an enclosing opener", path, name)
				}
			}
			for o, want := range pairsFromScan {
				if got := ctx.closerOf[o]; got != want {
					t.Fatalf("%s under %s: the two derivations disagree on a pairing", path, name)
				}
			}
		}
	}
}

// CRC: crc-BracketContext.md | Seq: seq-pair.md#1.5 | R86
func TestStaleStampRebuildsAndFreshDoesNot(t *testing.T) {
	d, ctx := Scan("a {b} c", 0, codeLang())
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
	d, ctx := Scan("a {b} c", 0, codeLang())
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
	d, ctx := Scan("# A markdown heading\n\nSome prose.\n", 0, &BracketLang{})
	if len(d.Nodes()) != 1 {
		t.Fatalf("an empty table should produce one Text node, got %d", len(d.Nodes()))
	}
	if len(ctx.closerOf) != 0 || len(ctx.openerOf) != 0 || len(ctx.enclosing) != 0 {
		t.Fatalf("a document with no brackets must carry no links")
	}
}

// CRC: crc-BracketContext.md | R80
// The context carries the language it scanned with, and hands it back.
func TestContextCarriesItsLanguage(t *testing.T) {
	lang := codeLang()
	_, ctx := Scan("a {b} c", 0, lang)
	if ctx.Language() != lang {
		t.Fatalf("the context must report the table it scanned with")
	}
}
