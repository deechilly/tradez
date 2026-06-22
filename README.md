# tradez

[![Test](https://github.com/deechilly/tradez/actions/workflows/test.yml/badge.svg)](https://github.com/deechilly/tradez/actions/workflows/test.yml)

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

![tradez demo](recording.gif)

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

Requires the Go version declared in `go.mod`.

---

## Save Slots

tradez opens on a **slot selection screen** with three save slots labelled Game 1, Game 2, and Game 3. Each slot stores the complete game state: your portfolio, all open and filled orders, the current news cycle, active market influences, and the timer to the next news event.

| Action | Key |
|--------|-----|
| Navigate slots | `↑` / `↓` or `j` / `k` |
| Load save / start new game (empty slot) | `enter` |
| Start a new game in the selected slot | `n` |
| Jump directly to slot | `1`, `2`, or `3` |

Selecting a slot with an existing save loads it immediately. The slot displays your saved portfolio value, P&L, difficulty, and how long ago it was saved.

**Saving in-game:** press `ctrl+s` from the Market screen at any time to overwrite the current slot. The header bar briefly confirms the save.

**Returning to the menu:** press `esc` from the Market screen to go back to the slot selection screen without quitting. The slot metadata refreshes so you can see your last saved state before switching or starting over.

Save files are stored in `~/.tradez/saves/slot{1,2,3}.json`.

---

## Difficulty & Starting Capital

At launch you make two choices independently.

**Difficulty** controls what information is visible:

| Mode   | Insight level                                                                 |
|--------|-------------------------------------------------------------------------------|
| Easy   | Influence countdown timers shown in Market Feed; full analyst view available  |
| Normal | Timers hidden — judge influence strength and duration from the bar chart alone |

**Starting capital** is chosen separately and works the same in both modes:

| Capital    | Challenge                                                      |
|------------|----------------------------------------------------------------|
| $1,000,000 | Comfortable margin for error; diversify freely                 |
| $50,000    | Limited capital forces prioritisation                          |
| $1,000     | One bad trade can end your run; timing is everything           |

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

The **first event fires within 30 seconds** of starting a game, so you have something to react to immediately. After that, events fire every **1 to 5 minutes**. A world event is generated and posted to the Market Feed. Events are drawn from 54 templates across 12 categories:

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
- **Active Influences** section: one row per affected group, showing direction (▲/▼), targets, strength bar, and (in Easy mode) time remaining — both winners and losers visible at once
- Full news history with category badge, time elapsed, sentiment arrow, and affected targets

The news panel can be toggled with `n` if terminal width is limited. Influence indicators (▲/▼) appear next to each affected stock in the main market table, coloured to show the direction for *that specific stock*.

---

## Achievements

Press `a` from the Market screen to open the Achievements screen. Unlocks persist across sessions in `~/.tradez/achievements.json`.

| Icon | Name | How to unlock |
|------|------|---------------|
| 💎 | Diamond Hands | Close a long position at a profit after it was 35%+ underwater at some point |
| ⛈ | Weather the Storm | Hold a position through a negative news event targeting your sector for 60+ seconds without selling |
| 📉 | Sell the Dip | Sell a stock 15%+ below your cost basis, then watch it climb 25%+ above your sell price |
| 💀 | Bust!!! | Portfolio value falls below 5% of starting capital |
| 💣 | BOOM | Close a single long position for 3× your cost basis (200%+ return on one trade) |
| 🩳 | Squeezed | A short position racks up unrealized losses ≥75% of the margin you posted |
| 🎲 | YOLO | Commit 85%+ of your total portfolio value to a single buy order |
| 🧻 | Paper Hands | Sell a position at a loss, then watch the stock climb 20%+ above your sell price |
| 🚀 | To The Moon | A long position you hold is up 100%+ from your average cost |
| 🍗 | Tendies | Realize profit ≥5× starting capital from closing a single position |
| 😱 | GUH | A single sell or cover realizes a loss ≥30% of your total portfolio value |
| 🛍 | Bag Holder | Hold a position 40%+ in the red for 90 consecutive seconds |
| 📸 | Loss Porn | Portfolio hits 70%+ below starting capital — and you're still playing |
| 🌙 | WAGMI | Portfolio reaches 5× starting capital. We're all gonna make it |
| 🦍 | Ape Strong | Hold 8+ simultaneous long positions across different stocks |
| 🐒 | I Like the Stock | Hold a position untouched for 5+ minutes — long-term investing by WSB standards |
| ☠ | Widow Maker | A single buy order consumes 95%+ of your available cash |
| 📊 | Not Financial Advice | Make a trade that's the exact opposite of the Analyst's STRONG recommendation |
| 🚌 | Short Bus | Profit from 3 separate short positions in one session |
| 🖨 | Infinite Money Glitch | Profit on both a long AND a short position on the same ticker in one session |
| 🏠 | Free Real Estate | A limit buy fills 10%+ below your limit price |

---

## Trading

### Screens

| Screen       | Key    | Description                                        |
|--------------|--------|----------------------------------------------------|
| Market       | (main) | Live table of all stocks with sparklines           |
| Detail       | enter  | Price chart, company details, and analyst view     |
| Portfolio    | p      | Open positions and total P&L                       |
| Orders       | o      | Pending and filled order history                   |
| Trade        | b/s/…  | Order entry dialog                                 |
| Achievements | a      | All 21 achievements and unlock status              |

### Order Types

#### Market Buy — `b`
Buys shares immediately at the current price. Cost is deducted from cash instantly. If you already hold a short position on the same stock, the buy nets against it first — covering the short before adding any remainder as long shares.

#### Market Sell — `s`
Sells shares you own at the current price. Proceeds are added to cash instantly.

#### Limit Buy — `l`
Places a buy order that only fills when the price drops to or below your specified limit. The full limit cost is reserved from your cash when you place the order. If the order fills at a price below your limit, the difference is refunded. Cancel a pending limit order from the Orders screen to reclaim your reserved cash.

#### Limit Sell — `x`
Places a sell order that only fills when the price rises to or above your specified limit. Your shares are reserved when you place the order and returned if you cancel.

#### Short Sell — `h`
Bets that a stock will fall. You borrow and immediately sell shares you don't own. **50% margin** of the position value is reserved from your cash as collateral. If the price drops, you profit when you cover. If the price rises, your loss grows. If you already hold a long position on the same stock, the short nets against it first — selling the long before borrowing any remainder.

#### Cover Short — `c`
Closes a short position. You buy shares at the current price to return the borrowed stock. Your margin collateral is returned plus or minus the P&L.

#### Buy Put — `p` (from stock detail screen)
Buys European-style put option contracts on the selected stock. A put gives you the right — but not the obligation — to profit if the stock falls below the strike price before the option expires.

The put dialog lets you choose:
- **Strike price** — five levels from deep in-the-money (+20%) to deep out-of-the-money (−20%). The premium, break-even price, and moneyness label are shown for each.
- **Expiry** — Short (~1 min / 30 ticks), Medium (~3 min / 90 ticks), or Long (~5 min / 150 ticks).
- **Contracts** — each contract covers 100 shares. You pay the full premium upfront.

Premiums are calculated using Black-Scholes, with the stock's volatility scaled to an annualised implied volatility and game ticks treated as the time dimension. OTM puts on volatile stocks feel like lottery tickets; ITM puts provide more certain intrinsic value but cost substantially more.

Options are **European-style**: you cannot exercise early. To close a position before expiry, navigate to your put in the Portfolio screen and press `x` to sell at the current mark-to-market value. At expiry, ITM contracts auto-settle for intrinsic value; OTM contracts expire worthless.

### Margin Calls

When you hold short positions, the game tracks your **account equity** against your **short exposure** (the total current market value of all borrowed shares). This ratio drives a three-state health system:

| Ratio | State | Display |
|-------|-------|---------|
| ≥ 50% | Healthy | Green bar on Portfolio screen |
| 25–50% | Warning | Yellow bar + WARNING label |
| < 25% | **Margin Call** | Red bar + countdown timer |

When equity falls below 25% of short exposure, a margin call is issued. You have **15 ticks (~30 seconds)** to restore your margin — cover your shorts, sell long positions, or do both. The margin health bar at the top of the Portfolio screen shows a live countdown.

If the countdown expires without intervention, **all short positions are force-covered at the current market price**. Depending on how far the shorted stocks have moved against you, this can wipe a significant portion of your portfolio in one tick.

If a force-cover leaves your portfolio below **5% of your starting capital**, the game is over. You are presented with a YOU LOSE screen showing your final portfolio value and the total loss. From there you can either keep watching the market in spectator mode (no trading) or return to the title screen to start a new game.

The health bar is always visible on the Portfolio screen when you hold open shorts. An active margin call also replaces the normal status display in the top-right of the market table.

### Risk

- **Shorts can theoretically lose more than you invested** — there is no ceiling on how high a price can go
- **Limit buy orders reserve cash** — placing many large limit orders can leave you unable to trade spot
- **Margin calls are enforced** — a short that runs against you will trigger a forced cover at market price if you don't act within the grace period
- **Put premiums are non-refundable** — an OTM put that expires worthless is a total loss of the premium paid; time decay accelerates as expiry approaches

---

## Analyst View

Each stock detail screen has a third tab — **Analysis** — showing a data-driven buy/sell recommendation. Press `tab` twice from the chart to reach it.

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
| `a`       | Open achievements screen                            |
| `1`–`5`   | Sort by symbol / price / change% / volume / mkt cap |
| `?`       | Open glossary screen                                |
| `ctrl+s`  | Save game to current slot                           |
| `pgup`    | Scroll news feed up                                 |
| `pgdn`    | Scroll news feed down                               |
| `esc`     | Return to slot selection screen                     |
| `q`       | Quit                                                |

### Stock Detail

| Key     | Action                                                            |
|---------|-------------------------------------------------------------------|
| `tab`   | Cycle tabs: Chart → Details → Analysis → Chart                   |
| `b`     | Market buy                                                        |
| `s`     | Market sell                                                       |
| `l`     | Limit buy                                                         |
| `x`     | Limit sell                                                        |
| `h`     | Short sell                                                        |
| `c`     | Cover short                                                       |
| `p`     | Buy put option                                                    |
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

### Achievements Screen

| Key              | Action        |
|------------------|---------------|
| `↑` / `k` / `pgup` | Scroll up  |
| `↓` / `j` / `pgdn` | Scroll down |
| `esc`            | Back          |

### Glossary Screen

| Key   | Action                        |
|-------|-------------------------------|
| `↑↓`  | Navigate glossary terms       |
| `esc` | Back                          |
---

## Strategy Tips

- **Watch both sides of each influence.** An oil conflict that lifts Energy also drags Consumer. Short the loser while going long the winner from the same headline.
- **Rate decisions are the most powerful rotation signal.** A surprise rate cut simultaneously benefits Finance, Real Estate, and Tech while making Utilities less attractive. A rate hike reverses all of that.
- **M&A and EARNINGS events are sharp but company-specific.** An acquisition rumour can spike a stock instantly. These are the fastest in-and-out opportunities — and the fastest risks if you're on the wrong side.
- **High-beta stocks amplify everything.** Crypto and Tech overshoot both up and down relative to other sectors under the same news event.
- **Limit orders let you pre-position.** If a DISASTER event just hit Real Estate, set limit buys below current price and wait for the panic to bottom out before the influence fades.
- **Influence strength bars tell you how much runway is left.** A nearly expired influence with a stock still moving is a contrarian signal — the drift is about to disappear. On Easy mode the countdown timer makes this exact; on Normal mode learn to read the bar decay.
- **Use the Analysis tab before entering a position.** Check whether the technicals and fundamentals agree with the news signal before committing capital.
- **The sparkline in the market table is your best friend** for spotting micro-trends without opening each stock — especially useful when capital is tight.

---

## MCP Integration

tradez can expose a live running game to MCP-compatible AI tools.

Start the TUI first:

```bash
go run .
```

Load or start a save slot so there is an active live game. Then configure your MCP client to launch:

```bash
go run . mcp
```

The MCP server uses stdio, so the AI client owns the MCP process lifecycle. The running TUI exposes a private bridge on `127.0.0.1` and writes a token-protected descriptor to `~/.tradez/bridge/current.json`. If no live TUI is running, `tradez mcp` exits with a clear error.

By default, MCP tools can read game state and execute trades in the active live game. For read-only access:

```bash
go run . mcp --read-only
```

### MCP Client Examples

Run these from the tradez repository root unless the example shows an explicit path. If you build a binary first, replace `go run . mcp` with `/absolute/path/to/tradez/tradez mcp`.

#### Claude Code

For a project-scoped server that is shared through `.mcp.json`:

```bash
claude mcp add --transport stdio --scope project tradez -- go run . mcp
```

Read-only variant:

```bash
claude mcp add --transport stdio --scope project tradez -- go run . mcp --read-only
```

Equivalent `.mcp.json`:

```json
{
  "mcpServers": {
    "tradez": {
      "command": "go",
      "args": ["run", ".", "mcp"]
    }
  }
}
```

For read-only JSON configuration, add `"--read-only"` to the `args` array.

After starting Claude Code in this repo, run `/mcp` to confirm the `tradez` server connected. Project-scoped MCP servers may require approval the first time Claude Code sees the `.mcp.json` file.

#### Codex CLI

For a user-level Codex MCP entry:

```bash
codex mcp add tradez -- go run . mcp
```

Read-only variant:

```bash
codex mcp add tradez -- go run . mcp --read-only
```

For project-local configuration, create or edit `.codex/config.toml`:

```toml
[mcp_servers.tradez]
command = "go"
args = ["run", ".", "mcp"]
cwd = ".."
```

Use this read-only variant if you want Codex to inspect the market without placing or cancelling trades:

```toml
[mcp_servers.tradez]
command = "go"
args = ["run", ".", "mcp", "--read-only"]
cwd = ".."
```

Start Codex from the repo and run `/mcp` to confirm the server is active. Codex loads project-scoped `.codex/config.toml` only for trusted projects.

Once connected, try prompts like:

- `Use tradez to summarize the current market and my portfolio.`
- `Review recent news and explain which sectors are being pushed up or down.`
- `Propose a trade plan, but ask before placing any orders.`

### MCP Features

Resources:

- `tradez://game/summary`
- `tradez://market/stocks`
- `tradez://market/news`
- `tradez://market/influences`
- `tradez://portfolio`
- `tradez://orders`
- `tradez://stock/{symbol}`

Read tools:

- `get_game_state`
- `list_stocks`
- `get_stock`
- `get_news`
- `get_portfolio`
- `get_orders`

Mutating tools, omitted by `--read-only`:

- `place_order`
- `cancel_order`
- `save_game`

Prompts:

- `analyze_market`
- `explain_news_impact`
- `review_portfolio`
- `propose_trade_plan`

MCP-triggered trades are routed through the TUI event loop, so portfolio state, order history, status messages, and achievement checks update the same way as keyboard-driven trades.

--- 

## Architecture

```
tradez/
├── main.go                    Entry point
└── internal/
    ├── analysis/
    │   └── analysis.go        Shared analyst scoring for TUI and MCP
    ├── livebridge/
    │   ├── client.go          Token-authenticated bridge client for MCP mode
    │   ├── server.go          Localhost bridge owned by the running TUI
    │   └── types.go           Shared bridge operations and payloads
    ├── mcpserver/
    │   └── server.go          Stdio MCP server, tools, resources, prompts
    ├── snapshot/
    │   └── snapshot.go        JSON-safe game, market, portfolio DTOs
    ├── market/
    │   ├── stock.go           Stock model and price point history
    │   ├── generator.go       Procedural market and company generation
    │   ├── market.go          Price engine, news scheduling, tick loop
    │   ├── news.go            54 news templates, InfluenceEffect per-group
    │   │                      sentiments, MarketInfluence decay model
    │   ├── options.go         Black-Scholes put pricing, ImpliedVol helper
    │   └── save.go            MarketState snapshot for save/load
    ├── game/
    │   ├── game.go            Game state, portfolio, all order type logic
    │   ├── options.go         PutContract model, BuyPut/SellPut/ProcessPuts, strike helpers
    │   ├── save.go            Save/load logic, slot metadata, JSON serialisation
    │   └── achievements.go    21 achievement definitions, JSON persistence
    └── tui/
        ├── model.go           Bubbletea model, all views and key handlers,
        │                      analyst scoring (fundamentals/technicals/news)
        ├── achievements.go    Achievement tracking, flash notifications,
        │                      achievements screen
        └── styles.go          Lipgloss colour palette and style definitions
```

**Dependencies:** [Bubbletea](https://github.com/charmbracelet/bubbletea) · [Lipgloss](https://github.com/charmbracelet/lipgloss) · [Bubbles](https://github.com/charmbracelet/bubbles) · [asciigraph](https://github.com/guptarohit/asciigraph)

---

## Recording

`record.sh` builds the binary, plays `tour.tape` through [VHS](https://github.com/charmbracelet/vhs), and writes the result to `recording.gif`. VHS requires **ffmpeg** and **ttyd** in addition to the `vhs` binary itself.

### macOS

```bash
brew install vhs ttyd ffmpeg
./record.sh
```

### Linux

VHS publishes pre-built binaries, distro packages, and a Go install path. Pick whichever fits your setup:

**Debian / Ubuntu**
```bash
# via Charm's apt repository
sudo mkdir -p /etc/apt/keyrings
curl -fsSL https://repo.charm.sh/apt/gpg.key | sudo gpg --dearmor -o /etc/apt/keyrings/charm.gpg
echo "deb [signed-by=/etc/apt/keyrings/charm.gpg] https://repo.charm.sh/apt/ * *" \
  | sudo tee /etc/apt/sources.list.d/charm.list
sudo apt update && sudo apt install vhs ttyd ffmpeg
```

**Fedora / RHEL / openSUSE**
```bash
# via Charm's yum repository
echo '[charm]
name=Charm
baseurl=https://repo.charm.sh/yum/
enabled=1
gpgcheck=1
gpgkey=https://repo.charm.sh/yum/gpg.key' | sudo tee /etc/yum.repos.d/charm.repo
sudo yum install vhs ttyd ffmpeg
```

**Arch Linux**
```bash
# vhs and ttyd are in the AUR
yay -S vhs ttyd
sudo pacman -S ffmpeg
```

**Any distro — via Go**
```bash
go install github.com/charmbracelet/vhs@latest
# grab a ttyd binary from https://github.com/tsl0922/ttyd/releases
# and place it somewhere on your PATH
sudo apt install ffmpeg   # or equivalent for your distro
```

**Any distro — pre-built binaries**

Download `vhs` from [github.com/charmbracelet/vhs/releases](https://github.com/charmbracelet/vhs/releases) and `ttyd` from [github.com/tsl0922/ttyd/releases](https://github.com/tsl0922/ttyd/releases). Place both somewhere on your `$PATH`.

Once dependencies are in place:

```bash
./record.sh
```
