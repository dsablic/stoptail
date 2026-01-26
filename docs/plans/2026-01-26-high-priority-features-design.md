# High Priority Features Design

## Overview

Three high-priority features for stoptail:

1. Shard allocation explain modal
2. Index settings toggle in Mappings
3. Cluster settings modal in Overview

---

## 1. Shard Allocation Explain Modal

**Trigger:** In Overview tab, when "Unassigned" row is selected, press `Enter`.

**API:** `GET /_cluster/allocation/explain`
```json
{"index": "<selected-index>", "shard": 0, "primary": true}
```

**Modal content:**
- Index name
- Shard number and type (primary/replica)
- Status
- Unassigned reason (e.g., INDEX_CREATED, NODE_LEFT)
- Allocation decision summary
- Details (e.g., no_valid_shard_copy)

**Implementation:**
- Add `FetchAllocationExplain(ctx, index, shard, primary)` to `internal/es/cluster.go`
- Add modal rendering in `overview.go`
- Handle `Enter` key on unassigned row

---

## 2. Index Settings Toggle in Mappings

**Trigger:** In Mappings tab, press `s` to toggle between mappings and settings.

**API:** `GET /{index}/_settings`

**Settings displayed:**
- number_of_shards, number_of_replicas
- refresh_interval, codec
- creation_date, uuid, version
- routing.allocation settings

**Implementation:**
- Add `viewMode` field to `MappingsModel` (mappings/settings)
- Add `FetchIndexSettings(ctx, index)` to `internal/es/cluster.go`
- Toggle on `s` key
- Update status bar: `s: settings` / `s: mappings`

---

## 3. Cluster Settings Modal in Overview

**Trigger:** In Overview tab, press `S` (shift+s).

**API:** `GET /_cluster/settings?include_defaults=false&flat_settings=true`

**Settings displayed (grouped):**
- Allocation: routing.allocation.enable, rebalance.enable
- Disk Watermarks: low, high, flood
- Recovery: node_concurrent_recoveries, max_bytes_per_sec

**Scope:** Read-only (editing via Workbench is safer).

**Implementation:**
- Add `FetchClusterSettings(ctx)` to `internal/es/cluster.go`
- Add `showClusterSettings` bool to `OverviewModel`
- Render modal on `S` key
- Group settings by category

---

## Documentation Updates Required

- README.md: Add keybindings for Enter (allocation explain), s (settings toggle), S (cluster settings)
- help.go: Update Overview and Mappings help sections
- CLAUDE.md: Update if new patterns introduced
