# Index and Alias Management Design

## Overview

Add index and alias management capabilities to the Overview tab via keyboard shortcuts and centered modal dialogs.

## Keybindings

| Key | Action | Confirmation |
|-----|--------|--------------|
| `c` | Create index | 3-step modal: name, shards, replicas |
| `d` | Delete index | Type index name to confirm |
| `a` | Add alias | Enter alias name |
| `A` | Remove alias | Enter alias name |

All actions except `c` require an index to be selected.

## Modal Dialog

**Appearance:**
- Centered box with rounded border
- Title at top: "Create Index", "Delete Index", "Add Alias", "Remove Alias"
- Input field(s) below title
- Help text at bottom: "Enter: confirm | Esc: cancel"

**Create Index flow (3-step):**
1. "Index name:" - required
2. "Number of shards (default 1):" - empty uses default
3. "Number of replicas (default 1):" - empty uses default

**Delete Index flow:**
1. "Type 'index-name' to confirm deletion:"
2. Must type exact name to proceed
3. Mismatch shows error, stays in modal

**Add/Remove Alias flow:**
1. "Alias name:" with text input
2. Enter confirms, Esc cancels

## ES API Calls

**Create Index:**
```
PUT /{index-name}
{
  "settings": {
    "number_of_shards": <shards>,
    "number_of_replicas": <replicas>
  }
}
```

**Delete Index:**
```
DELETE /{index-name}
```

**Add Alias:**
```
POST /_aliases
{
  "actions": [
    { "add": { "index": "<index>", "alias": "<alias>" } }
  ]
}
```

**Remove Alias:**
```
POST /_aliases
{
  "actions": [
    { "remove": { "index": "<index>", "alias": "<alias>" } }
  ]
}
```

## Error Handling

- Show error in modal (e.g., "Error: index already exists")
- User presses Esc to dismiss and return to Overview
- Refresh index list automatically on success

## Implementation

**New files:**
- `internal/ui/modal.go` - Reusable modal dialog with text input

**Modified files:**
- `internal/es/cluster.go` - Add CreateIndex, DeleteIndex, AddAlias, RemoveAlias
- `internal/ui/overview.go` - Add modal state, keybindings, action handling
- `internal/ui/help.go` - Document new keybindings
- `README.md` - Document new keybindings

**State in OverviewModel:**
```go
modal        *Modal
modalAction  string      // "create", "delete", "addAlias", "removeAlias"
modalStep    int         // for multi-step create flow
createState  CreateState // holds name, shards, replicas
```
