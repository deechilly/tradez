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
	screenSlotSelect screen = iota
	screenMenu
	screenCapital
	screenMarket
	screenStock
	screenPortfolio
	screenOrders
	screenTrade
	screenAchievements
)


type tradeMode int

const (
	tradeBuy tradeMode = iota
	tradeSell
	tradeLimitBuy
	tradeLimitSell
	tradeShort
	tradeCover
	tradePut
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
	case tradePut:
		return "Buy PUT"
	}
	return ""
}

const (
	newsPanelWidth    = 40
	newsPanelMinWidth = 112 // compact cols (~70) + panel (40) + border (1) + gutter
)

type Model struct {
	g                 *game.Game
	screen            screen
	activeSlot        int             // 1–3; 0 until a slot is chosen
	slotCursor        int             // 0–2 on the slot-select screen
	slotInfos         [3]game.SaveSlot
	pendingDifficulty game.Difficulty
	width             int
	height            int
	cursor            int
	stockSymbol       string
	sortCol           int
	sortAsc           bool
	tab               int
	tradeMode         tradeMode
	inputShares       textinput.Model
	inputPrice        textinput.Model
	inputFocus        int
	putStrikeCursor   int // 0–4: deep ITM → deep OTM
	putExpiryCursor   int // 0–2: Short / Medium / Long
	errMsg            string
	okMsg             string
	msgTimer          int
	stocks            []*market.Stock
	flashNews         *market.NewsEvent
	flashTicks        int
	newsScroll        int
	newsVisible       bool

	// Achievement state
	achStore          *game.AchievementStore
	posTrack          map[string]*positionTrack
	sellLog           []sellEntry
	stormWatch        map[string]time.Time // symbol → time negative news hit their sector
	seenFilledOrders  map[int]bool
	lastAnalysisScore float64
	sessionShortWins  int
	perSymbolWins     map[string][2]bool // [longWin, shortWin]
	flashAchieveID    string
	flashAchieveTick  int
	achieveQueue      []string
	achScroll         int
}

func NewModel() Model {
	ti := textinput.New()
	ti.Placeholder = "100"
	ti.CharLimit = 10
	ti.Width = 12

	tp := textinput.New()
	tp.Placeholder = "0.00"
	tp.CharLimit = 12
	tp.Width = 12

	m := Model{
		screen:           screenSlotSelect,
		sortCol:          0,
		sortAsc:          true,
		inputShares:      ti,
		inputPrice:       tp,
		newsVisible:      true,
		achStore:         game.LoadAchievements(),
		posTrack:         make(map[string]*positionTrack),
		stormWatch:       make(map[string]time.Time),
		seenFilledOrders: make(map[int]bool),
		perSymbolWins:    make(map[string][2]bool),
	}
	m.loadSlotInfos()
	return m
}

func (m *Model) loadSlotInfos() {
	for i := 0; i < 3; i++ {
		m.slotInfos[i] = game.SlotInfo(i + 1)
	}
}

func (m *Model) refreshStocks() {
	if m.g == nil {
		return
	}
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
		if m.g != nil {
			prevPortfolioVal := m.g.PortfolioValue()

			newEvent := m.g.Market.Tick()
			m.g.ProcessLimitOrders()
			m.g.ProcessPuts()
			m.checkNewLimitFills()
			m.refreshStocks()

			if newEvent != nil {
				m.flashNews = newEvent
				m.flashTicks = 6
				m.updateStormWatch(newEvent)
			}
			if m.flashTicks > 0 {
				m.flashTicks--
				if m.flashTicks == 0 {
					m.flashNews = nil
				}
			}

			m.checkTickAchievements(prevPortfolioVal)
			m.tickAchieveFlash()

			if m.msgTimer > 0 {
				m.msgTimer--
				if m.msgTimer == 0 {
					m.errMsg = ""
					m.okMsg = ""
				}
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
	case screenSlotSelect:
		return m.handleSlotSelectKey(msg)
	case screenMenu:
		return m.handleMenuKey(msg)
	case screenCapital:
		return m.handleCapitalKey(msg)
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
	case screenAchievements:
		return m.handleAchievementsKey(msg)
	}
	return m, nil
}

func (m Model) handleMenuKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "1":
		m.pendingDifficulty = game.DifficultyEasy
		m.screen = screenCapital
	case "2":
		m.pendingDifficulty = game.DifficultyNormal
		m.screen = screenCapital
	case "q", "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) handleCapitalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	startGame := func(cash float64) {
		m.g.Cash = cash
		m.g.StartingCash = cash
		m.g.Difficulty = m.pendingDifficulty
		m.g.Market.ScheduleFirstNews()
		m.screen = screenMarket
	}
	switch msg.String() {
	case "1":
		startGame(1_000_000)
	case "2":
		startGame(50_000)
	case "3":
		startGame(1_000)
	case "esc":
		m.screen = screenMenu
	case "q", "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) handleMarketKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.loadSlotInfos()
		m.screen = screenSlotSelect
		return m, nil
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
	case "a":
		m.screen = screenAchievements
		m.achScroll = 0
	case "ctrl+s":
		if m.activeSlot > 0 {
			if err := game.Save(m.g, m.activeSlot); err != nil {
				m.errMsg = fmt.Sprintf("Save failed: %v", err)
			} else {
				m.okMsg = fmt.Sprintf("Saved to Game %d", m.activeSlot)
			}
			m.msgTimer = 3
		}
	case "n":
		m.newsVisible = !m.newsVisible
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
	case "p":
		m.openTrade(tradePut)
	case "tab":
		m.tab = (m.tab + 1) % 3
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
	if mode == tradePut {
		m.putStrikeCursor = 2 // default ATM
		m.putExpiryCursor = 1 // default Medium
	}
	if s := m.g.Market.GetStock(m.stockSymbol); s != nil {
		m.lastAnalysisScore = m.analysisComposite(s)
	}
}

