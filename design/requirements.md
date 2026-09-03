# Requirements

## Feature: node protocol
**Source:** specs/node-protocol.md

- **R1:** `sdom` parses a source into a flat array of nodes held in document order.
- **R2:** Bytes the parse does not model are preserved exactly and re-emitted unchanged.
- **R3:** (inferred) The node array tiles the document: spans are contiguous and half-open, the first begins at offset 0 and the last ends at the end of the source.
- **R4:** Nesting is expressed as links owned by the layer that needs them, never as node children.
- **R5:** The package is `sdom`, located in `sdom/`, in Go module `github.com/zot/simple-dom`.
- **R6:** `Node` is an interface declaring exactly `Kids() []Node`, `Location() Loc`, `Render() (string, error)` and `Equals(Node) bool`.
- **R7:** `Kids` returns the node's children in document order; a leaf returns none.
- **R8:** `Render` returns the bytes the node stands for at the time of the call.
- **R9:** `Render` is given neither the document nor any other context.
- **R10:** `Equals` never compares `Location`.
- **R11:** `Parse` is not a method on `Node`; only concrete kinds that genuinely parse from text declare it.
- **R12:** Each schema's parse context is a concrete type rather than an interface.
- **R13:** Every concrete node kind declares its own `Equals`, whose minimum body asserts the argument to that kind and delegates to the embedded `Compound`.
- **R14:** A kind holding state its children do not carry compares that state in its `Equals`; a kind whose state is derived from its children compares nothing beyond them.
- **R15:** `Text` is a leaf node that holds bytes and has no children.
- **R16:** `Compound` is a node whose children tile its span, which renders by concatenating its children's renders and performs no parsing.
- **R17:** `Compound` is embedded by every compound defined in the layers above `sdom`, which supply only how their own children are computed.
- **R18:** A bracket group is never a node; no node kind may model one or be named one.
- **R19:** Compound nodes exist only for stenciling — a region a tool reads or writes as a value.

## Feature: location
**Source:** specs/location.md

- **R20:** A node's location separates provenance — where it was read from — from faithfulness — whether it still renders those bytes.
- **R21:** `Loc.Offset()` returns the source position the node was read from, or −1 when there is no provenance.
- **R22:** `Loc.Length()` returns the node's current rendered extent in bytes.
- **R23:** `Loc.Altered()` reports that the node was read from its offset and no longer renders the bytes there.
- **R24:** `Loc.Faithful()` is true when the node has provenance and is not altered.
- **R25:** A faithful node renders exactly the source span at its offset.
- **R26:** The zero value of a location means "no provenance", and never "offset 0".
- **R27:** A node that has been changed keeps its offset.
- **R28:** A compound is altered if any of its children is, computed on read rather than stamped at edit time and propagated upward. *(Provisional — see gap O1. The implementation also requires the children's spans to run contiguously from the compound's own offset; closing O1 rewrites this requirement and the spec sentence behind it.)*
- **R29:** For an altered node the offset is historical while the length is current, so the pair names a span the node never owned; a consumer uses the offset alone unless the node is faithful.
- **R30:** `Split` divides one node's span into two, each keeping the provenance of the part of the source it now covers.
- **R31:** `Merge` joins two adjacent nodes into one.
- **R32:** `Split` and `Merge` are `Doc` methods, because both change node membership.
- **R33:** `Merge` checks adjacency when both operands are faithful, from their two locations alone at the call site, and does not check it otherwise.
- **R34:** A merged node is faithful only if both operands are faithful.
- **R35:** A merged node takes the first operand's offset when that operand has provenance, and the second operand's offset otherwise.
- **R36:** A split followed by a merge of the same pair returns the original span with its provenance intact.
- **R88:** A node's offset is relative to the source of the document it belongs to; a document's base offset never enters a location.
- **R89:** A location carries the parse it came from, as an `Origin`.
- **R90:** `Origin` is a concrete type carrying a name, rather than an interface or an empty marker.
- **R91:** An origin is minted once per parse and shared by every node that parse produces, so two parses of the same source yield two origins.
- **R92:** Setting a location's origin is a chained operation; the constructor taking an offset and a length is unchanged.
- **R93:** A nil origin is unknown rather than different, and is compatible with any other origin.
- **~~R94:~~** (Retired T2 — see R118) Merging two locations whose origins are both present and different panics.
- **~~R95:~~** (Retired T3 — no replacement) That panic is not the mutation-window sentinel, so `Mutate` re-raises it rather than converting it to an error.

