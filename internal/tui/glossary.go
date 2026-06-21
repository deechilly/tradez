package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type glossaryEntry struct {
	term string
	def  string
}

type glossarySection struct {
	title   string
	entries []glossaryEntry
}

var glossarySections = []glossarySection{
	{
		title: "ORDER TYPES",
		entries: []glossaryEntry{
			{
				"Market Buy",
				"Purchase shares immediately at the current price. Fast and certain, but you get whatever the market is asking right now — which may have moved since you last looked.",
			},
			{
				"Market Sell",
				"Sell shares you own at the current price. Your holding is converted to cash immediately.",
			},
			{
				"Limit Buy",
				"A buy order that only fills when the price drops to or below your target. Your cash is reserved when you place it. Use limit buys to catch dips without watching the ticker constantly.",
			},
			{
				"Limit Sell",
				"A sell order that only fills when the price rises to or above your target. Your shares are reserved. Use limit sells to lock in a profit target automatically.",
			},
			{
				"Short Sell",
				"Borrow shares and sell them immediately, betting the price will fall. If the price drops, you buy them back cheaper (cover) and pocket the difference. Risk: if the stock rises, losses are unlimited — there is no ceiling on how high a price can go. Requires 50% of the position value as margin collateral.",
			},
			{
				"Cover Short",
				"Close a short position by buying back the borrowed shares at the current price. Your margin collateral is returned and the P&L (profit or loss) is settled.",
			},
			{
				"Buy Put",
				"Purchase a put options contract that profits if the stock falls below a chosen strike price before expiry. You pay a premium upfront, which is your maximum possible loss — unlike a short, a put can never blow up beyond the premium you paid.",
			},
		},
	},
	{
		title: "PUT OPTIONS",
		entries: []glossaryEntry{
			{
				"Put Contract",
				"A financial derivative giving you the right (but not the obligation) to profit if a stock trades below a strike price at expiry. One contract covers 100 shares. The put gains value as the stock falls; it expires worthless if the stock stays above the strike.",
			},
			{
				"Strike Price",
				"The reference price that determines a put's intrinsic value. A put is profitable when the stock trades below its strike ('in the money'). Selecting a lower strike is cheaper but requires a larger move to profit.",
			},
			{
				"Premium",
				"The price you pay per contract when buying a put. This is the maximum you can lose on the trade. It reflects both the option's intrinsic value and the time value remaining before expiry.",
			},
			{
				"Intrinsic Value",
				"The immediate, exercise-right-now value of an option. For a put: max(strike − current price, 0) × 100. An ATM or OTM put has zero intrinsic value — you are paying only for the possibility the stock moves before expiry.",
			},
			{
				"Time Value & Theta Decay",
				"The portion of the premium above intrinsic value. It reflects the probability that the stock reaches a profitable level before expiry. Time value erodes to zero by expiry — this erosion is called theta decay. OTM options lose their entire premium if the stock does not move. ITM options retain intrinsic value at expiry regardless of theta.",
			},
			{
				"In the Money (ITM)",
				"A put where the stock price is already below the strike. It has intrinsic value right now. ITM puts cost more but have a higher probability of returning a profit, and are less reliant on a big move.",
			},
			{
				"At the Money (ATM)",
				"Strike price approximately equals the current stock price. The put is entirely time value — it has the highest sensitivity to changes in implied volatility and benefits most from a sudden sharp move in either direction.",
			},
			{
				"Out of the Money (OTM)",
				"Strike price is below the current stock price. No intrinsic value; you need the stock to fall past the strike before expiry just to break even. Cheaper than ITM but high-risk: many expire worthless. Think of OTM puts as cheap lottery tickets on a crash.",
			},
			{
				"Break-even Price",
				"The stock price at which your put neither profits nor loses at expiry: strike − (premium per share). The stock must fall below this point for a net gain. Displayed in the put buy dialog so you can assess the required move.",
			},
			{
				"Implied Volatility (IV)",
				"The market's forward-looking estimate of how wildly a stock will swing, extracted from option prices. High-IV stocks (crypto, speculative tech) have more expensive premiums for the same strike and expiry — you pay for the possibility of a big move. tradez derives IV from the stock's simulated per-tick volatility using: IV = per-tick vol × √1000.",
			},
			{
				"Black-Scholes Model",
				"The mathematical framework used to price options. Inputs: current stock price, strike price, time to expiry, implied volatility, and a risk-free interest rate. It produces a theoretical 'fair value' for the option. tradez uses Black-Scholes to price all put contracts, with 1000 ticks per year and a 5% risk-free rate.",
			},
		},
	},
	{
		title: "ANALYSIS METRICS",
		entries: []glossaryEntry{
			{
				"P/E Ratio (Price-to-Earnings)",
				"Share price divided by annual earnings per share. A high P/E suggests investors expect strong future growth — or that the stock is overpriced. A low P/E may indicate good value or market pessimism. P/Es are most meaningful when compared within the same industry: tech companies routinely carry P/Es of 30–100 while utilities might be 12–18.",
			},
			{
				"EPS (Earnings Per Share)",
				"A company's net profit divided by its total shares outstanding. Positive EPS means the business is profitable; negative means it's losing money. The analyst's fundamentals score rewards high EPS as a proxy for business quality.",
			},
			{
				"YTD Growth (Year-to-Date)",
				"Percentage price change since the start of the year. The analyst uses this as a contrarian signal: stocks that have fallen sharply may be oversold and score more bullish (potential rebound); stocks that have soared may be overbought and score more bearish (potential mean reversion). Neither signal is guaranteed.",
			},
			{
				"Dividend Yield",
				"Annual dividend payment as a percentage of share price. A positive yield means the company returns cash to shareholders — the analyst treats it as a mild bullish quality signal. Stocks that do not pay dividends are reinvesting earnings for growth instead, which is neither inherently good nor bad.",
			},
			{
				"Beta",
				"A stock's sensitivity to broad market moves. Beta 1.0 = moves in lockstep with the market. Beta 2.0 = swings roughly twice as hard in either direction. Beta 0.5 = half as volatile. In tradez, high-beta stocks (tech, crypto) amplify gains from strong news events — but suffer proportionally larger losses when news turns negative. Use beta to size positions: smaller positions in high-beta names reduce risk.",
			},
			{
				"20-Period Moving Average",
				"The average closing price over the last 20 ticks (~40 real-world seconds). Price above the MA suggests short-term bullish momentum; below suggests bearish pressure. The analyst weights this in the technicals score.",
			},
			{
				"50-Period Moving Average",
				"The average price over the last 50 ticks (~100 seconds). A longer-term trend signal. When the 20-period MA crosses above the 50-period MA, traders call this a 'golden cross' — a bullish signal. The reverse (20 crossing below 50) is a 'death cross' — bearish. Both show up in the analyst's composite score.",
			},
			{
				"Momentum",
				"The direction and rate of recent price change, normalised to a −1 to +1 scale, calculated over the last 15 ticks. Positive momentum means prices have been climbing consistently; negative means falling. Momentum tends to persist in the short run but mean-revert over longer periods. The analyst includes it in the technicals score.",
			},
			{
				"Market Signals",
				"The net directional force of active news events on this stock right now. Each news event creates an influence that decays over time. The signals panel shows the current strength and direction. A stock with strong bullish signals and rising technicals is the analyst's strongest candidate.",
			},
			{
				"Composite Score",
				"A weighted average of Fundamentals (30%), Technicals (35%), and Market Signals (35%), mapped to a five-tier rating: Strong Buy → Buy → Hold → Sell → Strong Sell. Treat it as one data point among many — it summarises current conditions, it does not predict the future. News events can flip a Strong Buy to a Strong Sell within seconds.",
			},
		},
	},
	{
		title: "MARKET CONCEPTS",
		entries: []glossaryEntry{
			{
				"Volatility",
				"The standard deviation of price changes — how wildly a stock swings each tick. High-volatility stocks offer bigger potential gains but bigger potential losses. In tradez, crypto and speculative tech carry the highest volatility. Volatility also drives option premiums: a volatile stock has more expensive puts.",
			},
			{
				"Bull Market / Bear Market",
				"A bull market is a period of rising prices and optimism; a bear market is falling prices and pessimism. In tradez, news events create short sector-specific bull or bear runs. Identifying which sector a headline affects — and in which direction — is the core skill of the game.",
			},
			{
				"Sector Rotation",
				"The flow of capital from one industry to another, typically driven by macro news. A central bank rate hike simultaneously hurts Technology and Real Estate (higher borrowing costs) while benefiting Finance (wider net interest margins) and Utilities (bond-like appeal). Spotting rotation early lets you exit weakening sectors and enter strengthening ones before the crowd.",
			},
			{
				"Short Squeeze",
				"When a heavily-shorted stock rises sharply, short-sellers are forced to buy back shares to limit losses. This buying itself drives the price higher, triggering even more forced covering — a self-reinforcing spiral. Watch for stocks with negative recent performance and a sudden burst of strong positive news; the squeeze can be violent and fast.",
			},
			{
				"Margin",
				"Cash or collateral pledged against a leveraged position. In tradez, short selling requires 50% margin: if you short 100 shares at $50, you post $2,500 as collateral. The margin is returned when you cover; profit or loss is added on top. If you cover at $40, you recover $2,500 margin plus $1,000 profit (100 × $10 gain).",
			},
			{
				"Position Sizing",
				"How much capital you allocate to a single trade. Concentration amplifies gains when you're right — and losses when you're wrong. Spreading capital across multiple stocks and trade types (long, short, puts) reduces the risk of any single bad trade wiping out your portfolio. As a rule of thumb: risk no more than 5–10% of your total portfolio on any one position.",
			},
		},
	},
	{
		title: "READING THE FEED",
		entries: []glossaryEntry{
			{
				"Influence Strength",
				"Each news event creates a directional force on one or more stocks that decays linearly over its lifetime (roughly 2–10 minutes). The strength bar in the Market Feed shows remaining force. As it fades, the news-driven drift disappears and the stock reverts toward its underlying trend. On Easy difficulty the timer is visible; on Normal it is hidden.",
			},
			{
				"Sentiment (▲▲ ▲ ◆ ▼ ▼▼)",
				"The directional bias of a news event on a specific stock or sector. ▲▲ = strongly bullish (large upward price drift). ▲ = mildly bullish. ◆ = neutral. ▼ = mildly bearish. ▼▼ = strongly bearish. The same headline can be bullish for one sector and bearish for another simultaneously.",
			},
			{
				"Winners & Losers",
				"Most macro news events create opposite effects across sectors. An energy supply shock lifts Energy stocks while depressing Consumer and Transport names. A tech breakthrough boosts Technology while making competing sectors look stale. The feed shows both sides — going long the winner and short the loser from the same headline is a classic pair trade.",
			},
			{
				"News Cycle",
				"Events arrive on a random timer and each has a finite lifespan. Once an event fades, prices return to their baseline drift. The most profitable window is often shortly after a major event lands — prices are still moving in response, options premiums spike with IV, and there is time left on the influence before it decays.",
			},
		},
	},
}