func (m Model) handlePortfolioKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	syms := m.portfolioSymbols()
	totalRows := len(syms) + len(m.g.Puts)
	switch msg.String() {
	case "q", "esc":
		m.screen = screenMarket
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < totalRows-1 {
			m.cursor++
		}
	case "enter":
		if m.cursor < len(syms) {
			m.stockSymbol = syms[m.cursor]
			m.screen = screenStock
		}
	case "x":
		putIdx := m.cursor - len(syms)
		if putIdx >= 0 && putIdx < len(m.g.Puts) {
			put := m.g.Puts[putIdx]
			if err := m.g.SellPut(put.ID, put.Contracts); err != nil {
				m.errMsg = err.Error()
			} else {
				m.okMsg = "Put sold!"
			}
			m.msgTimer = 3
			if m.cursor >= totalRows-1 && m.cursor > 0 {
				m.cursor--
			}
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

var putExpiries = []game.PutExpiry{game.PutExpiryShort, game.PutExpiryMedium, game.PutExpiryLong}

func (m Model) handleTradeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	isLimit := m.tradeMode == tradeLimitBuy || m.tradeMode == tradeLimitSell

	// Put buy dialog has its own navigation
	if m.tradeMode == tradePut {
		switch msg.String() {
		case "esc":
			m.screen = screenMarket
		case "up", "k":
			if m.putStrikeCursor > 0 {
				m.putStrikeCursor--
			}
		case "down", "j":
			if m.putStrikeCursor < 4 {
				m.putStrikeCursor++
			}
		case "left", "h":
			if m.putExpiryCursor > 0 {
				m.putExpiryCursor--
			}
		case "right", "l":
			if m.putExpiryCursor < 2 {
				m.putExpiryCursor++
			}
		case "enter":
			s := m.g.Market.GetStock(m.stockSymbol)
			if s == nil {
				m.errMsg = "Stock not found"
				m.msgTimer = 5
				return m, nil
			}
			contractsStr := strings.TrimSpace(m.inputShares.Value())
			contracts, err := strconv.Atoi(contractsStr)
			if err != nil || contracts <= 0 {
				m.errMsg = "Enter a positive contract count"
				m.msgTimer = 5
				return m, nil
			}
			strikes := game.PutStrikes(s.Price)
			strike := strikes[m.putStrikeCursor]
			expiry := putExpiries[m.putExpiryCursor]
			if err := m.g.BuyPut(m.stockSymbol, strike, expiry, contracts); err != nil {
				m.errMsg = err.Error()
				m.msgTimer = 5
			} else {
				m.okMsg = "Put purchased!"
				m.msgTimer = 3
				m.screen = screenMarket
			}
		default:
			var cmd tea.Cmd
			m.inputShares, cmd = m.inputShares.Update(msg)
			return m, cmd
		}
		return m, nil
	}

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
		// Record pre-trade state for achievement checks
		s := m.g.Market.GetStock(m.stockSymbol)
		prePV := m.g.PortfolioValue()
		preCash := m.g.Cash
		var preAvgCost, preShortAvg float64
		var preShares, preShortShares int
		if s != nil {
			if pos := m.g.Positions[m.stockSymbol]; pos != nil {
				preAvgCost = pos.AvgCost
				preShortAvg = pos.ShortAvg
				preShares = pos.Shares
				preShortShares = pos.ShortShares
			}
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
			if !isLimit && s != nil {
				m.checkTradeAchievements(prePV, preCash, preAvgCost, preShortAvg,
					preShares, preShortShares, shares, m.tradeMode, s)
			}
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
	case screenSlotSelect:
		return m.viewSlotSelect()
	case screenMenu:
		return m.viewMenu()
	case screenCapital:
		return m.viewCapital()
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
	case screenAchievements:
		return m.viewAchievements()
	}
	return ""
}

func (m Model) handleSlotSelectKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.slotCursor > 0 {
			m.slotCursor--
		}
	case "down", "j":
		if m.slotCursor < 2 {
			m.slotCursor++
		}
	case "1":
		m.slotCursor = 0
		return m.activateSlot()
	case "2":
		m.slotCursor = 1
		return m.activateSlot()
	case "3":
		m.slotCursor = 2
		return m.activateSlot()
	case "enter":
		return m.activateSlot()
	case "n":
		return m.startNewGameInSlot()
	case "q", "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) activateSlot() (tea.Model, tea.Cmd) {
	slot := m.slotInfos[m.slotCursor]
	if slot.Exists {
		g, err := game.Load(slot.Number)
		if err == nil {
			m.g = g
			m.activeSlot = slot.Number
			m.refreshStocks()
			m.screen = screenMarket
			return m, nil
		}
	}
	return m.startNewGameInSlot()
}

func (m Model) startNewGameInSlot() (tea.Model, tea.Cmd) {
	mkt := market.New(time.Now().UnixNano())
	m.g = game.New(mkt)
	m.activeSlot = m.slotInfos[m.slotCursor].Number
	m.screen = screenMenu
	return m, nil
}

