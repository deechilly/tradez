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

tradez simulates a live stock market with ~50 procedurally generated companies across 10 industries. Prices update every 2 seconds. Random world events fire every 1–5 minutes, creating real lasting effects that push different industries in opposite directions. Your job is to read the market, interpret the news, and trade your way to profit.

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

At launch, choose a starting capital level. Difficulty also controls access to the analyst assist feature.

| Mode   | Starting Capital | Extras                                                   |
|--------|-----------------|----------------------------------------------------------|
| Easy   | $1,000,000      | Analyst View tab on every stock detail screen            |
| Medium | $50,000         | Standard — limited capital forces prioritisation         |
| Hard   | $1,000          | One bad trade can end your run; timing is everything     |

---

## The Market

### Companies & Industries

Each game session generates a fresh market. Companies are assigned to one of 10 industries, each with distinct volatility characteristics:

| Industry    | Volatility  | Notes                                            |
|-------------|-------------|--------------------------------------------------|
| Technology  | High        | Growth-driven; reacts strongly to tech news      |
| Finance     | Medium      | Sensitive to rate decisions and economic data    |
| Energy      | Medium-High | Moves with supply/demand news, OPEC events       |
| Healthcare  | Medium      | Driven by trial results and FDA approvals        |
| Consumer    | Low-Medium  | Tracks economic sentiment and retail data        |
| Industrial  | Medium      | Responds to infrastructure and trade news        |
| Crypto      | Very High   | Extreme volatility; regulation-sensitive         |
| Real Estate | Low         | Rate-sensitive; slow-moving                      |
| Materials   | Medium      | Tied to commodity prices and disaster events     |
| Utilities   | Very Low    | Defensive; barely moves without major catalysts  |

Each company has a full metadata profile: symbol, name, industry, market cap, P/E ratio, EPS, dividend yield, beta, and YTD growth.

### Price Engine

Prices update every **2 seconds** using a random walk model. Each tick:

1. A noise component is applied — random directional movement scaled by the stock's **volatility** and current price
2. A **trend drift** nudges the price slightly in a persistent direction (set at generation)
3. Active **market influences** from news events add a per-stock directional bias that decays over time
4. Rare spontaneous spikes add sudden sharp moves

Beta determines how much a stock amplifies broader swings. High-beta stocks (crypto, tech) swing harder; low-beta stocks (utilities, real estate) barely budge.

---

## News Events

Every **1 to 5 minutes**, a world event is generated and posted to the Market Feed. Events are drawn from 54 templates across 12 categories:

| Category   | Badge colour | Typical effect                                           |
|------------|-------------|-----------------------------------------------------------|
| CONFLICT   | Red         | Energy/Materials up; Consumer/Tech under pressure         |
| DISASTER   | Orange      | Finance/Real hurt; Materials/Industrial benefit (rebuild) |
| TECH       | Blue        | Technology surges or falls; rotation into/out of Energy   |
| M&A        | Purple      | Individual stock spikes on acquisition rumour             |
| REGULATION | Yellow      | Rate cuts lift Finance/Real/Tech; hikes benefit Utilities |
| ECONOMY    | Teal        | GDP/inflation prints rotate capital between sectors       |
| SPACE      | Light blue  | Technology and Industrial benefit                         |
| HEALTH     | Green       | Healthcare swings; pandemic events drag Consumer/Real     |
| ENERGY     | Red-orange  | Energy and Utilities rotate on supply/production news     |
| CRYPTO     | Gold        | Crypto sector; large moves on regulation or exchange news |
| CONSUMER   | Pink        | Consumer and Finance track retail/debt sentiment          |
| EARNINGS   | Mint        | Company-specific; large moves on beats, misses, buybacks  |

Some events target an entire **industry**. Others name a **specific company** from the live market — its real symbol and name appear in the headline.

### Winner/Loser Dynamics

Each news event carries **per-group effects** — different industries can move in opposite directions from the same headline. This models real capital rotation:

| Event                        | Winners                  | Losers                    |
|------------------------------|--------------------------|---------------------------|
| Rate hike                    | Finance +0.35, Utils +0.2 | Tech −0.6, Crypto −0.6, Real −0.6 |
| Oil conflict                 | Energy +0.5, Materials +0.5 | Consumer −0.25, Tech −0.25 |
| Pandemic alert               | Healthcare +0.7          | Consumer −0.5, Real −0.5  |
| Fusion energy breakthrough   | Tech +0.8, Utilities +0.4 | Energy −0.5              |
| Infrastructure stimulus      | Industrial +0.6, Materials +0.6 | —                  |
| Crypto exchange collapse     | —                        | Crypto −0.8, Finance −0.4 |

The Market Feed's **Active Influences** panel shows one line per group, so you can see both sides simultaneously.

### Market Influence

When a news event fires:

1. An **immediate price spike** hits each affected group at its own sentiment magnitude
2. A **MarketInfluence** is created that persists for a random duration (2–10 minutes)
3. Each price tick, the influence adds a per-stock directional drift — a Consumer stock and an Energy stock can receive opposite pushes from the same oil-conflict influence
4. The influence **decays linearly** over its lifetime, fading to zero at expiry

### Reading the Market Feed

The right-side **Market Feed** panel shows:

- **BREAKING** banner with the full headline when a new event fires
- **Active Influences** section: one row per affected group, showing direction (▲/▼), targets, strength bar, and time remaining — both winners and losers visible at once
- Full news history with category badge, time elapsed, sentiment arrow, and affected targets