## Feature: document
**Source:** specs/document.md

- **R37:** A `Doc` holds the source bytes, a base offset, the flat document-order node array, an open `data` slot, and two derived indices.
- **R38:** A `Doc`'s base offset is its own position within an outer document. *(See R88, which states the other half: node offsets are relative to the document's own source and never include the base. Tracked as gap O9 until the code matches.)*
- **R39:** A `Doc`'s derived indices cover its own node array and nothing else: one from node to position, one over lines.
- **R40:** A `Doc` holds no schema-specific state; any other derived state is owned by the layer that needs it.
- **R41:** `Prev` and `Next` navigate by node rather than by position.
- **R42:** A `Doc` exposes a monotonic structural generation.
- **R43:** The structural generation is bumped whenever node membership changes.
- **R44:** A content edit does not bump the structural generation.
- **R45:** A layer holding state derived from document structure stamps itself with the generation it was built against and rebuilds when that stamp is stale.
- **R46:** A `Doc` keeps no registry of derived indices and issues no invalidation callbacks.
- **R47:** `Mutate` brackets a set of edits on the document.
- **R48:** Edits inside a mutation window are applied directly; nothing is queued and nothing is deferred.
- **R49:** Inside a mutation window `Prev`, `Next`, the position lookup, the line lookup and reading the structural generation all refuse, by panicking with a typed sentinel.
- **R50:** `Mutate` converts that typed sentinel into an error.
- **R51:** A panic that is not the typed sentinel is re-raised unchanged.
- **R52:** The derived indices rebuild once, at the exit of a mutation window.
- **R53:** `Mutate` saves and restores its window state rather than counting it, so a nested call is a pass-through.
- **R54:** A mutation window provides no rollback.
- **R55:** An error or a panic escaping the mutation function poisons the document; there is no reset, and recovery is to re-parse.
- **R56:** `sdom` provides no operation log, queued edit plan, transaction or undo.
- **R118:** Every node in a document that names a parse names the same one; building a document from nodes of two different parses panics. A node with no origin makes no claim and is compatible with any document.
- **R119:** A document's nodes must describe its source. Beyond the shared-parse check this is not verified, because the guard would cost more than the failure, which is loud in every other test.
- **R120:** When both operands are faithful, a merged node's text is a slice of the document's source rather than a separate copy of it; when either is altered its bytes are not in the source, so a new string is unavoidable. The *values* are equal either way — what differs, and what is asserted, is whether the result shares the source's storage. *(See gap O12: whether the fast path also avoids building a string it discards is a separate claim, and an unmeasured one.)*

## Feature: parser protocol
**Source:** specs/parser-protocol.md

- **R155:** One source is walked once, and several parsers may contribute nodes to that single
  pass rather than a later layer splitting and re-carving the array.
- **R156:** `ParserState` owns the walk — the position and the nodes emitted so far, the last of
  which is always current — and exposes `Src`, `Pos`, `SetPos`, `Advance`, `At`, `Emit`,
  `NodeCount` and `Last`.
- **R157:** `ParserState` mints and owns the parse's `Origin`, so a document produced by several
  collaborating parsers carries exactly one.
- **R158:** `ParserState` exposes `Emit`, `NodeCount` and `Last` rather than the emitted node slice,
  so no live array is handed to a parser.
- **R224:** The array is the only parse state: the first byte nobody recognizes creates a `Text` and
  every further byte consumed as text extends it, so the last node is always current; `Advance`
  consumes and extends, `SetPos` only moves.
- **R225:** `Emit` ends the live text run and does not shrink it; a parser never emits a node over
  bytes already in the run — stated, not guarded.
- **R159:** A `Parser` either recognizes something at the head of the input and emits it, or does
  nothing at all.
- **R160:** `Parser.NodeType` reports the kind of the node `Parse` would emit at this position
  without emitting it, and reports separately that it would emit nothing — the two being different
  claims.
