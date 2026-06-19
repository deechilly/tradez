# tradez

A terminal-based stock trading simulator built in Go. Watch live markets tick, react to breaking news, build a portfolio, and try to turn seed money into a fortune — all from your terminal.

```
  ████████╗██████╗  █████╗ ██████╗ ███████╗███████╗
     ██╔══╝██╔══██╗██╔══██╗██╔══██╗██╔════╝╚══███╔╝
     ██║   ██████╔╝███████║██║  ██║█████╗    ███╔╝
     ██║   ██╔══██╗██╔══██║██║  ██║██╔══╝   ███╔╝
     ██║   ██║  ██║██║  ██║██████╔╝███████╗███████╗
     ╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝╚═════╝ ╚══════╝╚══════╝
```

---

## Overview

tradez simulates a live stock market with ~50 procedurally generated companies across 10 industries. Prices update every 2 seconds. Random world events fire every 1–5 minutes as news items that create real, lasting effects on the stocks they describe. Your job is to read the market, interpret the news, and trade your way to profit.

There is no win condition. Your score is your portfolio value vs. your starting capital.

---

## Getting Started

```bash
git clone https://github.com/deechilly/tradez
cd tradez
go run .
```

Or build a binary:

```bash
go build -o tradez .
./tradez
```

Requires Go 1.21+.

---

## Difficulty

At launch, choose a starting capital level. Difficulty affects nothing except how much money you begin with — markets are equally volatile on all modes.

| Mode   | Starting Capital | Challenge                                              |
|--------|-----------------|--------------------------------------------------------|
| Easy   | $1,000,000      | Comfortable margin for error; diversify freely         |
| Medium | $50,000         | Limited capital forces prioritisation                  |
| Hard   | $1,000          | One bad trade can end your run; timing is everything   |

---

## The Market

### Companies & Industries

Each game session generates a fresh market. Companies are assigned to one of 10 industries, each with distinct volatility characteristics:

| Industry    | Volatility | Notes                                           |
|-------------|------------|-------------------------------------------------|
| Technology  | High       | Growth-driven; reacts strongly to tech news     |
| Finance     | Medium     | Sensitive to rate decisions and economic data   |
| Energy      | Medium-High| Moves with supply/demand news, OPEC events      |
| Healthcare  | Medium     | Driven by trial results and FDA approvals       |
| Consumer    | Low-Medium | Tracks economic sentiment and retail data       |
| Industrial  | Medium     | Responds to infrastructure and trade news       |
| Crypto      | Very High  | Extreme volatility; regulation-sensitive        |
| Real Estate | Low        | Rate-sensitive; slow-moving                     |
| Materials   | Medium     | Tied to commodity prices and disaster events    |
| Utilities   | Very Low   | Defensive; barely moves without major catalysts |

Each company has a full metadata profile: symbol, name, industry, market cap, P/E ratio, EPS, dividend yield, beta, and YTD growth.

### Price Engine

Prices update every **2 seconds** using a random walk model. Each tick:

1. A noise component is applied — random directional movement scaled by the stock's **volatility** and current price
2. A **trend drift** nudges the price slightly in a persistent direction (set at generation)
3. Active **market influences** from news events add a directional bias that decays over time
4. Rare spontaneous news spikes (~0.2% chance per tick) add sudden sharp moves

Beta determines how much a stock amplifies broader market swings. High-beta stocks (crypto, tech) swing harder; low-beta stocks (utilities, real estate) barely budge.

---

## News Events

Every **1 to 5 minutes**, a world event is generated and posted to the Market Feed. Events are drawn from 55 templates across 12 categories:

| Category   | Badge colour | Typical effect                                          |
|------------|-------------|----------------------------------------------------------|
| CONFLICT   | Red         | Energy/Materials up; broad market uncertainty            |
| DISASTER   | Orange      | Hits Insurance, Consumer, Real Estate                    |
| TECH       | Blue        | Technology surges or falls; Finance affected by crypto   |
| M&A        | Purple      | Individual stock spikes 30–80% on acquisition rumour     |
| REGULATION | Yellow      | Rate decisions move everything; targeted rules hit sectors|
| ECONOMY    | Teal        | GDP/inflation prints cause broad sector rotations        |
| SPACE      | Light blue  | Technology and Industrial benefit                        |
| HEALTH     | Green       | Healthcare swings; pandemic events hit Consumer/Real Est.|
| ENERGY     | Red-orange  | Energy and Utilities rotate on supply/production news    |
| CRYPTO     | Gold        | Crypto sector; large moves on regulation or exchange news|
| CONSUMER   | Pink        | Consumer and Finance track retail/debt sentiment         |
| EARNINGS   | Mint        | Company-specific; large moves on beats, misses, buybacks |

Some events target an entire **industry**. Others name a **specific company** from the live market — its real symbol and name appear in the headline.

### Market Influence

When a news event fires:

1. An **immediate price spike** hits affected stocks (scaled to the event's magnitude and sentiment)
2. A **MarketInfluence** is created that persists for a random duration (2–10 minutes depending on the event type)
3. Each subsequent price tick, active influences add a directional drift bias to affected stocks — positive sentiment pushes prices up, negative pushes them down
4. The influence **decays linearly** over its lifetime, fading to zero at expiry

The result: a news event causes an immediate reaction, followed by a sustained trend that gradually normalises. A conflict disrupting oil supply will push energy stocks up for several minutes. A failed drug trial will punish a healthcare stock for a few minutes before the market stabilises.

### Reading the Feed

The right-side **Market Feed** panel shows:

- **BREAKING** banner with the headline when a new event fires
- **Active Influences** section: which industries/stocks are under pressure, current strength (█ bar), and time remaining
- Full news history with category badge, time elapsed, sentiment arrow (▲▼▲▲), and affected targets

Influence indicators (`↑` / `↓`) appear in the main market table next to each stock currently under active influence.

---

## Trading

### Screens

| Screen    | Key    | Description                                 |
|-----------|--------|---------------------------------------------|
| Market    | (main) | Live table of all stocks                    |
| Detail    | enter  | Price chart + company details for one stock |
| Portfolio | p      | Your open positions and P&L                 |
| Orders    | o      | Pending and filled order history            |
| Trade     | b/s/…  | Order entry dialog                          |

### Order Types

#### Market Buy — `b`
Buys shares immediately at the current price. Cost is deducted from cash instantly.

#### Market Sell — `s`
Sells shares you own at the current price. Proceeds are added to cash instantly.

#### Limit Buy — `l`
Places a buy order that only fills when the price drops to or below your specified limit. The full limit cost is reserved from your cash when you place the order. If the order fills at a price below your limit, the difference is refunded. Cancel a pending limit order from the Orders screen to reclaim your reserved cash.

#### Limit Sell — `x`
Places a sell order that only fills when the price rises to or above your specified limit. Your shares are reserved when you place the order and returned if you cancel.

#### Short Sell — `h`
Bets that a stock will fall. You borrow and immediately sell shares you don't own. **50% margin** of the position value is reserved from your cash as collateral. If the price drops, you profit when you cover. If the price rises, your loss grows.

#### Cover Short — `c`
Closes a short position. You buy shares at the current price to return the borrowed stock. Your margin collateral is returned plus or minus the P&L.

### Risk

- **Shorts can theoretically lose more than you invested** — there is no ceiling on how high a price can go
- **Limit buy orders reserve cash** — placing many large limit orders can leave you unable to trade spot
- **Margin calls are not enforced** in this version; a short gone badly wrong can push your cash negative

---

## Controls

### Market Table

| Key       | Action                        |
|-----------|-------------------------------|
| `↑` / `k` | Move cursor up                |
| `↓` / `j` | Move cursor down              |
| `enter`   | Open stock detail             |
| `b`       | Market buy selected stock     |
| `s`       | Market sell selected stock    |
| `p`       | Open portfolio                |
| `o`       | Open orders                   |
| `1`–`5`   | Sort by symbol/price/change%/volume/market cap |
| `pgup`    | Scroll news feed up           |
| `pgdn`    | Scroll news feed down         |
| `q`       | Quit                          |

### Stock Detail

| Key     | Action                        |
|---------|-------------------------------|
| `tab`   | Toggle chart / details        |
| `b`     | Market buy                    |
| `s`     | Market sell                   |
| `l`     | Limit buy                     |
| `x`     | Limit sell                    |
| `h`     | Short sell                    |
| `c`     | Cover short                   |
| `esc`   | Back to market table          |

### Orders Screen

| Key     | Action                        |
|---------|-------------------------------|
| `↑↓`   | Navigate pending orders       |
| `x`     | Cancel selected pending order |
| `esc`   | Back                          |

### Trade Dialog

| Key     | Action                           |
|---------|----------------------------------|
| `tab`   | Switch between shares/price field (limit orders only) |
| `enter` | Confirm order                    |
| `esc`   | Cancel                           |

---

## Strategy Tips

- **Watch the Market Feed, not just the prices.** A CONFLICT event hitting energy will push oil stocks for minutes — enough time to enter and exit a position.
- **Industries move together** under influence. If a rate cut is announced, Finance and Real Estate are both likely to benefit — you don't need to pick the single best stock.
- **M&A and EARNINGS events are company-specific and sharp.** An acquisition rumour can spike a stock 30–80% instantly. These events are the fastest opportunities and fastest risks.
- **High-beta stocks amplify everything.** Crypto and Tech will overshoot both up and down relative to other sectors under the same news event.
- **Limit orders let you pre-position.** If a DISASTER event just hit Real Estate, set limit buys below current price and wait for the panic to bottom out before the influence fades.
- **Influence decays.** The strength bars in the Active Influences panel tell you how long a trend has left. A nearly expired influence with a stock still moving is a contrarian signal.
- **On Hard mode**, the sparkline in the market table is your best friend — spot micro-trends without opening each stock.

---

## Architecture

```
tradez/
├── main.go                    Entry point
└── internal/
    ├── market/
    │   ├── stock.go           Stock model and price point history
    │   ├── generator.go       Procedural market and company generation
    │   ├── market.go          Price engine, news scheduling, tick loop
    │   └── news.go            News templates, events, and MarketInfluence
    ├── game/
    │   └── game.go            Game state, portfolio, all order type logic
    └── tui/
        ├── model.go           Bubbletea model, all views and key handlers
        └── styles.go          Lipgloss colour palette and style definitions
```

**Dependencies:** [Bubbletea](https://github.com/charmbracelet/bubbletea) · [Lipgloss](https://github.com/charmbracelet/lipgloss) · [Bubbles](https://github.com/charmbracelet/bubbles) · [asciigraph](https://github.com/guptarohit/asciigraph)
