package tui

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
	"tradez/internal/game"
	"tradez/internal/market"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/guptarohit/asciigraph"
)

type screen int

const (
	screenMenu screen = iota
	screenMarket
	screenStock
	screenPortfolio
	screenOrders
	screenTrade
)


type tradeMode int

const (
	tradeBuy tradeMode = iota
	tradeSell
	tradeLimitBuy
	tradeLimitSell
	tradeShort
	tradeCover
)

func (t tradeMode) label() string {
	switch t {
	case tradeBuy:
		return "Market BUY"
	case tradeSell:
		return "Market SELL"
	case tradeLimitBuy:
		return "Limit BUY"
	case tradeLimitSell:
		return "Limit SELL"
	case tradeShort:
		return "SHORT Sell"
	case tradeCover:
		return "Cover SHORT"
	}
	return ""
}

const (
	newsPanelWidth    = 44
	newsPanelMinWidth = 156 // terminal must be at least this wide to show the news panel
	// full column set totals ~107 chars; 107 + 44 panel + 5 gutter = 156
)

type Model struct {
	g            *game.Game
	screen       screen
	width        int
	height       int
	cursor       int
	stockSymbol  string
	sortCol      int
	sortAsc      bool
	tab          int
	tradeMode    tradeMode
	inputShares  textinput.Model
	inputPrice   textinput.Model
	inputFocus   int
	errMsg       string
	okMsg        string
	msgTimer     int
	stocks       []*market.Stock
	flashNews    *market.NewsEvent // newly arrived event, flashes briefly
	flashTicks   int
	newsScroll   int
}

func NewModel(g *game.Game) Model {
	ti := textinput.New()
	ti.Placeholder = "100"
	ti.CharLimit = 10
	ti.Width = 12

	tp := textinput.New()
	tp.Placeholder = "0.00"
	tp.CharLimit = 12
	tp.Width = 12

	m := Model{
		g:           g,
		screen:      screenMenu,
		sortCol:     0,
		sortAsc:     true,
		inputShares: ti,
		inputPrice:  tp,
	}
	m.refreshStocks()
	return m
}

func (m *Model) refreshStocks() {
	m.stocks = m.g.Market.Snapshot()
	m.sortStocks()
}

func (m *Model) sortStocks() {
	stocks := m.stocks
	asc := m.sortAsc
	switch m.sortCol {
	case 0:
		sort.Slice(stocks, func(i, j int) bool {
			if asc {
				return stocks[i].Symbol < stocks[j].Symbol
			}
			return stocks[i].Symbol > stocks[j].Symbol
		})
	case 1:
		sort.Slice(stocks, func(i, j int) bool {
			if asc {
				return stocks[i].Price < stocks[j].Price
			}
			return stocks[i].Price > stocks[j].Price
		})
	case 2:
		sort.Slice(stocks, func(i, j int) bool {
			if asc {
				return stocks[i].ChangePct() < stocks[j].ChangePct()
			}
			return stocks[i].ChangePct() > stocks[j].ChangePct()
		})
	case 3:
		sort.Slice(stocks, func(i, j int) bool {
			if asc {
				return stocks[i].Volume < stocks[j].Volume
			}
			return stocks[i].Volume > stocks[j].Volume
		})
	case 4:
		sort.Slice(stocks, func(i, j int) bool {
			if asc {
				return stocks[i].MarketCap < stocks[j].MarketCap
			}
			return stocks[i].MarketCap > stocks[j].MarketCap
		})
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return t
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case time.Time:
		// drive the market tick and schedule next one
		newEvent := m.g.Market.Tick()
		m.g.ProcessLimitOrders()
		m.refreshStocks()

		if newEvent != nil {
			m.flashNews = newEvent
			m.flashTicks = 6 // ~12 seconds of flash
		}
		if m.flashTicks > 0 {
			m.flashTicks--
			if m.flashTicks == 0 {
				m.flashNews = nil
			}
		}

		if m.msgTimer > 0 {
			m.msgTimer--
			if m.msgTimer == 0 {
				m.errMsg = ""
				m.okMsg = ""
			}
		}
		return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
			return t
		})

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenMenu:
		return m.handleMenuKey(msg)
	case screenMarket:
		return m.handleMarketKey(msg)
	case screenStock:
		return m.handleStockKey(msg)
	case screenPortfolio:
		return m.handlePortfolioKey(msg)
	case screenOrders:
		return m.handleOrdersKey(msg)
	case screenTrade:
		return m.handleTradeKey(msg)
	}
	return m, nil
}

func (m Model) handleMenuKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "1":
		m.g.Cash = game.DifficultyEasy.StartingCash()
		m.g.Difficulty = game.DifficultyEasy
		m.screen = screenMarket
	case "2":
		m.g.Cash = game.DifficultyMedium.StartingCash()
		m.g.Difficulty = game.DifficultyMedium
		m.screen = screenMarket
	case "3":
		m.g.Cash = game.DifficultyHard.StartingCash()
		m.g.Difficulty = game.DifficultyHard
		m.screen = screenMarket
	case "q", "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) handleMarketKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.stocks)-1 {
			m.cursor++
		}
	case "enter":
		if m.cursor < len(m.stocks) {
			m.stockSymbol = m.stocks[m.cursor].Symbol
			m.screen = screenStock
		}
	case "p":
		m.cursor = 0
		m.screen = screenPortfolio
	case "o":
		m.cursor = 0
		m.screen = screenOrders
	case "b":
		if m.cursor < len(m.stocks) {
			m.stockSymbol = m.stocks[m.cursor].Symbol
			m.openTrade(tradeBuy)
		}
	case "s":
		if m.cursor < len(m.stocks) {
			m.stockSymbol = m.stocks[m.cursor].Symbol
			m.openTrade(tradeSell)
		}
	case "1", "2", "3", "4", "5":
		col, _ := strconv.Atoi(msg.String())
		col--
		if col == m.sortCol {
			m.sortAsc = !m.sortAsc
		} else {
			m.sortCol = col
			m.sortAsc = true
		}
		m.sortStocks()
	case "pgup":
		m.newsScroll -= 3
		if m.newsScroll < 0 {
			m.newsScroll = 0
		}
	case "pgdown":
		m.newsScroll += 3
	}
	return m, nil
}

