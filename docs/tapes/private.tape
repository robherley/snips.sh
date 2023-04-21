Set Shell "bash"
Set FontSize 18
Set Width 1220
Set Height 500
Set TypingSpeed 100ms

Hide
Type "alias ssh='ssh -q' && clear"
Enter
Show

Type "echo 'my super secret message' | ssh snips.sh -- -private"
Sleep 1.5s
Enter

Sleep 3s

Type "ssh f:hbpp9OCfJA@snips.sh -- sign -ttl 5m"
Sleep 1s
Enter

Sleep 5s
