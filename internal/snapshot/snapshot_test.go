package snapshot

import (
	"testing"
	"tradez/internal/game"
	"tradez/internal/market"
)

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
