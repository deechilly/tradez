package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"tradez/internal/livebridge"
	"tradez/internal/snapshot"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Options struct {
	ReadOnly bool
}

type emptyInput struct{}

type stockInput struct {
	Symbol string `json:"symbol" jsonschema:"ticker symbol to inspect"`
}

type cancelInput struct {
	ID int `json:"id" jsonschema:"pending order id to cancel"`
}

type Server struct {
	bridge   *livebridge.Client
	readOnly bool
}

func Run(ctx context.Context, opts Options) error {
	bridge, err := livebridge.Connect()
	if err != nil {
		return err
	}
	s := &Server{bridge: bridge, readOnly: opts.ReadOnly}
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "tradez",
		Version: "v0.1.0",
	}, nil)
	s.registerResources(server)
	s.registerPrompts(server)
	s.registerTools(server)
	return server.Run(ctx, &mcp.StdioTransport{})
}

func (s *Server) registerTools(server *mcp.Server) {
	closedWorld := false
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_game_state",
		Title:       "Get Game State",
		Description: "Return the active tradez save slot, cash, portfolio value, P&L, news timer, and summary counts.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:  true,
			OpenWorldHint: &closedWorld,
		},
	}, s.getGameState)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_stocks",
		Title:       "List Stocks",
		Description: "Return all generated stocks with prices, fundamentals, current influence strength, and recommendations.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:  true,
			OpenWorldHint: &closedWorld,
		},
	}, s.listStocks)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_stock",
		Title:       "Get Stock",
		Description: "Return detailed information, price history, position data, and analysis for one ticker symbol.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:  true,
			OpenWorldHint: &closedWorld,
		},
	}, s.getStock)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_news",
		Title:       "Get News",
		Description: "Return recent market news events and their sector or symbol effects.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:  true,
			OpenWorldHint: &closedWorld,
		},
	}, s.getNews)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_portfolio",
		Title:       "Get Portfolio",
		Description: "Return cash, total portfolio value, P&L, and all long and short positions.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:  true,
			OpenWorldHint: &closedWorld,
		},
	}, s.getPortfolio)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_orders",
		Title:       "Get Orders",
		Description: "Return pending, filled, cancelled, and historical orders.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:  true,
			OpenWorldHint: &closedWorld,
		},
	}, s.getOrders)

	if s.readOnly {
		return
	}

	destructive := true
	mcp.AddTool(server, &mcp.Tool{
		Name:        "place_order",
		Title:       "Place Order",
		Description: "Place a market, limit, short, or cover order in the active live tradez game.",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: &destructive,
			OpenWorldHint:   &closedWorld,
			ReadOnlyHint:    false,
		},
	}, s.placeOrder)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "cancel_order",
		Title:       "Cancel Order",
		Description: "Cancel a pending limit order and release reserved cash or shares.",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: &destructive,
			OpenWorldHint:   &closedWorld,
			ReadOnlyHint:    false,
		},
	}, s.cancelOrder)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "save_game",
		Title:       "Save Game",
		Description: "Save the active live game to its current save slot.",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: &destructive,
			IdempotentHint:  true,
			OpenWorldHint:   &closedWorld,
			ReadOnlyHint:    false,
		},
	}, s.saveGame)
}

func (s *Server) getGameState(ctx context.Context, _ *mcp.CallToolRequest, _ emptyInput) (*mcp.CallToolResult, snapshot.GameState, error) {
	var out snapshot.GameState
	err := s.bridge.Call(ctx, livebridge.OpGameState, emptyInput{}, &out)
	return textResult(gameStateText(out)), out, err
}

func (s *Server) listStocks(ctx context.Context, _ *mcp.CallToolRequest, _ emptyInput) (*mcp.CallToolResult, struct {
	Stocks []snapshot.StockSummary `json:"stocks"`
}, error) {
	var stocks []snapshot.StockSummary
	err := s.bridge.Call(ctx, livebridge.OpListStocks, emptyInput{}, &stocks)
	out := struct {
		Stocks []snapshot.StockSummary `json:"stocks"`
	}{Stocks: stocks}
	return textResult(fmt.Sprintf("Returned %d stocks.", len(stocks))), out, err
}

func (s *Server) getStock(ctx context.Context, _ *mcp.CallToolRequest, in stockInput) (*mcp.CallToolResult, snapshot.StockSummary, error) {
	var out snapshot.StockSummary
	err := s.bridge.Call(ctx, livebridge.OpGetStock, livebridge.StockRequest{Symbol: in.Symbol}, &out)
	return textResult(fmt.Sprintf("%s %s: $%.2f, %s.", out.Symbol, out.Name, out.Price, out.Recommendation)), out, err
}

