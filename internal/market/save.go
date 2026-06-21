package market

import (
	"math/rand"
	"time"
)

// MarketState is a serializable snapshot of all market data.
type MarketState struct {
	Stocks     []Stock           `json:"stocks"`
	News       []NewsEvent       `json:"news"`
	Influences []MarketInfluence `json:"influences"`
	NextNewsID int               `json:"next_news_id"`
	NextNewsAt time.Time         `json:"next_news_at"`
}

// Export captures the current market state for serialization.
func (m *Market) Export() MarketState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stocks := make([]Stock, len(m.Stocks))
	for i, s := range m.Stocks {
		sc := *s
		hist := make([]PricePoint, len(s.History))
		copy(hist, s.History)
		sc.History = hist
		stocks[i] = sc
	}

	news := make([]NewsEvent, len(m.News))
	for i, n := range m.News {
		ne := *n
		ne.Effects = copyEffects(n.Effects)
		ne.AffectedIndustries = append([]Industry(nil), n.AffectedIndustries...)
		ne.AffectedSymbols = append([]string(nil), n.AffectedSymbols...)
		news[i] = ne
	}

	infs := make([]MarketInfluence, len(m.Influences))
	for i, inf := range m.Influences {
		ic := *inf
		ic.Effects = copyEffects(inf.Effects)
		infs[i] = ic
	}

	return MarketState{
		Stocks:     stocks,
		News:       news,
		Influences: infs,
		NextNewsID: m.nextNewsID,
		NextNewsAt: m.nextNewsAt,
	}
}

// NewFromState restores a Market from a saved state with a fresh RNG.
func NewFromState(s MarketState) *Market {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	stocks := make([]*Stock, len(s.Stocks))
	for i := range s.Stocks {
		sc := s.Stocks[i]
		stocks[i] = &sc
	}

	news := make([]*NewsEvent, len(s.News))
	for i := range s.News {
		nc := s.News[i]
		news[i] = &nc
	}

	infs := make([]*MarketInfluence, len(s.Influences))
	for i := range s.Influences {
		ic := s.Influences[i]
		infs[i] = &ic
	}

	return &Market{
		Stocks:     stocks,
		News:       news,
		Influences: infs,
		rng:        rng,
		nextNewsID: s.NextNewsID,
		nextNewsAt: s.NextNewsAt,
	}
}

func copyEffects(src []InfluenceEffect) []InfluenceEffect {
	dst := make([]InfluenceEffect, len(src))
	for i, e := range src {
		dst[i] = InfluenceEffect{
			Industries:      append([]Industry(nil), e.Industries...),
			Symbols:         append([]string(nil), e.Symbols...),
			Sentiment:       e.Sentiment,
			CompanyIndustry: e.CompanyIndustry,
		}
	}
	return dst
}
