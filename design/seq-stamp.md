# Sequence: a layer checking its derived index
**Requirements:** R42, R43, R44, R45, R46

How a layer above `sdom` keeps derived state fresh without `Doc` knowing the layer
exists. This is the whole of the stamped-not-registered protocol.

## 1. A layer uses its index

1. A layer needs state derived from document structure
   1.1. The layer reads `d.Generation()`
        1.1.1. Outside a mutation window, it gets the current generation
        1.1.2. Inside one, the read **refuses** with the typed sentinel — which is
               how the guard reaches this index without the layer ever writing one
   1.2. The layer compares the answer against its own stamp
   1.3. Stamps equal — the index is fresh and is used as it stands
   1.4. Stamps differ — the layer rebuilds from the document's nodes and re-stamps
        with the generation it just read
   1.5. `Doc` learns nothing about any of this: no registration, no callback, and a
        document type whose layers need no derived state carries none