func (m Model) handleStockKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.screen = screenMarket
	case "b":
		m.openTrade(tradeBuy)
	case "s":
		m.openTrade(tradeSell)
	case "l":
		m.openTrade(tradeLimitBuy)
	case "x":
		m.openTrade(tradeLimitSell)
	case "h":
		m.openTrade(tradeShort)
	case "c":
		m.openTrade(tradeCover)
	case "tab":
		m.tab = (m.tab + 1) % 2
	}
	return m, nil
}

func (m *Model) openTrade(mode tradeMode) {
	m.tradeMode = mode
	m.inputShares.Reset()
	m.inputPrice.Reset()
	m.inputShares.Focus()
	m.inputFocus = 0
	m.errMsg = ""
	m.okMsg = ""
	m.screen = screenTrade
}

func (m Model) handlePortfolioKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.screen = screenMarket
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.g.Positions)-1 {
			m.cursor++
		}
	case "enter":
		syms := m.portfolioSymbols()
		if m.cursor < len(syms) {
			m.stockSymbol = syms[m.cursor]
			m.screen = screenStock
		}
	}
	return m, nil
}

func (m Model) portfolioSymbols() []string {
	syms := make([]string, 0, len(m.g.Positions))
	for k := range m.g.Positions {
		syms = append(syms, k)
	}
	sort.Strings(syms)
	return syms
}

func (m Model) handleOrdersKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.screen = screenMarket
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		pending := m.g.PendingOrders()
		if m.cursor < len(pending)-1 {
			m.cursor++
		}
	case "x":
		pending := m.g.PendingOrders()
		if m.cursor < len(pending) {
			m.g.CancelOrder(pending[m.cursor].ID)
		}
	}
	return m, nil
}

func (m Model) handleTradeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	isLimit := m.tradeMode == tradeLimitBuy || m.tradeMode == tradeLimitSell

	switch msg.String() {
	case "esc":
		m.screen = screenMarket
		return m, nil
	case "tab":
		if isLimit {
			if m.inputFocus == 0 {
				m.inputShares.Blur()
				m.inputPrice.Focus()
				m.inputFocus = 1
			} else {
				m.inputPrice.Blur()
				m.inputShares.Focus()
				m.inputFocus = 0
			}
		}
		return m, nil
	case "enter":
		sharesStr := strings.TrimSpace(m.inputShares.Value())
		shares, err := strconv.Atoi(sharesStr)
		if err != nil || shares <= 0 {
			m.errMsg = "Invalid share count"
			m.msgTimer = 5
			return m, nil
		}
		var tradeErr error
		if isLimit {
			priceStr := strings.TrimSpace(m.inputPrice.Value())
			limit, err2 := strconv.ParseFloat(priceStr, 64)
			if err2 != nil || limit <= 0 {
				m.errMsg = "Invalid limit price"
				m.msgTimer = 5
				return m, nil
			}
			if m.tradeMode == tradeLimitBuy {
				tradeErr = m.g.PlaceLimitBuy(m.stockSymbol, shares, limit)
			} else {
				tradeErr = m.g.PlaceLimitSell(m.stockSymbol, shares, limit)
			}
		} else {
			switch m.tradeMode {
			case tradeBuy:
				tradeErr = m.g.BuyMarket(m.stockSymbol, shares)
			case tradeSell:
				tradeErr = m.g.SellMarket(m.stockSymbol, shares)
			case tradeShort:
				tradeErr = m.g.ShortSell(m.stockSymbol, shares)
			case tradeCover:
				tradeErr = m.g.CoverShort(m.stockSymbol, shares)
			}
		}
		if tradeErr != nil {
			m.errMsg = tradeErr.Error()
			m.msgTimer = 5
		} else {
			m.okMsg = "Order placed!"
			m.msgTimer = 3
			m.screen = screenMarket
		}
		return m, nil
	}

	var cmd tea.Cmd
	if m.inputFocus == 0 {
		m.inputShares, cmd = m.inputShares.Update(msg)
	} else {
		m.inputPrice, cmd = m.inputPrice.Update(msg)
	}
	return m, cmd
}

// ── Views ──────────────────────────────────────────────────────────────────

func (m Model) View() string {
	switch m.screen {
	case screenMenu:
		return m.viewMenu()
	case screenMarket:
		return m.viewMarket()
	case screenStock:
		return m.viewStock()
	case screenPortfolio:
		return m.viewPortfolio()
	case screenOrders:
		return m.viewOrders()
	case screenTrade:
		return m.viewTrade()
	}
	return ""
}