- **R161:** `Parse(src, base, parser)` returns only the document; each parser owns its own context,
  which a caller reads from the parser it constructed.
- **R162:** `ParserState` holds one parser, which may delegate to others. Composition and
  precedence are the parser's own business and the walk arbitrates nothing.
- **R163:** Each position is offered to the parser exactly once.
- **R164:** A position counts as unrecognized only when **neither** the position **nor** the
  emitted-node count changed, and the loop then takes one byte as text. Position alone misses a
  zero-length node; node count alone misses a parser that advances without emitting.
- **R165:** A parser may recognize one thing and return, leaving the loop to offer the next
  position, or take the loop and recurse until its own terminator.
- **R166:** A parser registered only in the outermost loop is offered no position inside a group,
  which is what makes suppression structural rather than a check.
- **R192:** `Parser.Done` is called once after the document is built, which is where a parser
  binds a derived index to it — nothing earlier can, the nodes being emitted first.
- **R167:** Progress is a byte or a node, so the loop terminates. A parser that emits at one
  position without ever advancing is a parser bug and is stated rather than guarded.

## Feature: bracket parser
**Source:** specs/bracket-parser.md

- **R57:** The bracket parser parses a source into a flat, document-order stream of `sdom` nodes.
- **R58:** The parser is table-driven: supporting a new language means adding a table entry, not code.
- **R59:** Strings and comments are expressed as bracket groups rather than as special cases.
- **R60:** `BracketLang` carries the language's bracket groups and no other language configuration.
- **R61:** `BracketGroup` declares `Open`, `Separators`, `Close`, `Escape`, `AllowedInner` and `AllowedParent`.
- **R62:** There is no comment configuration: a line comment is a group closing on a newline, and a block comment a group closing on its terminator.
- **R63:** A block comment nests only when its own opener is listed in its `AllowedInner`.
- **R64:** `AllowedInner` nil means code mode — every group's openers are recognized inside the group.
- **R65:** `AllowedInner` non-nil, including empty, means parse-restricted — only the group's own `Close`, its `Escape`, and the listed openers are recognized, and every other byte is literal.
- **R66:** `AllowedParent` nil means the group is recognized in any context; non-nil means it is recognized only while parsing inside one of the listed openers.
- **R67:** nil and an empty slice are semantically distinct in both `AllowedInner` and `AllowedParent`.
- **R68:** `BracketLang` carries no indent parameters and no flag enabling indentation; a language needing indent scope is described by a type that embeds `BracketLang`, and the type is the flag.
- **~~R69:~~** (Retired T1 — see R121) The package exports the language tables `LangGo`, `LangShell`, `LangPascal` and `LangJavaScript`, chosen so that every field of `BracketGroup` is exercised by at least one of them.
- **R70:** Language tables are Go values; the package ships no config-file format or loader for them.
- **R71:** A marker whose first byte is a word character is recognized only at a word boundary — neither preceded nor followed by a word character.
- **R72:** A group's separators are recognized only while that group is the one currently open.
- **R73:** When no other marker matches, any code-mode group's closer is recognized, so a stray closer lands as a bracket rather than derailing the parse.
- **R74:** The parse always consumes at least one byte.
- **R75:** A group left open at end of input closes there, and no bytes are dropped.
- **R76:** Whitespace is not a node of its own: it folds into text, so a text run is everything between two recognized markers.
- **R77:** The emitted stream is flat and in document order — an opener, everything between it and its closer, and the closer are siblings in the array.
- **R78:** The parser adds the node kinds `Opener`, `Closer` and `Separator`; every other byte becomes `Text`.
- **R79:** A marker node holds the bytes it matched and no reference to its bracket group; the active group comes from the parse context.
- **R80:** The parser's parse context is a concrete type, carrying the language during the parse.
- **R81:** The parse context outlives the parse and owns the bracket pairing links.
- **R82:** An opener knows its closer and its enclosing opener.
- **R83:** A closer knows its opener.
- **R84:** Any node that is not a bracket marker knows its enclosing opener.
- **R85:** The pairing links are owned by the parse context and never by `Doc`.
- **R86:** The pairing links are a derived index: the context stamps itself with the document's structural generation and rebuilds when that stamp is stale.
- **R87:** Every bracket link is derivable from the flat array alone: a consumer walking it with
  its own stack reaches the same closer, opener, enclosing opener and separators the context
  reports. The index is a convenience over structure the array already carries, never a fact
  only the index holds.
