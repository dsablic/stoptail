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

# Go to Mappings tab directly (press 'm')
Type "m"
Sleep 1s

# Navigate and select an index
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 300ms
Enter
Sleep 1.5s

# Search in mappings with Ctrl+F
Ctrl+f
Sleep 300ms
Type "name"
Sleep 800ms
Enter
Sleep 400ms
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
Sleep 1s
Type "3"
Sleep 1s
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