func (m Model) viewMenu() string {
	w := m.width
	if w == 0 {
		w = 80
	}
	logo := styleTitle.Render(`
  ████████╗██████╗  █████╗ ██████╗ ███████╗███████╗
     ██╔══╝██╔══██╗██╔══██╗██╔══██╗██╔════╝╚══███╔╝
     ██║   ██████╔╝███████║██║  ██║█████╗    ███╔╝
     ██║   ██╔══██╗██╔══██║██║  ██║██╔══╝   ███╔╝
     ██║   ██║  ██║██║  ██║██████╔╝███████╗███████╗
     ╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝╚═════╝ ╚══════╝╚══════╝`)

	sub := styleNeutral.Render("  A terminal trading simulator")

	opts := []string{
		stylePositive.Render("  [1] ") + styleWhiteStr("Easy   — $1,000,000 starting capital"),
		styleYellowStr("  [2] ") + styleWhiteStr("Medium — $50,000 starting capital"),
		styleNegative.Render("  [3] ") + styleWhiteStr("Hard   — $1,000 starting capital"),
		"",
		styleNeutral.Render("  [q] Quit"),
	}

	content := logo + "\n\n" + sub + "\n\n" +
		styleHeader.Render("  Select Difficulty") + "\n\n" +
		strings.Join(opts, "\n")

	return lipgloss.Place(w, m.height, lipgloss.Center, lipgloss.Center,
		styleBorder.Padding(2, 4).Render(content))
}

func (m Model) showNewsPanel() bool {
	w := m.width
	if w == 0 {
		w = 160
	}
	return w >= newsPanelMinWidth
}

func (m Model) tableWidth() int {
	w := m.width
	if w == 0 {
		w = 160
	}
	if m.showNewsPanel() {
		return w - newsPanelWidth - 1
	}
	return w
}

func (m Model) viewMarket() string {
	w := m.width
	if w == 0 {
		w = 160
	}
	tW := m.tableWidth()

	// ── header bar ──────────────────────────────────────────────────────
	portfolioVal := m.g.PortfolioValue()
	startVal := m.g.Difficulty.StartingCash()
	pnl := portfolioVal - startVal
	pnlStyle := colorForChange(pnl)

	nextNews := m.g.Market.NextNewsIn()
	nextStr := fmt.Sprintf("next news in %ds", int(nextNews.Seconds()))

	header := styleHeader.Render(" TRADEZ ") +
		styleNeutral.Render(" │ ") +
		styleWhiteStr("Cash: ") + stylePositive.Render(fmt.Sprintf("$%s", commaf(m.g.Cash))) +
		styleNeutral.Render("  ") +
		styleWhiteStr("Portfolio: ") + styleAccentStr(fmt.Sprintf("$%s", commaf(portfolioVal))) +
		styleNeutral.Render("  ") +
		styleWhiteStr("P&L: ") + pnlStyle.Render(fmt.Sprintf("%s$%s", signStr(pnl), commaf(math.Abs(pnl)))) +
		styleNeutral.Render("  ") +
		styleHint.Render(time.Now().Format("15:04:05")) +
		styleNeutral.Render("  ") +
		styleHint.Render(nextStr)

	// ── sort bar ─────────────────────────────────────────────────────────
	sortLabels := [5]string{"1:SYM", "2:PRI", "3:CHG%", "4:VOL", "5:CAP"}
	sortBar := styleNeutral.Render(" Sort: ")
	for i, sl := range sortLabels {
		if i == m.sortCol {
			if m.sortAsc {
				sortBar += styleAccentStr(sl + "▲")
			} else {
				sortBar += styleAccentStr(sl + "▼")
			}
		} else {
			sortBar += styleNeutral.Render(sl)
		}
		if i < len(sortLabels)-1 {
			sortBar += styleNeutral.Render("  ")
		}
	}

	// ── column headers ───────────────────────────────────────────────────
	colHeader := padR("SYMBOL", 7) + padR("NAME", 20) + padR("INDUSTRY", 13) +
		padR("PRICE", 9) + padR("CHG", 9) + padR("CHG%", 8) +
		padR("VOLUME", 11) + padR("MKT CAP", 10) + padR("P/E", 6) + padR("β", 5) + "SPARK"

	// ── rows ─────────────────────────────────────────────────────────────
	visibleHeight := m.height - 9
	start := 0
	if m.cursor >= visibleHeight {
		start = m.cursor - visibleHeight + 1
	}

	var rows strings.Builder
	for i := start; i < len(m.stocks) && i < start+visibleHeight; i++ {
		s := m.stocks[i]
		chg := s.Change()
		pct := s.ChangePct()
		cs := colorForChange(pct)

		inf := m.g.Market.StockInfluenceStrength(s.Symbol)
		infChar := " "
		if inf > 0.1 {
			infChar = "↑"
		} else if inf < -0.1 {
			infChar = "↓"
		}
		infStyled := infChar
		if inf > 0.1 {
			infStyled = stylePositive.Render(infChar)
		} else if inf < -0.1 {
			infStyled = styleNegative.Render(infChar)
		}

		chgStr := padR(fmt.Sprintf("%s%.2f", signStr(chg), chg), 9)
		pctStr := padR(fmt.Sprintf("%s%.2f%%", signStr(pct), pct), 8)
		spark := sparklineChars(s.History, 8)

		if i == m.cursor {
			// Plain text row — no embedded ANSI so styleSelected can measure it correctly
			plain := padR(s.Symbol, 7) +
				padR(truncate(s.Name, 19), 20) +
				padR(truncate(string(s.Industry), 12), 13) +
				padR(fmt.Sprintf("$%.2f", s.Price), 9) +
				chgStr +
				pctStr +
				padR(commafInt(s.Volume), 11) +
				padR(market.FormatMarketCap(s.MarketCap), 10) +
				padR(fmt.Sprintf("%.1f", s.PERatio), 6) +
				padR(fmt.Sprintf("%.2f", s.Beta), 5) +
				spark + infChar
			rows.WriteString(styleSelected.Render(plain) + "\n")
		} else {
			colored := padR(s.Symbol, 7) +
				padR(truncate(s.Name, 19), 20) +
				padR(truncate(string(s.Industry), 12), 13) +
				padR(fmt.Sprintf("$%.2f", s.Price), 9) +
				cs.Render(chgStr) +
				cs.Render(pctStr) +
				padR(commafInt(s.Volume), 11) +
				padR(market.FormatMarketCap(s.MarketCap), 10) +
				padR(fmt.Sprintf("%.1f", s.PERatio), 6) +
				padR(fmt.Sprintf("%.2f", s.Beta), 5) +
				sparkline(s.History, 8) + infStyled
			rows.WriteString(colored + "\n")
		}
	}

	// ── message / status line ─────────────────────────────────────────────
	msgLine := ""
	if m.errMsg != "" {
		msgLine = " " + styleError.Render("⚠ "+m.errMsg)
	} else if m.okMsg != "" {
		msgLine = " " + styleOk.Render("✓ "+m.okMsg)
	} else if len(m.g.Messages) > 0 {
		msgLine = " " + styleNeutral.Render(m.g.Messages[len(m.g.Messages)-1])
	}

	keys := styleHint.Render(" ↑↓/jk  enter=detail  b=buy  s=sell  p=portfolio  o=orders  1-5=sort  q=quit")

	div := styleNeutral.Render(strings.Repeat("─", tW))

	tableContent := header + "\n" +
		sortBar + "\n" +
		div + "\n" +
		lipgloss.NewStyle().Bold(true).Foreground(colorGray).Render(" "+colHeader) + "\n" +
		div + "\n" +
		rows.String() +
		div + "\n" +
		msgLine + "\n" +
		keys

	// ── news panel ────────────────────────────────────────────────────────
	if !m.showNewsPanel() {
		return tableContent
	}

	newsPanel := m.renderNewsPanel(m.height - 2)
	return lipgloss.JoinHorizontal(lipgloss.Top, tableContent, newsPanel)
}

