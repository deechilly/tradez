package snapshot

import (
	"fmt"
	"sort"
	"time"
	"tradez/internal/analysis"
	"tradez/internal/game"
	"tradez/internal/market"
)

type GameState struct {
	ActiveSlot        int      `json:"active_slot"`
	Difficulty        string   `json:"difficulty"`
	Cash              float64  `json:"cash"`
	StartingCash      float64  `json:"starting_cash"`
	PortfolioValue    float64  `json:"portfolio_value"`
	PnL               float64  `json:"pnl"`
	PnLPct            float64  `json:"pnl_pct"`
	StockCount        int      `json:"stock_count"`
	NewsCount         int      `json:"news_count"`
	ActiveInfluences  int      `json:"active_influences"`
	PendingOrders     int      `json:"pending_orders"`
	Positions         int      `json:"positions"`
	NextNewsInSeconds int      `json:"next_news_in_seconds"`
	Messages          []string `json:"messages,omitempty"`
}

type StockSummary struct {
	Symbol            string           `json:"symbol"`
	Name              string           `json:"name"`
	Industry          string           `json:"industry"`
	Price             float64          `json:"price"`
	PrevClose         float64          `json:"prev_close"`
	Open              float64          `json:"open"`
	High              float64          `json:"high"`
	Low               float64          `json:"low"`
	Change            float64          `json:"change"`
	ChangePct         float64          `json:"change_pct"`
	Volume            int64            `json:"volume"`
	MarketCap         float64          `json:"market_cap"`
	PERatio           float64          `json:"pe_ratio"`
	EPS               float64          `json:"eps"`
	DivYield          float64          `json:"dividend_yield"`
	Beta              float64          `json:"beta"`
	YTDGrowth         float64          `json:"ytd_growth"`
	InfluenceStrength float64          `json:"influence_strength"`
	Recommendation    string           `json:"recommendation"`
	Analysis          *analysis.Result `json:"analysis,omitempty"`
	Position          *Position        `json:"position,omitempty"`
	History           []PricePoint     `json:"history,omitempty"`
}

type PricePoint struct {
	Time  time.Time `json:"time"`
	Price float64   `json:"price"`
}

type Position struct {
	Symbol          string  `json:"symbol"`
	LongShares      int     `json:"long_shares"`
	LongAvgCost     float64 `json:"long_avg_cost"`
	LongMarketValue float64 `json:"long_market_value"`
	LongPnL         float64 `json:"long_pnl"`
	ShortShares     int     `json:"short_shares"`
	ShortAvg        float64 `json:"short_avg"`
	ShortPnL        float64 `json:"short_pnl"`
	ShortMargin     float64 `json:"short_margin"`
	CurrentPrice    float64 `json:"current_price"`
}

type Portfolio struct {
	Cash           float64    `json:"cash"`
	StartingCash   float64    `json:"starting_cash"`
	PortfolioValue float64    `json:"portfolio_value"`
	PnL            float64    `json:"pnl"`
	PnLPct         float64    `json:"pnl_pct"`
	Positions      []Position `json:"positions"`
}

type Order struct {
	ID         int     `json:"id"`
	Symbol     string  `json:"symbol"`
	Type       string  `json:"type"`
	Shares     int     `json:"shares"`
	LimitPrice float64 `json:"limit_price,omitempty"`
	FilledAt   float64 `json:"filled_at,omitempty"`
	Status     string  `json:"status"`
}

type Orders struct {
	Pending []Order `json:"pending"`
	Filled  []Order `json:"filled"`
	All     []Order `json:"all"`
}

type NewsEvent struct {
	ID                 int               `json:"id"`
	Headline           string            `json:"headline"`
	Category           string            `json:"category"`
	Sentiment          float64           `json:"sentiment"`
	SentimentLabel     string            `json:"sentiment_label"`
	Magnitude          float64           `json:"magnitude"`
	CreatedAt          time.Time         `json:"created_at"`
	ExpiresAt          time.Time         `json:"expires_at"`
	Active             bool              `json:"active"`
	AffectedIndustries []string          `json:"affected_industries"`
	AffectedSymbols    []string          `json:"affected_symbols"`
	Effects            []InfluenceEffect `json:"effects"`
}

type InfluenceEffect struct {
	Industries []string `json:"industries,omitempty"`
	Symbols    []string `json:"symbols,omitempty"`
	Sentiment  float64  `json:"sentiment"`
}

