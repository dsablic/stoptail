# VHS demo script for stoptail
# Run: vhs demo.tape

Output demo.gif

Set FontSize 14
Set Width 1200
Set Height 700
Set Theme "Dracula"
Set TypingSpeed 75ms

# Hide shell prompt, start stoptail, then show
Hide
Type "stoptail http://localhost:9200"
Enter
Sleep 2s
Show

# Overview tab - scroll through indices
Type "l"
Sleep 400ms
Type "l"
Sleep 400ms
Type "l"
Sleep 400ms

# Filter indices
Type "/"
Sleep 300ms
Type "logs"
Sleep 1s

# Clear filter
Escape
Sleep 500ms
Escape
Sleep 1s

# Switch to Workbench tab (focus is automatically on path)
Tab
Sleep 1s

# Go to start of path and prepend "/products" (default path is /_search)
Ctrl+a
Type "/products"
Sleep 500ms

# Tab to body, clear it, then type query
Tab
Sleep 300ms
Ctrl+l
Sleep 200ms
Type@50ms '{"query":{"match_all":{}},"size":5}'
Sleep 500ms

# Execute query
Ctrl+e
Sleep 2s

# Search in response with Ctrl+F
Ctrl+f
Sleep 300ms
Type "name"
Sleep 1s
Enter
Sleep 500ms
Enter
Sleep 500ms
Escape
Sleep 500ms

# Tab to response, then to Mappings tab
Tab
Sleep 300ms
Tab
Sleep 1s

# Select an index and load mappings
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms
Enter
Sleep 1.5s

# Search in mappings with Ctrl+F
Ctrl+f
Sleep 300ms
Type "price"
Sleep 1s
Escape
Sleep 500ms

# Toggle tree view
Type "t"
Sleep 1s
Type "t"
Sleep 500ms

# Go to Nodes tab
Tab
Sleep 1s

# Show different views
Type "2"
Sleep 1.5s
Type "3"
Sleep 1.5s
Type "1"
Sleep 1s

# Show help
Type "?"
Sleep 2s
Type "?"
Sleep 500ms

# Quit - hide the exit
Hide
Type "q"
Sleep 500ms
