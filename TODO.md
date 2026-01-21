# TODO

## Custom Text Selection in Query Editor

Implement mouse-based text selection within the body editor that behaves like a GUI:

1. Click and drag to select text (highlighted with background color)
2. Selection stays within the editor bounds (not full terminal lines)
3. Copy selection to clipboard on `Ctrl+C` or mouse release

Implementation approach:
- Track mouse press position as selection start
- On mouse drag, update selection end and re-render with highlight
- Handle mouse release to finalize selection
- Store selected text and copy to clipboard on demand

This requires custom selection handling since terminal native selection spans full lines across the entire terminal width.

## Mappings Tab

Add a new tab to browse and explore index mappings:

- List all indices with their mapping structure
- Tree view of field types and properties
- Search/filter fields by name or type
- Show field details (type, analyzer, index options)
- Possibly allow expanding/collapsing nested objects

## UI Eye-Candy Improvements

Enhance the UI using more Charm ecosystem features:

- Glamour for markdown rendering (help text, documentation)
- Huh for interactive forms (cluster selection, filters)
- Animations/transitions for tab switching
- Better color themes and gradients
- Progress bars for long operations (reindex, etc.)
- Sparklines for metrics visualization
- Better table styling with lipgloss