func (m Model) viewSlotSelect() string {
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

	var rows []string
	for i, slot := range m.slotInfos {
		cursor := "   "
		if i == m.slotCursor {
			cursor = styleAccentStr(" ▶ ")
		}
		label := styleWhiteStr(slot.Name)

		var detail string
		if slot.Exists {
			pnl := slot.PortfolioValue - slot.StartingCash
			pnlStr := fmt.Sprintf("%s$%s", signStr(pnl), commaf(math.Abs(pnl)))
			d := time.Since(slot.SavedAt)
			var agoStr string
			switch {
			case d < time.Minute:
				agoStr = fmt.Sprintf("%ds ago", int(d.Seconds()))
			case d < time.Hour:
				agoStr = fmt.Sprintf("%dm ago", int(d.Minutes()))
			case d < 24*time.Hour:
				agoStr = fmt.Sprintf("%dh ago", int(d.Hours()))
			default:
				agoStr = fmt.Sprintf("%dd ago", int(d.Hours()/24))
			}
			detail = styleNeutral.Render("  ·  ") +
				styleWhiteStr("Portfolio ") + styleAccentStr("$"+commaf(slot.PortfolioValue)) +
				styleNeutral.Render("  P&L ") + colorForChange(pnl).Render(pnlStr) +
				styleNeutral.Render("  ") + styleHint.Render(slot.Difficulty.String()) +
				styleNeutral.Render("  ·  ") + styleHint.Render(agoStr)
		} else {
			detail = styleNeutral.Render("  ·  ") + styleHint.Render("[empty — press enter to start new game]")
		}
		rows = append(rows, cursor+label+detail)
	}

	hints := styleHint.Render("  [enter] Load  ·  [n] New Game  ·  [↑↓ / jk] Navigate  ·  [q] Quit")

	content := logo + "\n\n" +
		styleHeader.Render("  Save Slots") + "\n\n" +
		strings.Join(rows, "\n") + "\n\n" +
		hints

	return lipgloss.Place(w, m.height, lipgloss.Center, lipgloss.Center,
		styleBorder.Padding(2, 4).Render(content))
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
		stylePositive.Render("  [1] ") + styleWhiteStr("Easy   — influence timers and full analyst insights visible"),
		styleYellowStr("  [2] ") + styleWhiteStr("Normal — market signals are hidden; read the tape yourself"),
		"",
		styleNeutral.Render("  [q] Quit"),
	}

	content := logo + "\n\n" + sub + "\n\n" +
		styleHeader.Render("  Select Difficulty") + "\n\n" +
		strings.Join(opts, "\n")

	return lipgloss.Place(w, m.height, lipgloss.Center, lipgloss.Center,
		styleBorder.Padding(2, 4).Render(content))
}

func (m Model) viewCapital() string {
	w := m.width
	if w == 0 {
		w = 80
	}

	diffLabel := stylePositive.Render("Easy")
	if m.pendingDifficulty == game.DifficultyNormal {
		diffLabel = styleYellowStr("Normal")
	}

	opts := []string{
		styleNeutral.Render("  [1] ") + styleWhiteStr("$1,000,000") + styleNeutral.Render("  — maximum buying power"),
		styleNeutral.Render("  [2] ") + styleWhiteStr("$50,000") + styleNeutral.Render("   — moderate start"),
		styleNeutral.Render("  [3] ") + styleWhiteStr("$1,000") + styleNeutral.Render("    — hard mode capital"),
		"",
		styleHint.Render("  [esc] Back"),
	}

	content := styleHeader.Render("  Starting Capital") +
		styleNeutral.Render("  (difficulty: ") + diffLabel + styleNeutral.Render(")") +
		"\n\n" + strings.Join(opts, "\n")

	return lipgloss.Place(w, m.height, lipgloss.Center, lipgloss.Center,
		styleBorder.Padding(2, 4).Render(content))
}

