# VHS tape file for recording the streaming Markdown demo.
# Install vhs: https://github.com/charmbracelet/vhs
#
# Usage:
#   task demo:record
#   # or: cd examples/demo && vhs demo.tape
#
# This produces demo.gif in examples/demo/.

Output demo.gif

Set Shell "bash"
Set FontSize 14
Set Width 960
Set Height 600
Set Padding 16
Set Theme "Molokai"
Set TypingSpeed 0ms

# Start the demo with recording-optimized settings.
Type "go run . --record --clear=false"
Enter
Sleep 18s
