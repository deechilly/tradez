package analysis

import (
	"testing"
	"time"
	"tradez/internal/market"
)

func TestAnalyzeProducesStrongBuyForCheapPositiveMomentumStock(t *testing.T) {
	now := time.Now()
	history := make([]market.PricePoint, 0, 60)
	for i := 0; i < 60; i++ {
		history = append(history, market.PricePoint{
			Time:  now.Add(time.Duration(i) * time.Second),
			Price: 80 + float64(i),
		})
	}
	s := &market.Stock{
		Symbol:    "TEST",
		Industry:  market.IndustryTech,
		Price:     140,
		PERatio:   5,
		EPS:       28,
		DivYield:  3,
		Beta:      1.2,
		YTDGrowth: -50,
		History:   history,
	}

	got := Analyze(nil, s)
	if got.Recommendation != "STRONG_BUY" {
		t.Fatalf("recommendation = %s, want STRONG_BUY; composite %.3f", got.Recommendation, got.CompositeScore)
	}
	if got.CompositeScore <= 0.45 {
		t.Fatalf("composite = %.3f, want > 0.45", got.CompositeScore)
	}
}
