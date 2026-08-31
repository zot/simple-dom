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
**Fire alarm:** Make the pass build its `DeclarationName` with `NewText` over the
trimmed string instead of splitting the node, so the surrounding spaces are
dropped. Red: the renders differ by exactly the skipped whitespace. Note that the
*marker counts* stay equal, so only the byte comparison sees it — which is why both
halves of this test are here.
**Inject:** sdom/schema/schema.go:carve

## Test: a name is sliced out of the middle, not taken whole
**Purpose:** R148 — the failure that renders identically and is still wrong
**Input:** `func /* a */ foo /* b */ (x) {`, where the name arrives as `Text " foo "`
**Expected:** the `DeclarationName` renders exactly `foo`; its left and right
siblings render `" "` each; the three together render the original node
**Refs:** crc-Declaration.md, seq-declare.md#2.3
**Code:** sdom/schema/declaration_test.go
**Alarm:** 3
**Fire alarm:** Re-type the whole node instead of splitting it. Red: the name
renders `" foo "`. The document round-trip stays green, because no byte moved — only
the ownership did, and only an assertion about the node's own bytes can see it.
**Inject:** sdom/schema/schema.go:carve

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