func (m Model) renderNewsPanel(height int) string {
	inner := newsPanelWidth - 2

	title := lipgloss.NewStyle().Bold(true).Foreground(colorAccent).
		Width(inner).Render("▸ MARKET FEED")

	// Active influences summary
	influences := m.g.Market.ActiveInfluences()
	var infLines strings.Builder
	if len(influences) > 0 {
		infLines.WriteString(styleNeutral.Render(strings.Repeat("─", inner)) + "\n")
		infLines.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colorGray).Render("ACTIVE INFLUENCES") + "\n")
		for _, inf := range influences {
			remaining := time.Until(inf.ExpiresAt)
			minsLeft := int(remaining.Minutes())
			secsLeft := int(remaining.Seconds()) % 60
			timeStr := fmt.Sprintf("%dm%ds", minsLeft, secsLeft)
			if minsLeft == 0 {
				timeStr = fmt.Sprintf("%ds", int(remaining.Seconds()))
			}

			arrow := stylePositive.Render("▲")
			if inf.Sentiment < 0 {
				arrow = styleNegative.Render("▼")
			}
			strength := math.Abs(inf.InfluenceStrength())
			bars := int(strength * 5)
			if bars > 5 {
				bars = 5
			}
			barStr := strings.Repeat("█", bars) + strings.Repeat("░", 5-bars)

			targets := industryListStr(inf.AffectedIndustries)
			if len(inf.AffectedSymbols) > 0 {
				targets = strings.Join(inf.AffectedSymbols, ",")
			}

			line := arrow + " " +
				lipgloss.NewStyle().Foreground(colorGray).Render(padR(truncate(targets, 14), 15)) +
				colorForChange(inf.Sentiment).Render(barStr) +
				styleNeutral.Render(" "+timeStr)
			infLines.WriteString(truncateLine(line, inner) + "\n")
		}
	}

	// Flash banner for new news
	flashBanner := ""
	if m.flashNews != nil {
		catColor := lipgloss.Color(market.CategoryColor(m.flashNews.Category))
		badge := lipgloss.NewStyle().
			Background(catColor).
			Foreground(lipgloss.Color("#000000")).
			Bold(true).
			Padding(0, 1).
			Render("● BREAKING")
		flashBanner = badge + "\n"
		wrapped := wrapText(m.flashNews.Headline, inner)
		for _, line := range wrapped {
			flashBanner += lipgloss.NewStyle().Bold(true).Foreground(colorWhite).Render(line) + "\n"
		}
		flashBanner += "\n"
	}

	// News list (most recent first)
	allNews := m.g.Market.RecentNews(50)
	// reverse so newest is at top
	for i, j := 0, len(allNews)-1; i < j; i, j = i+1, j-1 {
		allNews[i], allNews[j] = allNews[j], allNews[i]
	}

	// apply scroll
	if m.newsScroll > len(allNews)-1 {
		m.newsScroll = len(allNews) - 1
	}
	if m.newsScroll < 0 {
		m.newsScroll = 0
	}
	visible := allNews
	if m.newsScroll < len(visible) {
		visible = visible[m.newsScroll:]
	}

	var newsLines strings.Builder
	newsLines.WriteString(styleNeutral.Render(strings.Repeat("─", inner)) + "\n")

	linesUsed := 0
	maxLines := height - 8 - strings.Count(infLines.String(), "\n") - strings.Count(flashBanner, "\n")

	for _, n := range visible {
		if linesUsed >= maxLines {
			break
		}
		catColor := lipgloss.Color(market.CategoryColor(n.Category))
		badge := lipgloss.NewStyle().
			Background(catColor).
			Foreground(lipgloss.Color("#000000")).
			Bold(true).
			Padding(0, 1).
			Render(string(n.Category))

		ago := styleNeutral.Render(n.TimeAgo())
		sentLabel := n.SentimentLabel()
		sentStyle := colorForChange(n.Sentiment)

		// first line: badge + time + sentiment
		headerLine := badge + " " + ago + "  " + sentStyle.Render(sentLabel)
		newsLines.WriteString(headerLine + "\n")
		linesUsed++

		// headline (word-wrapped)
		wrapped := wrapText(n.Headline, inner)
		for i, line := range wrapped {
			if linesUsed >= maxLines {
				break
			}
			if i == 0 {
				newsLines.WriteString(styleWhiteStr(line) + "\n")
			} else {
				newsLines.WriteString(styleNeutral.Render("  "+line) + "\n")
			}
			linesUsed++
		}

		// affected targets line
		if linesUsed < maxLines {
			targets := industryListStr(n.AffectedIndustries)
			if len(n.AffectedSymbols) > 0 {
				targets = strings.Join(n.AffectedSymbols, ", ") + " · " + targets
			}
			if targets != "" {
				newsLines.WriteString(styleNeutral.Render("  → "+truncate(targets, inner-4)) + "\n")
				linesUsed++
			}
		}

		if linesUsed < maxLines {
			newsLines.WriteString("\n")
			linesUsed++
		}
	}

	if len(allNews) == 0 {
		newsLines.WriteString(styleNeutral.Render("  Awaiting market events...") + "\n")
	}

	scrollHint := ""
	if len(allNews) > 5 {
		scrollHint = "\n" + styleHint.Render(" pgup/pgdn to scroll")
	}

	panelContent := title + "\n" +
		flashBanner +
		infLines.String() +
		newsLines.String() +
		scrollHint

	return lipgloss.NewStyle().
		Width(newsPanelWidth).
		BorderStyle(lipgloss.NormalBorder()).
		BorderLeft(true).
		BorderForeground(colorBorder).
		Padding(0, 1).
		Render(panelContent)
}

