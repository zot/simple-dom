# Test Design: the declaration machinery
**Source:** crc-Declaration.md, crc-MutationWindow.md, crc-BracketContext.md

## Test: Replace keeps position and bumps the generation
**Purpose:** R125 — the one new structural verb behaves like its siblings
**Input:** a scanned document; inside a mutation window, replace one `*Text` with a
`*DeclarationName` carrying the same bytes and location
**Expected:** the render is byte-identical, the node array has the same length, the
replacement sits at the old node's index, and the structural generation advanced
**Refs:** crc-MutationWindow.md, seq-declare.md#2.2
**Code:** sdom/declaration_test.go
**Alarm:** 1
**Fire alarm:** Make `Replace` leave `dirty` unset. Red: the generation does not
advance, so a stamped index keeps a stale stamp and never rebuilds. Nothing about
the bytes changes — the render, the tiling and the round-trip all stay green — which
is the whole class of failure a stamped index has.
**Inject:** sdom/mutate.go:Doc.Replace
**Pulled:** 2026-08-31 — rang, and alone. Dropping `d.dirty = true` failed only
`TestReplaceKeepsPositionAndBumpsGeneration`, on exactly its own message: *generation
did not advance; a stamped index would never rebuild.* Every byte-level check stayed
green, which is the whole class of failure a stamped index has.

## Test: a declaration pass changes no bytes and no markers
**Purpose:** R124 — the additive property, structurally rather than by convention
**Input:** every Go file in `sdom/`, scanned twice: once plain, once with a
declaration pass run over it
**Expected:** the two renders are byte-identical, and the **flattened** node arrays
are equal kind-for-kind except where a `Text` became a `DeclarationType` or
`DeclarationName`; the count of `Opener`, `Closer` and `Separator` is unchanged
**Refs:** crc-Declaration.md, seq-declare.md#2.6
**Code:** sdom/schema/declaration_test.go
**Alarm:** 2
**Fire alarm:** Drop the top-level `Enclosing` guard in `goNext`, so commented-out
code and string contents are scanned too. Red: the count over-reports — 17 in
`doc.go` where 15 exist, 24 in `loc_test.go` where 19 do. Bytes and marker counts
stay identical, so only the count assertion sees it.
**Inject:** sdom/schema/golang.go:goNext
**Pulled:** 2026-08-31 — rang, over-counting across nine files at once.

*This injection replaced the one first written here, and why is worth keeping.* The
original was ~~build the `DeclarationName` with `NewText` over the trimmed string so
the surrounding spaces are dropped; the renders then differ~~ — and it is
**unbuildable**. Removing `carve`'s split also removes the right remainder a caller
threads when a keyword and its name share one node, so the suite SIGSEGVs before any
byte comparison runs: the absence of a result, not a red test. Re-aimed at the call
site, the suite runs and **the bytes stay identical**.

That is a fact about the design, not a gap in the test. The pass only `Split`s, which
preserves bytes, `Replace`s a node with one built from the same text, and never
removes a node — so **byte loss is unrepresentable**, which is what `R124` claims. The
same argument covers the marker counts. Both of those halves are structural
guarantees; the **count** is the half that can fail, and it is what this alarm now
injects against. It is also the half that caught the real defect, twice.

## Test: a name is sliced out of the middle, not taken whole
**Purpose:** R148 — the failure that renders identically and is still wrong
**Input:** `func /* a */ foo /* b */ (x) {`, where the name arrives as `Text " foo "`
**Expected:** the `DeclarationName` renders exactly `foo`; its left and right
siblings render `" "` each; the three together render the original node
**Refs:** crc-Declaration.md, seq-declare.md#2.3
**Code:** sdom/schema/declaration_test.go
**Alarm:** 3
**Fire alarm:** At the call site, carve the **whole node** for the name instead of
its identifier bounds — `carve(d, target, 0, len(tb), newName)`. Red: the name
renders with its surrounding spaces. The document round-trip stays green, because no
byte moved — only the ownership did, and only an assertion about the node's own
bytes can see it.
**Inject:** sdom/schema/golang.go:Go
**Pulled:** 2026-08-31 — rang: `func[ Index]` against `func[Index]`, the leading
space exactly as predicted, and the byte round-trip green throughout.

*The site was corrected to get there.* It first named `sdom/schema/schema.go:carve`,
and editing `carve` itself is too broad — it breaks the keyword carve, whose
remainder the name carve depends on, so the injection SIGSEGVs instead of asserting.
The `Inject:` field must name what the injection **edits**, and a site wider than the
defect makes an alarm unbuildable rather than strict.

## Test: a stale declaration accessor refuses
**Purpose:** R128 — the one index this context cannot rebuild
**Input:** a document with declaration links; make a structural edit, then ask a
`DeclarationType` for its names
**Expected:** a refusal naming the repair, not an empty slice and not the old map
**Refs:** crc-BracketContext.md, seq-declare.md#2.5
**Code:** sdom/declaration_test.go
**Alarm:** 4
**Fire alarm:** Return the map without comparing stamps. Red: the accessor answers
from a document that has changed underneath it. An empty answer would be worse than
a wrong one — it reads identically to *this keyword declares nothing* — which is
exactly why `IndexOf` refuses rather than returning `-1`.
**Inject:** sdom/context.go:BracketContext.Declarations
**Pulled:** 2026-08-31 — rang, and alone. Dropping the stamp comparison failed only
`TestStaleDeclarationsRefuse`, with *got <nil>, want ErrDeclarationsStale* — the
accessor answering happily from a document that had changed underneath it.

## Test: declarations survive a rebuild that follows them
**Purpose:** R126, R128 — the regression consolidating the index introduced
**Input:** mutate inside a window, then `SetDeclarations`, then call any accessor
that refreshes — the exact order every schema pass uses
**Expected:** the names are still there, and no refusal
**Refs:** crc-BracketContext.md, seq-declare.md#2.5
**Code:** sdom/declaration_test.go
**Alarm:** 6
**Fire alarm:** Remove the `refresh()` at the top of `SetDeclarations`. Red: the
names come back **empty with a nil error** — `rebuild` recreates the index the
declaration links now live in, then sets `stamp` to the generation `declStamp`
already held, so the loss is invisible to the staleness check. That empty answer is
the one the accessor's own comment promises to refuse, because it reads as *this
keyword declares nothing*. Note `TestStaleDeclarationsRefuse` stays **green** under
this injection: it edits after recording, so the stamps differ and it refuses
correctly. The hole is a rebuild firing while they agree.
**Inject:** sdom/context.go:BracketContext.SetDeclarations

## Test: SetDeclarations replaces rather than merges
**Purpose:** R126 — the second regression, and the one no caller would have hit yet
**Input:** two `SetDeclarations` calls, the second omitting the first's keyword
**Expected:** the omitted keyword keeps nothing
**Refs:** crc-BracketContext.md
**Code:** sdom/declaration_test.go
**Alarm:** 7
**Fire alarm:** Drop the clearing loop in `SetDeclarations`. Red: the omitted
keyword keeps its old names. Nothing fails today without this test — no caller runs
a second pass over one context — which is why it is written down rather than left to
the first consumer that does.
**Inject:** sdom/context.go:BracketContext.SetDeclarations
