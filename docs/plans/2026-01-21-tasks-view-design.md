# Tasks View Design

**Goal:** Add a "Tasks" tab showing long-running cancellable operations (reindex, snapshot, force merge, update-by-query, delete-by-query) with cancel functionality.

---

## Phase 1: Data Layer

**File:** `internal/es/cluster.go`

### TaskInfo struct

```go
type TaskInfo struct {
    ID            string
    Action        string
    Node          string
    Index         string
    RunningTime   string
    RunningTimeMs int64
    Description   string
    Cancellable   bool
    Progress      float64
}
```

### FetchTasks method

- Calls `_tasks?detailed=true&actions=*reindex*,*snapshot*,*forcemerge*,*update/byquery*,*delete/byquery*`
- Parses nested JSON (nodes -> tasks), flattens to slice
- Formats running time from nanos to human-readable (e.g., "2m 34s")
- Extracts progress from task status when available

### CancelTask method

- POST to `_tasks/{taskID}/_cancel`
- Returns error if cancel fails

---

## Phase 2: UI Component

**File:** `internal/ui/tasks.go` (new)

### TasksModel struct

```go
type TasksModel struct {
    tasks         []es.TaskInfo
    selectedRow   int
    scrollY       int
    width, height int
    loading       bool
    confirming    string
}
```

### Table columns

| Action | Index | Node | Running | Progress | Cancel |
|--------|-------|------|---------|----------|--------|

### Keybindings

- `up/down` or `j/k` - navigate rows
- `c` - initiate cancel on selected task
- `y` - confirm cancel
- `n` or `Esc` - abort cancel
- `r` - refresh (manual only)

### Cancel confirmation UX

- `c` sets `confirming` to task ID
- Row highlights red, bottom shows "Cancel task? y/n"
- `y` calls CancelTask, clears confirming, refreshes
- `n` or `Esc` clears confirming state

---

## Phase 3: Integration

**File:** `internal/ui/model.go`

### Tab constant

```go
const (
    TabOverview = iota
    TabWorkbench
    TabNodes
    TabTasks
)
```

### Message types

- `tasksMsg` - carries fetched []TaskInfo
- `taskCancelledMsg` - confirms cancel succeeded/failed

### Tab cycling

- Tab: Overview -> Workbench -> Nodes -> Tasks -> Overview
- Shift+Tab: reverse

---

## Phase 4: Documentation

**File:** `internal/ui/help.go`

Add Tasks section:
- c: Cancel selected task
- y/n: Confirm/abort cancel
- up/down: Navigate tasks
- r: Refresh

**File:** `README.md`

- Add Tasks tab to Features section
- Add Tasks keybindings table

---

## Verification

1. `go test ./...` - all tests pass
2. `docker compose up -d` - start ES
3. Start a long-running task: `curl -X POST "localhost:9200/_reindex?wait_for_completion=false" -H 'Content-Type: application/json' -d '{"source":{"index":"products"},"dest":{"index":"products-copy"}}'`
4. `go run .` - launch TUI
5. Tab to Tasks tab - verify task appears
6. Press c, then y - verify cancel works
7. Press r - verify refresh
