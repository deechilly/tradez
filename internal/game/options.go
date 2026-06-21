package game

import (
	"fmt"
	"math"
	"time"
	"tradez/internal/market"
)

// sharesPerContract is the number of underlying shares one put contract covers.
const sharesPerContract = 100

// PutExpiry labels the three available expiry durations.
type PutExpiry int

const (
	PutExpiryShort  PutExpiry = 30  // ~1 min
	PutExpiryMedium PutExpiry = 90  // ~3 min
	PutExpiryLong   PutExpiry = 150 // ~5 min
)

func (e PutExpiry) Label() string {
	switch e {
	case PutExpiryShort:
		return "Short (~1 min)"
	case PutExpiryMedium:
		return "Medium (~3 min)"
	case PutExpiryLong:
		return "Long (~5 min)"
	}
	return ""
}

// PutContract represents a long put position held by the player.
type PutContract struct {
	ID        int       `json:"id"`
	Symbol    string    `json:"symbol"`
	Strike    float64   `json:"strike"`
	Premium   float64   `json:"premium"`   // total cash paid (per-share × 100 × contracts)
	Contracts int       `json:"contracts"` // number of contracts
	ExpiresAt time.Time `json:"expires_at"`
	BoughtAt  time.Time `json:"bought_at"`
}

// TicksLeft returns ticks remaining until expiry, clamped to zero.
func (p *PutContract) TicksLeft() int {
	d := time.Until(p.ExpiresAt)
	if d <= 0 {
		return 0
	}
	ticks := int(d.Seconds() / 2)
	return ticks
}

// IsExpired reports whether the contract has passed its expiry time.
func (p *PutContract) IsExpired() bool {
	return time.Now().After(p.ExpiresAt)
}

// CurrentValue returns the current mark-to-market value of the full position.
func (p *PutContract) CurrentValue(spot float64, iv float64) float64 {
	price := market.PutPrice(spot, p.Strike, iv, p.TicksLeft())
	return price * float64(sharesPerContract) * float64(p.Contracts)
}

// UnrealizedPnL is current value minus premium paid.
func (p *PutContract) UnrealizedPnL(spot, iv float64) float64 {
	return p.CurrentValue(spot, iv) - p.Premium
}

// BuyPut purchases put contracts on the given symbol.
//
//   strike    – strike price chosen by the player
//   expiry    – PutExpiryShort / Medium / Long (tick count)
//   contracts – number of contracts (each covers 100 shares)
func (g *Game) BuyPut(symbol string, strike float64, expiry PutExpiry, contracts int) error {
	s := g.Market.GetStock(symbol)
	if s == nil {
		return fmt.Errorf("unknown symbol %s", symbol)
	}
	if contracts <= 0 {
		return fmt.Errorf("contract count must be positive")
	}
	iv := market.ImpliedVol(s.Volatility)
	pricePerShare := market.PutPrice(s.Price, strike, iv, int(expiry))
	totalPremium := pricePerShare * float64(sharesPerContract) * float64(contracts)
	if totalPremium > g.Cash {
		return fmt.Errorf("insufficient funds: need $%.2f, have $%.2f", totalPremium, g.Cash)
	}
	g.Cash -= totalPremium
	g.nextPutID++
	put := &PutContract{
		ID:        g.nextPutID,
		Symbol:    symbol,
		Strike:    strike,
		Premium:   totalPremium,
		Contracts: contracts,
		ExpiresAt: time.Now().Add(time.Duration(int(expiry)) * 2 * time.Second),
		BoughtAt:  time.Now(),
	}
	g.Puts = append(g.Puts, put)
	g.addMsg(fmt.Sprintf("Bought %d put contract(s) on %s @ $%.2f strike (paid $%.2f)", contracts, symbol, strike, totalPremium))
	return nil
}

// SellPut closes (or partially closes) a put position at current market value.
func (g *Game) SellPut(id, contracts int) error {
	for _, put := range g.Puts {
		if put.ID != id {
			continue
		}
		if contracts > put.Contracts {
			return fmt.Errorf("only hold %d contract(s)", put.Contracts)
		}
		s := g.Market.GetStock(put.Symbol)
		if s == nil {
			return fmt.Errorf("stock %s not found", put.Symbol)
		}
		iv := market.ImpliedVol(s.Volatility)
		pricePerShare := market.PutPrice(s.Price, put.Strike, iv, put.TicksLeft())
		proceeds := pricePerShare * float64(sharesPerContract) * float64(contracts)
		g.Cash += proceeds
		pnl := proceeds - (put.Premium/float64(put.Contracts))*float64(contracts)
		put.Contracts -= contracts
		if put.Contracts == 0 {
			g.removePut(id)
		}
		g.addMsg(fmt.Sprintf("Sold %d put(s) on %s for $%.2f (P&L: %s$%.2f)", contracts, put.Symbol, proceeds, signStr(pnl), math.Abs(pnl)))
		return nil
	}
	return fmt.Errorf("put contract #%d not found", id)
}

// ProcessPuts expires contracts that have passed their expiry time.
// ITM contracts are settled at intrinsic value; OTM contracts expire worthless.
// Returns the slice of newly expired contracts (for flash notifications).
func (g *Game) ProcessPuts() []*PutContract {
	var expired []*PutContract
	remaining := g.Puts[:0]
	for _, put := range g.Puts {
		if !put.IsExpired() {
			remaining = append(remaining, put)
			continue
		}
		s := g.Market.GetStock(put.Symbol)
		intrinsic := 0.0
		if s != nil {
			intrinsic = market.PutIntrinsic(s.Price, put.Strike) * float64(sharesPerContract) * float64(put.Contracts)
		}
		if intrinsic > 0 {
			g.Cash += intrinsic
			pnl := intrinsic - put.Premium
			g.addMsg(fmt.Sprintf("Put #%d on %s expired ITM — settled $%.2f (P&L: %s$%.2f)", put.ID, put.Symbol, intrinsic, signStr(pnl), math.Abs(pnl)))
		} else {
			g.addMsg(fmt.Sprintf("Put #%d on %s expired worthless", put.ID, put.Symbol))
		}
		expired = append(expired, put)
	}
	g.Puts = remaining
	return expired
}

func (g *Game) removePut(id int) {
	out := g.Puts[:0]
	for _, p := range g.Puts {
		if p.ID != id {
			out = append(out, p)
		}
	}
	g.Puts = out
}

func signStr(v float64) string {
	if v >= 0 {
		return "+"
	}
	return ""
}

// PutStrikes returns five candidate strike prices for a stock at standard
// moneyness levels: deep ITM, ITM, ATM, OTM, deep OTM.
func PutStrikes(price float64) [5]float64 {
	return [5]float64{
		roundStrike(price * 1.20), // deep ITM
		roundStrike(price * 1.10), // ITM
		roundStrike(price * 1.00), // ATM
		roundStrike(price * 0.90), // OTM
		roundStrike(price * 0.80), // deep OTM
	}
}

func roundStrike(v float64) float64 {
	// Round to nearest 0.50
	return math.Round(v*2) / 2
}