func (s *Server) getNews(ctx context.Context, _ *mcp.CallToolRequest, _ emptyInput) (*mcp.CallToolResult, struct {
	News []snapshot.NewsEvent `json:"news"`
}, error) {
	var news []snapshot.NewsEvent
	err := s.bridge.Call(ctx, livebridge.OpGetNews, emptyInput{}, &news)
	out := struct {
		News []snapshot.NewsEvent `json:"news"`
	}{News: news}
	return textResult(fmt.Sprintf("Returned %d news events.", len(news))), out, err
}

func (s *Server) getPortfolio(ctx context.Context, _ *mcp.CallToolRequest, _ emptyInput) (*mcp.CallToolResult, snapshot.Portfolio, error) {
	var out snapshot.Portfolio
	err := s.bridge.Call(ctx, livebridge.OpPortfolio, emptyInput{}, &out)
	return textResult(fmt.Sprintf("Portfolio value $%.2f, P&L $%.2f.", out.PortfolioValue, out.PnL)), out, err
}

func (s *Server) getOrders(ctx context.Context, _ *mcp.CallToolRequest, _ emptyInput) (*mcp.CallToolResult, snapshot.Orders, error) {
	var out snapshot.Orders
	err := s.bridge.Call(ctx, livebridge.OpOrders, emptyInput{}, &out)
	return textResult(fmt.Sprintf("%d pending orders, %d filled orders.", len(out.Pending), len(out.Filled))), out, err
}

func (s *Server) placeOrder(ctx context.Context, _ *mcp.CallToolRequest, in livebridge.PlaceOrderRequest) (*mcp.CallToolResult, snapshot.TradeResult, error) {
	var out snapshot.TradeResult
	err := s.bridge.Call(ctx, livebridge.OpPlaceOrder, in, &out)
	return textResult(out.Message), out, err
}

func (s *Server) cancelOrder(ctx context.Context, _ *mcp.CallToolRequest, in cancelInput) (*mcp.CallToolResult, snapshot.Orders, error) {
	var out snapshot.Orders
	err := s.bridge.Call(ctx, livebridge.OpCancel, livebridge.CancelOrderRequest{ID: in.ID}, &out)
	return textResult(fmt.Sprintf("Cancelled order #%d.", in.ID)), out, err
}

func (s *Server) saveGame(ctx context.Context, _ *mcp.CallToolRequest, _ emptyInput) (*mcp.CallToolResult, snapshot.GameState, error) {
	var out snapshot.GameState
	err := s.bridge.Call(ctx, livebridge.OpSave, emptyInput{}, &out)
	return textResult(fmt.Sprintf("Saved Game %d.", out.ActiveSlot)), out, err
}

func (s *Server) registerResources(server *mcp.Server) {
	resources := []mcp.Resource{
		{URI: "tradez://game/summary", Name: "game_summary", Title: "Game Summary", MIMEType: "application/json", Description: "Current live game summary."},
		{URI: "tradez://market/stocks", Name: "market_stocks", Title: "Market Stocks", MIMEType: "application/json", Description: "All live stocks with prices and signals."},
		{URI: "tradez://market/news", Name: "market_news", Title: "Market News", MIMEType: "application/json", Description: "Recent news events."},
		{URI: "tradez://market/influences", Name: "market_influences", Title: "Market Influences", MIMEType: "application/json", Description: "Active decaying market influences."},
		{URI: "tradez://portfolio", Name: "portfolio", Title: "Portfolio", MIMEType: "application/json", Description: "Current cash, P&L, and positions."},
		{URI: "tradez://orders", Name: "orders", Title: "Orders", MIMEType: "application/json", Description: "Pending and historical orders."},
	}
	for i := range resources {
		resource := resources[i]
		server.AddResource(&resource, s.readResource)
	}
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "tradez://stock/{symbol}",
		Name:        "stock_detail",
		Title:       "Stock Detail",
		MIMEType:    "application/json",
		Description: "Detailed stock snapshot for a ticker symbol.",
	}, s.readResource)
}

