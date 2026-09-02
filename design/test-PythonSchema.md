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
