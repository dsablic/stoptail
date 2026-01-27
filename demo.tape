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

# Go to Cluster tab (Tab once: Overview -> Cluster)
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

# Show Templates view
Type "7"
Sleep 1.5s

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
