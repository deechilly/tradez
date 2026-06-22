Output recording.gif

Set FontSize 13
Set Width 1500
Set Height 675
Set Framerate 20
Set PlaybackSpeed 1.0
Set Theme "GitHub Dark"

# ── launch ────────────────────────────────────────────────────────────────────
Type "./tradez"
Enter
Sleep 3s

# ── slot select ───────────────────────────────────────────────────────────────
# Start a fresh game in Game 1 (n = new game, regardless of existing save)
Type "n"
Sleep 400ms

# ── difficulty & capital ──────────────────────────────────────────────────────
Type "1"
Sleep 400ms

Type "2"
Sleep 7s

# ── market view ───────────────────────────────────────────────────────────────
# Sort by % change descending so the biggest movers are at the top
Type "3"
Sleep 200ms
Type "3"
Sleep 3s

# ── stock detail: chart → analysis ───────────────────────────────────────────
Enter
Sleep 2.5s

Tab
Sleep 2s

Tab
Sleep 3s

# ── buy 50 shares of the top gainer ──────────────────────────────────────────
Escape
Sleep 400ms
Type "b"
Sleep 700ms
Type "50"
Sleep 400ms
Enter
Sleep 3s

# ── navigate to another stock for the put demo ───────────────────────────────
Down
Sleep 250ms
Down
Sleep 250ms
Down
Sleep 400ms
Enter
Sleep 1.5s

# ── put dialog ────────────────────────────────────────────────────────────────
Type "p"
Sleep 1s

Down
Sleep 600ms
Down
Sleep 600ms
Up
Sleep 500ms
Up
Sleep 500ms

Right
Sleep 500ms
Left
Sleep 700ms

Type "2"
Sleep 400ms
Enter
Sleep 2.5s

# ── portfolio: see the stock position and the open put ────────────────────────
Type "p"
Sleep 2s

Down
Sleep 1.5s

Escape
Sleep 500ms

# ── glossary: see the help screen and glossry of terms ────────────────────────

Type "?"
Sleep 500ms

PageDown
Sleep 1s

Down
Sleep 1s

Down
Sleep 1s

PageDown
Sleep 1s

Escape
Sleep 2s

# ── save to Game 1 ────────────────────────────────────────────────────────────
Ctrl+S
Sleep 2.5s

# ── return to slot select: Game 1 now shows saved metadata ───────────────────
Escape
Sleep 3s

# ── quit ─────────────────────────────────────────────────────────────────────
Type "q"
Sleep 500ms