func (m Model) showNewsPanel() bool {
	if !m.newsVisible {
		return false
	}
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
	pnl := portfolioVal - m.g.StartingCash
	pnlStyle := colorForChange(pnl)

	nextNews := m.g.Market.NextNewsIn()
	var rightStatus string
	if m.okMsg != "" {
		rightStatus = styleOk.Render(m.okMsg)
	} else if m.errMsg != "" {
		rightStatus = styleError.Render(m.errMsg)
	} else {
		rightStatus = styleHint.Render(fmt.Sprintf("next news in %ds", int(nextNews.Seconds())))
	}

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
		rightStatus

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
	// Switch to a compact column set when the news panel is visible so both fit.
	// Compact drops: absolute CHG, VOLUME, P/E; narrows NAME and INDUSTRY.
	// Compact total: ~71 chars. Full total: ~107 chars.
	// compact: SYMBOL(7)+PRICE(9)+OPEN(9)+CHG%(8)+IND(9)+β(5)+SPARK(6)+inf(1) = 54
	// full:    SYMBOL(7)+IND(9)+PRICE(9)+OPEN(9)+HIGH(9)+LOW(9)+CHG(9)+CHG%(8)+VOL(11)+MCAP(10)+β(5)+SPARK(8)+inf(1) = 104
	compact := tW < 119

	var nameW int
	if compact {
		nameW = tW - 54
		if nameW < 14 {
			nameW = 14
		}
	} else {
		nameW = tW - 104
		if nameW < 15 {
			nameW = 15
		}
	}

	var colHeader string
	if compact {
		colHeader = padR("SYMBOL", 7) + padR("NAME", nameW) + padR("PRICE", 9) +
			padR("OPEN", 9) + padR("CHG%", 8) + padR("IND", 9) + padR("β", 5) + "SPARK"
	} else {
		colHeader = padR("SYMBOL", 7) + padR("NAME", nameW) + padR("IND", 9) +
			padR("PRICE", 9) + padR("OPEN", 9) + padR("HIGH", 9) + padR("LOW", 9) +
			padR("CHG", 9) + padR("CHG%", 8) + padR("VOLUME", 11) +
			padR("MKT CAP", 10) + padR("β", 5) + "SPARK"
	}

	// ── rows ─────────────────────────────────────────────────────────────
	visibleHeight := max(m.height-9, 1)
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

		pctStr := padR(fmt.Sprintf("%s%.2f%%", signStr(pct), pct), 8)

		if compact {
			spark := sparklineChars(s.History, 6)
			openStr := padR(fmt.Sprintf("$%.2f", s.Open), 9)
			var b strings.Builder
			if i == m.cursor {
				b.WriteString(padR(s.Symbol, 7))
				b.WriteString(padR(truncate(s.Name, nameW-1), nameW))
				b.WriteString(padR(fmt.Sprintf("$%.2f", s.Price), 9))
				b.WriteString(openStr)
				b.WriteString(pctStr)
				b.WriteString(padR(shortIndustry(s.Industry), 9))
				b.WriteString(padR(fmt.Sprintf("%.2f", s.Beta), 5))
				b.WriteString(spark)
				b.WriteString(infChar)
				rows.WriteString(styleSelected.Render(b.String()) + "\n")
			} else {
				openColored := openStr
				if s.Price > s.Open {
					openColored = stylePositive.Render(openStr)
				} else if s.Price < s.Open {
					openColored = styleNegative.Render(openStr)
				}
				b.WriteString(padR(s.Symbol, 7))
				b.WriteString(padR(truncate(s.Name, nameW-1), nameW))
				b.WriteString(padR(fmt.Sprintf("$%.2f", s.Price), 9))
				b.WriteString(openColored)
				b.WriteString(cs.Render(pctStr))
				b.WriteString(padR(shortIndustry(s.Industry), 9))
				b.WriteString(padR(fmt.Sprintf("%.2f", s.Beta), 5))
				b.WriteString(sparkline(s.History, 6))
				b.WriteString(infStyled)
				rows.WriteString(b.String() + "\n")
			}
		} else {
			chgStr := padR(fmt.Sprintf("%s%.2f", signStr(chg), chg), 9)
			spark := sparklineChars(s.History, 8)
			openStr := padR(fmt.Sprintf("$%.2f", s.Open), 9)
			highStr := padR(fmt.Sprintf("$%.2f", s.High), 9)
			lowStr := padR(fmt.Sprintf("$%.2f", s.Low), 9)
			var b strings.Builder
			if i == m.cursor {
				b.WriteString(padR(s.Symbol, 7))
				b.WriteString(padR(truncate(s.Name, nameW-1), nameW))
				b.WriteString(padR(shortIndustry(s.Industry), 9))
				b.WriteString(padR(fmt.Sprintf("$%.2f", s.Price), 9))
				b.WriteString(openStr)
				b.WriteString(highStr)
				b.WriteString(lowStr)
				b.WriteString(chgStr)
				b.WriteString(pctStr)
				b.WriteString(padR(commafInt(s.Volume), 11))
				b.WriteString(padR(market.FormatMarketCap(s.MarketCap), 10))
				b.WriteString(padR(fmt.Sprintf("%.2f", s.Beta), 5))
				b.WriteString(spark)
				b.WriteString(infChar)
				rows.WriteString(styleSelected.Render(b.String()) + "\n")
			} else {
				openColored := openStr
				if s.Price > s.Open {
					openColored = stylePositive.Render(openStr)
				} else if s.Price < s.Open {
					openColored = styleNegative.Render(openStr)
				}
				b.WriteString(padR(s.Symbol, 7))
				b.WriteString(padR(truncate(s.Name, nameW-1), nameW))
				b.WriteString(padR(shortIndustry(s.Industry), 9))
				b.WriteString(padR(fmt.Sprintf("$%.2f", s.Price), 9))
				b.WriteString(openColored)
				b.WriteString(stylePositive.Render(highStr))
				b.WriteString(styleNegative.Render(lowStr))
				b.WriteString(cs.Render(chgStr))
				b.WriteString(cs.Render(pctStr))
				b.WriteString(padR(commafInt(s.Volume), 11))
				b.WriteString(padR(market.FormatMarketCap(s.MarketCap), 10))
				b.WriteString(padR(fmt.Sprintf("%.2f", s.Beta), 5))
				b.WriteString(sparkline(s.History, 8))
				b.WriteString(infStyled)
				rows.WriteString(b.String() + "\n")
			}
		}
	}

	// ── message / status line ─────────────────────────────────────────────
	msgLine := ""
	if m.flashAchieveID != "" {
		def := game.AchievementByID(m.flashAchieveID)
		if def != nil {
			badge := lipgloss.NewStyle().
				Background(lipgloss.Color("#f9ca24")).
				Foreground(lipgloss.Color("#000000")).
				Bold(true).Padding(0, 1).
				Render("★ ACHIEVEMENT")
			msgLine = " " + badge + "  " + styleWhiteStr(def.Icon+" "+def.Name)
		}
	} else if m.errMsg != "" {
		msgLine = " " + styleError.Render("⚠ "+m.errMsg)
	} else if m.okMsg != "" {
		msgLine = " " + styleOk.Render("✓ "+m.okMsg)
	} else if len(m.g.Messages) > 0 {
		msgLine = " " + styleNeutral.Render(m.g.Messages[len(m.g.Messages)-1])
	}

	keys := styleHint.Render(" ↑↓/jk  enter=detail  b=buy  s=sell  p=portfolio  o=orders  a=achievements  1-5=sort  n=news  ctrl+s=save  esc=menu  q=quit")

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

			total := inf.ExpiresAt.Sub(inf.CreatedAt).Seconds()
			decay := 0.0
			if total > 0 {
				rem := remaining.Seconds()
				if rem > 0 {
					decay = rem / total
				}
			}

			for i, eff := range inf.Effects {
				strength := math.Abs(eff.Sentiment * inf.Magnitude * decay)
				bars := int(strength * 5)
				if bars > 5 {
					bars = 5
				}
				if bars == 0 && strength > 0.01 {
					bars = 1
				}
				barStr := strings.Repeat("█", bars) + strings.Repeat("░", 5-bars)

				arrow := stylePositive.Render("▲")
				if eff.Sentiment < 0 {
					arrow = styleNegative.Render("▼")
				}

				targets := industryListStr(eff.Industries)
				if len(eff.Symbols) > 0 {
					targets = strings.Join(eff.Symbols, ",")
				}

				t := ""
				if i == 0 && m.g.Difficulty.ShowInfluenceTimer() {
					t = styleNeutral.Render(" " + timeStr)
				}
				line := arrow + " " +
					lipgloss.NewStyle().Foreground(colorGray).Render(padR(truncate(targets, 14), 15)) +
					colorForChange(eff.Sentiment).Render(barStr) + t
				infLines.WriteString(truncateLine(line, inner) + "\n")
			}
		}
	}

	// Flash banner: achievement unlock takes priority over breaking news
	flashBanner := ""
	if m.flashAchieveID != "" {
		def := game.AchievementByID(m.flashAchieveID)
		if def != nil {
			badge := lipgloss.NewStyle().
				Background(lipgloss.Color("#f9ca24")).
				Foreground(lipgloss.Color("#000000")).
				Bold(true).Padding(0, 1).
				Render("★ ACHIEVEMENT")
			flashBanner = badge + "\n"
			flashBanner += lipgloss.NewStyle().Bold(true).Foreground(colorWhite).
				Render(def.Icon+" "+def.Name) + "\n"
			wrapped := wrapText(def.Desc, inner)
			for _, line := range wrapped {
				flashBanner += styleNeutral.Render(line) + "\n"
			}
			flashBanner += "\n"
		}
	} else if m.flashNews != nil {
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

	priceBar := lipgloss.NewStyle().Bold(true).Foreground(colorWhite).Render(fmt.Sprintf("$%.2f  ", s.Price)) +
		cs.Render(fmt.Sprintf("%s%.2f  (%s%.2f%%)", signStr(chg), chg, signStr(pct), pct))

	tabNames := []string{"Chart", "Details", "Analysis"}
	tabs := ""
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
	case 2:
		content = m.renderAnalysis(s)
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
		if !n.IsActive() {
			continue
		}
		if sent, ok := n.SentimentFor(s); ok {
			sentStyle := colorForChange(sent)
			badge := lipgloss.NewStyle().
				Background(lipgloss.Color(market.CategoryColor(n.Category))).
				Foreground(lipgloss.Color("#000000")).Bold(true).Padding(0, 1).
				Render(string(n.Category))
			newsForStock += "\n" + styleNeutral.Render("  ") + badge +
				"  " + sentStyle.Render(market.SentimentLabel(sent)) +
				"  " + styleWhiteStr(truncate(n.Headline, w-20))
		}
	}

	div := styleNeutral.Render(strings.Repeat("─", w))
	tabHint := "tab=chart/details/analysis"
	keys := styleHint.Render(" b=buy  s=sell  l=limit buy  x=limit sell  h=short  c=cover  " + tabHint + "  esc=back")

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

