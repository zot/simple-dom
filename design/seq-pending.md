# Sequence: the queue file
**Requirements:** R259, R260, R261, R263

## 1. Reading

1. `ParsePending(src)`
   1.1. Parse with the markdown base
   1.2. For each level-2 `Heading`: if the text after it opens `N.`, an entry begins;
        otherwise it is listed as unread, with its line
   1.3. The region runs to the next level-2-or-higher heading, or to a text holding a
        `---` line, where the run is cut
   1.4. Derive the values from the run's rendered bytes: the heading line, the
        `Source:` line, the `Next:` line

## 2. Placing

2. `Place(e, pos)`
   2.1. Refuse a position outside `1 … len+1`
   2.2. Render the canonical entry text, ending in a blank line
   2.3. Inside a window, `Insert` it as one synthetic text before the entry at `pos`, or
        at the end; re-read the document and re-derive the entries

## 3. Removing

3. `Remove(id)`
   3.1. Find the entry; unknown is an error
   3.2. Inside a window, split the shared tail text at the region's end if the run
        does not end on a node boundary, then `Remove` every node of the run
   3.3. Re-read the document and re-derive the entries