- **R121:** The package exports the language tables `LangGo`, `LangShell`, `LangPascal`,
  `LangJavaScript`, `LangTypeScript` and `LangLua`, chosen both so that every field of
  `BracketGroup` is exercised by at least one of them and so that the languages mini-spec reads
  are covered.

- **R152:** An opener knows the separators belonging to its group.
- **R153:** A separator knows its opener.
- **R154:** How the parse context stores its links is not part of its contract: what it owes is
  the answers, and whether it keeps one map or several is its own business.

- **R168:** `BracketGroup` carries an uninterpreted `Kind` label, which the bracket parser never
  reads and which leaves a group's shape and the parsing rules unchanged.
- **R169:** `LangPython` is shipped, and is the first `IndentLang`.
- **R170:** Only Python's `f`-prefixed string forms get groups of their own, parse-restricted with
  `{` as the one escape hatch; other prefixes need none, since the prefix falls through as text
  and the quote that follows opens the ordinary group.
- **R171:** Every Python string group carries the same escape, because a backslash escapes the
  closing quote even in a raw string.
- **R172:** Python's `f` groups precede the plain forms, longest quote form first, so `f"""` is
  matched before `f"`.
- **R173:** Python's f-string interpolation names the language's own code brace in `AllowedInner`
  and needs no group of its own, so the inside of `{…}` is full code mode.
- **R193:** `Opener` and `Closer` return the typed marker kinds, `*Opener` and `*Closer`, and nil
  for an unmatched marker.
- **R194:** `InnerText(n)` returns the bytes between a group's opener and its closer, where `n` is
  either; a group left open at end of input runs to the end of the source.
- **R195:** `OuterText(n)` returns the bytes from a group's opener through its closer, under the
  same rules as `InnerText`.
- **R196:** Accessors that return a slice return the context's own slice, and a consumer does not
  write through it; this is documented, not guarded, because every consumer discards its document
  within one operation.
- **R222:** `Doc()` returns the document the context is bound to, nil before the parse's `Done`.
- **R207:** `BracketLang` carries `Comment CommentStyle{Prefix, Suffix, Kind}`: how the language
  writes a comment, distinct from how its table recognizes one.
- **R208:** A comment constructed as `Prefix + body + Suffix` parses as a group whose kind equals
  `Comment.Kind`; the agreement is guarded by a per-language test, not a runtime check.
- **R209:** A language with no comment style has an empty `Prefix` and constructs nothing; `sdom`
  compares `Comment.Kind` and never branches on its value.

## Feature: indent scope
**Source:** specs/indent-parser.md

- **R174:** Indentation is significant only at bracket depth 0.
- **R175:** `IndentLang` embeds `BracketLang` and adds `Tab`, `Transparent` and `Continuation`;
  the type is the flag and there is no boolean.
- **R176:** `Transparent` names a `Kind` value rather than a syntax, so this package compares two
  configured strings and never learns what a comment is.
- **R177:** An `Indent` node holds the leading whitespace of the line whose level it announces.
- **R178:** One `Indent` node is emitted at every change of level and none where the level is
  unchanged; consecutive lines at one column share the frame opened by the last change, and their
  leading whitespace stays ordinary text.
- **R179:** A non-empty indent-parsed source opens with a zero-length root `Indent`, emitted by the
  indent parser when no node has yet been emitted; an empty source produces no nodes at all.
- **R180:** A return to column 0 is a zero-length `Indent`, having no whitespace to own.
- **R181:** A dedent is not a node of its own: the column is the level, so closing several levels
  at once is one node with a smaller column.
- **R182:** An `Indent` node's column is derived from its text with tabs expanded by `Tab`, and is
  never stored.
- **R183:** A blank line, and a line holding only groups of the `Transparent` kind, do not change
  the level.
