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
- **R19:** Compound nodes exist only for stenciling — a region a tool writes into.

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
- **R91:** An origin is minted once per parse and shared by every node that parse produces, so two scans of the same source yield two origins.
- **R92:** Setting a location's origin is a chained operation; the constructor taking an offset and a length is unchanged.
- **R93:** A nil origin is unknown rather than different, and is compatible with any other origin.
- **R94:** Merging two locations whose origins are both present and different panics.
- **R95:** That panic is not the mutation-window sentinel, so `Mutate` re-raises it rather than converting it to an error.

## Feature: document
**Source:** specs/document.md

- **R37:** A `Doc` holds the source bytes, a base offset, the flat document-order node array, an open `data` slot, and two derived indices.
- **R38:** A `Doc`'s base offset is its own position within an outer document. *(See R88, which states the other half: node offsets are relative to the document's own source and never include the base. Tracked as gap O9 until the code matches.)*
- **R39:** A `Doc`'s derived indices cover its own node array and nothing else: one from node to position, one over lines.
- **R40:** A `Doc` holds no lexicon-specific state; any other derived state is owned by the layer that needs it.
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

## Feature: bracket lexer
**Source:** specs/bracket-lexer.md

- **R57:** The bracket lexer scans a source into a flat, document-order stream of `sdom` nodes.
- **R58:** The lexer is table-driven: supporting a new language means adding a table entry, not code.
- **R59:** Strings and comments are expressed as bracket groups rather than as special cases.
- **R60:** `BracketLang` carries the language's bracket groups and no other lexical configuration.
- **R61:** `BracketGroup` declares `Open`, `Separators`, `Close`, `Escape`, `AllowedInner` and `AllowedParent`.
- **R62:** There is no comment configuration: a line comment is a group closing on a newline, and a block comment a group closing on its terminator.
- **R63:** A block comment nests only when its own opener is listed in its `AllowedInner`.
- **R64:** `AllowedInner` nil means code mode — every group's openers are recognized inside the group.
- **R65:** `AllowedInner` non-nil, including empty, means scan-restricted — only the group's own `Close`, its `Escape`, and the listed openers are recognized, and every other byte is literal.
- **R66:** `AllowedParent` nil means the group is recognized in any context; non-nil means it is recognized only while scanning inside one of the listed openers.
- **R67:** nil and an empty slice are semantically distinct in both `AllowedInner` and `AllowedParent`.
- **R68:** `BracketLang` carries no indent parameters and no flag enabling indentation; a language needing indent scope is described by a type that embeds `BracketLang`, and the type is the flag.
- **R69:** The package exports the language tables `LangGo`, `LangShell`, `LangPascal` and `LangJavaScript`, chosen so that every field of `BracketGroup` is exercised by at least one of them.
- **R70:** Language tables are Go values; the package ships no config-file format or loader for them.
- **R71:** A marker whose first byte is a word character is recognized only at a word boundary — neither preceded nor followed by a word character.
- **R72:** A group's separators are recognized only while that group is the one currently open.
- **R73:** When no other marker matches, any code-mode group's closer is recognized, so a stray closer lands as a bracket rather than derailing the scan.
- **R74:** The scan always consumes at least one byte.
- **R75:** A group left open at end of input closes there, and no bytes are dropped.
- **R76:** Whitespace is not a token: it folds into text, so a text run is everything between two recognized markers.
- **R77:** The emitted stream is flat and in document order — an opener, everything between it and its closer, and the closer are siblings in the array.
- **R78:** The lexer adds the node kinds `Opener`, `Closer` and `Separator`; every other byte becomes `Text`.
- **R79:** A marker node holds the bytes it matched and no reference to its bracket group; the active group comes from the parse context.
- **R80:** The lexer's parse context is a concrete type, carrying the language while scanning.
- **R81:** The parse context outlives the parse and owns the bracket pairing links.
- **R82:** An opener knows its closer and its enclosing opener.
- **R83:** A closer knows its opener.
- **R84:** Any node that is not a bracket marker knows its enclosing opener.
- **R85:** The pairing links are owned by the parse context and never by `Doc`.
- **R86:** The pairing links are a derived index: the context stamps itself with the document's structural generation and rebuilds when that stamp is stale.
- **R87:** A forward scan that skips whole bracket pairs finds a node's enclosing opener independently, and must agree with the index.

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
- **R115:** Only what a tool writes into is a bound field; minimality constrains a stencil's span, never the number of children within it.
- **R116:** Stenciled parts separated by text become several stencil nodes rather than one span wide enough to swallow the text between them.
- **R117:** `Done` leaves no two adjacent children that are both plain glue: an omitted group's text is merged with its neighbours, so omitting a group produces the identical child list to a regex that never named it.
