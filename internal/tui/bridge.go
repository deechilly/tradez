package tui

import (
	"encoding/json"
	"fmt"
	"strings"
	"tradez/internal/analysis"
	"tradez/internal/game"
	"tradez/internal/livebridge"
	"tradez/internal/snapshot"
)

func (m *Model) handleBridgeRequest(req livebridge.RequestMsg) {
	if req.Reply == nil {
		return
	}
	req.Reply <- m.bridgeResponse(req)
}

func (m *Model) bridgeResponse(req livebridge.RequestMsg) livebridge.Response {
	if m.g == nil {
		return livebridge.Response{OK: false, Error: "no active game; load or start a save slot in the TUI first"}
	}

	switch req.Operation {
	case livebridge.OpGameState:
		return livebridge.Response{OK: true, Data: snapshot.Game(m.g, m.activeSlot)}
	case livebridge.OpListStocks:
		return livebridge.Response{OK: true, Data: snapshot.Stocks(m.g, false)}
	case livebridge.OpGetStock:
		input, err := decodeBridgePayload[livebridge.StockRequest](req.Payload)
		if err != nil {
			return bridgeError(err)
		}
		stock, err := snapshot.Stock(m.g, strings.ToUpper(input.Symbol))
		if err != nil {
			return bridgeError(err)
		}
		return livebridge.Response{OK: true, Data: stock}
	case livebridge.OpGetNews:
		return livebridge.Response{OK: true, Data: snapshot.News(m.g)}
	case livebridge.OpPortfolio:
		return livebridge.Response{OK: true, Data: snapshot.PortfolioSnapshot(m.g)}
	case livebridge.OpOrders:
		return livebridge.Response{OK: true, Data: snapshot.OrdersSnapshot(m.g)}
	case livebridge.OpInfluences:
		return livebridge.Response{OK: true, Data: snapshot.Influences(m.g)}
	case livebridge.OpPlaceOrder:
		input, err := decodeBridgePayload[livebridge.PlaceOrderRequest](req.Payload)
		if err != nil {
			return bridgeError(err)
		}
		result, err := m.executeBridgeTrade(input)
		if err != nil {
			return bridgeError(err)
		}
		return livebridge.Response{OK: true, Data: result}
	case livebridge.OpCancel:
		input, err := decodeBridgePayload[livebridge.CancelOrderRequest](req.Payload)
		if err != nil {
			return bridgeError(err)
		}
		if input.ID <= 0 {
			return bridgeError(fmt.Errorf("order id must be positive"))
		}
		if !m.g.CancelOrder(input.ID) {
			return bridgeError(fmt.Errorf("pending order #%d not found", input.ID))
		}
		m.okMsg = fmt.Sprintf("MCP cancelled order #%d", input.ID)
		m.errMsg = ""
		m.msgTimer = 3
		return livebridge.Response{OK: true, Data: snapshot.OrdersSnapshot(m.g)}
	case livebridge.OpSave:
		if m.activeSlot <= 0 {
			return bridgeError(fmt.Errorf("no active save slot"))
		}
		if m.g.StartingCash <= 0 {
			return bridgeError(fmt.Errorf("cannot save before starting capital is selected"))
		}
		if err := game.Save(m.g, m.activeSlot); err != nil {
			return bridgeError(fmt.Errorf("save failed: %w", err))
		}
		m.okMsg = fmt.Sprintf("Saved to Game %d via MCP", m.activeSlot)
		m.errMsg = ""
		m.msgTimer = 3
		return livebridge.Response{OK: true, Data: snapshot.Game(m.g, m.activeSlot)}
	default:
		return bridgeError(fmt.Errorf("unknown bridge operation %q", req.Operation))
	}
}