func (m Model) renderAnalysis(s *market.Stock) string {
	w := m.width - 4
	if w < 60 {
		w = 60
	}

	div := styleNeutral.Render(strings.Repeat("─", w))

	// ── Fundamental score ────────────────────────────────────────
	fundScore := 0.0

	pe := s.PERatio
	avgPE := industryAvgPE(s.Industry)
	peScore := 0.0
	peLabel := "fair value"
	switch {
	case pe <= 0:
		peScore, peLabel = -0.3, "no earnings"
	case pe < avgPE*0.65:
		peScore, peLabel = 0.7, "undervalued"
	case pe < avgPE*0.9:
		peScore, peLabel = 0.35, "cheap"
	case pe < avgPE*1.15:
		peScore, peLabel = 0.05, "fair value"
	case pe < avgPE*1.5:
		peScore, peLabel = -0.3, "rich"
	default:
		peScore, peLabel = -0.6, "expensive"
	}
	fundScore += peScore * 0.40

	epsScore := 0.0
	epsLabel := "break-even"
	if s.EPS > 0 {
		epsScore, epsLabel = 0.3, "profitable"
	} else if s.EPS < 0 {
		epsScore, epsLabel = -0.5, "loss-making"
	}
	fundScore += epsScore * 0.25

	ytdScore := 0.0
	ytdLabel := "moderate"
	switch {
	case s.YTDGrowth < -40:
		ytdScore, ytdLabel = 0.5, "deeply oversold"
	case s.YTDGrowth < -20:
		ytdScore, ytdLabel = 0.25, "oversold"
	case s.YTDGrowth > 60:
		ytdScore, ytdLabel = -0.4, "overbought"
	case s.YTDGrowth > 30:
		ytdScore, ytdLabel = -0.15, "extended"
	}
	fundScore += ytdScore * 0.20

	divScore := math.Min(s.DivYield/6.0, 0.5)
	divLabel := "none"
	if s.DivYield > 0 {
		divLabel = fmt.Sprintf("%.2f%% yield", s.DivYield)
	}
	fundScore += divScore * 0.15

	if fundScore > 1.0 {
		fundScore = 1.0
	} else if fundScore < -1.0 {
		fundScore = -1.0
	}

	// ── Technical score ───────────────────────────────────────────
	hist := s.History
	sma20 := calcSMA(hist, 20)
	sma50 := calcSMA(hist, 50)
	momentum := calcMomentum(hist, 15)

	techScore := 0.0
	ma20Label := "no data"
	if sma20 > 0 {
		if s.Price > sma20 {
			techScore += 0.30
			ma20Label = "▲ above"
		} else {
			techScore -= 0.30
			ma20Label = "▼ below"
		}
	}
	ma50Label := "no data"
	if sma50 > 0 {
		if s.Price > sma50 {
			techScore += 0.25
			ma50Label = "▲ above"
		} else {
			techScore -= 0.25
			ma50Label = "▼ below"
		}
	}
	techScore += momentum * 0.45
	momLabel := "neutral"
	switch {
	case momentum > 0.5:
		momLabel = "strong uptrend"
	case momentum > 0.2:
		momLabel = "uptrend"
	case momentum < -0.5:
		momLabel = "strong downtrend"
	case momentum < -0.2:
		momLabel = "downtrend"
	}
	if techScore > 1.0 {
		techScore = 1.0
	} else if techScore < -1.0 {
		techScore = -1.0
	}

	// ── News score ────────────────────────────────────────────────
	netInf := m.g.Market.StockInfluenceStrength(s.Symbol)
	newsScore := math.Max(-1.0, math.Min(1.0, netInf*4))

	posEvents, negEvents := 0, 0
	for _, n := range m.g.Market.RecentNews(30) {
		if !n.IsActive() {
			continue
		}
		if sent, ok := n.SentimentFor(s); ok {
			if sent > 0 {
				posEvents++
			} else {
				negEvents++
			}
		}
	}
	newsLabel := "neutral"
	switch {
	case netInf > 0.2:
		newsLabel = "strong tailwind"
	case netInf > 0.05:
		newsLabel = "tailwind"
	case netInf < -0.2:
		newsLabel = "strong headwind"
	case netInf < -0.05:
		newsLabel = "headwind"
	}

	// ── Composite ─────────────────────────────────────────────────
	composite := fundScore*0.30 + techScore*0.35 + newsScore*0.35
	if composite > 1.0 {
		composite = 1.0
	} else if composite < -1.0 {
		composite = -1.0
	}

	recLabel := "◆ HOLD"
	recStyle := styleNeutral
	switch {
	case composite > 0.45:
		recLabel, recStyle = "▲▲ STRONG BUY", stylePositive
	case composite > 0.15:
		recLabel, recStyle = "▲  BUY", stylePositive
	case composite < -0.45:
		recLabel, recStyle = "▼▼ STRONG SELL", styleNegative
	case composite < -0.15:
		recLabel, recStyle = "▼  SELL", styleNegative
	}

	confidence := int(math.Abs(composite) * 100)

	// Price targets (use volatility + beta for stop distance)
	stopPct := math.Max(0.03, s.Volatility*math.Max(s.Beta, 0.5)*0.5)
	targetPct := math.Max(0.02, math.Abs(composite)*0.15)
	var stopPrice, targetPrice float64
	var targetDir string
	if composite >= 0 {
		stopPrice = s.Price * (1 - stopPct)
		targetPrice = s.Price * (1 + targetPct)
		targetDir = "upside"
	} else {
		stopPrice = s.Price * (1 + stopPct)
		targetPrice = s.Price * (1 - targetPct)
		targetDir = "downside"
	}

	// ── Render ────────────────────────────────────────────────────
	bar8 := func(score float64) string {
		n := int(math.Abs(score) * 8)
		if n > 8 {
			n = 8
		}
		return colorForChange(score).Render(strings.Repeat("█", n) + strings.Repeat("░", 8-n))
	}
	scoreTag := func(score float64) string {
		return colorForChange(score).Render(market.SentimentLabel(score) + fmt.Sprintf(" %+.2f", score))
	}

	var b strings.Builder

	b.WriteString(styleHint.Render("  Analyst View") + "\n")
	b.WriteString(div + "\n\n")

	b.WriteString(styleHeader.Render("  FUNDAMENTALS") + "  " + scoreTag(fundScore) + "\n")
	b.WriteString(div + "\n")
	b.WriteString(fmt.Sprintf("  %-14s  %s  %s\n", "P/E Ratio",
		bar8(peScore), styleNeutral.Render(fmt.Sprintf("%.1f  (sector avg ~%.0f)  %s", pe, avgPE, peLabel))))
	b.WriteString(fmt.Sprintf("  %-14s  %s  %s\n", "EPS",
		bar8(epsScore), styleNeutral.Render(fmt.Sprintf("$%.2f  %s", s.EPS, epsLabel))))
	b.WriteString(fmt.Sprintf("  %-14s  %s  %s\n", "YTD Growth",
		bar8(ytdScore), styleNeutral.Render(fmt.Sprintf("%+.1f%%  %s", s.YTDGrowth, ytdLabel))))
	b.WriteString(fmt.Sprintf("  %-14s  %s  %s\n", "Dividend",
		bar8(divScore), styleNeutral.Render(divLabel)))
	b.WriteString(fmt.Sprintf("  %-14s  %s  %s\n\n", "Beta",
		bar8(-math.Min(s.Beta/2.5, 1.0)), styleNeutral.Render(fmt.Sprintf("%.2f  %s", s.Beta, betaRiskLabel(s.Beta)))))

	b.WriteString(styleHeader.Render("  TECHNICALS") + "  " + scoreTag(techScore) + "\n")
	b.WriteString(div + "\n")
	if sma20 > 0 {
		b.WriteString(fmt.Sprintf("  20-period MA   $%-10.2f  %s\n", sma20, styleNeutral.Render(ma20Label)))
	}
	if sma50 > 0 {
		b.WriteString(fmt.Sprintf("  50-period MA   $%-10.2f  %s\n", sma50, styleNeutral.Render(ma50Label)))
	}
	b.WriteString(fmt.Sprintf("  %-14s  %s  %s\n\n", "Momentum",
		bar8(momentum), styleNeutral.Render(momLabel)))

	b.WriteString(styleHeader.Render("  MARKET SIGNALS") + "  " + scoreTag(newsScore) + "\n")
	b.WriteString(div + "\n")
	b.WriteString(fmt.Sprintf("  Active bullish:  %s\n", stylePositive.Render(fmt.Sprintf("%d event(s)", posEvents))))
	b.WriteString(fmt.Sprintf("  Active bearish:  %s\n", styleNegative.Render(fmt.Sprintf("%d event(s)", negEvents))))
	b.WriteString(fmt.Sprintf("  Net influence:   %s  %s\n\n",
		colorForChange(netInf).Render(fmt.Sprintf("%+.3f", netInf)),
		styleNeutral.Render(newsLabel)))

	border := styleNeutral.Render(strings.Repeat("═", w))
	b.WriteString(border + "\n")
	b.WriteString(fmt.Sprintf("  RECOMMENDATION:   %s    %s\n\n",
		recStyle.Bold(true).Render(recLabel),
		styleNeutral.Render(fmt.Sprintf("confidence: %d%%", confidence))))
	if composite >= 0 {
		b.WriteString(fmt.Sprintf("  Entry:   $%.2f    Stop-loss: $%.2f  (%+.1f%%)\n",
			s.Price, stopPrice, (stopPrice/s.Price-1)*100))
	} else {
		b.WriteString(fmt.Sprintf("  Short:   $%.2f    Stop-loss: $%.2f  (%+.1f%%)\n",
			s.Price, stopPrice, (stopPrice/s.Price-1)*100))
	}
	b.WriteString(fmt.Sprintf("  Target:  $%.2f    %s: %+.1f%%\n",
		targetPrice, targetDir, (targetPrice/s.Price-1)*100))
	b.WriteString(border + "\n")

	return b.String()
}

