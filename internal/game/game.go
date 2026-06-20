package game

import (
	"fmt"
	"tradez/internal/market"
)

type Difficulty int

const (
	DifficultyEasy   Difficulty = iota
	DifficultyNormal
)

func (d Difficulty) String() string {
	switch d {
	case DifficultyEasy:
		return "Easy"
	case DifficultyNormal:
		return "Normal"
	}
	return "Unknown"
}

// ShowInfluenceTimer controls whether the time remaining on active market
// influences is visible in the news panel.
func (d Difficulty) ShowInfluenceTimer() bool {
	return d == DifficultyEasy
}

type OrderType int

const (
	OrderMarketBuy OrderType = iota
	OrderMarketSell
	OrderLimitBuy
	OrderLimitSell
	OrderShortSell
	OrderShortCover
)

func (o OrderType) String() string {
	switch o {
	case OrderMarketBuy:
		return "BUY"
	case OrderMarketSell:
		return "SELL"
	case OrderLimitBuy:
		return "LIMIT BUY"
	case OrderLimitSell:
		return "LIMIT SELL"
	case OrderShortSell:
		return "SHORT"
	case OrderShortCover:
		return "COVER"
	}
	return "?"
}

type OrderStatus int

const (
	OrderPending   OrderStatus = iota
	OrderFilled
	OrderCancelled
)

type Order struct {
	ID         int
	Symbol     string
	Type       OrderType
	Shares     int
	LimitPrice float64
	FilledAt   float64
	Status     OrderStatus
}

type Position struct {
	Symbol     string
	Shares     int
	AvgCost    float64
	ShortShares int
	ShortAvg   float64
}

func (p *Position) MarketValue(price float64) float64 {
	return float64(p.Shares) * price
}

func (p *Position) ShortPnL(price float64) float64 {
	return float64(p.ShortShares) * (p.ShortAvg - price)
}

func (p *Position) LongPnL(price float64) float64 {
	return float64(p.Shares) * (price - p.AvgCost)
}

type Game struct {
	Market      *market.Market
	Cash        float64
	StartingCash float64
	Positions   map[string]*Position
	Orders      []*Order
	nextOrderID int
	Difficulty  Difficulty
	Messages    []string
}

func New(m *market.Market) *Game {
	return &Game{
		Market:    m,
		Positions: make(map[string]*Position),
	}
}

func (g *Game) addMsg(msg string) {
	g.Messages = append(g.Messages, msg)
	if len(g.Messages) > 50 {
		g.Messages = g.Messages[len(g.Messages)-50:]
	}
}

func (g *Game) BuyMarket(symbol string, shares int) error {
	s := g.Market.GetStock(symbol)
	if s == nil {
		return fmt.Errorf("unknown symbol %s", symbol)
	}
	cost := s.Price * float64(shares)
	if cost > g.Cash {
		return fmt.Errorf("insufficient funds: need $%.2f, have $%.2f", cost, g.Cash)
	}
	g.Cash -= cost

	pos := g.getOrCreatePos(symbol)
	total := float64(pos.Shares)*pos.AvgCost + cost
	pos.Shares += shares
	if pos.Shares > 0 {
		pos.AvgCost = total / float64(pos.Shares)
	}

	g.nextOrderID++
	g.Orders = append(g.Orders, &Order{
		ID: g.nextOrderID, Symbol: symbol, Type: OrderMarketBuy,
		Shares: shares, FilledAt: s.Price, Status: OrderFilled,
	})
	g.addMsg(fmt.Sprintf("Bought %d shares of %s @ $%.2f", shares, symbol, s.Price))
	return nil
}

func (g *Game) SellMarket(symbol string, shares int) error {
	pos, ok := g.Positions[symbol]
	if !ok || pos.Shares < shares {
		available := 0
		if ok {
			available = pos.Shares
		}
		return fmt.Errorf("insufficient shares: have %d, selling %d", available, shares)
	}
	s := g.Market.GetStock(symbol)
	if s == nil {
		return fmt.Errorf("unknown symbol %s", symbol)
	}
	proceeds := s.Price * float64(shares)
	g.Cash += proceeds
	pos.Shares -= shares

	g.nextOrderID++
	g.Orders = append(g.Orders, &Order{
		ID: g.nextOrderID, Symbol: symbol, Type: OrderMarketSell,
		Shares: shares, FilledAt: s.Price, Status: OrderFilled,
	})
	g.addMsg(fmt.Sprintf("Sold %d shares of %s @ $%.2f", shares, symbol, s.Price))
	return nil
}

func (g *Game) PlaceLimitBuy(symbol string, shares int, limit float64) error {
	s := g.Market.GetStock(symbol)
	if s == nil {
		return fmt.Errorf("unknown symbol %s", symbol)
	}
	cost := limit * float64(shares)
	if cost > g.Cash {
		return fmt.Errorf("insufficient funds to reserve: need $%.2f", cost)
	}
	g.Cash -= cost
	g.nextOrderID++
	g.Orders = append(g.Orders, &Order{
		ID: g.nextOrderID, Symbol: symbol, Type: OrderLimitBuy,
		Shares: shares, LimitPrice: limit, Status: OrderPending,
	})
	g.addMsg(fmt.Sprintf("Limit BUY %d %s @ $%.2f placed", shares, symbol, limit))
	return nil
}