- **R184:** A line following a `Continuation` marker is not indented, and the marker counts only at
  bracket depth 0 — so one inside a comment group does not continue, and one inside a string needs
  no rule because the group is still open at the next line start.
- **R185:** An indent frame's parent is the nearest preceding `Indent` with a strictly smaller
  column, so two frames at one column under one parent are siblings.
- **R186:** The indent context answers a frame's parent and its direct children, and owns one index.
- **R187:** Every indent link is derivable from the flat array alone: an `Indent` node carries its
  own whitespace, so a consumer walking with a stack of open columns reaches the same parent and
  children the context reports, needing no language knowledge to do it.
- **R188:** A dedent to a column matching no open level parents to the nearest smaller column
  rather than being refused, `sdom` being no syntax checker.

## Feature: stencils
**Source:** specs/stencils.md

- **R96:** A stencil is a compound that parses itself from text with a regex whose named groups are the fields its schema binds.
- **R97:** `NewStencilBuilder` reports whether the regex matched; what a non-match means is the schema's decision, not the builder's.
- **R98:** On construction the builder lays out a `Text` for the head of the match, one for every gap between consecutive participating named groups, and one for the tail.
- **R99:** Each participating named group starts as a nil slot for the schema to fill, rather than defaulting to a `Text`.
- **R100:** A named group that did not participate in the match is given no slot and no glue, and owes nothing.
- **R101:** `Group` returns a participating group's matched text and its provenance, and the zero location for a group that did not participate.
- **R102:** `Put` patches a node the schema built into its group's slot.
- **R103:** `Omit` folds a participating group's span into the surrounding glue, so a schema need not bind a group it does not want as a field.
- **R104:** `Done` returns the children and the text the match did not consume.
- **R105:** `Done` panics when a group's slot was left nil.
- **R106:** `Done` panics when a plugged node's span does not match its group's.
- **R107:** The builder returns no name-to-node map; a schema keeps references to the nodes it built as it builds them.
- **R108:** Binding is by group name, never by position.
- **R109:** Nested capture groups are skipped rather than rejected.
- **R110:** A stencil's group indices are half-open, and a zero-length group is a valid `[k,k)` needing no special case.
- **R111:** `Bool` is a typed view over a `Text` node and stores no value of its own.
- **R112:** `Bool.Value` derives the value from the text on every read.
- **R113:** `Bool.Set` writes through to the text, which keeps its offset and becomes altered.
- **R114:** A bound value points at the node already in the child list rather than replacing it.
- **~~R115:~~** (Retired T4 — see R223) Only what a tool writes into is a bound field; minimality constrains a stencil's span, never the number of children within it.
- **R116:** Stenciled parts separated by text become several stencil nodes rather than one span wide enough to swallow the text between them.
- **R223:** A bound field is what a tool reads or writes through a typed view — binding serves
  access as well as editing; minimality constrains a stencil's span, never the number of
  children within it.
- **R117:** `Done` leaves no two adjacent children that are both plain glue: an omitted group's text is merged with its neighbours, so omitting a group produces the identical child list to a regex that never named it.

## Feature: declarations
**Source:** specs/declarations.md

- **R122:** A declaration is not a node with a span: it is a `DeclarationType` carrying the
  keyword and one or more `DeclarationName`s carrying the names, all ordinary siblings in the
  flat array.
- **R123:** `DeclarationType` and `DeclarationName` are node kinds over `Text`.
- **R124:** A declaration pass only splits text nodes and re-types halves, so the flattened
  array is identical with and without it and no marker stops being recognized.
- **R125:** `Doc.Replace` swaps one node for another in the document, preserving position; it
  changes membership, so it bumps the structural generation and requires an open mutation
  window.
- **R126:** A `DeclarationType` holds no reference to its names; the declaration links live on
  `BracketContext`.
- **R127:** The declaration link is one-to-many: a keyword maps to every name it declares, one
  for a plain declaration and several for a group.
- **R128:** `sdom` provides the declaration link map and a schema fills it in, because filling
  it in requires knowing what announces a declaration.
- **R197:** The declaration link is typed at both ends: a schema records
  `map[*DeclarationType][]*DeclarationName`, and the context stores it so.