func (m Model) viewGlossary() string {
	w := m.width
	if w == 0 {
		w = 120
	}

	div := styleNeutral.Render(strings.Repeat("─", w))
	termW := 26
	defW := w - termW - 4
	if defW < 40 {
		termW = 0
		defW = w - 4
	}
	if defW > 110 {
		defW = 110
	}

	// Build all content lines once, then window them.
	var lines []string
	styleTermName := lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
	styleDefText := lipgloss.NewStyle().Foreground(colorGray)
	styleSectionTitle := lipgloss.NewStyle().Bold(true).Foreground(colorGold)

	for si, sec := range glossarySections {
		if si > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, styleSectionTitle.Render("  "+sec.title))
		lines = append(lines, div)

		for _, e := range sec.entries {
			defLines := wrapText(e.def, defW)
			if termW > 0 {
				// Side-by-side: term column | definition column
				pad := termW - len(e.term)
				if pad < 1 {
					pad = 1
				}
				firstLine := styleTermName.Render(e.term) +
					strings.Repeat(" ", pad) +
					styleDefText.Render(defLines[0])
				lines = append(lines, "  "+firstLine)
				for _, dl := range defLines[1:] {
					lines = append(lines, "  "+strings.Repeat(" ", termW)+styleDefText.Render(dl))
				}
			} else {
				// Narrow terminal: stacked layout
				lines = append(lines, "  "+styleTermName.Render(e.term))
				for _, dl := range defLines {
					lines = append(lines, "    "+styleDefText.Render(dl))
				}
			}
			lines = append(lines, "")
		}
	}

	// 4 fixed lines: header + div + div + hints
	visH := m.height - 4
	if visH < 1 {
		visH = 1
	}

	scroll := m.glossaryScroll
	maxScroll := len(lines) - visH
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scroll > maxScroll {
		scroll = maxScroll
	}
	if scroll < 0 {
		scroll = 0
	}

	end := scroll + visH
	if end > len(lines) {
		end = len(lines)
	}
	body := strings.Join(lines[scroll:end], "\n")
	if len(lines[scroll:end]) < visH {
		body += strings.Repeat("\n", visH-len(lines[scroll:end]))
	}

	scrollHint := ""
	if len(lines) > visH {
		pct := 0
		if maxScroll > 0 {
			pct = scroll * 100 / maxScroll
		}
		scrollHint = styleNeutral.Render(fmt.Sprintf("  %d%%  ↑↓/pgup/pgdn to scroll", pct))
	}
	hints := styleHint.Render(" esc=back") + scrollHint

	header := styleTitle.Render(" GLOSSARY ") +
		styleNeutral.Render("  financial terms & concepts")

	return header + "\n" + div + "\n" + body + "\n" + div + "\n" + hints
}

func (m Model) handleGlossaryKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.screen = m.prevScreen
	case "up", "k":
		if m.glossaryScroll > 0 {
			m.glossaryScroll--
		}
	case "down", "j":
		m.glossaryScroll++
	case "pgup":
		m.glossaryScroll -= m.height / 2
		if m.glossaryScroll < 0 {
			m.glossaryScroll = 0
		}
	case "pgdown":
		m.glossaryScroll += m.height / 2
	case "home", "g":
		m.glossaryScroll = 0
	case "end", "G":
		m.glossaryScroll = 999999 // clamped to maxScroll in view
	}
	return m, nil
}