func (m Model) viewStock() string {
	s := m.g.Market.GetStock(m.stockSymbol)
	if s == nil {
		return "Stock not found"
	}

	w := m.width
	if w == 0 {
		w = 100
	}

	pct := s.ChangePct()
	chg := s.Change()
	cs := colorForChange(pct)

	pos := m.g.Positions[s.Symbol]

	// influence indicator for this stock
	inf := m.g.Market.StockInfluenceStrength(s.Symbol)
	infStr := ""
	if math.Abs(inf) > 0.05 {
		label := "Market tailwind"
		st := stylePositive
		if inf < 0 {
			label = "Market headwind"
			st = styleNegative
		}
		bars := int(math.Abs(inf) * 8)
		if bars > 8 {
			bars = 8
		}
		infStr = "  " + st.Render(label+": "+strings.Repeat("█", bars)+strings.Repeat("░", 8-bars))
	}

	title := styleTitle.Render(s.Symbol) + "  " +
		styleWhiteStr(s.Name) + "  " +
		styleNeutral.Render(string(s.Industry)) + infStr

	priceBar := lipgloss.NewStyle().Bold(true).Foreground(colorWhite).Render(fmt.Sprintf("$%.2f  ")) +
		cs.Render(fmt.Sprintf("%s%.2f  (%s%.2f%%)", signStr(chg), chg, signStr(pct), pct))

	tabs := ""
	tabNames := []string{"Chart", "Details"}
	for i, t := range tabNames {
		if i == m.tab {
			tabs += styleTabActive.Render(t)
		} else {
			tabs += styleTab.Render(t)
		}
	}

	var content string
	switch m.tab {
	case 0:
		content = m.renderChart(s, w-4)
	case 1:
		content = m.renderDetails(s)
	}

	posSummary := ""
	if pos != nil && (pos.Shares > 0 || pos.ShortShares > 0) {
		pnlLong := pos.LongPnL(s.Price)
		posSummary = "\n" + styleBorder.Padding(0, 1).Render(
			styleWhiteStr("Position: ") +
				stylePositive.Render(fmt.Sprintf("%d long", pos.Shares)) +
				styleNeutral.Render(" │ ") +
				styleNegative.Render(fmt.Sprintf("%d short", pos.ShortShares)) +
				styleNeutral.Render(fmt.Sprintf("  Avg: $%.2f", pos.AvgCost)) +
				"  P&L: " + colorForChange(pnlLong).Render(fmt.Sprintf("%s$%.2f", signStr(pnlLong), math.Abs(pnlLong))),
		)
	}

	// show relevant active news for this stock
	newsForStock := ""
	for _, n := range m.g.Market.RecentNews(20) {
		for _, sym := range n.AffectedSymbols {
			if sym == s.Symbol && n.IsActive() {
				sentStyle := colorForChange(n.Sentiment)
				newsForStock += "\n" + styleNeutral.Render("  ") +
					lipgloss.NewStyle().
						Background(lipgloss.Color(market.CategoryColor(n.Category))).
						Foreground(lipgloss.Color("#000000")).Bold(true).Padding(0, 1).
						Render(string(n.Category)) +
					"  " + sentStyle.Render(n.SentimentLabel()) +
					"  " + styleWhiteStr(truncate(n.Headline, w-20))
			}
		}
		for _, ind := range n.AffectedIndustries {
			if ind == s.Industry && n.IsActive() && len(n.AffectedSymbols) == 0 {
				sentStyle := colorForChange(n.Sentiment)
				newsForStock += "\n" + styleNeutral.Render("  ") +
					lipgloss.NewStyle().
						Background(lipgloss.Color(market.CategoryColor(n.Category))).
						Foreground(lipgloss.Color("#000000")).Bold(true).Padding(0, 1).
						Render(string(n.Category)) +
					"  " + sentStyle.Render(n.SentimentLabel()) +
					"  " + styleWhiteStr(truncate(n.Headline, w-20))
			}
		}
	}

	div := styleNeutral.Render(strings.Repeat("─", w))
	keys := styleHint.Render(" b=buy  s=sell  l=limit buy  x=limit sell  h=short  c=cover  tab=chart/details  esc=back")

	return title + "\n" + priceBar + "\n" +
		div + "\n" +
		tabs + "\n" +
		div + "\n" +
		content +
		newsForStock +
		posSummary + "\n" +
		div + "\n" +
		keys
}

