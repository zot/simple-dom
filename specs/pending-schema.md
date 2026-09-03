# The pending schema

The second file schema: it embeds the markdown base, **owns the pending file's DOM**, and
adds the queue entry. An entry is a `## N.` heading and everything beneath it to the next
heading of level 2 or higher, or a `---` rule outside a fence.

```go
type Pending struct { /* the document, its contexts, the entries */ }

func ParsePending(src string) *Pending
func (p *Pending) Doc() *sdom.Doc
func (p *Pending) Render() (string, error)

func (p *Pending) Entries() []*Entry      // in file order — the top entry is active
func (p *Pending) Entry(id int) *Entry
func (p *Pending) MaxID() int              // 0 when empty
func (p *Pending) Unread() []string        // level-2 headings that are not entries

func (p *Pending) Place(e EntryText, pos int) error   // 1-based among entries; len+1 appends
func (p *Pending) After(id int) (int, error)          // the position that follows a live entry
func (p *Pending) Remove(id int) error

type Entry struct {
    ID                int
    Title, Skill, Status string   // derived from the heading line
    SourceDoc, PartKey string     // derived from the Source: line
    Next              string      // derived from the Next: line, "" when absent
}

type EntryText struct { ID int; Title, Skill, Status, SourceDoc, PartKey, Next string }
```

**An entry is a view, not a node.** Unlike a part line it has no field a tool writes into:
the tool mints whole entries and removes whole entries, and reads the number, the Source
pointer and the Next line. So the schema keeps each entry as the run of flat nodes from
its heading to the end of its region, derives its values from that run's rendered bytes at
the positions `trajectory-format.md` names — `## N.` at column 0, the `Source:` line
beneath, the `#key` after `part` — and never re-cuts them. Everything in the run stays
exactly as the base parsed it.

**A region ends at the next heading of level 2 or higher, or at a `---` line outside a
fence.** A fence in a body is that entry's: the base emits no heading inside one, so a
quoted `## 5.` cannot end an entry, by construction.

**Placement is a node placement, never a byte splice.** `Place(e, pos)` renders the
canonical entry — heading line, `Source:` line, `Next:` line when given, a trailing blank
line — as one synthetic text and inserts it before the entry at `pos`, or at the end when
`pos` is one past the last. A position outside `1 … len+1` is **refused, not clamped**:
a clamp silently reinterprets an instruction the caller was specific about. `After(id)`
resolves the position following a live entry and refuses an unknown one. `--next`, which
consults the current file, is resolved by the caller before it reaches here.

**Removal drops the run.** `Remove(id)` takes the entry's nodes out of the array — splitting
the last text at the region's end when the next entry's bytes share it — inside one
mutation window.

**What it could not read is reported.** A level-2 heading whose text does not open `N.`
is not an entry; it is listed by `Unread`, since a shape-based reader is blind to what is
outside its shape and must say so.

**After each write the document is re-read from its bytes** and the view rebuilt with it.
A placed entry is one synthetic text until it is parsed, and nothing holds a node of this
document across a write, so re-reading costs nothing a consumer can see.
