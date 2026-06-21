package mcpserver

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestReadOnlyModeOmitsMutatingTools(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	server := mcp.NewServer(&mcp.Implementation{Name: "tradez-test"}, nil)
	(&Server{readOnly: true}).registerTools(server)

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Run(ctx, serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "tradez-test-client"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer session.Close()

	result, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}

	tools := make(map[string]bool)
	for _, tool := range result.Tools {
		tools[tool.Name] = true
	}
	for _, name := range []string{"get_game_state", "list_stocks", "get_stock", "get_news", "get_portfolio", "get_orders"} {
		if !tools[name] {
			t.Fatalf("missing read tool %s", name)
		}
	}
	for _, name := range []string{"place_order", "cancel_order", "save_game"} {
		if tools[name] {
			t.Fatalf("mutating tool %s should not be registered in read-only mode", name)
		}
	}

	cancel()
	select {
	case <-errCh:
	default:
	}
}
