# vhs docs/demo.tape

Output docs/demo.gif

Set FontSize 14
Set Width 1000
Set Height 600
Set Theme "Catppuccin Mocha"

# use the locally built binary
Hide
Type 'export PATH="$PWD/bin:$PATH"'
Enter
Type "clear"
Enter
Show

Type "cronkit explain '*/15 2-5 * * 1-5'"
Enter
Sleep 2s

Type "cronkit next '0 9 * * 1-5' --count 3"
Enter
Sleep 3s

Type "cronkit timeline '*/15 * * * *' --view day --width 90"
Enter
Sleep 3s

Type "cronkit check '0 0 1 * 1' --verbose"
Enter
Sleep 3s
