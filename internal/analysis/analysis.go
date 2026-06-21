package analysis

import (
	"math"
	"tradez/internal/market"
)

type Component struct {
	Score float64 `json:"score"`
	Label string  `json:"label"`
}

type Result struct {
	FundamentalScore float64 `json:"fundamental_score"`
	TechnicalScore   float64 `json:"technical_score"`
	NewsScore        float64 `json:"news_score"`
	CompositeScore   float64 `json:"composite_score"`
	Recommendation   string  `json:"recommendation"`
	Confidence       int     `json:"confidence"`

	PERatio       Component `json:"pe_ratio"`
	EPS           Component `json:"eps"`
	YTDGrowth     Component `json:"ytd_growth"`
	Dividend      Component `json:"dividend"`
	BetaRisk      Component `json:"beta_risk"`
	MA20          Component `json:"ma20"`
	MA50          Component `json:"ma50"`
	Momentum      Component `json:"momentum"`
	ActiveBullish int       `json:"active_bullish"`
	ActiveBearish int       `json:"active_bearish"`
	NetInfluence  float64   `json:"net_influence"`
	NewsLabel     string    `json:"news_label"`

	EntryPrice  float64 `json:"entry_price,omitempty"`
	ShortPrice  float64 `json:"short_price,omitempty"`
	StopPrice   float64 `json:"stop_price"`
	TargetPrice float64 `json:"target_price"`
	TargetDir   string  `json:"target_dir"`
}

func Analyze(m *market.Market, s *market.Stock) Result {
	if s == nil {
		return Result{Recommendation: "HOLD"}
	}

	pe := s.PERatio
	avgPE := IndustryAvgPE(s.Industry)
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
	fundScore := peScore * 0.40

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
		divLabel = "yield"
	}
	fundScore += divScore * 0.15
	fundScore = clamp(fundScore, -1, 1)

	hist := s.History
	sma20 := SMA(hist, 20)
	sma50 := SMA(hist, 50)
	momentum := Momentum(hist, 15)

	techScore := 0.0
	ma20Label := "no data"
	if sma20 > 0 {
		if s.Price > sma20 {
			techScore += 0.30
			ma20Label = "above"
		} else {
			techScore -= 0.30
			ma20Label = "below"
		}
	}
	ma50Label := "no data"
	if sma50 > 0 {
		if s.Price > sma50 {
			techScore += 0.25
			ma50Label = "above"
		} else {
			techScore -= 0.25
			ma50Label = "below"
		}
	}
	techScore += momentum * 0.45
	techScore = clamp(techScore, -1, 1)

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

	netInf := 0.0
	posEvents, negEvents := 0, 0
	if m != nil {
		netInf = m.StockInfluenceStrength(s.Symbol)
		for _, n := range m.RecentNews(30) {
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
	}
	newsScore := clamp(netInf*4, -1, 1)
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

	composite := clamp(fundScore*0.30+techScore*0.35+newsScore*0.35, -1, 1)
	rec := Recommendation(composite)
	confidence := int(math.Abs(composite) * 100)

	stopPct := math.Max(0.03, s.Volatility*math.Max(s.Beta, 0.5)*0.5)
	targetPct := math.Max(0.02, math.Abs(composite)*0.15)
	stopPrice := s.Price * (1 - stopPct)
	targetPrice := s.Price * (1 + targetPct)
	targetDir := "upside"
	entryPrice := s.Price
	shortPrice := 0.0
	if composite < 0 {
		stopPrice = s.Price * (1 + stopPct)
		targetPrice = s.Price * (1 - targetPct)
		targetDir = "downside"
		entryPrice = 0
		shortPrice = s.Price
	}

	return Result{
		FundamentalScore: fundScore,
		TechnicalScore:   techScore,
		NewsScore:        newsScore,
		CompositeScore:   composite,
		Recommendation:   rec,
		Confidence:       confidence,
		PERatio:          Component{Score: peScore, Label: peLabel},
		EPS:              Component{Score: epsScore, Label: epsLabel},
		YTDGrowth:        Component{Score: ytdScore, Label: ytdLabel},
		Dividend:         Component{Score: divScore, Label: divLabel},
		BetaRisk:         Component{Score: -math.Min(s.Beta/2.5, 1.0), Label: BetaRiskLabel(s.Beta)},
		MA20:             Component{Score: compareMA(s.Price, sma20), Label: ma20Label},
		MA50:             Component{Score: compareMA(s.Price, sma50), Label: ma50Label},
		Momentum:         Component{Score: momentum, Label: momLabel},
		ActiveBullish:    posEvents,
		ActiveBearish:    negEvents,
		NetInfluence:     netInf,
		NewsLabel:        newsLabel,
		EntryPrice:       entryPrice,
		ShortPrice:       shortPrice,
		StopPrice:        stopPrice,
		TargetPrice:      targetPrice,
		TargetDir:        targetDir,
	}
}

func Recommendation(score float64) string {
	switch {
	case score > 0.45:
		return "STRONG_BUY"
	case score > 0.15:
		return "BUY"
	case score < -0.45:
		return "STRONG_SELL"
	case score < -0.15:
		return "SELL"
	default:
		return "HOLD"
	}
}

func IndustryAvgPE(ind market.Industry) float64 {
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

func SMA(hist []market.PricePoint, n int) float64 {
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

func Momentum(hist []market.PricePoint, n int) float64 {
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
	return clamp((last-first)/first/0.10, -1, 1)
}

func BetaRiskLabel(beta float64) string {
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

func compareMA(price, ma float64) float64 {
	if ma == 0 {
		return 0
	}
	if price > ma {
		return 1
	}
	if price < ma {
		return -1
	}
	return 0
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
