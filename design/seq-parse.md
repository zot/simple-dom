# Sequences: the scan
**Requirements:** R57, R64, R72, R73, R74, R75, R77, R291, R294, R309, R346, R347, R349, R353, R356

Three diagrams: a code-mode group, a scan-restricted one, and the rules that keep
the scan from stalling or losing bytes.

## 1. A code-mode group

1. The parser meets an opener
   1.1. The marker matches, and the group's `AllowedParent` permits the context
        currently open
   1.2. An `Opener` node is constructed and **appended to the document's array**
   1.3. The group's loop runs, receiving the group as the enclosing context — the loop
        *is* the context; the parser's stack of open (group, opened text) frames is
        consulted only for step 1.6
        1.3.1. Whatever the loop recognizes is appended as a **sibling**, never as
               a child
        1.3.2. A nested opener recurses; the nesting lives on the call stack
        1.3.3. A separator of this group is appended and the loop continues
   1.4. This group's closer matches — the literal `Close`, or with `CloseIsOpen` the
        very bytes that opened it, for a pattern group the pattern's match equal to them,
        under the group's `BeforeClose`; a `Closer` is appended and the loop returns
        1.4.1. A run of the pattern that is not the opener's text: content, whole — or,
               longer and rejected, an unbalanced `Closer` that ends the group
   1.5. Nothing of the nesting survives in the array — opener, contents and closer
        are siblings, and the pairing is a derived index
   1.6. Not this group's closer and not a separator, but the closer of a group further
        down the stack, the nearest first: the loop returns unclosed without consuming
        1.6.1. `open` ends the group as at end of input — demoted if `DemoteUnclosed`,
               otherwise left unclosed — and pops its frame
        1.6.2. The enclosing loop meets the same position: its own closer closes it, or
               it returns in turn, until the group that closer belongs to is reached

## 2. A scan-restricted group

2. The parser meets a string, or a comment
   2.1. Its opener is appended exactly as in diagram 1
   2.2. Inside, only three things are recognized
        2.2.1. This group's own closer, which ends it — literal, or the opener's text
        2.2.2. Its `Escape`, which consumes itself and the following byte as
               literal
        2.2.3. An opener of a group named in `AllowedInner` — by a literal opener or by
               its pattern — which recurses into whatever mode that group declares
   2.3. Every other byte accumulates as literal `Text` — comments inside strings
        are not comments, and brackets inside comments are not brackets
   2.4. A comment is this case with a newline or a terminator for its closer,
        which is what makes it non-nesting without a rule saying so

## 3. The scan cannot stall, and cannot lose bytes

3. Nothing recognized matches here
   3.1. The any-close fallback is tried, once no enclosing group closes here (1.6): any
        code-mode group's literal closer is recognized, so a stray `}` lands as a `Closer`
        rather than derailing the scan
   3.2. Otherwise a text run begins, and **advances at least one byte** before
        testing again — which holds structurally, since this branch is only
        reached once no marker matched at this position
   3.3. The run ends where a marker would start, or at end of input
   3.4. At end of input every group still open closes where it stands — unless it demotes
        3.4.1. A `DemoteUnclosed` group reaching end of input, or a `BlankLineBound` one
               reaching a blank line — unless `LineHeadUnbound` and the opener stood at a
               line head — returns to `open` *unclosed*
        3.4.2. `open` rewinds the state to the node count and position before its opener,
               takes the opener's bytes as text, and records the demotion on the context —
               dropping any demotion recorded inside the group
        3.4.3. `open` returns to the enclosing loop, which re-reads the bytes in its own
               mode: markers the group had enclosed are recognized, and a later opener of
               the same group opens afresh
   3.5. The array tiles the source: every byte belongs to exactly one node
