# Test Design: Python declarations
**Source:** crc-PythonSchema.md

Fire alarms are added in the Implementation phase.

## Test: a method inside a class is found
**Purpose:** R190 — the whole point: indent frames are not bracket groups
**Input:** `class Widget:` / `    def draw(self):` / `        pass`
**Expected:** both `class Widget` and `def draw` are recognized, the second being
a top-level text node despite sitting two indent levels in
**Refs:** crc-PythonSchema.md, seq-declare.md
**Code:** sdom/schema/python_test.go
**Alarm:** 1
**Fire alarm:** In `schema.isSpace`, restore the `n.(*sdom.Text)` assertion in place of the kind test. **This is the real defect, hit while writing this schema:** `Indent` embeds `Text` but is its own type, so the backward statement-start walk stops recognizing an indent as whitespace and **every declaration at a line start is rejected** — `class Widget:` at offset 0 finds nothing. Red: every Python test that expects a declaration. Go, Lua and Shell stay green throughout, their documents containing no `Indent` nodes, which is why this shipped invisibly until a second parser existed.
**Inject:** sdom/schema/schema.go:isSpace
**Pulled:** 2026-09-01 — rang, and the prediction held exactly. Three test functions failed, all of them Python's; the **entire `sdom` package stayed green**, and so did the Go, Lua and Shell schema tests sitting in the same package as the failure. **119 of 122 tests pass under the real defect** — which is the claim this alarm exists to make. A node kind added in one package silently broke a consumer in another, and nothing short of a parser that actually emits `Indent` could see it.

## Test: a def inside a docstring is not a declaration
**Purpose:** R133 in Python's form — a comment or string interior is inside a group
**Input:** a module docstring containing the text `def notreal():`
**Expected:** it is not recognized
**Refs:** crc-PythonSchema.md
**Code:** sdom/schema/python_test.go

## Test: a decorated declaration is found, and the decorator is not one
**Purpose:** R189 — decorators are ordinary text above
**Input:** `@cache` / `def f():` / `    pass`
**Expected:** `def f` is recognized; `@cache` produces nothing
**Refs:** crc-PythonSchema.md
**Code:** sdom/schema/python_test.go

## Test: a def after a comment line begins a statement
**Purpose:** R134 — the structural half of the statement-start rule
**Input:** `# CRC: crc-Doc.md` / `def f():`
**Expected:** recognized, the preceding top-level content ending in the comment's
closing newline `Closer` rather than in a byte of text
**Refs:** crc-PythonSchema.md
**Code:** sdom/schema/python_test.go

## Test: the pass is additive
**Purpose:** R124 in Python's form
**Input:** every Python fixture, parsed once plain and once with the schema
**Expected:** the same bytes, the same tiling, and a recognition count that does
not drop
**Refs:** crc-PythonSchema.md
**Code:** sdom/schema/python_test.go

## Test: only spaces may separate the keyword from the name
**Purpose:** R189 — the guard the happy path never exercises
**Input:** `def\tfoo`, `def    foo`, `def -foo`, `class *A`
**Expected:** the first two are declarations; the last two are not
**Refs:** crc-PythonSchema.md
**Code:** sdom/schema/python_test.go
**Alarm:** 2
**Fire alarm:** Delete the loop in `pyNameAfter` that requires everything between the
keyword and the identifier to be a space or a tab. Red: `def -foo():` is recognized as
declaring `foo`. Python has no block comment and a newline there does not compile, so
anything else between them means this is not the declaration it looks like — but no
valid Python reaches the guard, which is exactly why nothing was asking.
**Inject:** sdom/schema/python.go:pyNameAfter
**Pulled:** 2026-09-01 — **found by injecting past the alarm list.** Deleting the guard
outright left the whole suite green beforehand. With this test it fails on both
negative cases, `got "def[foo] " want ""`.
