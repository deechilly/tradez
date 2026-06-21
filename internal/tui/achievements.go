package tui

import (
	"fmt"
	"math"
	"strings"
	"time"
	"tradez/internal/analysis"
	"tradez/internal/game"
	"tradez/internal/market"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// positionTrack holds per-position state for achievement detection.
type positionTrack struct {
	openedAt      time.Time
	lastTouchedAt time.Time
	minPnLPct     float64   // most negative P&L% seen (e.g. -0.40 = 40% down)
	underwaterAt  time.Time // when it went below -40%; zero if currently above
}

// sellEntry records a sell for post-sell price tracking (paper hands / sell the dip).
type sellEntry struct {
	symbol    string
	sellPrice float64
	costBasis float64 // avg cost at time of sell
	soldAt    time.Time
}

// unlockAchievement marks an achievement unlocked and queues the flash notification.
func (m *Model) unlockAchievement(id string) {
	if m.achStore.Unlock(id) {
		m.achieveQueue = append(m.achieveQueue, id)
		if m.flashAchieveTick == 0 && len(m.achieveQueue) > 0 {
			m.flashAchieveID = m.achieveQueue[0]
			m.achieveQueue = m.achieveQueue[1:]
			m.flashAchieveTick = 8
		}
	}
}

// tickAchieveFash advances the flash timer and pops the next queued achievement.
func (m *Model) tickAchieveFlash() {
	if m.flashAchieveTick > 0 {
		m.flashAchieveTick--
		if m.flashAchieveTick == 0 {
			m.flashAchieveID = ""
			if len(m.achieveQueue) > 0 {
				m.flashAchieveID = m.achieveQueue[0]
				m.achieveQueue = m.achieveQueue[1:]
				m.flashAchieveTick = 8
			}
		}
	}
}

// updateStormWatch adds positions to the storm watch when a negative news event fires.
func (m *Model) updateStormWatch(event *market.NewsEvent) {
	for symbol, pos := range m.g.Positions {
		if pos.Shares == 0 {
			continue
		}
		s := m.g.Market.GetStock(symbol)
		if s == nil {
			continue
		}
		sent, ok := event.SentimentFor(s)
		if ok && sent < -0.2 {
			if _, already := m.stormWatch[symbol]; !already {
				m.stormWatch[symbol] = time.Now()
			}
		}
	}
}

// checkNewLimitFills checks limit buy fills for the Free Real Estate achievement.
func (m *Model) checkNewLimitFills() {
	for _, o := range m.g.Orders {
		if o.Status != game.OrderFilled || o.Type != game.OrderLimitBuy {
			continue
		}
		if m.seenFilledOrders[o.ID] {
			continue
		}
		m.seenFilledOrders[o.ID] = true
		if o.LimitPrice > 0 {
			discount := 1.0 - o.FilledAt/o.LimitPrice
			if discount >= 0.10 {
				m.unlockAchievement("free_real_estate")
			}
		}
	}
}

// checkTickAchievements runs checks that depend on the current market state each tick.
func (m *Model) checkTickAchievements(prevPortfolioVal float64) {
	now := time.Now()
	portfolioVal := m.g.PortfolioValue()
	startingCash := m.g.StartingCash
	if startingCash == 0 {
		return
	}

	// ── portfolio-level checks ───────────────────────────────────────────────

	// WAGMI: portfolio ≥ 5× starting capital
	if portfolioVal >= startingCash*5 {
		m.unlockAchievement("wagmi")
	}

	// BUST: portfolio < 5% of starting capital
	if portfolioVal < startingCash*0.05 {
		m.unlockAchievement("bust")
	}

	// LOSS PORN: portfolio < 30% of starting capital (70%+ down)
	if portfolioVal < startingCash*0.30 {
		m.unlockAchievement("loss_porn")
	}

	// APE STRONG: 8+ simultaneous long positions
	longCount := 0
	for _, pos := range m.g.Positions {
		if pos.Shares > 0 {
			longCount++
		}
	}
	if longCount >= 8 {
		m.unlockAchievement("ape_strong")
	}

	// ── position-level checks ────────────────────────────────────────────────

	for symbol, pt := range m.posTrack {
		pos := m.g.Positions[symbol]
		if pos == nil || pos.Shares == 0 {
			delete(m.posTrack, symbol)
			continue
		}
		s := m.g.Market.GetStock(symbol)
		if s == nil || pos.AvgCost == 0 {
			continue
		}
		pnlPct := s.Price/pos.AvgCost - 1.0

		// Update low-water mark
		if pnlPct < pt.minPnLPct {
			pt.minPnLPct = pnlPct
		}

		// TO THE MOON: position up 100%+
		if pnlPct >= 1.0 {
			m.unlockAchievement("to_the_moon")
		}

		// BAG HOLDER: 40%+ underwater for 90+ seconds
		if pnlPct <= -0.40 {
			if pt.underwaterAt.IsZero() {
				pt.underwaterAt = now
			} else if now.Sub(pt.underwaterAt) >= 90*time.Second {
				m.unlockAchievement("bag_holder")
			}
		} else {
			pt.underwaterAt = time.Time{}
		}

		// I LIKE THE STOCK: position untouched for 5+ minutes
		if !pt.lastTouchedAt.IsZero() && now.Sub(pt.lastTouchedAt) >= 5*time.Minute {
			m.unlockAchievement("i_like_the_stock")
		}
	}

	// SQUEEZED: short unrealized loss ≥ 75% of margin posted
	for symbol, pos := range m.g.Positions {
		if pos.ShortShares == 0 || pos.ShortAvg == 0 {
			continue
		}
		s := m.g.Market.GetStock(symbol)
		if s == nil {
			continue
		}
		margin := pos.ShortAvg * float64(pos.ShortShares) * 0.5
		unrealizedLoss := math.Max(0, (s.Price-pos.ShortAvg)*float64(pos.ShortShares))
		if margin > 0 && unrealizedLoss/margin >= 0.75 {
			m.unlockAchievement("squeezed")
		}
	}

	// PAPER HANDS & SELL THE DIP: monitor post-sell price action
	cutoff := now.Add(-5 * time.Minute)
	validSells := m.sellLog[:0]
	for _, se := range m.sellLog {
		if se.soldAt.Before(cutoff) {
			continue // expire after 5 minutes
		}
		validSells = append(validSells, se)
		s := m.g.Market.GetStock(se.symbol)
		if s == nil {
			continue
		}
		// PAPER HANDS: sold at any loss, price now ≥ 20% above sell price
		if se.costBasis > se.sellPrice && s.Price >= se.sellPrice*1.20 {
			m.unlockAchievement("paper_hands")
		}
		// SELL THE DIP: sold ≥ 15% below cost, price now ≥ 25% above sell price
		if se.sellPrice < se.costBasis*0.85 && s.Price >= se.sellPrice*1.25 {
			m.unlockAchievement("sell_the_dip")
		}
	}
	m.sellLog = validSells

	// WEATHER THE STORM: held through 60 seconds after negative news hit
	for symbol, hitTime := range m.stormWatch {
		if now.Sub(hitTime) < 60*time.Second {
			continue
		}
		pos := m.g.Positions[symbol]
		if pos != nil && pos.Shares > 0 {
			m.unlockAchievement("weather_storm")
		}
		delete(m.stormWatch, symbol)
	}

	_ = prevPortfolioVal
}

// checkTradeAchievements fires after a market order completes successfully.
// prePortfolioVal, preCash, preAvgCost, preShortAvg are all recorded BEFORE the trade call.
func (m *Model) checkTradeAchievements(
	prePortfolioVal, preCash, preAvgCost, preShortAvg float64,
	preShares, preShortShares, orderShares int,
	mode tradeMode, s *market.Stock,
) {
	fillPrice := s.Price

	switch mode {
	case tradeBuy:
		cost := fillPrice * float64(orderShares)

		// YOLO: single buy ≥ 85% of total portfolio value
		if prePortfolioVal > 0 && cost/prePortfolioVal >= 0.85 {
			m.unlockAchievement("yolo")
		}

		// WIDOW MAKER: buy consumes ≥ 95% of available cash
		if preCash > 0 && cost/preCash >= 0.95 {
			m.unlockAchievement("widow_maker")
		}

		// Update / create position track
		if pt := m.posTrack[s.Symbol]; pt == nil {
			m.posTrack[s.Symbol] = &positionTrack{
				openedAt:      time.Now(),
				lastTouchedAt: time.Now(),
			}
		} else {
			pt.lastTouchedAt = time.Now()
		}

		// NOT FINANCIAL ADVICE: buy when STRONG SELL
		if m.lastAnalysisScore < -0.45 {
			m.unlockAchievement("not_financial_advice")
		}

	case tradeSell:
		if preAvgCost > 0 {
			profitPct := fillPrice/preAvgCost - 1.0
			profit := (fillPrice - preAvgCost) * float64(orderShares)

			// BOOM: 200%+ return on a single close (3× cost)
			if profitPct >= 2.0 {
				m.unlockAchievement("boom")
			}

			// TENDIES: profit ≥ 5× starting capital from single close
			if profit >= m.g.StartingCash*5 {
				m.unlockAchievement("tendies")
			}

			// GUH: realized loss ≥ 30% of portfolio value
			if profitPct < 0 {
				realizedLoss := -profit
				if prePortfolioVal > 0 && realizedLoss/prePortfolioVal >= 0.30 {
					m.unlockAchievement("guh")
				}
			}

			// DIAMOND HANDS: closed at profit, was 35%+ underwater
			if profitPct > 0 {
				if pt := m.posTrack[s.Symbol]; pt != nil && pt.minPnLPct <= -0.35 {
					m.unlockAchievement("diamond_hands")
				}
			}

			// Log the sell for paper hands / sell the dip tracking (only if at a loss)
			if fillPrice < preAvgCost {
				m.sellLog = append(m.sellLog, sellEntry{
					symbol:    s.Symbol,
					sellPrice: fillPrice,
					costBasis: preAvgCost,
					soldAt:    time.Now(),
				})
				if len(m.sellLog) > 30 {
					m.sellLog = m.sellLog[len(m.sellLog)-30:]
				}
			}

			// INFINITE MONEY GLITCH: profit on long AND short of same ticker
			if profitPct > 0 {
				wins := m.perSymbolWins[s.Symbol]
				wins[0] = true
				m.perSymbolWins[s.Symbol] = wins
				if wins[1] {
					m.unlockAchievement("infinite_money_glitch")
				}
			}
		}

		// NOT FINANCIAL ADVICE: sell when STRONG BUY
		if m.lastAnalysisScore > 0.45 {
			m.unlockAchievement("not_financial_advice")
		}

		// Update / remove position track
		if pt := m.posTrack[s.Symbol]; pt != nil {
			pt.lastTouchedAt = time.Now()
		}
		pos := m.g.Positions[s.Symbol]
		if pos == nil || pos.Shares == 0 {
			delete(m.posTrack, s.Symbol)
		}
		// Sold it — no longer weathering the storm
		delete(m.stormWatch, s.Symbol)

	case tradeShort:
		// short opened — nothing to check on entry

	case tradeCover:
		if preShortAvg > 0 {
			profit := (preShortAvg - fillPrice) * float64(orderShares)

			// GUH: covering at a big loss
			if profit < 0 {
				realizedLoss := -profit
				if prePortfolioVal > 0 && realizedLoss/prePortfolioVal >= 0.30 {
					m.unlockAchievement("guh")
				}
			}

			// SHORT BUS: profit from 3 short positions in session
			if profit > 0 {
				m.sessionShortWins++
				if m.sessionShortWins >= 3 {
					m.unlockAchievement("short_bus")
				}

				// INFINITE MONEY GLITCH
				wins := m.perSymbolWins[s.Symbol]
				wins[1] = true
				m.perSymbolWins[s.Symbol] = wins
				if wins[0] {
					m.unlockAchievement("infinite_money_glitch")
				}
			}
		}
	}

	_ = preShares
	_ = preShortShares
}

// analysisComposite computes the weighted analyst composite score for a stock.
// Mirrors the calculation in renderAnalysis without the rendering.
func (m Model) analysisComposite(s *market.Stock) float64 {
	if s == nil || m.g == nil {
		return 0
	}
	return analysis.Analyze(m.g.Market, s).CompositeScore
}

// ── Achievement screen ────────────────────────────────────────────────────────

func (m Model) viewAchievements() string {
	w := m.width
	if w == 0 {
		w = 100
	}

	unlocked := m.achStore.UnlockedCount()
	total := len(game.AllAchievements)

	div := styleNeutral.Render(strings.Repeat("─", w))

	// Build the list: unlocked first, then locked
	var allDefs []game.AchievementDef
	for _, def := range game.AllAchievements {
		if m.achStore.IsUnlocked(def.ID) {
			allDefs = append(allDefs, def)
		}
	}
	for _, def := range game.AllAchievements {
		if !m.achStore.IsUnlocked(def.ID) {
			allDefs = append(allDefs, def)
		}
	}

	descW := w - 8
	if descW < 20 {
		descW = 20
	}

	// Budget: header(1) + div(1) + body + div(1) + keys(1) = 4 fixed lines.
	// Body = achievement lines + optional "make some trades" message (1 line).
	messageLines := 0
	if unlocked == 0 {
		messageLines = 1
	}
	visH := max(m.height-4-messageLines, 1)

	maxScroll := max(len(allDefs)*3-visH, 0)
	scroll := m.achScroll
	if scroll > maxScroll {
		scroll = maxScroll
	}
	if scroll < 0 {
		scroll = 0
	}

	// Build the scrollable body, tracking rendered lines independently of scroll offset.
	body := ""
	lineCount := 0
	bodyLines := 0
	for _, def := range allDefs {
		isUnlocked := m.achStore.IsUnlocked(def.ID)
		if lineCount+3 > scroll {
			if isUnlocked {
				t := m.achStore.Unlocked[def.ID]
				nameLine := stylePositive.Render(" ✓ ") + styleWhiteStr(def.Icon+" "+def.Name) +
					styleNeutral.Render("   Unlocked "+t.Format("Jan 2, 2006"))
				body += nameLine + "\n"
				body += styleNeutral.Render("     "+truncate(def.Desc, descW)) + "\n"
			} else {
				grayStyle := lipgloss.NewStyle().Foreground(colorGray)
				body += styleNeutral.Render(" ✗ ") + grayStyle.Render(def.Icon+" "+def.Name) + "\n"
				body += grayStyle.Render("     "+truncate(def.Desc, descW)) + "\n"
			}
			body += "\n"
			bodyLines += 3
		}
		lineCount += 3
		if bodyLines >= visH {
			break
		}
	}

	if unlocked == 0 {
		body += styleNeutral.Render("  Make some trades to unlock your first achievement!") + "\n"
	}

	scrollHint := ""
	if len(allDefs)*3 > visH {
		scrollHint = "  pgup/pgdn to scroll"
	}
	keys := styleHint.Render(" esc=back" + scrollHint)

	header := styleTitle.Render(" ACHIEVEMENTS ") +
		styleNeutral.Render(fmt.Sprintf("  %d / %d unlocked", unlocked, total))

	return header + "\n" + div + "\n" + body + div + "\n" + keys
}

func (m Model) handleAchievementsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.screen = screenMarket
	case "pgup", "up", "k":
		m.achScroll -= 3
		if m.achScroll < 0 {
			m.achScroll = 0
		}
	case "pgdown", "down", "j":
		m.achScroll += 3
	}
	return m, nil
}