type Influence struct {
	NewsID             int               `json:"news_id"`
	Magnitude          float64           `json:"magnitude"`
	CreatedAt          time.Time         `json:"created_at"`
	ExpiresAt          time.Time         `json:"expires_at"`
	SecondsRemaining   int               `json:"seconds_remaining"`
	PrimarySentiment   float64           `json:"primary_sentiment"`
	PrimaryAbsStrength float64           `json:"primary_abs_strength"`
	Effects            []InfluenceEffect `json:"effects"`
}

type TradeResult struct {
	Message   string        `json:"message"`
	Order     *Order        `json:"order,omitempty"`
	GameState GameState     `json:"game_state"`
	Portfolio Portfolio     `json:"portfolio"`
	Stock     *StockSummary `json:"stock,omitempty"`
}

func Game(g *game.Game, activeSlot int) GameState {
	if g == nil || g.Market == nil {
		return GameState{ActiveSlot: activeSlot}
	}
	value := g.PortfolioValue()
	pnl := value - g.StartingCash
	pnlPct := 0.0
	if g.StartingCash != 0 {
		pnlPct = pnl / g.StartingCash * 100
	}
	positions := 0
	for _, pos := range g.Positions {
		if pos != nil && (pos.Shares > 0 || pos.ShortShares > 0) {
			positions++
		}
	}
	return GameState{
		ActiveSlot:        activeSlot,
		Difficulty:        g.Difficulty.String(),
		Cash:              g.Cash,
		StartingCash:      g.StartingCash,
		PortfolioValue:    value,
		PnL:               pnl,
		PnLPct:            pnlPct,
		StockCount:        len(g.Market.Snapshot()),
		NewsCount:         len(g.Market.RecentNews(1000)),
		ActiveInfluences:  len(g.Market.ActiveInfluences()),
		PendingOrders:     len(g.PendingOrders()),
		Positions:         positions,
		NextNewsInSeconds: int(g.Market.NextNewsIn().Seconds()),
		Messages:          append([]string(nil), g.Messages...),
	}
}

func Stocks(g *game.Game, includeAnalysis bool) []StockSummary {
	if g == nil || g.Market == nil {
		return nil
	}
	stocks := g.Market.Snapshot()
	out := make([]StockSummary, 0, len(stocks))
	for _, s := range stocks {
		out = append(out, stock(g, s, includeAnalysis, false))
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Symbol < out[j].Symbol
	})
	return out
}

func Stock(g *game.Game, symbol string) (*StockSummary, error) {
	if g == nil || g.Market == nil {
		return nil, fmt.Errorf("no active game")
	}
	s := g.Market.GetStock(symbol)
	if s == nil {
		return nil, fmt.Errorf("unknown symbol %s", symbol)
	}
	snap := stock(g, s, true, true)
	return &snap, nil
}

func PortfolioSnapshot(g *game.Game) Portfolio {
	if g == nil || g.Market == nil {
		return Portfolio{}
	}
	value := g.PortfolioValue()
	pnl := value - g.StartingCash
	pnlPct := 0.0
	if g.StartingCash != 0 {
		pnlPct = pnl / g.StartingCash * 100
	}
	syms := make([]string, 0, len(g.Positions))
	for sym := range g.Positions {
		syms = append(syms, sym)
	}
	sort.Strings(syms)

	positions := make([]Position, 0, len(syms))
	for _, sym := range syms {
		pos := g.Positions[sym]
		if pos == nil || (pos.Shares == 0 && pos.ShortShares == 0) {
			continue
		}
		s := g.Market.GetStock(sym)
		if s == nil {
			continue
		}
		positions = append(positions, position(sym, pos, s.Price))
	}

	return Portfolio{
		Cash:           g.Cash,
		StartingCash:   g.StartingCash,
		PortfolioValue: value,
		PnL:            pnl,
		PnLPct:         pnlPct,
		Positions:      positions,
	}
}

func OrdersSnapshot(g *game.Game) Orders {
	if g == nil {
		return Orders{}
	}
	all := make([]Order, 0, len(g.Orders))
	pending := make([]Order, 0)
	filled := make([]Order, 0)
	for _, o := range g.Orders {
		snap := order(o)
		all = append(all, snap)
		switch o.Status {
		case game.OrderPending:
			pending = append(pending, snap)
		case game.OrderFilled:
			filled = append(filled, snap)
		}
	}
	return Orders{Pending: pending, Filled: filled, All: all}
}

func OrderSnapshot(o *game.Order) Order {
	return order(o)
}

