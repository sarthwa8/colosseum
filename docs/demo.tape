# Regenerate the demo GIF with:  vhs docs/demo.tape   (run from the repo root)
Output docs/demo.gif

Set FontSize 15
Set Width 1080
Set Height 780
Set Theme "Catppuccin Mocha"
Set Padding 24
Set Margin 10
Set BorderRadius 8

# Start from a clean prompt (hidden setup)
Hide
Type "clear"
Enter
Show
Sleep 600ms

# The wow demo: one model breaks the other's code, judged in a real sandbox.
Type "./colosseum match --problem max-subarray --a mock:wrong --b mock:reference --format ad"
Sleep 1200ms
Enter
Sleep 6s
