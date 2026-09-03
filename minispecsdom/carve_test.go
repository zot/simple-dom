package minispecsdom

import (
	"slices"
	"strings"
	"testing"
)

func keysOf(ps []*Part) string {
	var ks []string
	for _, p := range ps {
		ks = append(ks, p.Key())
	}
	return strings.Join(ks, ",")
}

// CRC: crc-Carve.md | Seq: seq-carve.md#1 | R248, R249, R250, R251, R252, R256
func TestTheFixturesStatusBlockReadsAsParts(t *testing.T) {
	src := fixture(t)
	c := ParseCarve(src)
	if !c.HasStatus() {
		t.Fatal("no status block found")
	}
	if got := keysOf(c.Parts()); got != "Item 1,2.1,2.2,Item 4" {
		t.Fatalf("parts %q", got)
	}
	depths, parents := []int{}, []string{}
	for _, p := range c.Parts() {
		depths = append(depths, p.Depth)
		if p.Parent == nil {
			parents = append(parents, "-")
		} else {
			parents = append(parents, p.Parent.Key())
		}
	}
	if !slices.Equal(depths, []int{0, 2, 2, 0}) {
		t.Errorf("depths %v, want [0 2 2 0]", depths)
	}
	if got := strings.Join(parents, ","); got != "-,Item 1,Item 1,-" {
		t.Errorf("parents %q", got)
	}
	if len(c.Stateless()) != 1 || c.Stateless()[0].Key() != "Item 2" {
		t.Errorf("stateless %d", len(c.Stateless()))
	}
	if r, _ := c.Render(); r != src {
		t.Errorf("render differs")
	}
	// A bullet outside the region is not a part; a level-3 heading does not end the
	// region, a level-2 one does.
	outside := ParseCarve("## Status\n\n- [ ] **Item 1 — a.** **OPEN (not queued.)**\n\n### Sub\n\n- [ ] **Item 2 — still inside.**\n\n## Notes\n\n- [ ] **Item 9 — not a part.**\n")
	if got := keysOf(outside.Parts()); got != "Item 1,Item 2" {
		t.Errorf("a bullet outside the region became a part: %q", got)
	}
}

// CRC: crc-Carve.md | Seq: seq-carve.md#1.2 | R249, R250
func TestNoStatusBlockAndAFencedOne(t *testing.T) {
	for _, src := range []string{"# Carve\n\nprose\n\n## Decisions\n\n- [ ] **Item 1 — a.**\n", "# Carve\n\n```\n## Status\n\n- [ ] **Item 1 — a.**\n```\n"} {
		c := ParseCarve(src)
		if c.HasStatus() || len(c.Parts()) != 0 {
			t.Errorf("%q: status=%v parts=%d", src, c.HasStatus(), len(c.Parts()))
		}
	}
}

// CRC: crc-Carve.md | Seq: seq-carve.md#2.3 | R253, R254
func TestSetMarkerFollowsTheToolsRule(t *testing.T) {
	c := ParseCarve(fixture(t))
	if err := c.SetMarker("Item 4", "OPEN", "#9."); err != nil {
		t.Fatal(err)
	}
	r, _ := c.Render()
	if !strings.Contains(r, "- [ ] **Item 4 — fail fast.** **OPEN (#9.)**\n") {
		t.Errorf("Item 4 after SetMarker:\n%s", r)
	}
	src := "## Status\n\n- [ ] **Item 1 — a.** **NOT VERIFIED.** **OPEN (#8.)**\n- [ ] **Item 2 — b.**\n- [ ] **Item 3 — c.** **OPEN (#4.)** **OPEN (#5.)**\n"
	c = ParseCarve(src)
	if err := c.SetMarker("Item 1", "LANDED", "x"); err != nil {
		t.Fatal(err)
	}
	if err := c.SetMarker("Item 2", "DEFERRED", "Bill, 2026-09-03"); err != nil {
		t.Fatal(err)
	}
	// Two transients on one line: the first is replaced, the second removed.
	if err := c.SetMarker("Item 3", "OPEN", "#6."); err != nil {
		t.Fatal(err)
	}
	if err := c.SetMarker("Item 7", "OPEN", "#1."); err == nil {
		t.Errorf("an unknown key was accepted")
	}
	want := "## Status\n\n- [ ] **Item 1 — a.** **NOT VERIFIED.** **LANDED (x)**\n- [ ] **Item 2 — b.** **DEFERRED (Bill, 2026-09-03)**\n- [ ] **Item 3 — c.** **OPEN (#6.)**\n"
	if r, _ := c.Render(); r != want {
		t.Errorf("got:\n%s\nwant:\n%s", r, want)
	}
}

// CRC: crc-Carve.md | Seq: seq-carve.md#2 | R255
func TestLandIsThreeMarkingsInOneAct(t *testing.T) {
	c := ParseCarve(fixture(t))
	if err := c.Land("2.2", "`abc`, 2026-09-03 — `#8`."); err != nil {
		t.Fatal(err)
	}
	r, _ := c.Render()
	if !strings.Contains(r, "  - [x] ~~**2.2 — the config move.**~~ **LANDED (`abc`, 2026-09-03 — `#8`.)**\n") {
		t.Errorf("after Land:\n%s", r)
	}
}
