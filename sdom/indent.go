package sdom

import "strings"

// CRC: crc-IndentLang.md | R175, R176
//
// IndentLang describes a language whose indentation carries scope. It EMBEDS
// BracketLang and the TYPE IS THE FLAG, so a brace language carries none of these
// settings rather than meaningless zeroed ones.
type IndentLang struct {
	BracketLang

	// R182: a tab advances to the next multiple of this.
	Tab int

	// R176: the Kind marking groups that do not affect the level — a language marks
	// its comment groups with some label and names that label here. So this package
	// compares two configured strings and NEVER LEARNS WHAT A COMMENT IS. Writing
	// the value in here rather than in sdom is the whole reason that holds.
	//
	// The distinction has to come from a label because no property of a group's
	// shape supplies it: a comment-only line does not change the level and a
	// string-only line does, and both open a parse-restricted group.
	Transparent string

	// R184: a line following this marker is never indented.
	Continuation string
}

// column expands tabs in ws to the next multiple of Tab. The rule lives on the
// language because Tab does, and because the parse and the derivation both need it
// — one function, so the two cannot drift apart.
func (l *IndentLang) column(ws string) int {
	col := 0
	for _, r := range ws {
		if r == '\t' && l.Tab > 0 {
			col += l.Tab - col%l.Tab
		} else {
			col++
		}
	}
	return col
}

// CRC: crc-IndentParser.md | R177
//
// Indent announces a change of level, holding the leading whitespace of the line
// whose level it announces. Two of them are ZERO-LENGTH: the root, and a return to
// column 0, which has no whitespace to own. Half-open indices make both free.
//
// It embeds Text like the bracket markers, and declares its own Equals for the same
// reason: a promoted one cannot see the outer type.
type Indent struct{ Text }

// CRC: crc-IndentParser.md | R177
func NewIndent(text string, loc Loc) *Indent { return &Indent{Text{text: text, loc: loc}} }

// CRC: crc-IndentParser.md | R177
func (i *Indent) Equals(other Node) bool {
	x, ok := other.(*Indent)
	return ok && i.Text.Equals(&x.Text)
}

// CRC: crc-IndentParser.md | Seq: seq-indent.md | R174, R178, R179, R180, R183, R184
//
// IndentParser turns line starts into scope. It holds a BracketParser and hands off
// whenever it does not match, so ONE PASS produces both kinds of node.
//
// R174: it is registered only in the outermost loop, and the bracket parser TAKES
// THE LOOP on an opener — so this parser is never offered a position inside a group,
// and "indentation is significant only at bracket depth 0" holds structurally with no
// depth check anywhere.
//
// R187: it records no links. The frames are implied by the columns it emits, so the
// context derives them on demand rather than keeping a second copy that can disagree.
type IndentParser struct {
	lang     *IndentLang
	brackets *BracketParser
	ctx      *IndentContext
	levels   []int // open columns; parse state, not an index
}

// CRC: crc-IndentParser.md | R175
func NewIndentParser(lang *IndentLang) *IndentParser {
	return &IndentParser{
		lang:     lang,
		brackets: NewBracketParser(&lang.BracketLang),
		ctx:      &IndentContext{lang: lang},
	}
}

// Context returns the frame links this parse implies. Concrete, so no caller asserts.
func (ip *IndentParser) Context() *IndentContext { return ip.ctx }

// Brackets returns the bracket parser this one delegates to, and the bracket links
// with it. One pass produces both indexes; a consumer usually wants both.
func (ip *IndentParser) Brackets() *BracketParser { return ip.brackets }

// CRC: crc-IndentParser.md | Seq: seq-indent.md#1 | R178, R179, R180
//
// Parse emits an Indent where the level changes, and delegates everywhere else.
func (ip *IndentParser) Parse(st *ParserState) {
	// R179: the root, emitted HERE rather than by the walk — which keeps the walk
	// ignorant of indent and gives a non-indent parse no root at all. The walk sees
	// the node count move, does not advance, and offers this same position again,
	// where the level stack is no longer empty and this branch cannot fire twice.
	// An empty source produces no nodes, since the loop never calls Parse.
	if st.NodeCount() == 0 {
		st.Emit(NewIndent("", st.At(st.Pos(), 0)))
		ip.levels = []int{0}
		return
	}
	if ws, col, ok := ip.change(st); ok {
		st.Emit(NewIndent(ws, st.At(st.Pos(), len(ws))))
		ip.enter(col)
		return
	}
	ip.brackets.Parse(st)
}

