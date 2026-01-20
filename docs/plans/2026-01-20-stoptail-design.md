# stoptail Design

A TUI Elasticsearch client with cluster visualization and request workbench. Think elasticsearch-head, but as a fast terminal app.

## Goals

- Cluster overview with the iconic shard grid visualization
- Full request tool (any method, any path, any body)
- Fast, local, no CORS issues
- ES 8.x and 9.x support

## Tech Stack

- **Framework:** `github.com/charmbracelet/bubbletea`
- **Components:** `github.com/charmbracelet/bubbles` (table, textarea, viewport, textinput)
- **Styling:** `github.com/charmbracelet/lipgloss`
- **JSON Highlighting:** `github.com/alecthomas/chroma`
- **ES Client:** `github.com/elastic/go-elasticsearch/v8`

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  stoptail · https://elastic:***@localhost:9200   [Connected]│
├─────────────────────────────────────────────────────────────┤
│  [Overview]  [Workbench]                              Tab/← →│
├─────────────────────────────────────────────────────────────┤
│                                                             │
│                     Active View                             │
│                                                             │
├─────────────────────────────────────────────────────────────┤
│  Filter: collections_*                        q:quit  ?:help│
└─────────────────────────────────────────────────────────────┘
```

**Main Model Structure:**
- `activeTab` (0=Overview, 1=Workbench)
- `overview` sub-model (shard grid, filters)
- `workbench` sub-model (method, path, body, response)
- `client` (ES connection)
- `cluster` (cached cluster state)

## Connection & Configuration

**URL format:**
```
https://username:password@hostname:9200
```

**Configuration precedence (highest to lowest):**
1. CLI flag: `--url`
2. Environment variable: `ES_URL`
3. Default: `http://localhost:9200` (no auth)

**Usage:**
```bash
# From env
export ES_URL="https://elastic:secret@prod.example.com:9200"
stoptail

# Override with flag
stoptail --url "https://elastic:dev@localhost:9200"

# Local dev (no auth)
stoptail --url "http://localhost:9200"
```

**Connection handling:**
- On startup: validate connection with `GET /`
- Show connection error screen if unreachable
- Header shows masked URL (`elastic:***@host:9200`) + status
- No auto-reconnect for v1

## Overview Tab - Shard Grid

The signature elasticsearch-head visualization:

```
┌─────────────────────────────────────────────────────────────────────┐
│ Filter: [collections_*        ]  Aliases: [1:collections] [2:embed] │
├──────────┬──────────────────────────────────────────────────────────┤
│          │ collections_v1    collections_v2    embeddings_v3        │
│          │ 745G · 1.54G docs  980G · 1.54G     393G · 20.7M         │
│          │ [collections]      [collections]    [embeddings]         │
├──────────┼──────────────────────────────────────────────────────────┤
│ip-10-0-  │   ┌0─┐ ┌1─┐        ┌0─┐ ┌2─┐        ┌0─┐ ┌3─┐           │
│11-150    │   │P │ │R │        │R │ │P │        │P │ │R │           │
│  ● healthy   └──┘ └──┘        └──┘ └──┘        └──┘ └──┘           │
├──────────┼──────────────────────────────────────────────────────────┤
│ip-10-0-  │   ┌2─┐ ┌3─┐        ┌1─┐ ┌3─┐        ┌1─┐ ┌2─┐           │
│11-151    │   │P │ │R │        │P │ │R │        │R │ │P │           │
│  ● healthy   └──┘ └──┘        └──┘ └──┘        └──┘ └──┘           │
└──────────┴──────────────────────────────────────────────────────────┘
```

**Shard box colors:**
- Green = Primary shard (P)
- Blue = Replica shard (R)
- Yellow = Relocating
- Red = Unassigned

**Index header colors:**
- Green = healthy
- Yellow = yellow status
- Red = red status

**Interactions:**
- `/` - Focus filter input
- `Esc` - Clear filter / unfocus
- `←` `→` `↑` `↓` - Navigate grid
- `Enter` - Open selected index in Workbench
- `1` `2` `3`... - Toggle alias filters
- `r` - Refresh

**Data sources:**
- `GET /_cat/indices?format=json`
- `GET /_cat/nodes?format=json`
- `GET /_cat/shards?format=json`
- `GET /_cat/aliases?format=json`

## Workbench Tab - Request Tool

Split-pane layout:

```
┌────────────────────────────────────────┬────────────────────────────────────────┐
│ [GET ▼] [/collections_v1/_search     ] │  Response                    200 OK 45ms│
├────────────────────────────────────────┼────────────────────────────────────────┤
│ {                                      │ {                                      │
│   "query": {                           │   "took": 5,                           │
│     "bool": {                          │   "timed_out": false,                  │
│       "must": [                        │   "_shards": {                         │
│         { "match": { "title": "foo" }} │     "total": 5,                        │
│       ]                                │     "successful": 5                    │
│     }                                  │   },                                   │
│   },                                   │   "hits": {                            │
│   "size": 10                           │     "total": { "value": 142 },         │
│ }                                      │     "hits": [...]                      │
│                                        │   }                                    │
│                                        │ }                                      │
├────────────────────────────────────────┴────────────────────────────────────────┤
│ [✓ Valid JSON]                                            Ctrl+Enter: Execute   │
└─────────────────────────────────────────────────────────────────────────────────┘
```

**Left pane (Request):**
- Method selector: GET, POST, PUT, DELETE, HEAD
- Path input: textinput
- Body editor: textarea
- JSON validation indicator (green ✓ / red ✗)

**Right pane (Response):**
- Syntax-highlighted JSON via chroma
- Status: HTTP code + response time
- Scrollable viewport

**Interactions:**
- `Tab` - Cycle focus: method → path → body → response
- `Ctrl+M` - Cycle HTTP method
- `Ctrl+Enter` - Execute request
- `Ctrl+L` - Clear body
- `Ctrl+P` - Pretty-print body
- `y` (in response) - Copy to clipboard

**Smart defaults:**
- From Overview Enter: pre-fills `GET /{index}/_search` with `{}`
- Paths ending in `_search` default to POST

## Key Bindings

### Global
| Key | Action |
|-----|--------|
| `Tab` | Switch Overview ↔ Workbench |
| `q` / `Ctrl+C` | Quit |
| `?` | Toggle help overlay |
| `r` | Refresh data |

### Overview
| Key | Action |
|-----|--------|
| `/` | Focus filter |
| `Esc` | Clear filter / unfocus |
| `←` `→` `↑` `↓` | Navigate grid |
| `Enter` | Open index in Workbench |
| `1` `2` `3`... | Toggle alias filter |

### Workbench
| Key | Action |
|-----|--------|
| `Tab` | Cycle focus |
| `Ctrl+M` | Cycle HTTP method |
| `Ctrl+Enter` | Execute |
| `Ctrl+L` | Clear body |
| `Ctrl+P` | Pretty-print |

### Response Viewport
| Key | Action |
|-----|--------|
| `↑` `↓` `PgUp` `PgDn` | Scroll |
| `y` | Copy to clipboard |

## Project Structure

```
stoptail/
├── main.go                 # Entry point, CLI flags, ES client init
├── internal/
│   ├── ui/
│   │   ├── model.go        # Root bubbletea model, tab switching
│   │   ├── overview.go     # Overview tab model + shard grid
│   │   ├── workbench.go    # Workbench tab model + split pane
│   │   ├── styles.go       # lipgloss styles
│   │   └── help.go         # Help overlay
│   ├── es/
│   │   ├── client.go       # ES client wrapper, URL parsing
│   │   ├── indices.go      # Fetch indices
│   │   ├── nodes.go        # Fetch nodes
│   │   ├── shards.go       # Fetch shards
│   │   └── request.go      # Generic request executor
│   └── config/
│       └── config.go       # URL parsing, env/flag handling
├── go.mod
└── go.sum
```

## Data Flow

**Startup:**
```
parse URL (flag > env > default)
→ create ES client
→ GET / (validate connection)
→ fetch indices + nodes + shards + aliases
→ build grid model
→ render
```

**Overview → Workbench:**
```
Enter on index
→ set workbench.method = "GET"
→ set workbench.path = "/{index}/_search"
→ set workbench.body = "{}"
→ switch to Workbench tab
```

**Execute request:**
```
Ctrl+Enter
→ validate JSON body
→ execute via ES client
→ capture response + timing
→ syntax highlight with chroma
→ render in viewport
```

## Out of Scope (v1)

- Multi-cluster support
- API keys / Elastic Cloud auth
- Index creation/deletion from UI
- Saved queries
- Auto-reconnect
- Real-time input syntax highlighting
