package main

import (
	"fmt"
	"os"
	"time"
	"tradez/internal/game"
	"tradez/internal/market"
	"tradez/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	seed := time.Now().UnixNano()

	m := market.New(seed)
	g := game.New(m)

	model := tui.NewModel(g)

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
