# Query Editor Auto-completion Design

**Goal:** Add auto-completion to the Workbench body editor for ES query DSL keywords and index field names.

**Architecture:** Context-aware completion system that triggers on `"` and `:` characters, shows dropdown with suggestions, fetches index mappings for field names.

---

## Trigger Behavior

- After typing `"` at a key position: show DSL keyword completions
- After typing `:`: show value completions (field names, enum values)
- Typing filters the list; backspace reopens if dismissed
- Esc dismisses, Enter/Tab selects

## Data Sources

### Static DSL Keywords

Hardcoded map of context to completions:

| Context | Completions |
|---------|-------------|
| Root | query, aggs, size, from, sort, _source, highlight |
| Inside query | bool, match, match_all, term, terms, range, exists, prefix, wildcard, nested |
| Inside bool | must, should, must_not, filter, minimum_should_match |
| Inside aggs | terms, avg, sum, min, max, cardinality, date_histogram |
| Inside sort | order, mode, nested |

### Index Field Names

- Fetched from `GET /{index}/_mapping` when path contains an index
- Extract index name: first path segment not prefixed with `_`
- Cache per index for session duration
- Fetch async: show static completions immediately, add fields when ready
- On error: silently fall back to static completions only

## UI Component

### Dropdown

- Appears below current cursor line in body editor
- Max 8 items visible, scrollable
- Selected item highlighted with background color
- Shows text and kind hint: `title (field)`, `bool (keyword)`

### Keyboard Navigation

| Key | Action |
|-----|--------|
| Up/Down | Move selection (overrides textarea while open) |
| Enter/Tab | Insert selected, close dropdown |
| Esc | Close without inserting |
| Typing | Filters list, closes if no matches |

### Insertion Behavior

- Key completion after `"`: inserts `key": ` (adds closing quote and colon)
- Value completion after `:`: inserts ` "value"` or bare value for numbers/booleans

## JSON Context Detection

Track cursor position within JSON structure:

```go
type JSONContext struct {
    Path    []string  // e.g., ["query", "bool", "must"]
    InKey   bool
    InValue bool
}
```

Detection logic:
1. Tokenize from start to cursor position
2. Track `{` `}` `[` `]` depth and current key names
3. After `"` with even quote count at object level: key position
4. After `:`: value position

## State Structure

```go
type CompletionState struct {
    active      bool
    items       []CompletionItem
    filtered    []CompletionItem
    selectedIdx int
    triggerPos  int
}

type CompletionItem struct {
    Text string
    Kind string // "keyword", "field", "value"
}
```

## Tab Behavior Fix

**Current:** Tab cycles focus within Workbench (Method → Path → Body → Response)

**New:**
- Tab switches main tabs only when NOT in text input
- Inside body editor: Tab accepts completion (if open), otherwise no-op
- Add `WorkbenchModel.HasActiveInput() bool` for main model to check

## Files to Modify

- `internal/ui/workbench.go` - Add completion state, trigger logic, dropdown rendering
- `internal/ui/completion.go` (new) - CompletionState, DSL keywords map, context detection
- `internal/es/cluster.go` - Add FetchMapping method
- `internal/ui/model.go` - Update tab handling to check HasActiveInput()

## Verification

1. `go test ./...` - all tests pass
2. `go run .` - launch TUI
3. Navigate to Workbench, type `{"` - verify dropdown appears with root keywords
4. Select `query`, type `{"` - verify query-level completions
5. Test with index path `/products/_search` - verify field names appear
6. Test Tab key - verify it switches tabs when not in text input