func (g *Game) PlaceLimitSell(symbol string, shares int, limit float64) error {
	pos, ok := g.Positions[symbol]
	if !ok || pos.Shares < shares {
		available := 0
		if ok {
			available = pos.Shares
		}
		return fmt.Errorf("insufficient shares: have %d", available)
	}
	pos.Shares -= shares
	g.nextOrderID++
	g.Orders = append(g.Orders, &Order{
		ID: g.nextOrderID, Symbol: symbol, Type: OrderLimitSell,
		Shares: shares, LimitPrice: limit, Status: OrderPending,
	})
	g.addMsg(fmt.Sprintf("Limit SELL %d %s @ $%.2f placed", shares, symbol, limit))
	return nil
}

func (g *Game) ShortSell(symbol string, shares int) error {
	s := g.Market.GetStock(symbol)
	if s == nil {
		return fmt.Errorf("unknown symbol %s", symbol)
	}
	margin := s.Price * float64(shares) * 0.5
	if margin > g.Cash {
		return fmt.Errorf("insufficient margin: need $%.2f (50%% of position)", margin)
	}
	g.Cash -= margin
	pos := g.getOrCreatePos(symbol)
	total := float64(pos.ShortShares)*pos.ShortAvg + s.Price*float64(shares)
	pos.ShortShares += shares
	if pos.ShortShares > 0 {
		pos.ShortAvg = total / float64(pos.ShortShares)
	}
	g.nextOrderID++
	g.Orders = append(g.Orders, &Order{
		ID: g.nextOrderID, Symbol: symbol, Type: OrderShortSell,
		Shares: shares, FilledAt: s.Price, Status: OrderFilled,
	})
	g.addMsg(fmt.Sprintf("Short sold %d shares of %s @ $%.2f", shares, symbol, s.Price))
	return nil
}

func (g *Game) CoverShort(symbol string, shares int) error {
	pos, ok := g.Positions[symbol]
	if !ok || pos.ShortShares < shares {
		available := 0
		if ok {
			available = pos.ShortShares
		}
		return fmt.Errorf("insufficient short position: have %d short", available)
	}
	s := g.Market.GetStock(symbol)
	if s == nil {
		return fmt.Errorf("unknown symbol %s", symbol)
	}
	pnl := float64(shares) * (pos.ShortAvg - s.Price)
	margin := pos.ShortAvg * float64(shares) * 0.5
	g.Cash += margin + pnl
	pos.ShortShares -= shares
	g.nextOrderID++
	g.Orders = append(g.Orders, &Order{
		ID: g.nextOrderID, Symbol: symbol, Type: OrderShortCover,
		Shares: shares, FilledAt: s.Price, Status: OrderFilled,
	})
	g.addMsg(fmt.Sprintf("Covered %d short %s @ $%.2f (P&L: $%.2f)", shares, symbol, s.Price, pnl))
	return nil
}

func (g *Game) CancelOrder(id int) bool {
	for _, o := range g.Orders {
		if o.ID == id && o.Status == OrderPending {
			o.Status = OrderCancelled
			s := g.Market.GetStock(o.Symbol)
			if o.Type == OrderLimitBuy && s != nil {
				g.Cash += o.LimitPrice * float64(o.Shares)
			} else if o.Type == OrderLimitSell {
				pos := g.getOrCreatePos(o.Symbol)
				pos.Shares += o.Shares
			}
			g.addMsg(fmt.Sprintf("Order #%d cancelled", id))
			return true
		}
	}
	return false
}

func (g *Game) ProcessLimitOrders() {
	for _, o := range g.Orders {
		if o.Status != OrderPending {
			continue
		}
		s := g.Market.GetStock(o.Symbol)
		if s == nil {
			continue
		}
		switch o.Type {
		case OrderLimitBuy:
			if s.Price <= o.LimitPrice {
				refund := (o.LimitPrice - s.Price) * float64(o.Shares)
				g.Cash += refund
				pos := g.getOrCreatePos(o.Symbol)
				total := float64(pos.Shares)*pos.AvgCost + s.Price*float64(o.Shares)
				pos.Shares += o.Shares
				if pos.Shares > 0 {
					pos.AvgCost = total / float64(pos.Shares)
				}
				o.FilledAt = s.Price
				o.Status = OrderFilled
				g.addMsg(fmt.Sprintf("Limit BUY filled: %d %s @ $%.2f", o.Shares, o.Symbol, s.Price))
			}
		case OrderLimitSell:
			if s.Price >= o.LimitPrice {
				g.Cash += s.Price * float64(o.Shares)
				o.FilledAt = s.Price
				o.Status = OrderFilled
				g.addMsg(fmt.Sprintf("Limit SELL filled: %d %s @ $%.2f", o.Shares, o.Symbol, s.Price))
			}
		}
	}
}

func (g *Game) PortfolioValue() float64 {
	total := g.Cash
	for symbol, pos := range g.Positions {
		s := g.Market.GetStock(symbol)
		if s == nil {
			continue
		}
		total += pos.MarketValue(s.Price)
		total += pos.ShortPnL(s.Price) + float64(pos.ShortShares)*pos.ShortAvg*0.5
	}
	return total
}

func (g *Game) PendingOrders() []*Order {
	var out []*Order
	for _, o := range g.Orders {
		if o.Status == OrderPending {
			out = append(out, o)
		}
	}
	return out
}

func (g *Game) FilledOrders() []*Order {
	var out []*Order
	for i := len(g.Orders) - 1; i >= 0 && len(out) < 30; i-- {
		if g.Orders[i].Status == OrderFilled {
			out = append(out, g.Orders[i])
		}
	}
	return out
}

func (g *Game) getOrCreatePos(symbol string) *Position {
	if p, ok := g.Positions[symbol]; ok {
		return p
	}
	p := &Position{Symbol: symbol}
	g.Positions[symbol] = p
	return p
}
