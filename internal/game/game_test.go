package game_test

import (
	"testing"
	"tradez/internal/game"
	"tradez/internal/market"
)

// testGame returns a seeded game with ample cash and the first stock's symbol and price.
func testGame(t *testing.T) (*game.Game, string, float64) {
	t.Helper()
	g := game.New(market.New(42))
	g.Cash = 1_000_000
	g.StartingCash = 1_000_000
	stocks := g.Market.Snapshot()
	if len(stocks) == 0 {
		t.Fatal("no stocks generated")
	}
	sym := stocks[0].Symbol
	price := g.Market.GetStock(sym).Price
	return g, sym, price
}

// ── Netting: BuyMarket into open short ──────────────────────────────────────

func TestBuyMarketNetsFullShort(t *testing.T) {
	g, sym, _ := testGame(t)

	if err := g.ShortSell(sym, 10); err != nil {
		t.Fatalf("ShortSell: %v", err)
	}
	if err := g.BuyMarket(sym, 10); err != nil {
		t.Fatalf("BuyMarket: %v", err)
	}

	pos := g.Positions[sym]
	if pos.Shares != 0 {
		t.Errorf("long shares = %d, want 0", pos.Shares)
	}
	if pos.ShortShares != 0 {
		t.Errorf("short shares = %d, want 0", pos.ShortShares)
	}
	if pos.ShortAvg != 0 {
		t.Errorf("ShortAvg = %f, want 0 after full net-cover", pos.ShortAvg)
	}
}

func TestBuyMarketNetsPartialShort(t *testing.T) {
	g, sym, _ := testGame(t)

	if err := g.ShortSell(sym, 10); err != nil {
		t.Fatalf("ShortSell: %v", err)
	}
	if err := g.BuyMarket(sym, 15); err != nil {
		t.Fatalf("BuyMarket: %v", err)
	}

	pos := g.Positions[sym]
	if pos.ShortShares != 0 {
		t.Errorf("short shares = %d, want 0 after net", pos.ShortShares)
	}
	if pos.Shares != 5 {
		t.Errorf("long shares = %d, want 5 (remainder after netting 10-share short)", pos.Shares)
	}
}

// ── Netting: ShortSell into open long ───────────────────────────────────────

func TestShortSellNetsFullLong(t *testing.T) {
	g, sym, _ := testGame(t)

	if err := g.BuyMarket(sym, 10); err != nil {
		t.Fatalf("BuyMarket: %v", err)
	}
	if err := g.ShortSell(sym, 10); err != nil {
		t.Fatalf("ShortSell: %v", err)
	}

	pos := g.Positions[sym]
	if pos.Shares != 0 {
		t.Errorf("long shares = %d, want 0", pos.Shares)
	}
	if pos.ShortShares != 0 {
		t.Errorf("short shares = %d, want 0", pos.ShortShares)
	}
	if pos.AvgCost != 0 {
		t.Errorf("AvgCost = %f, want 0 after full net-sell", pos.AvgCost)
	}
}

func TestShortSellNetsPartialLong(t *testing.T) {
	g, sym, _ := testGame(t)

	if err := g.BuyMarket(sym, 10); err != nil {
		t.Fatalf("BuyMarket: %v", err)
	}
	if err := g.ShortSell(sym, 15); err != nil {
		t.Fatalf("ShortSell: %v", err)
	}

	pos := g.Positions[sym]
	if pos.Shares != 0 {
		t.Errorf("long shares = %d, want 0 after net", pos.Shares)
	}
	if pos.ShortShares != 5 {
		t.Errorf("short shares = %d, want 5 (remainder after netting 10-share long)", pos.ShortShares)
	}
}

// ── Stale average cleanup ────────────────────────────────────────────────────

func TestSellMarketClearsAvgCost(t *testing.T) {
	g, sym, _ := testGame(t)

	if err := g.BuyMarket(sym, 10); err != nil {
		t.Fatalf("BuyMarket: %v", err)
	}
	if err := g.SellMarket(sym, 10); err != nil {
		t.Fatalf("SellMarket: %v", err)
	}

	if pos := g.Positions[sym]; pos.AvgCost != 0 {
		t.Errorf("AvgCost = %f after full sell, want 0", pos.AvgCost)
	}
}

func TestCoverShortClearsShortAvg(t *testing.T) {
	g, sym, _ := testGame(t)

	if err := g.ShortSell(sym, 10); err != nil {
		t.Fatalf("ShortSell: %v", err)
	}
	if err := g.CoverShort(sym, 10); err != nil {
		t.Fatalf("CoverShort: %v", err)
	}

	if pos := g.Positions[sym]; pos.ShortAvg != 0 {
		t.Errorf("ShortAvg = %f after full cover, want 0", pos.ShortAvg)
	}
}

// ── Margin call: grace period and force cover ────────────────────────────────

func TestMarginCallForceCoverAllShorts(t *testing.T) {
	g, sym, shortPrice := testGame(t)

	// Give just enough cash to post the short margin.
	shares := 100
	margin := shortPrice * float64(shares) * 0.5
	g.Cash = margin
	g.StartingCash = margin * 10

	if err := g.ShortSell(sym, shares); err != nil {
		t.Fatalf("ShortSell: %v", err)
	}

	// Spike price 3× so equity/exposure ratio drops well below 25%.
	g.Market.GetStock(sym).Price = shortPrice * 3

	// First CheckMargin should activate the call.
	g.CheckMargin()
	if !g.MarginCallActive {
		t.Fatal("margin call not active after price spike")
	}

	// Exhaust the grace period; force cover fires on the final tick.
	for i := 0; i < game.MarginCallGrace; i++ {
		g.CheckMargin()
	}

	pos := g.Positions[sym]
	if pos.ShortShares != 0 {
		t.Errorf("short shares = %d after force cover, want 0", pos.ShortShares)
	}
	if pos.ShortAvg != 0 {
		t.Errorf("ShortAvg = %f after force cover, want 0", pos.ShortAvg)
	}
	if g.MarginCallActive {
		t.Error("MarginCallActive should be false after force cover")
	}
}

func TestMarginCallClearsWhenEquityRestored(t *testing.T) {
	g, sym, shortPrice := testGame(t)

	shares := 100
	margin := shortPrice * float64(shares) * 0.5
	g.Cash = margin
	g.StartingCash = margin * 10

	if err := g.ShortSell(sym, shares); err != nil {
		t.Fatalf("ShortSell: %v", err)
	}

	// Spike price to trigger call.
	g.Market.GetStock(sym).Price = shortPrice * 3
	g.CheckMargin()
	if !g.MarginCallActive {
		t.Fatal("margin call not active after price spike")
	}

	// Cover all shorts to restore equity.
	if err := g.CoverShort(sym, shares); err != nil {
		t.Fatalf("CoverShort: %v", err)
	}

	// Next tick should see no shorts and clear the call.
	g.CheckMargin()
	if g.MarginCallActive {
		t.Error("margin call should clear once shorts are covered")
	}
}
