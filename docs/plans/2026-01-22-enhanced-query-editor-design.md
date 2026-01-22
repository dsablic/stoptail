# Enhanced Query Editor Design

## Overview

Improve the workbench body editor with line numbers, syntax highlighting, smarter validation, and context-aware completion.

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                    Body Editor                       │
├─────────────────────────────────────────────────────┤
│  Textarea (raw input)                               │
│       ↓                                             │
│  Tree-sitter JSON parser                            │
│       ↓                                             │
│  Syntax-highlighted render + line numbers           │
├─────────────────────────────────────────────────────┤
│  Validation Pipeline:                               │
│                                                     │
│  1. encoding/json (instant)                         │
│     → JSON syntax errors                            │
│                                                     │
│  2. _validate API (debounced 500ms)                 │
│     → ES DSL semantic errors                        │
├─────────────────────────────────────────────────────┤
│  Context-Aware Completion:                          │
│                                                     │
│  Tree-sitter AST path → DSL schema lookup           │
│     → Valid keys/values for cursor position         │
└─────────────────────────────────────────────────────┘
```

## Components

### 1. Line Number Gutter

Render a fixed-width column (3-4 chars) left of the textarea content:

```
┌──────────────────────────┐
│ Body  ✓                  │
│  1 │ {                   │
│  2 │   "query": {        │
│  3 │     "match": {}     │
│  4 │   }                 │
│  5 │ }                   │
└──────────────────────────┘
```

- Gray text, subtle background
- Error lines marked with red indicator
- Syncs with textarea scroll position

### 2. Syntax Highlighting

Use tree-sitter with JSON grammar for AST-based highlighting:

- Keys: blue
- Strings: green
- Numbers: yellow
- Brackets/braces: white
- Null/booleans: purple

The textarea handles editing; we overlay highlighted content for display.

### 3. Validation Pipeline

**Layer 1: JSON syntax (instant)**
- Uses `encoding/json.Unmarshal`
- Reports line/column for parse errors
- Already implemented

**Layer 2: ES _validate API (debounced)**
- Fires 500ms after typing stops
- Validates semantic correctness
- Catches: typos in field names, invalid query types, wrong parameters
- Cancel in-flight requests if user keeps typing

**UI feedback:**

```
JSON valid + ES valid:     Body  ✓
JSON valid + ES pending:   Body  ✓ ⋯
JSON valid + ES invalid:   Body  ✓ ✗ unknown field [matchh]
JSON invalid:              Body  ✗ 3:5 unexpected comma
```

### 4. Context-Aware Completion

Tree-sitter provides the AST path to cursor position. Map paths to valid completions:

| Cursor Context | Completions |
|----------------|-------------|
| Root object | `query`, `size`, `from`, `sort`, `aggs`, `_source`, `highlight` |
| `query.*` | `match`, `match_phrase`, `term`, `terms`, `bool`, `range`, `exists` |
| `query.bool.*` | `must`, `should`, `must_not`, `filter`, `minimum_should_match` |
| `query.match.<field>.*` | `query`, `operator`, `fuzziness`, `analyzer`, `boost` |
| `query.range.<field>.*` | `gte`, `gt`, `lte`, `lt`, `format`, `boost` |
| `aggs.<name>.*` | `terms`, `avg`, `sum`, `min`, `max`, `date_histogram` |
| `aggs.<name>.terms.*` | `field`, `size`, `order`, `missing` |

**Field-type awareness:**
- `match`/`match_phrase` → suggest `text` fields
- `term`/`terms` → suggest `keyword` fields
- `range` → suggest `date`/`numeric` fields

### 5. Mouse Selection

Handle mouse events for cursor positioning and text selection:

**Click behaviors:**
- Single click: position cursor at click location
- Click + drag: select text range
- Double-click: select word
- Triple-click: select entire line
- Shift + click: extend selection from cursor to click position

**Implementation:**
- Translate screen coordinates (X, Y) to text position (line, column)
- Account for line number gutter offset
- Track selection start/end positions
- Render selection with inverted/highlighted background
- Support copy (Ctrl+C / Cmd+C) of selected text

**Visual feedback:**
```
┌──────────────────────────┐
│ Body  ✓                  │
│  1 │ {                   │
│  2 │   "query": {        │
│  3 │     "██████": {}    │  ← selected text highlighted
│  4 │   }                 │
│  5 │ }                   │
└──────────────────────────┘
```

**Scroll on drag:**
- When dragging selection past visible area, auto-scroll the viewport

## New Dependencies

- `github.com/smacker/go-tree-sitter` - Tree-sitter Go bindings
- JSON grammar bundled via go-tree-sitter

## Files Changed

| File | Changes |
|------|---------|
| `internal/ui/editor.go` | New: tree-sitter integration, highlighting, gutter rendering |
| `internal/ui/dsl_schema.go` | New: ES DSL schema for completion |
| `internal/es/validate.go` | New: _validate API wrapper |
| `internal/ui/workbench.go` | Wire up editor, debounced validation |
| `internal/ui/completion.go` | Replace manual parsing with tree-sitter path detection |

## Graceful Degradation

- Tree-sitter parse failure → fall back to plain text
- _validate API failure → show JSON-only validation
- ES version without _validate → skip semantic validation

## Performance

- Tree-sitter parsing: ~1ms for typical queries
- JSON validation: instant
- _validate API: debounced, with request cancellation