func (s *Server) readResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	uri := req.Params.URI
	var data any
	switch {
	case uri == "tradez://game/summary":
		var out snapshot.GameState
		if err := s.bridge.Call(ctx, livebridge.OpGameState, emptyInput{}, &out); err != nil {
			return nil, err
		}
		data = out
	case uri == "tradez://market/stocks":
		var out []snapshot.StockSummary
		if err := s.bridge.Call(ctx, livebridge.OpListStocks, emptyInput{}, &out); err != nil {
			return nil, err
		}
		data = out
	case uri == "tradez://market/news":
		var out []snapshot.NewsEvent
		if err := s.bridge.Call(ctx, livebridge.OpGetNews, emptyInput{}, &out); err != nil {
			return nil, err
		}
		data = out
	case uri == "tradez://market/influences":
		var out []snapshot.Influence
		if err := s.bridge.Call(ctx, livebridge.OpInfluences, emptyInput{}, &out); err != nil {
			return nil, err
		}
		data = out
	case uri == "tradez://portfolio":
		var out snapshot.Portfolio
		if err := s.bridge.Call(ctx, livebridge.OpPortfolio, emptyInput{}, &out); err != nil {
			return nil, err
		}
		data = out
	case uri == "tradez://orders":
		var out snapshot.Orders
		if err := s.bridge.Call(ctx, livebridge.OpOrders, emptyInput{}, &out); err != nil {
			return nil, err
		}
		data = out
	case strings.HasPrefix(uri, "tradez://stock/"):
		symbol := strings.TrimPrefix(uri, "tradez://stock/")
		var out snapshot.StockSummary
		if err := s.bridge.Call(ctx, livebridge.OpGetStock, livebridge.StockRequest{Symbol: symbol}, &out); err != nil {
			return nil, err
		}
		data = out
	default:
		return nil, mcp.ResourceNotFoundError(uri)
	}
	return jsonResource(uri, data)
}

func (s *Server) registerPrompts(server *mcp.Server) {
	prompts := []mcp.Prompt{
		{
			Name:        "analyze_market",
			Title:       "Analyze Market",
			Description: "Guide an AI assistant through broad market and sector analysis.",
		},
		{
			Name:        "explain_news_impact",
			Title:       "Explain News Impact",
			Description: "Explain how active news should affect sectors and selected stocks.",
		},
		{
			Name:        "review_portfolio",
			Title:       "Review Portfolio",
			Description: "Review risk, concentration, longs, shorts, and pending orders.",
		},
		{
			Name:        "propose_trade_plan",
			Title:       "Propose Trade Plan",
			Description: "Create a concrete trade plan using market, news, portfolio, and stock context.",
			Arguments: []*mcp.PromptArgument{
				{Name: "symbol", Title: "Symbol", Description: "Optional ticker symbol to focus on."},
			},
		},
	}
	for i := range prompts {
		prompt := prompts[i]
		server.AddPrompt(&prompt, s.getPrompt)
	}
}

func (s *Server) getPrompt(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	name := req.Params.Name
	symbol := strings.ToUpper(strings.TrimSpace(req.Params.Arguments["symbol"]))
	text := ""
	switch name {
	case "analyze_market":
		text = "Use tradez://game/summary, tradez://market/stocks, tradez://market/news, and tradez://market/influences to identify the strongest sector rotations, then rank actionable long and short candidates. Be explicit about evidence and uncertainty."
	case "explain_news_impact":
		text = "Use tradez://market/news and tradez://market/influences to explain which industries or symbols are being helped or hurt by active events. Include likely second-order effects."
	case "review_portfolio":
		text = "Use tradez://portfolio, tradez://orders, and tradez://game/summary to review current risk, P&L, concentration, pending order exposure, and possible next actions."
	case "propose_trade_plan":
		if symbol != "" {
			text = fmt.Sprintf("Use tradez://stock/%s plus portfolio, news, and market resources to propose a trade plan for %s. Include order type, share count sizing logic, invalidation, and what would change your mind.", symbol, symbol)
		} else {
			text = "Use market, portfolio, orders, news, and stock resources to propose a concise trade plan. Include order type, sizing logic, risk, invalidation, and what would change your mind."
		}
	default:
		return nil, fmt.Errorf("unknown prompt %q", name)
	}
	if s.readOnly {
		text += " This MCP server is running in read-only mode, so do not call mutating trade tools."
	}
	return &mcp.GetPromptResult{
		Description: name,
		Messages: []*mcp.PromptMessage{
			{
				Role:    mcp.Role("user"),
				Content: &mcp.TextContent{Text: text},
			},
		},
	}, nil
}

func textResult(text string) *mcp.CallToolResult {
	if strings.TrimSpace(text) == "" {
		text = "OK"
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}
}

func jsonResource(uri string, data any) (*mcp.ReadResourceResult, error) {
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, err
	}
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      uri,
				MIMEType: "application/json",
				Text:     string(raw),
			},
		},
	}, nil
}

func gameStateText(state snapshot.GameState) string {
	return fmt.Sprintf("Game %d: cash $%.2f, portfolio $%.2f, P&L $%.2f, next news in %ds.",
		state.ActiveSlot, state.Cash, state.PortfolioValue, state.PnL, state.NextNewsInSeconds)
}