func (m Model) renderChart(s *market.Stock, width int) string {
	if len(s.History) < 2 {
		return styleNeutral.Render("  Collecting data...")
	}
	n := width - 12
	if n < 10 {
		n = 10
	}
	history := s.History
	if len(history) > n {
		history = history[len(history)-n:]
	}
	prices := make([]float64, len(history))
	for i, p := range history {
		prices[i] = p.Price
	}

	h := m.height - 14
	if h < 8 {
		h = 8
	}
	if h > 24 {
		h = 24
	}

	chart := asciigraph.Plot(prices,
		asciigraph.Height(h),
		asciigraph.Width(width-14),
		asciigraph.Caption(fmt.Sprintf("Last %d ticks (2s each)", len(prices))),
	)

	first := prices[0]
	last := prices[len(prices)-1]
	trendPct := (last - first) / first * 100
	trendStyle := colorForChange(trendPct)

	return chart + "\n" +
		styleNeutral.Render(fmt.Sprintf("  Open: $%.2f  ", s.Open)) +
		stylePositive.Render(fmt.Sprintf("H: $%.2f  ", s.High)) +
		styleNegative.Render(fmt.Sprintf("L: $%.2f  ", s.Low)) +
		trendStyle.Render(fmt.Sprintf("Trend: %s%.2f%%", signStr(trendPct), trendPct))
}

func (m Model) renderDetails(s *market.Stock) string {
	fields := [][2]string{
		{"Symbol", s.Symbol},
		{"Name", s.Name},
		{"Industry", string(s.Industry)},
		{"Price", fmt.Sprintf("$%.2f", s.Price)},
		{"Prev Close", fmt.Sprintf("$%.2f", s.PrevClose)},
		{"Open", fmt.Sprintf("$%.2f", s.Open)},
		{"Day High", fmt.Sprintf("$%.2f", s.High)},
		{"Day Low", fmt.Sprintf("$%.2f", s.Low)},
		{"Volume", commaf(float64(s.Volume))},
		{"Market Cap", market.FormatMarketCap(s.MarketCap)},
		{"P/E Ratio", fmt.Sprintf("%.1f", s.PERatio)},
		{"EPS", fmt.Sprintf("$%.2f", s.EPS)},
		{"Div Yield", fmt.Sprintf("%.2f%%", s.DivYield)},
		{"Beta", fmt.Sprintf("%.2f", s.Beta)},
		{"YTD Growth", fmt.Sprintf("%.2f%%", s.YTDGrowth)},
	}
	var sb strings.Builder
	for _, f := range fields {
		label := lipgloss.NewStyle().Width(14).Foreground(colorGray).Render(f[0])
		sb.WriteString("  " + label + "  " + styleWhiteStr(f[1]) + "\n")
	}
	return sb.String()
}

func (m Model) viewPortfolio() string {
	w := m.width
	if w == 0 {
		w = 100
	}

	portfolioVal := m.g.PortfolioValue()
	startVal := m.g.Difficulty.StartingCash()
	totalPnL := portfolioVal - startVal

	header := styleTitle.Render(" PORTFOLIO ") +
		styleNeutral.Render("  Value: ") + styleAccentStr(fmt.Sprintf("$%s", commaf(portfolioVal))) +
		styleNeutral.Render("  P&L: ") + colorForChange(totalPnL).Render(fmt.Sprintf("%s$%s", signStr(totalPnL), commaf(math.Abs(totalPnL)))) +
		styleNeutral.Render("  Cash: ") + stylePositive.Render(fmt.Sprintf("$%s", commaf(m.g.Cash)))

	colHeader := padR("SYMBOL", 8) + padR("SHARES", 8) + padR("AVG COST", 12) +
		padR("CURR PRICE", 12) + padR("MKT VALUE", 14) + padR("LONG P&L", 14) +
		padR("SHORT", 8) + padR("SHORT AVG", 12) + padR("SHORT P&L", 12)

	syms := m.portfolioSymbols()
	var rows strings.Builder
	for i, sym := range syms {
		pos := m.g.Positions[sym]
		if pos == nil || (pos.Shares == 0 && pos.ShortShares == 0) {
			continue
		}
		s := m.g.Market.GetStock(sym)
		if s == nil {
			continue
		}
		pnl := pos.LongPnL(s.Price)
		shortPnL := pos.ShortPnL(s.Price)

		row := padR(sym, 8) +
			padR(fmt.Sprintf("%d", pos.Shares), 8) +
			padR(fmt.Sprintf("$%.2f", pos.AvgCost), 12) +
			padR(fmt.Sprintf("$%.2f", s.Price), 12) +
			padR(fmt.Sprintf("$%s", commaf(pos.MarketValue(s.Price))), 14) +
			colorForChange(pnl).Render(padR(fmt.Sprintf("%s$%.2f", signStr(pnl), math.Abs(pnl)), 14)) +
			padR(fmt.Sprintf("%d", pos.ShortShares), 8) +
			padR(fmt.Sprintf("$%.2f", pos.ShortAvg), 12) +
			colorForChange(shortPnL).Render(padR(fmt.Sprintf("%s$%.2f", signStr(shortPnL), math.Abs(shortPnL)), 12))

		if i == m.cursor {
			rows.WriteString(styleSelected.Render(row) + "\n")
		} else {
			rows.WriteString(lipgloss.NewStyle().Foreground(colorWhite).Render(row) + "\n")
		}
	}
	if rows.Len() == 0 {
		rows.WriteString(styleNeutral.Render("  No open positions.") + "\n")
	}

	div := styleNeutral.Render(strings.Repeat("─", w))
	keys := styleHint.Render(" ↑↓ navigate  enter=stock detail  esc=back")

	return header + "\n" + div + "\n" +
		lipgloss.NewStyle().Bold(true).Foreground(colorGray).Render(" "+colHeader) + "\n" +
		div + "\n" + rows.String() + div + "\n" + keys
}

