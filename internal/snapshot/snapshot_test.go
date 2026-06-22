package snapshot

import (
	"testing"
	"tradez/internal/game"
	"tradez/internal/market"
)

// TestPortfolioSnapshotFullNetting verifies that buying into a same-size short
// leaves no position in the MCP portfolio snapshot (the get_portfolio path).
func TestPortfolioSnapshotFullNetting(t *testing.T) {
	g := game.New(market.New(42))
	g.Cash = 1_000_000
	g.StartingCash = 1_000_000

	stocks := g.Market.Snapshot()
	sym := stocks[0].Symbol

	if err := g.ShortSell(sym, 10); err != nil {
		t.Fatalf("ShortSell: %v", err)
	}
	if err := g.BuyMarket(sym, 10); err != nil {
		t.Fatalf("BuyMarket (net): %v", err)
	}

	portfolio := PortfolioSnapshot(g)
	if len(portfolio.Positions) != 0 {
		t.Errorf("positions = %d after full net, want 0", len(portfolio.Positions))
	}

	state := Game(g, 1)
	if state.Positions != 0 {
		t.Errorf("GameState.Positions = %d after full net, want 0", state.Positions)
	}
}

// TestPortfolioSnapshotPartialNetting verifies that shorting more than an
// existing long leaves only the short remainder in the MCP portfolio snapshot.
func TestPortfolioSnapshotPartialNetting(t *testing.T) {
	g := game.New(market.New(42))
	g.Cash = 1_000_000
	g.StartingCash = 1_000_000

	stocks := g.Market.Snapshot()
	sym := stocks[0].Symbol

	if err := g.BuyMarket(sym, 10); err != nil {
		t.Fatalf("BuyMarket: %v", err)
	}
	// Short 15 into 10 long → nets to 5 short.
	if err := g.ShortSell(sym, 15); err != nil {
		t.Fatalf("ShortSell (net): %v", err)
	}

	portfolio := PortfolioSnapshot(g)
	if len(portfolio.Positions) != 1 {
		t.Fatalf("positions = %d after partial net, want 1", len(portfolio.Positions))
	}
	pos := portfolio.Positions[0]
	if pos.LongShares != 0 {
		t.Errorf("LongShares = %d, want 0", pos.LongShares)
	}
	if pos.ShortShares != 5 {
		t.Errorf("ShortShares = %d, want 5", pos.ShortShares)
	}

	// Confirm the per-stock snapshot (get_stock MCP path) agrees.
	stock, err := Stock(g, sym)
	if err != nil {
		t.Fatalf("Stock: %v", err)
	}
	if stock.Position == nil {
		t.Fatal("stock.Position is nil, want short position")
	}
	if stock.Position.LongShares != 0 || stock.Position.ShortShares != 5 {
		t.Errorf("stock.Position = %+v, want LongShares=0 ShortShares=5", stock.Position)
	}
}

func TestSnapshotsIncludePortfolioAndStockState(t *testing.T) {
	g := game.New(market.New(1))
	g.Cash = 1_000_000
	g.StartingCash = 1_000_000
	g.Difficulty = game.DifficultyEasy

	stocks := g.Market.Snapshot()
	if len(stocks) == 0 {
		t.Fatal("expected generated stocks")
	}
	symbol := stocks[0].Symbol
	if err := g.BuyMarket(symbol, 3); err != nil {
		t.Fatalf("BuyMarket failed: %v", err)
	}

	state := Game(g, 2)
	if state.ActiveSlot != 2 {
		t.Fatalf("active slot = %d, want 2", state.ActiveSlot)
	}
	if state.StockCount == 0 {
		t.Fatal("stock count should be populated")
	}

	portfolio := PortfolioSnapshot(g)
	if len(portfolio.Positions) != 1 {
		t.Fatalf("positions = %d, want 1", len(portfolio.Positions))
	}
	if portfolio.Positions[0].Symbol != symbol {
		t.Fatalf("position symbol = %s, want %s", portfolio.Positions[0].Symbol, symbol)
	}

	stock, err := Stock(g, symbol)
	if err != nil {
		t.Fatalf("Stock failed: %v", err)
	}
	if stock.Position == nil || stock.Position.LongShares != 3 {
		t.Fatalf("stock position = %#v, want 3 long shares", stock.Position)
	}
	if len(stock.History) == 0 {
		t.Fatal("stock detail should include history")
	}
}
