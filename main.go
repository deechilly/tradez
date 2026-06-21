package main

import (
	"fmt"
	"os"
	"tradez/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	model := tui.NewModel()
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