func (m Model) viewOrders() string {
	w := m.width
	if w == 0 {
		w = 100
	}

	pending := m.g.PendingOrders()
	filled := m.g.FilledOrders()

	header := styleTitle.Render(" ORDERS")
	colHeader := padR("#", 6) + padR("TYPE", 14) + padR("SYMBOL", 8) +
		padR("SHARES", 8) + padR("LIMIT", 12) + padR("FILLED@", 12) + "STATUS"

	var rows strings.Builder
	for i, o := range pending {
		row := padR(fmt.Sprintf("#%d", o.ID), 6) +
			styleYellowStr(padR(o.Type.String(), 14)) +
			padR(o.Symbol, 8) +
			padR(fmt.Sprintf("%d", o.Shares), 8) +
			padR(fmt.Sprintf("$%.2f", o.LimitPrice), 12) +
			padR("-", 12) +
			styleYellowStr("PENDING")
		if i == m.cursor {
			rows.WriteString(styleSelected.Render(row) + "\n")
		} else {
			rows.WriteString(lipgloss.NewStyle().Foreground(colorWhite).Render(row) + "\n")
		}
	}
	if len(pending) == 0 {
		rows.WriteString(styleNeutral.Render("  No pending orders.") + "\n")
	}

	rows.WriteString("\n" + lipgloss.NewStyle().Bold(true).Foreground(colorGray).Render(" Recent Filled Orders") + "\n")
	for _, o := range filled {
		var typeStyle string
		switch o.Type {
		case game.OrderMarketBuy, game.OrderLimitBuy:
			typeStyle = stylePositive.Render(padR(o.Type.String(), 14))
		default:
			typeStyle = styleNegative.Render(padR(o.Type.String(), 14))
		}
		row := padR(fmt.Sprintf("#%d", o.ID), 6) +
			typeStyle +
			padR(o.Symbol, 8) +
			padR(fmt.Sprintf("%d", o.Shares), 8) +
			padR("-", 12) +
			padR(fmt.Sprintf("$%.2f", o.FilledAt), 12) +
			styleOk.Render("FILLED")
		rows.WriteString(lipgloss.NewStyle().Foreground(colorWhite).Render(row) + "\n")
	}

	div := styleNeutral.Render(strings.Repeat("─", w))
	keys := styleHint.Render(" ↑↓ navigate  x=cancel selected  esc=back")

	return header + "\n" + div + "\n" +
		lipgloss.NewStyle().Bold(true).Foreground(colorGray).Render(" "+colHeader) + "\n" +
		div + "\n" + rows.String() + div + "\n" + keys
}

func (m Model) viewTrade() string {
	s := m.g.Market.GetStock(m.stockSymbol)
	if s == nil {
		return "Stock not found"
	}

	isLimit := m.tradeMode == tradeLimitBuy || m.tradeMode == tradeLimitSell
	pct := s.ChangePct()
	cs := colorForChange(pct)

	title := styleTitle.Render(m.tradeMode.label()) + "  " + styleWhiteStr(m.stockSymbol) + "  " + styleWhiteStr(s.Name)
	priceInfo := styleWhiteStr("Current: ") +
		lipgloss.NewStyle().Bold(true).Foreground(colorWhite).Render(fmt.Sprintf("$%.2f  ")) +
		cs.Render(fmt.Sprintf("(%s%.2f%%)", signStr(pct), pct))

	var inputSection string
	if isLimit {
		sharesLabel := styleNeutral.Render("Shares:   ")
		priceLabel := styleNeutral.Render("Limit $:  ")
		if m.inputFocus == 0 {
			inputSection = sharesLabel + styleInput.Render(m.inputShares.View()) + "\n" +
				priceLabel + styleNeutral.Render(m.inputPrice.View()) + "\n" +
				styleHint.Render("  tab to switch fields")
		} else {
			inputSection = sharesLabel + styleNeutral.Render(m.inputShares.View()) + "\n" +
				priceLabel + styleInput.Render(m.inputPrice.View()) + "\n" +
				styleHint.Render("  tab to switch fields")
		}
	} else {
		inputSection = styleNeutral.Render("Shares:   ") + styleInput.Render(m.inputShares.View())
	}

	cashLine := styleNeutral.Render("Cash: ") + stylePositive.Render(fmt.Sprintf("$%s", commaf(m.g.Cash)))

	pos := m.g.Positions[m.stockSymbol]
	posLine := ""
	if pos != nil {
		posLine = "\n" + styleNeutral.Render(fmt.Sprintf("Long: %d @ $%.2f  Short: %d @ $%.2f",
			pos.Shares, pos.AvgCost, pos.ShortShares, pos.ShortAvg))
	}

	// show active news affecting this stock in trade dialog
	relevantNews := ""
	for _, n := range m.g.Market.RecentNews(20) {
		if !n.IsActive() {
			continue
		}
		affected := false
		for _, sym := range n.AffectedSymbols {
			if sym == s.Symbol {
				affected = true
				break
			}
		}
		for _, ind := range n.AffectedIndustries {
			if ind == s.Industry {
				affected = true
				break
			}
		}
		if affected {
			sentStyle := colorForChange(n.Sentiment)
			relevantNews += "\n" + sentStyle.Render(n.SentimentLabel()) + " " + styleNeutral.Render(truncate(n.Headline, 40))
		}
	}
	if relevantNews != "" {
		relevantNews = "\n" + styleNeutral.Render("Active signals:") + relevantNews
	}

	feedback := ""
	if m.errMsg != "" {
		feedback = "\n" + styleError.Render("⚠ "+m.errMsg)
	} else if m.okMsg != "" {
		feedback = "\n" + styleOk.Render("✓ "+m.okMsg)
	}

	keys := styleHint.Render(" enter=confirm  esc=cancel")

	inner := title + "\n" + priceInfo + "\n\n" +
		cashLine + posLine + relevantNews + "\n\n" +
		inputSection + feedback + "\n\n" +
		keys

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
		styleBorder.Padding(1, 3).Render(inner))
}