func News(g *game.Game) []NewsEvent {
	if g == nil || g.Market == nil {
		return nil
	}
	news := g.Market.RecentNews(50)
	out := make([]NewsEvent, 0, len(news))
	for i := len(news) - 1; i >= 0; i-- {
		n := news[i]
		out = append(out, NewsEvent{
			ID:                 n.ID,
			Headline:           n.Headline,
			Category:           string(n.Category),
			Sentiment:          n.Sentiment,
			SentimentLabel:     n.SentimentLabel(),
			Magnitude:          n.Magnitude,
			CreatedAt:          n.CreatedAt,
			ExpiresAt:          n.ExpiresAt,
			Active:             n.IsActive(),
			AffectedIndustries: industries(n.AffectedIndustries),
			AffectedSymbols:    append([]string(nil), n.AffectedSymbols...),
			Effects:            effects(n.Effects),
		})
	}
	return out
}

func Influences(g *game.Game) []Influence {
	if g == nil || g.Market == nil {
		return nil
	}
	infs := g.Market.ActiveInfluences()
	out := make([]Influence, 0, len(infs))
	for _, inf := range infs {
		sent, strength := inf.PrimaryEffect()
		remaining := int(time.Until(inf.ExpiresAt).Seconds())
		if remaining < 0 {
			remaining = 0
		}
		out = append(out, Influence{
			NewsID:             inf.NewsID,
			Magnitude:          inf.Magnitude,
			CreatedAt:          inf.CreatedAt,
			ExpiresAt:          inf.ExpiresAt,
			SecondsRemaining:   remaining,
			PrimarySentiment:   sent,
			PrimaryAbsStrength: strength,
			Effects:            effects(inf.Effects),
		})
	}
	return out
}

func stock(g *game.Game, s *market.Stock, includeAnalysis, includeHistory bool) StockSummary {
	change := s.Change()
	changePct := s.ChangePct()
	res := analysis.Analyze(g.Market, s)
	snap := StockSummary{
		Symbol:            s.Symbol,
		Name:              s.Name,
		Industry:          string(s.Industry),
		Price:             s.Price,
		PrevClose:         s.PrevClose,
		Open:              s.Open,
		High:              s.High,
		Low:               s.Low,
		Change:            change,
		ChangePct:         changePct,
		Volume:            s.Volume,
		MarketCap:         s.MarketCap,
		PERatio:           s.PERatio,
		EPS:               s.EPS,
		DivYield:          s.DivYield,
		Beta:              s.Beta,
		YTDGrowth:         s.YTDGrowth,
		InfluenceStrength: g.Market.StockInfluenceStrength(s.Symbol),
		Recommendation:    res.Recommendation,
	}
	if includeAnalysis {
		snap.Analysis = &res
	}
	if pos := g.Positions[s.Symbol]; pos != nil && (pos.Shares > 0 || pos.ShortShares > 0) {
		p := position(s.Symbol, pos, s.Price)
		snap.Position = &p
	}
	if includeHistory {
		history := s.History
		if len(history) > 120 {
			history = history[len(history)-120:]
		}
		snap.History = make([]PricePoint, len(history))
		for i, p := range history {
			snap.History[i] = PricePoint{Time: p.Time, Price: p.Price}
		}
	}
	return snap
}

func position(symbol string, pos *game.Position, price float64) Position {
	return Position{
		Symbol:          symbol,
		LongShares:      pos.Shares,
		LongAvgCost:     pos.AvgCost,
		LongMarketValue: pos.MarketValue(price),
		LongPnL:         pos.LongPnL(price),
		ShortShares:     pos.ShortShares,
		ShortAvg:        pos.ShortAvg,
		ShortPnL:        pos.ShortPnL(price),
		ShortMargin:     pos.ShortAvg * float64(pos.ShortShares) * 0.5,
		CurrentPrice:    price,
	}
}

func order(o *game.Order) Order {
	if o == nil {
		return Order{}
	}
	return Order{
		ID:         o.ID,
		Symbol:     o.Symbol,
		Type:       o.Type.String(),
		Shares:     o.Shares,
		LimitPrice: o.LimitPrice,
		FilledAt:   o.FilledAt,
		Status:     orderStatus(o.Status),
	}
}

func orderStatus(status game.OrderStatus) string {
	switch status {
	case game.OrderPending:
		return "pending"
	case game.OrderFilled:
		return "filled"
	case game.OrderCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

func effects(src []market.InfluenceEffect) []InfluenceEffect {
	out := make([]InfluenceEffect, 0, len(src))
	for _, eff := range src {
		out = append(out, InfluenceEffect{
			Industries: industries(eff.Industries),
			Symbols:    append([]string(nil), eff.Symbols...),
			Sentiment:  eff.Sentiment,
		})
	}
	return out
}

func industries(src []market.Industry) []string {
	out := make([]string, 0, len(src))
	for _, ind := range src {
		out = append(out, string(ind))
	}
	return out
}
