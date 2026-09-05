package minispecsdom

import (
	"fmt"

	"github.com/zot/simple-dom/sdom"
)

// CRC: crc-Done.md | R300, R301, R302, R303
//
// unclosed lists every group the base's context reports open at end of input, at its
// opener's line, as an Unread naming the marker. One helper for four readers: a fence
// or span that runs to end of file takes every later entry with it and leaves nothing
// entry-like to list, so each reader must say this itself rather than wait to notice.
// Appended after a reader's other unread lines, which is file order: nothing structured
// can follow a group that is still open at the end.
func unclosed(ctx *sdom.BracketContext) []Unread {
	doc := ctx.Doc()
	var out []Unread
	for _, o := range ctx.Unclosed() {
		marker, _ := o.Render()
		line := doc.Line(o.Location().Offset())
		out = append(out, Unread{line, fmt.Sprintf("`%s` open to end of input", marker)})
	}
	return out
}
