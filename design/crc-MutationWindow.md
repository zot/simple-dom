# MutationWindow
**Requirements:** R31, R32, R33, R47, R48, R49, R50, R51, R52, R53, R54, R55, R56, R120

The bracket around a set of edits. **Edits are direct** — the window's whole
mechanism is that nothing can observe the document while they land.

## Knows
- whether a mutation is in progress
- the window state it saved on entry, to restore on exit
- whether the document has been poisoned

## Does
- `Mutate(f) error`: opens the window, runs `f`, closes it
- refuses every read that could observe a half-edited document: `Prev`, `Next`,
  the position lookup, the line lookup, and **reading the structural generation**
- converts the typed navigation sentinel into an error; **re-raises any other
  panic unchanged**, so real bugs keep their stack
- rebuilds the derived indices **once**, at the exit
- `Split(n, at)` and `Merge(a, b)`: membership changes, which is why they are
  `Doc` methods rather than node methods
- checks `Merge`'s adjacency **when both operands are faithful**, from their two
  locations at the call site, and not at all otherwise

## Constraints
- **Resolve targets before entering.** Navigation is legal outside; node
  references survive whatever the mutation does, and positions do not
- **No rollback.** Edits landed as they were made, so there is nothing to restore
  to, and a partial restore would produce a plausible wrong state
- **Saves and restores rather than counting**, so a nested call is a pass-through
- **An escaping error or panic poisons the document.** What was applied by then is
  unknown. There is no reset; recovery is to re-parse
- **Refusal, not a stale answer.** Rebuilding an index mid-window would not rescue
  a caller whose node was removed: a position lookup returning −1 and a `Next`
  returning nothing are indistinguishable from end-of-document
- **No operation log, queued plan, transaction or undo** — and none of them is
  prevented. This window is the seam a later layer would attach one to, and
  identity-keyed edits are already the primitive such a system needs

## Collaborators
- Doc: the document it brackets, and whose indices it rebuilds
- Node: edited directly, through each kind's own doors
- Loc: supplies the merged-location rules and the adjacency arithmetic

## Sequences
- seq-mutate.md