// CRC: crc-IndentParser.md | R160
// NodeType reports what Parse would emit here without emitting it. An Indent belongs
// to no group and so carries no kind.
func (ip *IndentParser) NodeType(st *ParserState) (string, bool) {
	if st.NodeCount() == 0 {
		return "", true
	}
	if _, _, ok := ip.change(st); ok {
		return "", true
	}
	return ip.brackets.NodeType(st)
}

// CRC: crc-IndentParser.md | R192
// Done binds both contexts to the finished document.
func (ip *IndentParser) Done(d *Doc) {
	ip.brackets.Done(d)
	ip.ctx.attach(d)
}

// CRC: crc-IndentParser.md | Seq: seq-indent.md#1 | R174, R183, R184
//
// change reports the whitespace and column of a line whose level differs from the
// one now open, and whether there is such a change here at all.
func (ip *IndentParser) change(st *ParserState) (ws string, col int, ok bool) {
	if !ip.atLineStart(st) || ip.continued(st) {
		return "", 0, false
	}
	src, p := st.Src(), st.Pos()
	e := p
	for e < len(src) && (src[e] == ' ' || src[e] == '\t') {
		e++
	}
	// R183: a blank line changes nothing.
	if e >= len(src) || src[e] == '\n' {
		return "", 0, false
	}
	// R183: nor does a line holding only a Transparent group. Deciding needs the
	// delegate, because a comment and a string both open a restricted group and only
	// the Kind separates them — which is why a lookahead reporting mere presence
	// could not answer it.
	if ip.transparentAt(st, e) {
		return "", 0, false
	}
	ws = src[p:e]
	col = ip.column(ws)
	if col == ip.levels[len(ip.levels)-1] {
		return "", 0, false
	}
	return ws, col, true
}

// atLineStart reports whether pos is the first byte of a line. Position 0 is the
// root's business, not this.
func (ip *IndentParser) atLineStart(st *ParserState) bool {
	return st.Pos() > 0 && st.Src()[st.Pos()-1] == '\n'
}

// CRC: crc-IndentParser.md | Seq: seq-indent.md#1.3 | R184
//
// continued reports whether the previous line ended with a continuation marker that
// counts — which needs no depth check of its own, because a marker counts exactly
// when it is still PENDING TEXT.
//
// A marker no group claimed is unflushed text, so it sits in src[textStart:pos]. One
// inside a comment was consumed by that group, whose closer is the newline, leaving
// nothing pending. One inside a string needs no rule at all: the group is still open
// at the next line start, so this parser is never offered the position.
func (ip *IndentParser) continued(st *ParserState) bool {
	if ip.lang.Continuation == "" || st.Pos() == 0 {
		return false
	}
	pending := st.Src()[st.textStart:st.Pos()]
	return strings.HasSuffix(pending, ip.lang.Continuation+"\n")
}

// transparentAt reports whether a group of the Transparent kind opens at e.
func (ip *IndentParser) transparentAt(st *ParserState, e int) bool {
	if ip.lang.Transparent == "" {
		return false
	}
	was := st.Pos()
	st.SetPos(e)
	kind, ok := ip.brackets.NodeType(st)
	st.SetPos(was)
	return ok && kind == ip.lang.Transparent
}

// CRC: crc-IndentParser.md | R182
// column expands tabs to the next multiple of Tab. The text is the storage; this is
// derived on every read and never kept.
func (ip *IndentParser) column(ws string) int { return ip.lang.column(ws) }

// CRC: crc-IndentParser.md | Seq: seq-indent.md#1.6 | R181, R188
//
// enter records the new level. Closing several at once is ONE node with a smaller
// column, because the column is the level; and a column matching no open level is
// not refused — it lands beside the nearest smaller one, sdom being no syntax
// checker.
func (ip *IndentParser) enter(col int) {
	for len(ip.levels) > 1 && ip.levels[len(ip.levels)-1] >= col {
		ip.levels = ip.levels[:len(ip.levels)-1]
	}
	// Nothing handles a column that is not deeper than the top, and nothing needs
	// to: after that loop it can only mean column 0 with the root alone on the
	// stack, which is the state already. The root is never popped and a column is
	// never negative.
	if col > ip.levels[len(ip.levels)-1] {
		ip.levels = append(ip.levels, col)
	}
}