func industryAvgPE(ind market.Industry) float64 {
	switch ind {
	case market.IndustryTech:
		return 35
	case market.IndustryFinance:
		return 14
	case market.IndustryEnergy:
		return 12
	case market.IndustryHealthcare:
		return 28
	case market.IndustryConsumer:
		return 22
	case market.IndustryIndustrial:
		return 18
	case market.IndustryCrypto:
		return 45
	case market.IndustryReal:
		return 20
	case market.IndustryMaterials:
		return 15
	case market.IndustryUtilities:
		return 17
	default:
		return 20
	}
}

func calcSMA(hist []market.PricePoint, n int) float64 {
	if len(hist) == 0 {
		return 0
	}
	if len(hist) < n {
		n = len(hist)
	}
	recent := hist[len(hist)-n:]
	sum := 0.0
	for _, p := range recent {
		sum += p.Price
	}
	return sum / float64(n)
}

// calcMomentum returns a normalised slope of the last n price points in [-1, 1].
// ±10% total move over n periods maps to ±1.0.
func calcMomentum(hist []market.PricePoint, n int) float64 {
	if len(hist) < 2 {
		return 0
	}
	if len(hist) < n {
		n = len(hist)
	}
	recent := hist[len(hist)-n:]
	first := recent[0].Price
	last := recent[len(recent)-1].Price
	if first == 0 {
		return 0
	}
	normalized := (last - first) / first / 0.10
	if normalized > 1.0 {
		return 1.0
	} else if normalized < -1.0 {
		return -1.0
	}
	return normalized
}

