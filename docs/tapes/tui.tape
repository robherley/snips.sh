Set Shell "bash"
Set FontSize 20
Set Width 1400
Set Height 830
Set TypingSpeed 100ms

# route `ssh snips.sh` to the local demo server (see record-tui.sh)
Hide
Type `alias ssh='ssh -F "$SNIPS_DEMO_SSH_CONFIG"'`
Enter
Type "clear"
Enter
Show

Type "ssh snips.sh"
Sleep 1s
Enter
Sleep 2.5s

# browse the list
Down@400ms 3
Up@400ms 1
Sleep 1s

# view a file
Enter
Sleep 3s
Escape
Sleep 1s

# the options window
Tab
Sleep 2.5s
Down@800ms 2
Sleep 1.5s
Escape
Sleep 1s

# settings: pick a new theme color
Ctrl+p
Sleep 1.5s
Enter
Sleep 1s
Down@600ms 3
Sleep 1s
Enter
Sleep 1.5s
Ctrl+p
Sleep 3s