// CRC: crc-IndentContext.md | R186
//
// IndentInfo is everything this context knows about one frame.
type IndentInfo struct {
	parent   Node
	children []Node
}

// CRC: crc-IndentContext.md | Seq: seq-indent.md#2 | R185, R186, R187, R188
//
// IndentContext carries the language during the parse and OUTLIVES it to own the
// frame links.
//
// R186: ONE index, and it is the outer one — brackets cannot contain an indent
// region, so this context sits above the bracket context rather than beside it. A
// brace language constructs none of this and pays nothing.
//
// R187: the parse records nothing here. Every link is derivable from the array
// alone, since an Indent carries its own whitespace, so this derives them once on
// demand rather than keeping a copy that can disagree.
type IndentContext struct {
	lang  *IndentLang
	doc   *Doc
	info  map[Node]IndentInfo
	built bool
	stamp uint64
}

// Language returns the table this context parsed with.
func (ic *IndentContext) Language() *IndentLang { return ic.lang }

// CRC: crc-IndentContext.md | R186
// Parent returns the frame containing this one, or nil for a top-level frame.
func (ic *IndentContext) Parent(frame Node) Node {
	ic.refresh()
	return ic.info[frame].parent
}

// CRC: crc-IndentContext.md | R186
// Children returns the frames opened directly inside this one, in document order.
func (ic *IndentContext) Children(frame Node) []Node {
	ic.refresh()
	return ic.info[frame].children
}

// attach binds the context to the document it was parsed from. It does NOT stamp:
// the index is empty until the first read builds it, and a stamp here would report a
// current index that knows nothing.
func (ic *IndentContext) attach(d *Doc) { ic.doc = d }

// CRC: crc-IndentContext.md | Seq: seq-stamp.md | R186
//
// refresh rebuilds when the stamp has gone stale, or when nothing has been built.
// Reading the generation is the one call every stamped index makes, which extends
// the mutation-window guard to this index without it writing a guard of its own.
func (ic *IndentContext) refresh() {
	if ic.doc == nil {
		return
	}
	if g := ic.doc.Generation(); !ic.built || g != ic.stamp {
		ic.rebuild()
		ic.stamp = g
		ic.built = true
	}
}

// CRC: crc-IndentContext.md | Seq: seq-indent.md#2.3 | R185, R187, R188
//
// rebuild derives every frame link by walking the finished array with a stack of
// open columns. It needs NO LANGUAGE KNOWLEDGE beyond the tab width: an Indent node
// carries its own whitespace, so the column is read off the node rather than
// recomputed from the source.
//
// R185: a frame's parent is the nearest preceding Indent with a strictly smaller
// column — except that the FIRST frame is never popped, so a return to column 0
// lands inside the root rather than beside it. That is what makes Children(root) the
// top-level frames, and what keeps the answer in this one index instead of a
// separate list of roots.
func (ic *IndentContext) rebuild() {
	ic.info = map[Node]IndentInfo{}
	type frame struct {
		node Node
		col  int
	}
	var stack []frame
	for _, n := range ic.doc.Nodes() {
		ind, ok := n.(*Indent)
		if !ok {
			continue
		}
		text, err := ind.Render()
		if err != nil {
			continue // a node that cannot render cannot be placed
		}
		col := ic.lang.column(text)
		for len(stack) > 1 && stack[len(stack)-1].col >= col {
			stack = stack[:len(stack)-1]
		}
		// Whatever survived the popping is the parent. R188: a column matching no
		// open level, or matching the one frame that is never popped, lands inside
		// that frame rather than being refused.
		var parent Node
		if len(stack) > 0 {
			parent = stack[len(stack)-1].node
		}
		if parent != nil {
			e := ic.info[parent]
			e.children = append(e.children, n)
			ic.info[parent] = e
		}
		e := ic.info[n]
		e.parent = parent
		ic.info[n] = e
		stack = append(stack, frame{n, col})
	}
}
