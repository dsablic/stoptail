# Mappings Tab Design

## Overview

Add a new Mappings tab to browse and explore index mappings and custom analyzers.

## Layout

**Tab Position:** Between Workbench and Nodes in the tab bar.

**Two-Pane Split:**
- Left pane (~25% width): Index list with filter
- Right pane (~75% width): Selected index mappings and analyzers

## Left Pane - Index List

- Index names sorted alphabetically
- Filter input at top (activated with `/`)
- Arrow keys to navigate, Enter or `l`/`→` to select
- Selected index highlighted
- Shows field count: `products (24 fields)`

## Right Pane - Mappings View

### Fields Section

**Flat view (default):**
```
user.name                text      analyzer=custom_text
user.email               keyword
user.address.city        keyword
user.address.zip         keyword   index=false
created_at               date
price                    float
tags                     keyword   doc_values=false
```

**Tree view (toggle with `t`):**
```
▼ user                   object
    name                 text      analyzer=custom_text
    email                keyword
  ▼ address              object
      city               keyword
      zip                keyword   index=false
created_at               date
price                    float
tags                     keyword   doc_values=false
```

**Columns:**
- Field name: left-aligned
- Type: left-aligned, ~10 chars
- Attributes: dimmed, only non-defaults

**Non-default attributes to display:**
- `analyzer`, `search_analyzer`
- `index=false`
- `doc_values=false`
- `norms=false`
- `store=true`
- `null_value=X`

### Analyzers Section

Appears below fields when custom analyzers exist:

```
─── Custom Analyzers ───────────────────────────────────

custom_text              analyzer
  tokenizer: standard
  filter: lowercase, snowball

edge_ngram_tokenizer     tokenizer
  type: edge_ngram
  min_gram: 2, max_gram: 10
```

Shows:
- Custom analyzers with tokenizer + filters
- Custom tokenizers with key settings
- Custom filters with type and key settings

## Keyboard Navigation

| Key | Action |
|-----|--------|
| `Tab`/`Shift+Tab` | Switch between app tabs |
| `/` | Activate filter in left pane |
| `←`/`→` or `h`/`l` | Switch focus between panes |
| `↑`/`↓` | Navigate/scroll within focused pane |
| `t` | Toggle flat/tree view |
| `r` | Refresh current index mappings |
| `Esc` | Clear filter / deactivate |

## Data Fetching

**ES API Calls:**
- Index list: Reuse `FetchClusterState` (already has indices)
- Mappings: `GET /{index}/_mapping`
- Settings: `GET /{index}/_settings` (for custom analyzers)

**Strategy:**
- Fetch on index selection (not upfront)
- Cache mappings per index during session
- `r` refreshes current index

## Data Structures

```go
type MappingField struct {
    Name       string
    Type       string
    Properties map[string]string
    Children   []MappingField
}

type AnalyzerInfo struct {
    Name     string
    Kind     string // "analyzer", "tokenizer", "filter"
    Settings map[string]string
}

type IndexMappings struct {
    IndexName string
    Fields    []MappingField
    Analyzers []AnalyzerInfo
}
```

## Implementation

New file: `internal/ui/mappings.go`

Follow existing patterns from `nodes.go` and `tasks.go`:
- `MappingsModel` struct
- `NewMappings()` constructor
- `SetSize()`, `Update()`, `View()` methods
- Add `TabMappings` constant to `model.go`
- Wire up in main `Model` struct and tab switching logic