func (m *Model) executeBridgeTrade(input livebridge.PlaceOrderRequest) (snapshot.TradeResult, error) {
	symbol := strings.ToUpper(strings.TrimSpace(input.Symbol))
	if symbol == "" {
		return snapshot.TradeResult{}, fmt.Errorf("symbol is required")
	}
	if input.Shares <= 0 {
		return snapshot.TradeResult{}, fmt.Errorf("shares must be positive")
	}
	mode, isLimit, err := bridgeTradeMode(input.Type)
	if err != nil {
		return snapshot.TradeResult{}, err
	}
	if isLimit && input.LimitPrice <= 0 {
		return snapshot.TradeResult{}, fmt.Errorf("limit_price must be positive for %s", input.Type)
	}

	s := m.g.Market.GetStock(symbol)
	if s == nil {
		return snapshot.TradeResult{}, fmt.Errorf("unknown symbol %s", symbol)
	}

	prePV := m.g.PortfolioValue()
	preCash := m.g.Cash
	var preAvgCost, preShortAvg float64
	var preShares, preShortShares int
	if pos := m.g.Positions[symbol]; pos != nil {
		preAvgCost = pos.AvgCost
		preShortAvg = pos.ShortAvg
		preShares = pos.Shares
		preShortShares = pos.ShortShares
	}

	m.lastAnalysisScore = analysis.Analyze(m.g.Market, s).CompositeScore
	beforeOrders := len(m.g.Orders)

	var tradeErr error
	switch mode {
	case tradeBuy:
		tradeErr = m.g.BuyMarket(symbol, input.Shares)
	case tradeSell:
		tradeErr = m.g.SellMarket(symbol, input.Shares)
	case tradeLimitBuy:
		tradeErr = m.g.PlaceLimitBuy(symbol, input.Shares, input.LimitPrice)
	case tradeLimitSell:
		tradeErr = m.g.PlaceLimitSell(symbol, input.Shares, input.LimitPrice)
	case tradeShort:
		tradeErr = m.g.ShortSell(symbol, input.Shares)
	case tradeCover:
		tradeErr = m.g.CoverShort(symbol, input.Shares)
	}
	if tradeErr != nil {
		m.errMsg = tradeErr.Error()
		m.okMsg = ""
		m.msgTimer = 5
		return snapshot.TradeResult{}, tradeErr
	}

	if !isLimit {
		m.checkTradeAchievements(prePV, preCash, preAvgCost, preShortAvg,
			preShares, preShortShares, input.Shares, mode, s)
	}

	m.okMsg = "MCP order placed"
	m.errMsg = ""
	m.msgTimer = 3

	var orderSnap *snapshot.Order
	if len(m.g.Orders) > beforeOrders {
		o := snapshot.OrderSnapshot(m.g.Orders[len(m.g.Orders)-1])
		orderSnap = &o
	}
	stockSnap, _ := snapshot.Stock(m.g, symbol)
	return snapshot.TradeResult{
		Message:   m.okMsg,
		Order:     orderSnap,
		GameState: snapshot.Game(m.g, m.activeSlot),
		Portfolio: snapshot.PortfolioSnapshot(m.g),
		Stock:     stockSnap,
	}, nil
}

func bridgeTradeMode(raw string) (tradeMode, bool, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "market_buy":
		return tradeBuy, false, nil
	case "market_sell":
		return tradeSell, false, nil
	case "limit_buy":
		return tradeLimitBuy, true, nil
	case "limit_sell":
		return tradeLimitSell, true, nil
	case "short_sell":
		return tradeShort, false, nil
	case "cover_short":
		return tradeCover, false, nil
	default:
		return tradeBuy, false, fmt.Errorf("unsupported order type %q", raw)
	}
}

func decodeBridgePayload[T any](raw json.RawMessage) (T, error) {
	var out T
	if len(raw) == 0 || string(raw) == "null" {
		raw = []byte("{}")
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, fmt.Errorf("invalid payload: %w", err)
	}
	return out, nil
}

func bridgeError(err error) livebridge.Response {
	return livebridge.Response{OK: false, Error: err.Error()}
}
