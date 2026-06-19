package tui

import "github.com/charmbracelet/lipgloss"

var (
	colorGreen  = lipgloss.Color("#00d26a")
	colorRed    = lipgloss.Color("#ff4757")
	colorYellow = lipgloss.Color("#ffa502")
	colorBlue   = lipgloss.Color("#1e90ff")
	colorGray   = lipgloss.Color("#636e72")
	colorWhite  = lipgloss.Color("#dfe6e9")
	colorBg     = lipgloss.Color("#0d1117")
	colorBg2    = lipgloss.Color("#161b22")
	colorBorder = lipgloss.Color("#30363d")
	colorAccent = lipgloss.Color("#58a6ff")
	colorGold   = lipgloss.Color("#e6c87e")

	styleBase = lipgloss.NewStyle().
			Background(colorBg).
			Foreground(colorWhite)

	styleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent).
			Background(colorBg)

	styleBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder)

	stylePositive = lipgloss.NewStyle().Foreground(colorGreen)
	styleNegative = lipgloss.NewStyle().Foreground(colorRed)
	styleNeutral  = lipgloss.NewStyle().Foreground(colorGray)
	styleTitle    = lipgloss.NewStyle().Bold(true).Foreground(colorGold)
	styleSelected = lipgloss.NewStyle().
			Background(lipgloss.Color("#1f2937")).
			Foreground(colorAccent).
			Bold(true)

	styleTab = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(colorGray)
	styleTabActive = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(colorAccent).
			Bold(true).
			Underline(true)

	styleHint = lipgloss.NewStyle().Foreground(colorGray).Italic(true)

	styleInput = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent).
			Padding(0, 1)

	styleError = lipgloss.NewStyle().Foreground(colorRed).Bold(true)
	styleOk    = lipgloss.NewStyle().Foreground(colorGreen).Bold(true)
)

func colorForChange(pct float64) lipgloss.Style {
	if pct > 0 {
		return stylePositive
	} else if pct < 0 {
		return styleNegative
	}
	return styleNeutral
}

func signStr(v float64) string {
	if v > 0 {
		return "+"
	}
	return ""
}
