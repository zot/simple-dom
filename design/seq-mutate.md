# Sequences: the mutation window
**Requirements:** R31, R33, R47, R48, R49, R50, R51, R52, R53, R54, R55

Three diagrams: an ordinary mutation, a `Merge` inside one, and a failure.

## 1. An ordinary mutation, start to finish

1. A caller changes the document
   1.1. Outside the window, the caller navigates with `Prev` / `Next` and resolves
        the nodes it means to edit
   1.2. The caller holds **node references** — a position would not survive what
        happens next
   1.3. `d.Mutate(f)` saves the current window state and opens the window
   1.4. `f` runs, editing directly
        1.4.1. A content write changes the node in place, through that kind's own
               door
        1.4.2. A structural change alters the node array in place
        1.4.3. Any `Prev`, `Next`, position lookup, line lookup or generation read
               panics with the typed sentinel
   1.5. `f` returns nil; `Mutate` restores the saved window state and closes
   1.6. The two derived indices rebuild **once**, and the structural generation
        bumps if membership changed

## 2. A `Merge` inside the window

2. Two adjacent nodes become one
   2.1. Both operands were resolved outside the window and are held by the caller
   2.2. `d.Merge(a, b)` checks adjacency
        2.2.1. Both faithful — their two locations prove it at the call site, and a
               non-adjacent pair is an error returned there
        2.2.2. Either unfaithful — no check runs, since an altered node's offset is
               historical while its length is current
   2.3. The merged location is computed
        2.3.1. Faithful only if **both** operands are
        2.3.2. Offset is `a`'s when `a` has provenance, and `b`'s otherwise
   2.4. The merged node replaces `a` and `b` in the array
   2.5. Membership changed, so the generation will bump when the window closes

## 3. A failure inside the window

3. Something the caller did not handle gets out
   3.1. `f` returns an error, or a panic escapes it
   3.2. `Mutate` inspects what came out
        3.2.1. The typed navigation sentinel becomes an ordinary error
        3.2.2. Any other panic is re-raised unchanged, so real bugs keep their stack
   3.3. The document is **poisoned**: what was applied before the failure is
        unknown, and nothing is undone
   3.4. The window state is restored and the error returned
   3.5. There is no reset — the caller re-parses, because a poisoned document means
        a bug in the code that wrote to it
