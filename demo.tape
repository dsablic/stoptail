# VHS demo script for stoptail
# Run: vhs demo.tape

Output demo.gif

Set FontSize 14
Set Width 1200
Set Height 700
Set Theme "Dracula"
Set TypingSpeed 75ms

# Start stoptail
Type "stoptail"
Enter
Sleep 3s

# Overview tab - scroll through indices
Type "l"
Sleep 400ms
Type "l"
Sleep 400ms
Type "l"
Sleep 400ms
Type "j"
Sleep 400ms
Type "j"
Sleep 400ms

# Filter indices
Type "/"
Sleep 300ms
Type "logs"
Sleep 1s
Escape
Sleep 500ms

# Clear filter
Escape
Sleep 1s

# Switch to Workbench tab
Tab
Sleep 1s

# Type a search query
Tab
Sleep 300ms
Type "/products/_search"
Sleep 500ms

# Add request body
Tab
Sleep 300ms
Type@50ms '{"query":{"match_all":{}},"size":5}'
Sleep 500ms

# Execute query
Ctrl+e
Sleep 2s

# Switch to Nodes tab
Tab
Sleep 1s

# Show different views
Type "2"
Sleep 1.5s
Type "3"
Sleep 1.5s
Type "4"
Sleep 1.5s
Type "1"
Sleep 1.5s

# Show help
Type "?"
Sleep 2s
Type "?"
Sleep 500ms

# Quit
Type "q"
Sleep 500ms
