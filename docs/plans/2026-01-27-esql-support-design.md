# ES|QL Support in Workbench

## Overview

Add ES|QL query support to the workbench with a toggle to switch between DSL (JSON) and ES|QL modes.

## Design

### Mode Toggle

- New `QueryMode` type: `ModeDSL` (default) and `ModeESSQL`
- Toggle button in toolbar: `[DSL]` / `[ES|QL]`
- Keyboard shortcut: `Ctrl+E` to toggle
- In ES|QL mode: method selector hidden (always POST), path defaults to `/_query`

### Editor Behavior

**DSL mode (current behavior):**
- JSON syntax highlighting
- JSON validation with error indicators
- Keyword/field completion
- Bracket auto-pairing
- Format button works

**ES|QL mode:**
- SQL syntax highlighting (chroma "SQL" lexer)
- No JSON validation
- No completion
- No bracket auto-pairing
- Format button disabled

Each mode preserves its content separately when switching.

### Execution

**DSL mode:** Send method + path + body as-is (current behavior)

**ES|QL mode:**
- Always POST
- Path editable, defaults to `/_query`
- Body wrapped at execution: `{"query": "<escaped content>"}`
- Response displayed normally

### History

- Entries tagged with `mode` field (`"esql"` or empty for DSL)
- History navigation filters by current mode
- Last used mode persisted

### Storage Changes

```go
type HistoryEntry struct {
    Method string `json:"method"`
    Path   string `json:"path"`
    Body   string `json:"body"`
    Mode   string `json:"mode,omitempty"` // "esql" or "" (DSL)
}
```

## Files to Modify

- `internal/ui/workbench.go` - mode toggle, editor behavior, execution
- `internal/storage/history.go` - mode field in HistoryEntry
- `internal/ui/help.go` - document Ctrl+E shortcut
- `README.md` - document ES|QL support

## UI Mockup

```
[ES|QL]  ╭──────────────────────╮  ▶ Run  ◀▶
         │ /_query              │
         ╰──────────────────────╯

╭─ Body (ES|QL) ─────────────────╮  ╭─ Response ─────────────────────╮
│ FROM logs-*                    │  │ {                              │
│ | WHERE @timestamp > NOW()     │  │   "columns": [...],            │
│ | STATS count = COUNT(*)       │  │   "values": [...]              │
│ | LIMIT 10                     │  │ }                              │
╰────────────────────────────────╯  ╰────────────────────────────────╯
```