func betaRiskLabel(beta float64) string {
	switch {
	case beta > 2.0:
		return "very high risk"
	case beta > 1.5:
		return "high risk"
	case beta > 1.0:
		return "above avg risk"
	case beta > 0.7:
		return "moderate risk"
	default:
		return "low risk"
	}
}

func (m Model) viewPortfolio() string {
	w := m.width
	if w == 0 {
		w = 100
	}

	portfolioVal := m.g.PortfolioValue()
	totalPnL := portfolioVal - m.g.StartingCash

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

	// ── puts section ──────────────────────────────────────────────────────
	var putsSection strings.Builder
	if len(m.g.Puts) > 0 {
		putColHeader := padR("#", 5) + padR("SYMBOL", 8) + padR("STRIKE", 10) +
			padR("CONTR", 7) + padR("EXPIRES", 11) + padR("CURR VALUE", 12) +
			padR("COST", 12) + "P&L"
		putsSection.WriteString("\n" + lipgloss.NewStyle().Bold(true).Foreground(colorGray).Render(" PUTS") + "\n")
		putsSection.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colorGray).Render(" "+putColHeader) + "\n")
		for pi, put := range m.g.Puts {
			s := m.g.Market.GetStock(put.Symbol)
			currVal := 0.0
			if s != nil {
				iv := market.ImpliedVol(s.Volatility)
				currVal = put.CurrentValue(s.Price, iv)
			}
			pnl := currVal - put.Premium
			tl := put.TicksLeft()
			var expiresStr string
			switch {
			case tl <= 0:
				expiresStr = "expired"
			case tl < 30:
				expiresStr = fmt.Sprintf("%ds", tl*2)
			default:
				expiresStr = fmt.Sprintf("%dm%ds", (tl*2)/60, (tl*2)%60)
			}
			row := padR(fmt.Sprintf("#%d", put.ID), 5) +
				padR(put.Symbol, 8) +
				padR(fmt.Sprintf("$%.2f", put.Strike), 10) +
				padR(fmt.Sprintf("%d", put.Contracts), 7) +
				padR(expiresStr, 11) +
				padR(fmt.Sprintf("$%.2f", currVal), 12) +
				padR(fmt.Sprintf("$%.2f", put.Premium), 12) +
				colorForChange(pnl).Render(fmt.Sprintf("%s$%.2f", signStr(pnl), math.Abs(pnl)))

			rowIdx := len(syms) + pi
			if rowIdx == m.cursor {
				putsSection.WriteString(styleSelected.Render(row) + "\n")
			} else {
				putsSection.WriteString(lipgloss.NewStyle().Foreground(colorWhite).Render(row) + "\n")
			}
		}
	}

	div := styleNeutral.Render(strings.Repeat("─", w))
	keys := styleHint.Render(" ↑↓ navigate  enter=stock detail  x=sell put  esc=back")

	statusLine := ""
	if m.okMsg != "" {
		statusLine = "\n" + styleOk.Render("✓ "+m.okMsg)
	} else if m.errMsg != "" {
		statusLine = "\n" + styleError.Render("⚠ "+m.errMsg)
	}

	return header + "\n" + div + "\n" +
		lipgloss.NewStyle().Bold(true).Foreground(colorGray).Render(" "+colHeader) + "\n" +
		div + "\n" + rows.String() + putsSection.String() + div + statusLine + "\n" + keys
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

	if m.tradeMode == tradePut {
		return m.viewTradePut(s)
	}

	isLimit := m.tradeMode == tradeLimitBuy || m.tradeMode == tradeLimitSell
	pct := s.ChangePct()
	cs := colorForChange(pct)

	title := styleTitle.Render(m.tradeMode.label()) + "  " + styleWhiteStr(m.stockSymbol) + "  " + styleWhiteStr(s.Name)
	priceInfo := styleWhiteStr("Current: ") +
		lipgloss.NewStyle().Bold(true).Foreground(colorWhite).Render(fmt.Sprintf("$%.2f  ", s.Price)) +
		cs.Render(fmt.Sprintf("(%s%.2f%%)", signStr(pct), pct))

	var inputSection string
	if isLimit {
		sharesBox := styleInputBlur.Render(m.inputShares.View())
		priceBox := styleInputBlur.Render(m.inputPrice.View())
		if m.inputFocus == 0 {
			sharesBox = styleInput.Render(m.inputShares.View())
		} else {
			priceBox = styleInput.Render(m.inputPrice.View())
		}
		sharesRow := lipgloss.JoinHorizontal(lipgloss.Center, styleNeutral.Render("Shares:  "), sharesBox)
		priceRow := lipgloss.JoinHorizontal(lipgloss.Center, styleNeutral.Render("Limit $: "), priceBox)
		inputSection = sharesRow + "\n" + priceRow + "\n" + styleHint.Render("  tab to switch fields")
	} else {
		inputSection = lipgloss.JoinHorizontal(lipgloss.Center,
			styleNeutral.Render("Shares:  "),
			styleInput.Render(m.inputShares.View()))
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
		if sent, ok := n.SentimentFor(s); ok {
			sentStyle := colorForChange(sent)
			relevantNews += "\n" + sentStyle.Render(market.SentimentLabel(sent)) + " " + styleNeutral.Render(truncate(n.Headline, 40))
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

var putMoneynessLabels = [5]string{"Deep ITM +20%", "ITM +10%", "ATM", "OTM -10%", "Deep OTM -20%"}

func (m Model) viewTradePut(s *market.Stock) string {
	pct := s.ChangePct()
	cs := colorForChange(pct)
	title := styleTitle.Render("Buy PUT") + "  " + styleWhiteStr(s.Symbol) + "  " + styleWhiteStr(s.Name)
	priceInfo := styleWhiteStr("Current: ") +
		lipgloss.NewStyle().Bold(true).Foreground(colorWhite).Render(fmt.Sprintf("$%.2f  ", s.Price)) +
		cs.Render(fmt.Sprintf("(%s%.2f%%)", signStr(pct), pct))

	iv := market.ImpliedVol(s.Volatility)
	strikes := game.PutStrikes(s.Price)
	expiries := putExpiries

	// Strike table
	strikeHeader := styleNeutral.Render(
		"  " + padR("STRIKE", 10) + padR("MONEYNESS", 15) + padR("PREM/CONTRACT", 16) + "BREAK-EVEN")
	var strikeRows strings.Builder
	for i, strike := range strikes {
		pricePerShare := market.PutPrice(s.Price, strike, iv, int(expiries[m.putExpiryCursor]))
		premiumPerContract := pricePerShare * 100
		breakeven := strike - pricePerShare
		row := padR(fmt.Sprintf("$%.2f", strike), 10) +
			padR(putMoneynessLabels[i], 15) +
			padR(fmt.Sprintf("$%.2f", premiumPerContract), 16) +
			fmt.Sprintf("$%.2f", breakeven)
		if i == m.putStrikeCursor {
			strikeRows.WriteString(styleSelected.Render("▶ "+row) + "\n")
		} else {
			strikeRows.WriteString(styleNeutral.Render("  "+row) + "\n")
		}
	}

	// Expiry row
	var expiryParts []string
	expiryLabels := []string{"Short (~1m)", "Medium (~3m)", "Long (~5m)"}
	for i, lbl := range expiryLabels {
		if i == m.putExpiryCursor {
			expiryParts = append(expiryParts, styleSelected.Render(" "+lbl+" "))
		} else {
			expiryParts = append(expiryParts, styleNeutral.Render(" "+lbl+" "))
		}
	}
	expiryRow := styleNeutral.Render("Expiry:  ") + strings.Join(expiryParts, styleNeutral.Render("·"))

	// Cost preview
	selectedStrike := strikes[m.putStrikeCursor]
	selectedExpiry := expiries[m.putExpiryCursor]
	pricePerShare := market.PutPrice(s.Price, selectedStrike, iv, int(selectedExpiry))
	contractsStr := strings.TrimSpace(m.inputShares.Value())
	contracts, _ := strconv.Atoi(contractsStr)
	var costLine string
	if contracts > 0 {
		total := pricePerShare * 100 * float64(contracts)
		costLine = styleNeutral.Render("Total cost: ") + styleAccentStr(fmt.Sprintf("$%.2f", total))
		if total > m.g.Cash {
			costLine += "  " + styleError.Render("insufficient funds")
		}
	}

	contractInput := lipgloss.JoinHorizontal(lipgloss.Center,
		styleNeutral.Render("Contracts: "),
		styleInput.Render(m.inputShares.View()))

	cashLine := styleNeutral.Render("Cash: ") + stylePositive.Render(fmt.Sprintf("$%s", commaf(m.g.Cash)))

	feedback := ""
	if m.errMsg != "" {
		feedback = "\n" + styleError.Render("⚠ "+m.errMsg)
	}

	keys := styleHint.Render(" ↑↓=strike  ◀▶=expiry  enter=buy  esc=cancel")

	inner := title + "\n" + priceInfo + "\n\n" +
		strikeHeader + "\n" + strikeRows.String() + "\n" +
		expiryRow + "\n\n" +
		contractInput + "\n" +
		cashLine + "    " + costLine + feedback + "\n\n" +
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

func shortIndustry(ind market.Industry) string {
	switch ind {
	case market.IndustryTech:
		return "Tech"
	case market.IndustryFinance:
		return "Finance"
	case market.IndustryEnergy:
		return "Energy"
	case market.IndustryHealthcare:
		return "Health"
	case market.IndustryConsumer:
		return "Consumer"
	case market.IndustryIndustrial:
		return "Indust"
	case market.IndustryCrypto:
		return "Crypto"
	case market.IndustryReal:
		return "R.Estate"
	case market.IndustryMaterials:
		return "Matls"
	case market.IndustryUtilities:
		return "Utils"
	}
	return string(ind)
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