- **R198:** `DeclarationNames(t *DeclarationType) ([]*DeclarationName, error)` is a method on the
  context returning the document's own name nodes, and it refuses with `ErrDeclarationsStale`
  when the document has changed since the links were recorded.

## Feature: declaration schemas
**Source:** specs/declaration-schemas.md
- **R129:** The package bundles declaration schemas for Go, TypeScript, JavaScript, Python, Lua
  and Shell. **Provisional — see gap O17:** Go, Lua, Shell and Python are implemented;
  TypeScript and JavaScript are not, and closing that gap either adds them or rewrites this
  requirement to name what is bundled.
- **R130:** There is no shared recognition rule: each schema recognizes its own language's
  declarations from the parse it is given.
- **R131:** A declaration keyword may be a substring of a text node or a bracket marker node,
  and a schema handles whichever its language uses.
- **R132:** A declaration's name is found by walking forward in document order from whatever
  announced it, which may be inside the group that keyword opened.
- **R133:** Commented-out code is never recognized as a declaration, because a comment is a
  bracket group and its interior is not a top-level text node.
- **R134:** A match at the start of a top-level text node begins a statement when the preceding
  top-level content, after the skipping of R145, ends with a statement separator.
- **R135:** A schema whose language announces declarations both by keyword and by keyword-less
  shape uses one alternation, whose non-participating branch yields the zero location.
- **R136:** Shell's assignment form forbids whitespace around `=` and Lua's permits it, so the
  two schemas do not share a pattern.
- **R137:** A declaration pattern consumes its statement separator and ends on a word boundary,
  because Go's regexp supports neither lookbehind nor lookahead.
- **R138:** `import` is in no schema's keyword set.
- **R139:** A grouped declaration's names come from a second pass over the single text node
  inside its parens, and each name becomes its own `DeclarationName`.
- **R140:** A name list is captured as one group and split into identifiers afterwards, because
  a repeated capture group reports only its last iteration.
- **R141:** Declarations are sought over top-level nodes; scanning another depth is the same
  operation against a different node set.
- **R142:** Indentation is not part of a declaration: a declaration has no indent field and no
  indent node of its own. Where indentation carries scope it is a separate mechanism with its
  own nodes, and not a declaration's business.
- **R143:** Which declarations ought to carry a traceability comment is a reader's policy, not a
  schema's.
- **R144:** Each schema carries its own recognition pass, and generalizing them into a shared
  tool waits until three schemas exist.
- **R145:** A schema skips whole comment groups and whitespace-only text nodes, in any number
  and any interleaving, whenever it walks the array — backward to test whether a match begins
  a statement, and forward to find a name. Skipping a comment group backward is one hop, because
  a closer names its opener.
- **R146:** Which groups are comments is the language layer's business, said once in the table: a
  language marks its comment groups with a `Kind` and a schema reads that back from an opener. A
  comment and a string remain the same shape, which is why no property of a group could answer it.
- **R147:** The skip of R145 runs before every decision in the walk, not once at its start; a
  receiver group is recognized as an opener met after skipping, and is jumped to its closer
  before skipping again.
- **R148:** A name is an identifier inside a text node that may carry whitespace on either side,
  so slicing it out splits that node at both edges rather than retyping it whole.
- **R149:** Skipped whitespace may contain statement separators, so a declaration may span lines
  and the forward walk does not stop at one; the backward statement-start test is a different
  question and is unaffected.
- **R150:** `sdom` neither validates text nor requires a schema to be lax about it: nothing in
  `sdom` is a syntax checker, and how strict a schema's own walk is, is that schema's choice.
- **R151:** Go's schema requires a func's name to reach its opening parenthesis without
  crossing a newline — not to abut it, since a space or a comment between them is legal while a
  comment carrying a newline is not; the rule binds func alone and no other schema inherits it.
- **R189:** Python announces a declaration with `def` or `class` at a statement start, and the
  name follows in the same text with no group intervening.
- **R190:** Python's declarations sit at bracket depth 0 at any indent depth, indent frames not
  being bracket groups, so the top-level predicate is unchanged.
- **R191:** Which depths a language's declarations live at is a fact about that language rather
  than a parameter, which is why recognition is a per-schema pass and not a shared driver setting.

