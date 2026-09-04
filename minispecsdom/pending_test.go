package minispecsdom

import (
	"os"
	"slices"
	"strings"
	"testing"
)

func pendingFixture(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("testdata/pending-sample.md")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func ids(p *Pending) []int {
	var out []int
	for _, e := range p.Entries() {
		out = append(out, e.ID)
	}
	return out
}

// CRC: crc-Pending.md | Seq: seq-pending.md#1 | R258, R259, R260, R264
func TestTheFixturesEntriesReadBack(t *testing.T) {
	src := pendingFixture(t)
	p := ParsePending(src)
	if r, _ := p.Render(); r != src {
		t.Fatal("render differs from source")
	}
	if got := ids(p); !slices.Equal(got, []int{8, 12, 3}) {
		t.Fatalf("ids %v, want [8 12 3]", got)
	}
	e := p.Entry(8)
	if e.Title != "The reader" || e.Skill != "mini-spec" || e.Status != "Design settled; ready to build" ||
		e.SourceDoc != "carves/x.md" || e.PartKey != "7" || e.Next != "write the spec." {
		t.Errorf("entry 8: %+v", *e)
	}
	if e := p.Entry(12); e.Skill != "" || e.PartKey != "2.1" || e.Next != "" || e.Status != "Parked with context" {
		t.Errorf("entry 12: %+v", *e)
	}
	if p.MaxID() != 12 {
		t.Errorf("MaxID %d", p.MaxID())
	}
	if u := p.Unread(); len(u) != 1 || u[0] != (Unread{25, "Notes"}) {
		t.Errorf("unread %+v", u)
	}
	// R283: every entry knows its line, 1-based.
	if p.Entry(8).Line() != 8 || p.Entry(12).Line() != 12 || p.Entry(3).Line() != 19 {
		t.Errorf("lines %d %d %d", p.Entry(8).Line(), p.Entry(12).Line(), p.Entry(3).Line())
	}
	// Entry 12's region holds its ### sub-item; entry 3's holds its fence.
	if !strings.Contains(runText(p.Entry(12)), "### Parked context") {
		t.Errorf("the sub-item is not inside entry 12's region")
	}
	if !strings.Contains(runText(p.Entry(3)), "## 9. **not an entry**") {
		t.Errorf("the fenced heading is not inside entry 3's region")
	}
}

// runText renders an entry's region: its run, cut where a rule ends it.
func runText(e *Entry) string {
	var b strings.Builder
	for i, n := range e.run {
		s, _ := n.Render()
		if i == len(e.run)-1 && e.tail != nil {
			s = s[:e.cut]
		}
		b.WriteString(s)
	}
	return b.String()
}

// CRC: crc-Pending.md | Seq: seq-pending.md#2 | R261, R262, R265
func TestPlaceByPositionRefusedNotClamped(t *testing.T) {
	p := ParsePending(pendingFixture(t))
	e := EntryText{ID: 20, Title: "New", Status: "Fresh.", SourceDoc: "carves/z.md", PartKey: "1"}
	if err := p.Place(e, 1); err != nil {
		t.Fatal(err)
	}
	if got := ids(p); !slices.Equal(got, []int{20, 8, 12, 3}) {
		t.Fatalf("after Place at 1: %v, want [20 8 12 3]", got)
	}
	r, _ := p.Render()
	if !strings.Contains(r, "---\n\n## 20. **New**. Fresh.\n   Source: [carves/z.md](carves/z.md), part `#1`.\n\n## 8. ") {
		t.Errorf("placed at 1:\n%s", r)
	}
	pos, err := p.After(12)
	if err != nil || pos != 4 {
		t.Fatalf("After(12) = %d, %v", pos, err)
	}
	if err := p.Place(EntryText{ID: 21, Title: "After twelve", Status: "S.", SourceDoc: "d", PartKey: "2"}, pos); err != nil {
		t.Fatal(err)
	}
	if err := p.Place(EntryText{ID: 22, Title: "Last", Status: "S.", SourceDoc: "d", PartKey: "3"}, len(p.Entries())+1); err != nil {
		t.Fatal(err)
	}
	if got := ids(p); !slices.Equal(got, []int{20, 8, 12, 21, 3, 22}) {
		t.Fatalf("ids %v, want [20 8 12 21 3 22]", got)
	}
	r, _ = p.Render()
	if !strings.Contains(r, "kept here until resumed.\n\n## 21. **After twelve**") || !strings.HasSuffix(r, "Not an entry either.\n\n## 22. **Last**. S.\n   Source: [d](d), part `#3`.\n\n") {
		t.Errorf("after and last:\n%s", r)
	}
	before, _ := p.Render()
	if err := p.Place(e, 0); err == nil {
		t.Errorf("position 0 accepted")
	}
	if err := p.Place(e, len(p.Entries())+2); err == nil {
		t.Errorf("position len+2 accepted")
	}
	if after, _ := p.Render(); after != before {
		t.Errorf("a refused Place changed the document")
	}
	if _, err := p.After(99); err == nil {
		t.Errorf("After(99) resolved")
	}
}

// CRC: crc-Pending.md | Seq: seq-pending.md#3 | R263
func TestRemoveDropsExactlyTheRun(t *testing.T) {
	src := pendingFixture(t)
	p := ParsePending(src)
	if err := p.Remove(12); err != nil {
		t.Fatal(err)
	}
	i := strings.Index(src, "## 12.")
	j := strings.Index(src, "## 3.")
	want := src[:i] + src[j:]
	if r, _ := p.Render(); r != want {
		t.Errorf("after Remove(12):\n%s\nwant:\n%s", r, want)
	}
	if got := ids(p); !slices.Equal(got, []int{8, 3}) {
		t.Errorf("ids %v, want [8 3]", got)
	}
	if err := p.Remove(99); err == nil {
		t.Errorf("Remove(99) accepted")
	}
}

// CRC: crc-Pending.md | Seq: seq-pending.md#1.3 | R259, R263
//
// Found by injecting past the alarm list: the fixture's only rule precedes its
// entries, so nothing proved a `---` after an entry ends its region.
func TestARuleEndsARegion(t *testing.T) {
	src := "# Pending\n\n---\n\n## 4. **Only**. S.\n   Source: [d](d), part `#1`.\n\n---\n\nTrailing prose after the rule.\n"
	p := ParsePending(src)
	e := p.Entry(4)
	if e == nil || strings.Contains(runText(e), "Trailing prose") {
		t.Fatalf("the region ran past the rule: %q", runText(e))
	}
	if err := p.Remove(4); err != nil {
		t.Fatal(err)
	}
	if r, _ := p.Render(); r != "# Pending\n\n---\n\n---\n\nTrailing prose after the rule.\n" {
		t.Errorf("after Remove:\n%q", r)
	}
}
