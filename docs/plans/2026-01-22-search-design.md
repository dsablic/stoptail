# Search in Text Views Design

## Overview

Add Ctrl+F search functionality to Mappings, Nodes, and Tasks views, matching the existing Workbench search UX.

## Scope

**Views getting search:**
- Mappings (right pane - fields and analyzers)
- Nodes (all views - Memory, Disk, Fielddata)
- Tasks (task list)

**Views excluded:**
- Overview - already has `/` filter for indices
- Workbench - already has Ctrl+F search

## Behavior

Matching Workbench pattern:
- `Ctrl+F` opens search bar at bottom of view
- Type to search (case-insensitive)
- `Enter` jumps to next match
- `Shift+Enter` jumps to previous match
- `Esc` closes search
- Shows "N/M" counter (current/total matches)
- Shows "No matches" when query has no results
- First match auto-jumped to on typing

## Implementation

### Reusable Search Component

Create `internal/ui/search.go` with `SearchBar` struct:
- Wraps `textinput.Model`
- Tracks `matches []int` (line indices) and `currentIdx`
- Provides `Update()`, `View()`, `FindMatches(lines []string, query string)`
- Handles Enter/Shift+Enter/Esc key logic

### Per-View Integration

Each model adds:
```go
search       SearchBar
searchActive bool
```

**Mappings:**
- Search applies to right pane content
- Matches against field names, types, property values
- Updates `mappingScroll` to show matched line

**Nodes:**
- Search applies to table content
- Matches against node names, values, versions
- Updates `scrollY` to show matched line
- Works across all sub-views

**Tasks:**
- Search applies to task list
- Matches against action, description, index, node
- Updates scroll to show matched task

### Status Bar

When search active, status bar shows:
```
Esc: close  Enter: next  Shift+Enter: prev
```

## Files

**New:**
- `internal/ui/search.go`

**Modified:**
- `internal/ui/mappings.go`
- `internal/ui/nodes.go`
- `internal/ui/tasks.go`
- `internal/ui/help.go`
- `README.md`
