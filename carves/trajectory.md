# Carve: the trajectory reader

> **DRAFT (2026-09-03).** A brainstorm, per Bill's method: the split below is a proposal,
> the forks are open, and only `## Decisions` is settled. Iterate here, then design.

The mini-spec tool reads and **writes** four kinds of file — `PENDING.md`, `CURRENT.md`,
`DONE.md`, and `carves/*.md` — and today does it with its own older DOM plus fourteen
regexes across two readers and a part-line lexicon. This carve builds those readers over
`sdom`, as stencils in `minispecsdom`, so an edit is a write into a field rather than a
line rewrite, and every byte the tool does not touch comes back exactly as it was.

**The shapes are not restated here.** `~/.claude/skills/mini-spec/trajectory-format.md`
is normative, and its *what the tool reads* table — file, position, what is taken — is
this carve's requirement list. What this document holds is the split, the decisions, and
what is still open.

## Status

- [x] ~~**Item 9 — the live text node.**~~ **LANDED (`3fd278e`, 2026-09-03 — `#16`.)**
- [x] ~~**Item 1 — the markdown schema.**~~ **LANDED (`f5de2b5`, 2026-09-03 — `#17`.)**
- [x] ~~**Item 2 — the part line.**~~ **LANDED (`46a8f41`, 2026-09-03 — `#18`.)**
- [x] ~~**Item 3 — the carve status block.**~~ **LANDED (`7be2e38`, 2026-09-03 — `#19`.)**
- [x] ~~**Item 4 — the pending file.**~~ **LANDED (`24cfb6f`, 2026-09-03 — `#20`.)**
- [ ] **Item 5 — the done file.** **OPEN (#21.)**
- [ ] **Item 6 — the current file.** **OPEN (not queued.)**
- [ ] **Item 7 — say what could not be read.** **OPEN (not queued.)**
- [ ] **Item 8 — cut the tool over.** **OPEN (not queued.)**

## Decisions

**DECIDED (Bill, 2026-09-03): the mini-spec schemas are built here, in this project.**
The point of the project is to support the mini-spec tool. This is the decision that
withdrew `carves/done/simple-dom.md` Item 9: generalizing the language schemas was a
small step in this direction, and building the schemas themselves eclipses it.

**DECIDED (Bill, 2026-09-03): the trajectory files come first**, ahead of requirements,
design, CRC cards, sequences and test designs. They have the most exacting format rules,
the tool already writes them, and the normative reference exists to measure a reader by.

**DECIDED (inherited from trajectory-format.md): the reader ingests by position.** What
it reads as data comes from fixed positions — a `## N.` heading at column 0, the
`Source:` line beneath it, a `- **` at column 0, the slot between the em dash and the
colon, a `- [ ]` inside `## Status`. Everything else is prose, and a stencil that binds
only those positions cannot mistake a sentence for a record. This is the reason the work
is a set of stencils and not a markdown parser with opinions.

**DECIDED (inherited): the reader reports what it could not read**, always — a count of
entry-like lines it did not recognize and of references it could not reach — because a
shape-based check is blind to whatever is outside the shape and reports clean over it.

**DECIDED (Bill, 2026-09-03): committed tests rely only on code in this repository;
the tool is a development instrument, and we wean off it before the switch.** The
mini-spec tool is being upgraded to replace its current techniques with `sdom`, so it
will not remain a reliable oracle — after the switch the two would be tested against
each other. While a part is being built, comparing the new reader against the tool's
answers over live files is a fine way to find what the reader misses; none of that
becomes a test. Acceptance for every part is: fixtures **committed here** (the
trajectory files are private in a live project, so samples are copied in, not read in
place), expectations written by hand from `trajectory-format.md`'s table, and the byte
round-trip over that corpus. No committed test shells out to `minispec`, and none reads
`~/work/mini-spec`.

**DECIDED (Bill, 2026-09-03): one schema per kind of file, and each owns the DOM for its
kind.** A pending schema, a current schema, a done schema, a carve schema. Markdown is
reused heavily between them but is **not** the schema: a markdown base exists only if the
four inherit from it — the shape `LangPython` already takes, embedding `BracketLang` and
adding what indentation needs. So a file's schema is *markdown base plus this file's
stencils and writes*, and nothing parses a pending file with a bare markdown schema.

## Item 9

`sdom` machinery, placed ahead of Item 1 because Item 1's line-head detection wants it.
**Added 2026-09-03 (Bill).**

Today the walk keeps two representations of parse state: the node array, and a pending
text run in `src[textStart:pos]` that becomes a node only at `FlushText`. The change: the
walk creates a `Text` on the first byte it declines and **extends it as `pos` advances**,
so the last node in the array is always a faithful picture of where the parse stands.

**What it deletes.** `textStart`, `FlushText`, and the flush step in `Emit` — a text run
simply ends where the next marker begins. It also closes a leak: the indent parser's
continuation check reads the pending run out of the state's unexported field today; with
a live node it reads the last node like any consumer.

**What it costs.** Nothing per byte — extending is a substring of the source and a length
bump. The extend is an in-package operation that keeps the location faithful; `SetText`
marks a node altered and must not be the path.

**What it adds to the surface.** One accessor for the last node. R158 says the surface is
`Emit` plus `NodeCount`, deliberately, so it widens by a method rather than by exporting
the slice — O2 and O21's shape.

**What stays true.** Progress is still a byte or a node and the walk's no-progress test
still reads both; the root-indent check on a count of zero still fires first, since the
text node appears only after the first byte is declined; the final array is identical, so
the round-trip and every existing test are unmoved.

**One invariant, stated at `Emit` rather than guarded:** a parser never emits a node over
bytes already absorbed into the live text. None does — each is offered every position and
never backs up — but the walk now relies on it.

**Landing obligation: reconcile the design at its source, so nothing reads as if the
pending run still exists.** The model is written down in these places, found by grep on
2026-09-03; each is rewritten or retired when the part lands, not annotated:

- `specs/parser-protocol.md` — the `ParserState` comment (*the position, the pending
  text*), `Emit`'s signature comment (*flush pending text*), and the walk description
  near line 94.
- `design/requirements.md` — **R156** (*the position, the pending text and the nodes*)
  is reworded in place; **R158** widens by the last-node accessor; a new requirement
  states the live text node and the `Emit` invariant.
- `design/crc-ParserState.md` — *Knows: where the pending text run began*; *Does:
  takes one byte as pending text; flushes pending text before any node is emitted*.
- `design/seq-collaborate.md#1.6` and the step that *takes one byte as pending text*.
- `design/test-Protocol.md` — the test *pending text is flushed before an emitted
  node* and its alarm, which inject a flush order that will no longer exist; replace
  with the live-node property (the last node is the current text and ends at `pos`)
  and an alarm against it. The alarm in `test-Indent.md` on `IndentParser.continued`
  names `st.textStart` and is re-anchored to the last-node read.
- `design/test-BracketContext.md` — the note that *`take` flushes pending text before
  emitting*.
- `sdom/indent.go` — the continuation check's comment and its read of the pending run.

Completion test, from the skill: could an agent reading only specs and design be led to
put `FlushText` back? Every line above is a place that would say yes.

## Item 1

The markdown base the four file schemas embed — never used on its own. Not markdown in
general: **the subset the four files use**, and only the structure the readers bind.

What the old readers consumed: headings (with level), list items (with depth and a
checkbox), fenced code (so a fence's contents are never read as structure), code spans
(so a backquoted `` `#7` `` is a mention, not a use), and the inline runs `**` and `~~`
that dress a part line. Links appear in the `Source:` line only.

**DECIDED (Bill, 2026-09-03): one pass, a `Parser` that wraps `IndentParser`.** Markdown
is line-structured at the top and bracketed inside a line, so the base is an `IndentLang`
— depth is a list item's indentation — with a `BracketLang` carrying the fence, the code
span, `**` and `~~` (symmetric groups, the shape a quote already has). The markdown
parser implements the three-method `Parser` interface and holds an `IndentParser`, the
way that one holds a `BracketParser`: its `Parse` delegates, and when nothing was emitted
and the position is a **line head** — scan back over spaces and tabs to a newline or the
start of the source, so no state is kept — it recognizes `## `, `- ` and a checkbox and
emits a marker node for it. A heading or list item is therefore a line-head marker like
`Indent`, holding only the bytes it matched; its extent, the rest of the line, is derived
by a context from the array, as the indent frames are. **No post-pass and no `Replace`.**
`NodeType` delegates the same way, so lookahead and transparency keep working.

**DECIDED (Bill, 2026-09-03): narrow.** The base models what Items 2–6 bind — headings,
list items, the checkbox, the fence, the code span, `**`, `~~`, one link shape — and no
more. As the parsers accumulate, commonality among them is what earns generalization;
nothing is generalized ahead of it. The seam is a `BracketLang` entry, not machinery.

**DECIDED (Bill, 2026-09-03): the base lives in `sdom/schema`**, beside Go and Lua —
markdown is a language, nothing in the base knows what a queue is, and microfts2
indexes markdown. The four file schemas in `minispecsdom` embed it.

## Item 2

The part line, as one stencil — what `partline` does today with six regexes:

    - [x] ~~**Item 1 — record and resolve.**~~ **LANDED (`4c6e974`, 2026-08-04 — `#3`.)**

Bound fields: the checkbox, the key (`Item N` or `N.M`, nothing else), the title, and
the marker's verb, attribution and queue `#N`. Glue: the opening and closing `~~`/`**`
runs, the em dash, the parentheses. The writes the tool performs today: flip the
checkbox, replace the marker (`SetMarker`), strike the title on landing.

Deviations are part of the contract, not an error path: an unkeyed line, a non-conforming
checkbox interior, a mis-cased verb are each *reported with the shape they must take*,
and the checkbox still counts. A stencil whose regex fails to match must still yield the
list item and say which rule it broke.

**DECIDED (Bill, 2026-09-03): an item node with bound values and a list of marker
stencils.** The part line is one node over the list item. Its **checkbox** and its
**key** (`Item 1`, `2.2`) are bound values; its **markers** are a list of marker stencil
nodes, each tiling `**VERB (attribution)**` and owning its contents *including* the `**`
opener and closer the base already emitted — the shape `TraceabilityComment` takes over
a comment group; everything between is interspersed text. Which bold run is which is
decided by content: the head's first bold run carries the key and title, and a later bold
run whose interior reads as `VERB (…)` is a marker, which is how a `SPLIT` line carries a
marker and prose after it. **Striking the title on landing is a structural edit**, not a
field write — wrapping the head in `~~` inserts nodes around the bold run inside a
mutation window, the way the comment splice does — **and the item node hides it behind
high-level accessors**: `IsStruck() bool` derives from the children, `Strike(bool)`
performs the edit. A consumer never touches `~~` nodes; the API is the same shape as
`Bool.Value` / `Bool.Set`, one level up (Bill, 2026-09-03).

## Item 3

**The carve schema.** It embeds the markdown base, owns a carve file's DOM, and adds the
`## Status` block: the region from that heading to the next of the same level or
higher, the part lines inside it with their depth, a subpart as an indented sibling, a
`SPLIT` parent with no checkbox. Reads: every part with key, state, queue ID, depth.
Writes: mark a part's marker by key (`pending add-item`, `finish`, `revert`).

Depends on Items 1 and 2.

## Item 4

**The pending schema.** It embeds the markdown base, owns the pending file's DOM, and
adds the entry heading `## N. **title** (skill). status.` with its
`Source:` and `Next:` lines beneath, and the sub-items a paused entry nests. Reads: the
number, the document and `#key` after `part`. Writes: mint an entry at a position
(`--next`, `--nth`, `--after`, `--last`), remove an entry, lift context into a sub-item.

## Item 5

**The done schema.** It embeds the markdown base, owns the done file's DOM, and adds the
entry `- **date — ids: title.** (commit) Part `doc#key`.` and its
body. Reads: every `#N` in the identifier slot and nothing outside it; the backquoted part
pointer. Writes: prepend an entry with its body. The companion count — entry-like lines
(`- ` at column 0) that did not read as entries — is Item 7's, fed from here.

## Item 6

**The current schema.** It embeds the markdown base, owns the current file's DOM, and
adds the one region that matters: exactly one `## Active` region, cleared or replaced as a unit; standing
sections untouched. The 2026-08-18 incident — a reset that deleted 320 lines of standing
context — is the property to guard: the write reaches the region and nothing else, and a
file with two `## Active` headings is refused rather than guessed.

## Item 7

The report: what was found and what could not be read, printed even when nothing else
is. Per file, the count of shape-like lines not recognized and of references not reached.
This is a reader obligation across Items 3–6 rather than a feature; it is a part so that
it is scheduled rather than assumed.

## Item 8

Cutting the tool over: `parser/trajectory.go`, `parser/carve.go` and `partline` in
`~/work/mini-spec/tool` re-pointed at the new readers, and the tool's own older `sdom`
retired from those paths. This is the consumer's work and lives in that repository; it is
listed here so the carve says where it ends.

**DECIDED (Bill, 2026-09-03): early comparisons, then wean, then one cut.** The old
reader is a useful instrument while Items 1–7 are being built and stops being one the
moment the tool is switched, so every comparison is throwaway and the acceptance behind
the cut is this repository's own tests. The cutover is then a consumer change with those
tests already green behind it.