// ── helpers ────────────────────────────────────────────────────────────────

func padR(s string, width int) string {
	runes := []rune(s)
	if len(runes) >= width {
		return string(runes[:width])
	}
	return s + strings.Repeat(" ", width-len(runes))
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}

// commafInt formats an integer with thousands separators and no decimal places.
func commafInt(v int64) string {
	s := fmt.Sprintf("%d", v)
	neg := ""
	if strings.HasPrefix(s, "-") {
		neg = "-"
		s = s[1:]
	}
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return neg + string(result)
}

// sparklineChars returns the spark bar characters without any ANSI color codes,
// used when the row will be wrapped in styleSelected (which sets its own colors).
func sparklineChars(history []market.PricePoint, width int) string {
	if len(history) < 2 {
		return strings.Repeat(" ", width)
	}
	bars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	n := width
	if len(history) < n {
		n = len(history)
	}
	pts := history[len(history)-n:]
	min, max := pts[0].Price, pts[0].Price
	for _, p := range pts {
		if p.Price < min {
			min = p.Price
		}
		if p.Price > max {
			max = p.Price
		}
	}
	rng := max - min
	result := make([]rune, len(pts))
	for i, p := range pts {
		idx := 0
		if rng > 0 {
			idx = int((p.Price-min)/rng*float64(len(bars)-1))
		}
		result[i] = bars[idx]
	}
	return string(result)
}

func truncateLine(s string, max int) string {
	// strip ANSI before measuring isn't trivial; just return as-is for now
	return s
}

func commaf(v float64) string {
	s := fmt.Sprintf("%.2f", v)
	parts := strings.Split(s, ".")
	intPart := parts[0]
	neg := ""
	if strings.HasPrefix(intPart, "-") {
		neg = "-"
		intPart = intPart[1:]
	}
	var result []byte
	for i, c := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	if len(parts) > 1 {
		return neg + string(result) + "." + parts[1]
	}
	return neg + string(result)
}

func sparkline(history []market.PricePoint, width int) string {
	if len(history) < 2 {
		return strings.Repeat(" ", width)
	}
	bars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	n := width
	if len(history) < n {
		n = len(history)
	}
	pts := history[len(history)-n:]
	min, max := pts[0].Price, pts[0].Price
	for _, p := range pts {
		if p.Price < min {
			min = p.Price
		}
		if p.Price > max {
			max = p.Price
		}
	}
	rng := max - min
	result := make([]rune, len(pts))
	for i, p := range pts {
		idx := 0
		if rng > 0 {
			idx = int((p.Price - min) / rng * float64(len(bars)-1))
		}
		result[i] = bars[idx]
	}
	first := pts[0].Price
	last := pts[len(pts)-1].Price
	s := string(result)
	if last > first {
		return stylePositive.Render(s)
	} else if last < first {
		return styleNegative.Render(s)
	}
	return styleNeutral.Render(s)
}

func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}
	words := strings.Fields(text)
	var lines []string
	var current strings.Builder
	for _, w := range words {
		if current.Len()+len(w)+1 > width && current.Len() > 0 {
			lines = append(lines, current.String())
			current.Reset()
		}
		if current.Len() > 0 {
			current.WriteByte(' ')
		}
		current.WriteString(w)
	}
	if current.Len() > 0 {
		lines = append(lines, current.String())
	}
	return lines
}

func industryListStr(industries []market.Industry) string {
	parts := make([]string, len(industries))
	for i, ind := range industries {
		parts[i] = string(ind)
	}
	return strings.Join(parts, ", ")
}

func styleWhiteStr(s string) string {
	return lipgloss.NewStyle().Foreground(colorWhite).Render(s)
}

func styleYellowStr(s string) string {
	return lipgloss.NewStyle().Foreground(colorYellow).Render(s)
}

func styleAccentStr(s string) string {
	return lipgloss.NewStyle().Foreground(colorAccent).Render(s)
}
