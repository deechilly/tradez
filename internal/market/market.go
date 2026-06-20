package market

import (
	"math/rand"
	"sync"
	"time"
)

type Market struct {
	Stocks     []*Stock
	News       []*NewsEvent
	Influences []*MarketInfluence
	mu         sync.RWMutex
	rng        *rand.Rand
	nextNewsID int
	nextNewsAt time.Time
}

func New(seed int64) *Market {
	rng := rand.New(rand.NewSource(seed))
	stocks := GenerateMarket(rng)
	m := &Market{
		Stocks:     stocks,
		rng:        rng,
		nextNewsID: 1,
	}
	m.scheduleNextNews()
	return m
}

func (m *Market) scheduleNextNews() {
	// fire next news event between 1–5 minutes from now
	seconds := 60 + m.rng.Intn(4*60)
	m.nextNewsAt = time.Now().Add(time.Duration(seconds) * time.Second)
}

// ScheduleFirstNews resets the news timer so the first event fires within
// 30 seconds of the player starting a game.
func (m *Market) ScheduleFirstNews() {
	m.mu.Lock()
	defer m.mu.Unlock()
	seconds := 5 + m.rng.Intn(26) // 5–30 seconds
	m.nextNewsAt = time.Now().Add(time.Duration(seconds) * time.Second)
}

func (m *Market) Tick() (newEvent *NewsEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()

	// Generate news if scheduled time has passed
	if now.After(m.nextNewsAt) {
		event, influence := GenerateNews(m.rng, m.Stocks, m.nextNewsID)
		m.nextNewsID++
		m.News = append(m.News, event)
		m.Influences = append(m.Influences, influence)
		// keep at most 50 news items
		if len(m.News) > 50 {
			m.News = m.News[len(m.News)-50:]
		}
		m.scheduleNextNews()
		newEvent = event

		// Immediate price spike for affected stocks
		for _, s := range m.Stocks {
			if influence.Targets(s) {
				spike := influence.InfluenceStrengthFor(s) * s.Volatility * s.Price * 8
				s.Price += spike
				if s.Price < 0.01 {
					s.Price = 0.01
				}
				s.Price = roundTo(s.Price, 2)
			}
		}
	}

	// Prune expired influences
	active := m.Influences[:0]
	for _, inf := range m.Influences {
		if inf.IsActive() {
			active = append(active, inf)
		}
	}
	m.Influences = active

	// Tick each stock
	for _, s := range m.Stocks {
		m.tickStock(s, now)
	}

	return newEvent
}

func (m *Market) tickStock(s *Stock, now time.Time) {
	noise := (m.rng.Float64()*2 - 1) * s.Volatility * s.Price
	drift := s.Trend * s.Price

	// Sum active influences targeting this stock
	influenceDrift := 0.0
	for _, inf := range m.Influences {
		strength := inf.InfluenceStrengthFor(s)
		if strength != 0 {
			influenceDrift += strength * s.Volatility * s.Price * 3
		}
	}

	delta := noise + drift + influenceDrift

	newPrice := s.Price + delta
	if newPrice < 0.01 {
		newPrice = 0.01
	}
	newPrice = roundTo(newPrice, 2)

	if newPrice > s.High {
		s.High = newPrice
	}
	if newPrice < s.Low {
		s.Low = newPrice
	}

	s.Price = newPrice
	s.Volume += int64(m.rng.Intn(10000))

	s.History = append(s.History, PricePoint{Time: now, Price: newPrice})
	if len(s.History) > 500 {
		s.History = s.History[len(s.History)-500:]
	}
}

func (m *Market) GetStock(symbol string) *Stock {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, s := range m.Stocks {
		if s.Symbol == symbol {
			return s
		}
	}
	return nil
}

func (m *Market) Snapshot() []*Stock {
	m.mu.RLock()
	defer m.mu.RUnlock()
	snap := make([]*Stock, len(m.Stocks))
	copy(snap, m.Stocks)
	return snap
}

func (m *Market) RecentNews(n int) []*NewsEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	news := m.News
	if len(news) <= n {
		out := make([]*NewsEvent, len(news))
		copy(out, news)
		return out
	}
	out := make([]*NewsEvent, n)
	copy(out, news[len(news)-n:])
	return out
}

func (m *Market) ActiveInfluences() []*MarketInfluence {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*MarketInfluence, 0, len(m.Influences))
	for _, inf := range m.Influences {
		if inf.IsActive() {
			out = append(out, inf)
		}
	}
	return out
}

// StockInfluenceStrength returns the net influence on a given stock right now.
func (m *Market) StockInfluenceStrength(symbol string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var s *Stock
	for _, st := range m.Stocks {
		if st.Symbol == symbol {
			s = st
			break
		}
	}
	if s == nil {
		return 0
	}
	total := 0.0
	for _, inf := range m.Influences {
		total += inf.InfluenceStrengthFor(s)
	}
	return total
}

func (m *Market) NextNewsIn() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	d := time.Until(m.nextNewsAt)
	if d < 0 {
		return 0
	}
	return d
}