The news panel can be toggled with `n` if terminal width is limited. Influence indicators (▲/▼) appear next to each affected stock in the main market table, coloured to show the direction for *that specific stock*.

---

## Trading

### Screens

| Screen    | Key    | Description                                        |
|-----------|--------|----------------------------------------------------|
| Market    | (main) | Live table of all stocks with sparklines           |
| Detail    | enter  | Price chart, company details, and analyst view     |
| Portfolio | p      | Open positions and total P&L                       |
| Orders    | o      | Pending and filled order history                   |
| Trade     | b/s/…  | Order entry dialog                                 |

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

## Analyst View (Easy Mode)

On Easy difficulty, each stock detail screen has a third tab — **Analysis** — showing a data-driven buy/sell recommendation. Press `tab` twice from the chart to reach it.

The recommendation is built from three scored components:

### Fundamentals (30%)
- **P/E ratio** vs the industry's historical average — cheap relative to sector peers scores bullish; expensive scores bearish
- **EPS** — profitability signal; loss-making companies carry a penalty
- **YTD growth** as a contrarian indicator — deeply oversold stocks score higher; overextended runners score lower
- **Dividend yield** — a positive carry signal

### Technicals (35%)
- **20-period moving average** — is the current price above or below its recent average?
- **50-period moving average** — longer trend context
- **Momentum slope** — the directional rate of change over the last 15 price ticks, normalised to ±1

### Market Signals (35%)
- Net active influence strength targeting this stock right now
- Count of active bullish vs bearish news events affecting this stock or its industry

### Output

Each component shows an 8-block strength bar and a signed score. The weighted composite score maps to:

| Score      | Recommendation   |
|------------|-----------------|
| > +0.45    | ▲▲ STRONG BUY   |
| +0.15–0.45 | ▲  BUY          |
| ±0.15      | ◆  HOLD         |
| −0.15–−0.45| ▼  SELL         |
| < −0.45    | ▼▼ STRONG SELL  |

A confidence percentage and suggested entry price, stop-loss, and price target are also displayed. These are model outputs intended to assist decision-making — they are not guaranteed.

---

## Controls

### Market Table

| Key       | Action                                              |
|-----------|-----------------------------------------------------|
| `↑` / `k` | Move cursor up                                      |
| `↓` / `j` | Move cursor down                                    |
| `enter`   | Open stock detail                                   |
| `b`       | Market buy selected stock                           |
| `s`       | Market sell selected stock                          |
| `p`       | Open portfolio                                      |
| `o`       | Open orders                                         |
| `n`       | Toggle Market Feed panel                            |
| `1`–`5`   | Sort by symbol / price / change% / volume / mkt cap |
| `pgup`    | Scroll news feed up                                 |
| `pgdn`    | Scroll news feed down                               |
| `q`       | Quit                                                |

### Stock Detail

| Key     | Action                                                            |
|---------|-------------------------------------------------------------------|
| `tab`   | Cycle tabs: Chart → Details → Analysis (Easy) → Chart            |
| `b`     | Market buy                                                        |
| `s`     | Market sell                                                       |
| `l`     | Limit buy                                                         |
| `x`     | Limit sell                                                        |
| `h`     | Short sell                                                        |
| `c`     | Cover short                                                       |
| `esc`   | Back to market table                                              |

### Orders Screen

| Key   | Action                        |
|-------|-------------------------------|
| `↑↓`  | Navigate pending orders       |
| `x`   | Cancel selected pending order |
| `esc` | Back                          |

### Trade Dialog

| Key     | Action                                             |
|---------|----------------------------------------------------|
| `tab`   | Switch between shares / price field (limit orders) |
| `enter` | Confirm order                                      |
| `esc`   | Cancel                                             |

---

## Strategy Tips

- **Watch both sides of each influence.** An oil conflict that lifts Energy also drags Consumer. Short the loser while going long the winner from the same headline.
- **Rate decisions are the most powerful rotation signal.** A surprise rate cut simultaneously benefits Finance, Real Estate, and Tech while making Utilities less attractive. A rate hike reverses all of that.
- **M&A and EARNINGS events are sharp but company-specific.** An acquisition rumour can spike a stock instantly. These are the fastest in-and-out opportunities — and the fastest risks if you're on the wrong side.
- **High-beta stocks amplify everything.** Crypto and Tech overshoot both up and down relative to other sectors under the same news event.
- **Limit orders let you pre-position.** If a DISASTER event just hit Real Estate, set limit buys below current price and wait for the panic to bottom out before the influence fades.
- **Influence strength bars tell you how much runway is left.** A nearly expired influence with a stock still moving is a contrarian signal — the drift is about to disappear.
- **On Easy mode, use the Analysis tab before entering a position.** Check whether the technicals and fundamentals agree with the news signal before committing capital.
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
    │   └── news.go            54 news templates, InfluenceEffect per-group
    │                          sentiments, MarketInfluence decay model
    ├── game/
    │   └── game.go            Game state, portfolio, all order type logic
    └── tui/
        ├── model.go           Bubbletea model, all views and key handlers,
        │                      analyst scoring (fundamentals/technicals/news)
        └── styles.go          Lipgloss colour palette and style definitions
```

**Dependencies:** [Bubbletea](https://github.com/charmbracelet/bubbletea) · [Lipgloss](https://github.com/charmbracelet/lipgloss) · [Bubbles](https://github.com/charmbracelet/bubbles) · [asciigraph](https://github.com/guptarohit/asciigraph)
