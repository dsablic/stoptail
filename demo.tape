# VHS demo script for stoptail
# Run: vhs demo.tape
# Prerequisites:
# - Build first: go build .
# - Clear history: rm ~/.stoptail/history.json
# - Uses ./stoptail (local build, not homebrew version)

Output demo.gif

Set FontSize 14
Set Width 1200
Set Height 700
Set Theme "Dracula"
Set TypingSpeed 75ms

# Hide shell prompt, start stoptail, then show
Hide
Type "./stoptail http://localhost:9200"
Enter
Sleep 2s
Show

# Overview tab - scroll through indices
Sleep 500ms
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

# Go to Nodes tab (Tab 3 times: Overview -> Workbench -> Mappings -> Nodes)
Tab
Sleep 500ms
Tab
Sleep 500ms
Tab
Sleep 1s

# Show Memory view with heap% bars (already on view 1)
Sleep 1s

# Show Disk view with disk% bars
Type "2"
Sleep 1.5s

# Show Fielddata view
Type "3"
Sleep 1s

# Back to Memory
Type "1"
Sleep 1s

# Show help (glamour rendered)
Type "?"
Sleep 2s
Type "?"
Sleep 500ms

# Go back to Workbench (Shift+Tab twice: Nodes -> Mappings -> Workbench)
Shift+Tab
Sleep 500ms
Shift+Tab
Sleep 500ms

# Type a path
Ctrl+a
Ctrl+k
Type "/products/_search"
Sleep 500ms

# Tab to body
Tab
Sleep 300ms

# Type a query
Ctrl+a
Ctrl+k
Type@75ms '{"query":{"match_all":{}},"size":5}'
Sleep 500ms

# Execute
Ctrl+Enter
Sleep 1.5s

# Quit - hide the exit
Hide
Type "q"
Sleep 500ms