## Feature: lists
**Source:** specs/lists.md

- **R199:** A `List` is a compound with exactly one child, the `Text` of the whole field.
- **R200:** `Items` derives the values from the literal on every call — split on commas, whitespace
  trimmed — and stores nothing.
- **R201:** `SetItems` rewrites the whole literal canonically, items joined by `", "`; an unedited
  field is never touched by an edit to another.
- **R202:** A list parses `WS? item ( WS? "," WS? item )* WS?`, an item being a run with no
  whitespace and no comma; `ParseList` returns what it did not consume.
- **R203:** `SetItems` is guarded by re-parsing its own render: the re-parse must consume all of it
  and yield the same items, or the write is refused with an error and the literal is unchanged.
- **R204:** A `RequirementList` item is `Rn`, `Rn-Rm` or `Rn-m`; `Items() []int` expands ranges,
  and a reversed range contributes only its low ref.
- **R205:** `RequirementList.SetItems([]int)` sorts, de-duplicates and condenses runs of three or
  more consecutive numbers to `Rn-m`; it cannot be refused.
- **R206:** Only the requirement-list parser accepts ranges; the plain parser does not, so a range
  in a plain list cannot exist by construction.

## Feature: traceability comment
**Source:** specs/traceability-comment.md

- **R210:** `TraceabilityComment` is one node kind for every language, tiling the whole comment
  from opener through closer.
- **R211:** The interior is fields separated by `|` — `CRC:`, `Seq:`, `Test:` each with a plain
  list, and a keyword-less requirement list — at most one of each, in any order, followed by an
  optional description after a separator.
- **R212:** A `:` is a field-key colon only immediately after `CRC`, `Seq` or `Test`; anywhere else
  it is the description separator.
- **R213:** A `Seq` item may carry `#step`, and the typed view splits path from step.
- **R214:** Whitespace, keywords, `|` and the separator are computed glue; an unedited comment
  renders back byte-exact; the separator is preserved when unedited and written as `--` on a fresh
  write; the description is bound and writable.
- **R215:** Recognition is a parse that consumes the whole interior; a comment that leads with a
  field but leaves bytes uncovered is not a traceability comment.
- **R216:** `Parse(cmt *Opener, ctx *BracketContext) bool` fills the node off to the side, touching
  no document, and returns false when the interior is not a single text node or is not consumed.
- **R217:** On success the children are the original `*Opener`, the interior's glue and fields, and
  the original `*Closer` — reused, not recreated.
- **R218:** The interior is parsed by a segment walk with one stencil per `|`-segment, and the
  results splice flat into the node.
- **R219:** `Comments(d, ctx)` is the second pass: every opener whose group kind equals the
  language's `Comment.Kind` is a candidate, each success replaces its run from opener to closer
  inside its own mutation window, and nothing else is touched.
- **R220:** `New(lang, Fields)` assembles the canonical interior — CRC, Seq, Test, refs, `--`
  description — and runs the same interior walk at a synthetic location inside synthetic markers,
  so no child list is hand-built and the node has no origin.
- **R221:** The node exposes typed accessors `CRC`, `Seq`, `Test`, `Refs` and `Description`, nil
  when the field is absent.

## Feature: markdown base
**Source:** specs/markdown.md

- **R226:** `LangMarkdown` is an `IndentLang` in `sdom/schema`, embedded by the trajectory file
  schemas and never used to parse a file alone; it models only what those schemas bind.
- **R227:** Its bracket groups are the fence and the code span, both restricted with kind `code`,
  and `**` and `~~` restricted with escape hatches — bold admits code spans, strike admits bold
  and code spans — in that match order; links are not groups. A symmetric marker is never a
  code-mode group, since there it reopens rather than closes.
- **R228:** `MarkdownParser` implements `Parser`, holds an `IndentParser`, and delegates first; when
  the delegate emitted nothing and moved nothing at a line head, it emits a line-head marker.
- **R229:** A line head is read from the array: `Last()` is an `Indent`, or a `Text` ending in a
  newline followed only by spaces or tabs; no state is kept for it.
