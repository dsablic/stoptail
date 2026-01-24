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

# Overview tab - show primary/total sizes and document counts
Sleep 1s

# Scroll through indices
Type "l"
Sleep 400ms
Type "l"
Sleep 400ms
Type "l"
Sleep 800ms

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

# Go to Browser tab (Tab twice: Overview -> Workbench -> Browser)
Tab
Sleep 500ms
Tab
Sleep 1s

# Select an index and load documents
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms
Enter
Sleep 1.5s

# Navigate through documents
Type "j"
Sleep 400ms
Type "j"
Sleep 400ms
Type "j"
Sleep 400ms

# Move to detail pane
Type "l"
Sleep 1s

# Back to doc list
Type "h"
Sleep 500ms

# Go to Nodes tab (Tab twice: Browser -> Mappings -> Nodes)
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

# Quit - hide the exit
Hide
Type "q"
Sleep 500ms
