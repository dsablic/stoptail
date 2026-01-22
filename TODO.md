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

## UI Eye-Candy Improvements

Enhance the UI using more Charm ecosystem features:

- [x] Glamour for markdown rendering (help text)
- [x] Huh for interactive forms (create/delete index, add/remove alias)
- [x] Progress spinners for ES operations
- [x] Bar charts for metrics visualization (heap%, disk%)
- [x] Better table styling with alternating row backgrounds
- [x] Gradient bars with color transitions
- Animations/transitions for tab switching (limited in terminal UIs)
