# VHS script for harific demo
# This script demonstrates the TUI features including search and navigation

# Setup
Output harific-demo.gif
Set FontSize 14
Set Width 1200
Set Height 800
Set Theme "Dracula"

# Start harific with the ultra.mega.har file
Type "./bin/harific ultra.mega.har"
Enter
Sleep 3s  # Wait for the large file to load

# Initial display - let user see the entry list
Sleep 1s

# Scroll down through entries for 3 seconds
Down@100ms 30  # Scroll down repeatedly over 3 seconds
Sleep 500ms

# Open global search with Ctrl+F
Ctrl+F
Sleep 200ms

# Type search term
Type "/search"
Sleep 500ms

# Close search with Escape
Escape

# Navigate to second result
PageUp
Down
Sleep 500ms

# Open detail view of selected entry
Enter
Sleep 1s

# Navigate through detail view tabs
Tab
Sleep 500ms
Enter
Sleep 1s

# Scroll down in detail view for 2 seconds
Down@100ms 20  # Scroll down repeatedly over 2 seconds
Sleep 500ms

# Open search in detail view with Ctrl+F
Ctrl+F
Sleep 200ms

# Type "event" to search in the JSON
Type "event"
Sleep 500ms

# Toggle filtered view
Enter
Sleep 1s

# Toggle back to full view
Enter
Sleep 1s

# Exit search mode
Escape

Ctrl+F
Sleep 200ms

# Type "event" to search in the JSON
Type "ttl"
Sleep 500ms

# Toggle filtered view
Enter
Sleep 2s

# Toggle back to full view
Enter
Sleep 2s

# Exit search mode
Escape
Sleep 300ms

# Exit detail modal
Escape
Sleep 300ms

# Exit back to main list
Escape
Sleep 100ms

# One more escape to be sure we're at the main level
Escape

# Quit the application
Type "q"

Sleep 500ms