- **R230:** `Heading` holds one to six `#` and the space and derives its level from them; `ListItem`
  holds `- `; `Checkbox` holds `[ ]` or `[x]`, is recognized only immediately after a `ListItem`,
  and derives `Checked` from its bytes with `SetChecked` writing the interior through.
- **R231:** No line-head marker shares a first byte with any bracket opener in the table, which is
  what lets the wrapper check after delegating; guarded by a test over the table.
- **R232:** Inside a fence or a code span the wrapper is never offered a position, so a line-head
  shape there is text with no rule needed.
- **R233:** `NodeType` answers for the three line-head kinds and otherwise delegates; `Done`
  delegates.
- **R234:** A marker holds only the bytes it matched; extents — a heading's line and region, a list
  item's body — are a consumer's derivation from the array, and the base records nothing.
- **R235:** Each marker kind is its own type embedding `Text`, with its own `Equals`.

## Feature: part line
**Source:** specs/part-line.md

- **R236:** `PartLine` is one node over a list item line, from the `- ` marker to the byte before
  the newline, reusing the `ListItem`, `Checkbox` and every bracket marker the base emitted.
- **R237:** Every list item line parses; `Parse` returns false only when the item is not in the
  document, and belonging to a status block is the carve schema's business.
- **R238:** The head is the first bold run after the checkbox; the line is keyed when its interior
  begins `Item N — ` or `N.M — ` — the word required without a dot and forbidden with one, the em
  dash and nothing else — and the key is a bound `Text`, the separator glue, the title the rest of
  that text.
- **R239:** A later bold run whose interior reads `VERB (attribution)`, the verb in capitals as one
  or more words joined by spaces or hyphens, is a `MarkerSpan`; anything else is interspersed text.
- **R240:** `Checkbox()` is the base's own `Checkbox` node, nil when the line has none.
- **R241:** `IsStruck()` derives from whether the head sits inside a `~~` group; `Strike(bool)`
  inserts or removes the `~~` pair around the head's bold run among the node's own children — the
  flat array is untouched, so no mutation window is involved; no consumer touches a `~~` node.
- **R242:** Deviations are reported, each naming its rule and target shape — unkeyed head,
  non-em-dash separator, non-conforming checkbox interior, verb not in capitals, an `OPEN`
  attribution not exactly `#N.` or `not queued.` — and the line still parses.
- **R243:** `MarkerSpan` tiles `**` through `**` reusing both; its verb is a bound `Text`; its
  attribution and queue ID are derived by rendering the nodes between the parentheses.
- **R244:** `MarkerSpan.Set` rewrites the interior canonically as one text under the guarded write:
  the render re-parsed as a marker must yield the same verb and attribution, or the write is
  refused and the literal unchanged.
- **R245:** `Splice` replaces the line's run with the node inside a mutation window; `PartLines`
  parses and splices every list item in document order.
- **R246:** `PartLine` and `MarkerSpan` each declare their own `Equals`, comparing children.
- **R247:** Bound texts and glue are re-cut from the interior texts; no bytes are lost or
  normalised on read, and an unedited line renders back byte-exact.

## Feature: carve schema
**Source:** specs/carve-schema.md

- **R248:** `Carve` embeds the markdown base and owns a carve file's DOM; `ParseCarve` is the
  only way one is made and `Render` re-emits it.
- **R249:** The status block is the region from the level-2 heading `Status` to the next heading
  of level 2 or higher or the end of file; `HasStatus` is false when there is none.
- **R250:** Only list items inside the status region become part lines; other lists in the body
  are left as the base parsed them, and a fenced sample is invisible by construction.
- **R251:** A status line with a checkbox is a `Part`; one without is `Stateless`, recorded and
  never given a state.
- **R252:** `Depth` is the bullet's leading whitespace, and `Parent` is the nearest preceding
  part with a smaller depth, or nil.
- **R253:** `Part(key)` finds a part by its key; a write to a key no part carries is an error.
- **R254:** `SetMarker` replaces the first transient marker — verb `OPEN` — removes any other
  transient, and appends a marker when the line carries none; it selects by what it replaces.
- **R255:** `Land` checks the box, strikes the head, and sets `LANDED (attribution)` through the
  marker rule, in one act.
- **R256:** Every part carries its line's deviations.
