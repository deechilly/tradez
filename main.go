package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"tradez/internal/livebridge"
	"tradez/internal/mcpserver"
	"tradez/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "mcp" {
		if err := runMCP(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if err := runTUI(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runTUI() error {
	bridge, err := livebridge.NewServer()
	if err != nil {
		return err
	}
	model := tui.NewModel(tui.WithBridge(bridge))
	p := tea.NewProgram(model, tea.WithAltScreen())
	bridge.SetSender(p)
	if err := bridge.Start(); err != nil {
		return err
	}
	defer bridge.Close()
	_, err = p.Run()
	return err
}

func runMCP(args []string) error {
	fs := flag.NewFlagSet("tradez mcp", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	readOnly := fs.Bool("read-only", false, "omit mutating MCP tools")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return mcpserver.Run(context.Background(), mcpserver.Options{ReadOnly: *readOnly})
}
