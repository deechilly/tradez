package market_test

import (
	"math"
	"math/rand"
	"testing"
	"time"
	"tradez/internal/market"
)

// ── Stock ────────────────────────────────────────────────────────────────────

func TestStockChange(t *testing.T) {
	s := &market.Stock{Price: 110, PrevClose: 100}
	if got := s.Change(); got != 10 {
		t.Errorf("Change() = %f, want 10", got)
	}
}

func TestStockChangePct(t *testing.T) {
	s := &market.Stock{Price: 110, PrevClose: 100}
	if got := s.ChangePct(); math.Abs(got-10) > 0.001 {
		t.Errorf("ChangePct() = %f, want 10%%", got)
	}
}

func TestStockChangePctZeroPrevClose(t *testing.T) {
	s := &market.Stock{Price: 50, PrevClose: 0}
	if got := s.ChangePct(); got != 0 {
		t.Errorf("ChangePct() with zero PrevClose = %f, want 0", got)
	}
}

// ── Generator ────────────────────────────────────────────────────────────────

func TestGenerateMarketStockCount(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	stocks := market.GenerateMarket(rng)
	if len(stocks) < 70 || len(stocks) > 90 {
		t.Errorf("stock count = %d, want 70–90", len(stocks))
	}
}

func TestGenerateMarketAllIndustries(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	stocks := market.GenerateMarket(rng)

	industries := map[market.Industry]bool{}
	for _, s := range stocks {
		industries[s.Industry] = true
	}

	all := []market.Industry{
		market.IndustryTech, market.IndustryFinance, market.IndustryEnergy,
		market.IndustryHealthcare, market.IndustryConsumer, market.IndustryIndustrial,
		market.IndustryCrypto, market.IndustryReal, market.IndustryMaterials,
		market.IndustryUtilities,
	}
	for _, ind := range all {
		if !industries[ind] {
			t.Errorf("industry %q not represented in generated market", ind)
		}
	}
}

func TestGenerateMarketUniqueSymbols(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	stocks := market.GenerateMarket(rng)

	seen := map[string]bool{}
	for _, s := range stocks {
		if seen[s.Symbol] {
			t.Errorf("duplicate symbol %q", s.Symbol)
		}
		seen[s.Symbol] = true
	}
}

func TestGenerateMarketSymbolLength(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	stocks := market.GenerateMarket(rng)
	for _, s := range stocks {
		if len(s.Symbol) != 4 {
			t.Errorf("symbol %q has length %d, want 4", s.Symbol, len(s.Symbol))
		}
	}
}

func TestGenerateMarketPricesPositive(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for _, s := range market.GenerateMarket(rng) {
		if s.Price <= 0 {
			t.Errorf("%s: price = %f, want > 0", s.Symbol, s.Price)
		}
		if s.Beta <= 0 {
			t.Errorf("%s: beta = %f, want > 0", s.Symbol, s.Beta)
		}
		if s.Volatility <= 0 {
			t.Errorf("%s: volatility = %f, want > 0", s.Symbol, s.Volatility)
		}
	}
}

func TestGenerateMarketHistoryPopulated(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for _, s := range market.GenerateMarket(rng) {
		if len(s.History) == 0 {
			t.Errorf("%s: history is empty", s.Symbol)
		}
	}
}

func TestGenerateMarketDifferentSeedsProduceDifferentMarkets(t *testing.T) {
	// Two different seeds should not produce identical symbol sets.
	a := market.GenerateMarket(rand.New(rand.NewSource(1)))
	b := market.GenerateMarket(rand.New(rand.NewSource(2)))
	setA := map[string]bool{}
	for _, s := range a {
		setA[s.Symbol] = true
	}
	overlap := 0
	for _, s := range b {
		if setA[s.Symbol] {
			overlap++
		}
	}
	// Some symbols may coincidentally match, but a full overlap would be
	// vanishingly improbable with distinct seeds and 70–90 stocks per market.
	if overlap == len(a) && overlap == len(b) {
		t.Error("different seeds produced identical symbol sets")
	}
}

func TestFormatMarketCap(t *testing.T) {
	cases := []struct {
		cap  float64
		want string
	}{
		{2.5e12, "$2.50T"},
		{1.2e9, "$1.20B"},
		{500e6, "$500.00M"},
		{999, "$999"},
	}
	for _, c := range cases {
		if got := market.FormatMarketCap(c.cap); got != c.want {
			t.Errorf("FormatMarketCap(%g) = %q, want %q", c.cap, got, c.want)
		}
	}
}

// ── Market ───────────────────────────────────────────────────────────────────

func TestNewMarketHasStocks(t *testing.T) {
	m := market.New(42)
	if len(m.Snapshot()) == 0 {
		t.Fatal("new market has no stocks")
	}
}

func TestGetStockFound(t *testing.T) {
	m := market.New(42)
	sym := m.Snapshot()[0].Symbol
	s := m.GetStock(sym)
	if s == nil {
		t.Fatalf("GetStock(%q) returned nil", sym)
	}
	if s.Symbol != sym {
		t.Errorf("GetStock returned wrong stock: %q", s.Symbol)
	}
}

func TestGetStockNotFound(t *testing.T) {
	m := market.New(42)
	if s := m.GetStock("ZZZZ"); s != nil {
		t.Errorf("GetStock(unknown) = %+v, want nil", s)
	}
}

func TestTickAppendsHistory(t *testing.T) {
	m := market.New(42)
	s := m.Snapshot()[0]
	before := len(s.History)
	m.Tick()
	if len(s.History) != before+1 {
		t.Errorf("history length: before=%d after=%d, want before+1", before, len(s.History))
	}
}

func TestTickPriceFloor(t *testing.T) {
	m := market.New(42)
	s := m.GetStock(m.Snapshot()[0].Symbol)
	s.Price = 0.001 // force below the $0.01 floor
	m.Tick()
	if s.Price < 0.01 {
		t.Errorf("price = %f after tick, want >= 0.01", s.Price)
	}
}

func TestSnapshotLength(t *testing.T) {
	m := market.New(42)
	snap := m.Snapshot()
	if len(snap) == 0 {
		t.Fatal("snapshot is empty")
	}
	// A second snapshot reflects the same count.
	if len(m.Snapshot()) != len(snap) {
		t.Error("successive snapshots have different lengths")
	}
}

func TestNextNewsInIsNonNegative(t *testing.T) {
	m := market.New(42)
	if d := m.NextNewsIn(); d < 0 {
		t.Errorf("NextNewsIn() = %v, want >= 0", d)
	}
}

// ── News & Influence ─────────────────────────────────────────────────────────

func TestSentimentLabel(t *testing.T) {
	cases := []struct {
		s    float64
		want string
	}{
		{0.5, "▲▲"},
		{0.2, "▲"},
		{0.0, "◆"},
		{-0.2, "▼"},
		{-0.5, "▼▼"},
	}
	for _, c := range cases {
		if got := market.SentimentLabel(c.s); got != c.want {
			t.Errorf("SentimentLabel(%v) = %q, want %q", c.s, got, c.want)
		}
	}
}

func TestNewsEventSentimentForIndustry(t *testing.T) {
	event := &market.NewsEvent{
		Effects: []market.InfluenceEffect{
			{Industries: []market.Industry{market.IndustryTech}, Sentiment: 0.5},
		},
	}
	s := &market.Stock{Industry: market.IndustryTech}
	sent, ok := event.SentimentFor(s)
	if !ok {
		t.Fatal("SentimentFor tech stock in tech event = false, want true")
	}
	if sent != 0.5 {
		t.Errorf("sentiment = %f, want 0.5", sent)
	}
}

func TestNewsEventSentimentForSymbol(t *testing.T) {
	event := &market.NewsEvent{
		Effects: []market.InfluenceEffect{
			{Symbols: []string{"ABCD"}, Sentiment: -0.8},
		},
	}
	s := &market.Stock{Symbol: "ABCD", Industry: market.IndustryFinance}
	sent, ok := event.SentimentFor(s)
	if !ok {
		t.Fatal("SentimentFor targeted symbol = false, want true")
	}
	if sent != -0.8 {
		t.Errorf("sentiment = %f, want -0.8", sent)
	}
}

func TestNewsEventSentimentForUnrelated(t *testing.T) {
	event := &market.NewsEvent{
		Effects: []market.InfluenceEffect{
			{Industries: []market.Industry{market.IndustryTech}, Sentiment: 0.5},
		},
	}
	s := &market.Stock{Symbol: "XYZW", Industry: market.IndustryUtilities}
	_, ok := event.SentimentFor(s)
	if ok {
		t.Error("SentimentFor unrelated stock = true, want false")
	}
}

func TestMarketInfluenceTargetsIndustry(t *testing.T) {
	inf := &market.MarketInfluence{
		Effects: []market.InfluenceEffect{
			{Industries: []market.Industry{market.IndustryEnergy}, Sentiment: 0.4},
		},
		ExpiresAt: time.Now().Add(time.Minute),
	}
	s := &market.Stock{Industry: market.IndustryEnergy}
	if !inf.Targets(s) {
		t.Error("Targets energy stock in energy influence = false, want true")
	}
	other := &market.Stock{Industry: market.IndustryFinance}
	if inf.Targets(other) {
		t.Error("Targets finance stock in energy influence = true, want false")
	}
}

func TestMarketInfluenceStrengthExpired(t *testing.T) {
	inf := &market.MarketInfluence{
		Effects: []market.InfluenceEffect{
			{Industries: []market.Industry{market.IndustryTech}, Sentiment: 1.0},
		},
		Magnitude: 1.0,
		CreatedAt: time.Now().Add(-2 * time.Minute),
		ExpiresAt: time.Now().Add(-time.Second), // already expired
	}
	s := &market.Stock{Industry: market.IndustryTech}
	if strength := inf.InfluenceStrengthFor(s); strength != 0 {
		t.Errorf("expired influence strength = %f, want 0", strength)
	}
}

func TestMarketInfluenceStrengthActiveNonZero(t *testing.T) {
	inf := &market.MarketInfluence{
		Effects: []market.InfluenceEffect{
			{Industries: []market.Industry{market.IndustryTech}, Sentiment: 0.5},
		},
		Magnitude: 0.8,
		CreatedAt: time.Now().Add(-time.Second),
		ExpiresAt: time.Now().Add(time.Minute),
	}
	s := &market.Stock{Industry: market.IndustryTech}
	if strength := inf.InfluenceStrengthFor(s); strength == 0 {
		t.Error("active influence strength = 0, want non-zero for targeted stock")
	}
}

func TestGenerateNewsHasHeadlineAndEffects(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	stocks := market.GenerateMarket(rng)
	event, influence := market.GenerateNews(rng, stocks, 1)
	if event.Headline == "" {
		t.Error("generated news event has empty headline")
	}
	if len(event.Effects) == 0 {
		t.Error("generated news event has no effects")
	}
	if influence.Magnitude <= 0 {
		t.Errorf("influence magnitude = %f, want > 0", influence.Magnitude)
	}
	if !event.ExpiresAt.After(event.CreatedAt) {
		t.Error("event ExpiresAt is not after CreatedAt")
	}
}

func TestGenerateNewsEventAndInfluenceShareID(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	stocks := market.GenerateMarket(rng)
	event, influence := market.GenerateNews(rng, stocks, 42)
	if event.ID != 42 {
		t.Errorf("event.ID = %d, want 42", event.ID)
	}
	if influence.NewsID != 42 {
		t.Errorf("influence.NewsID = %d, want 42", influence.NewsID)
	}
}

// ── Options (Black-Scholes) ──────────────────────────────────────────────────

func TestImpliedVol(t *testing.T) {
	// ImpliedVol(v) = v * sqrt(1000); verify monotonicity and scale
	low := market.ImpliedVol(0.01)
	high := market.ImpliedVol(0.02)
	if high <= low {
		t.Errorf("ImpliedVol not monotonic: ImpliedVol(0.02)=%f <= ImpliedVol(0.01)=%f", high, low)
	}
	expected := 0.01 * math.Sqrt(1000)
	if math.Abs(low-expected) > 1e-10 {
		t.Errorf("ImpliedVol(0.01) = %f, want %f", low, expected)
	}
}

func TestPutIntrinsic(t *testing.T) {
	cases := []struct {
		spot, strike, want float64
	}{
		{90, 100, 10},  // ITM
		{100, 100, 0},  // ATM
		{110, 100, 0},  // OTM
	}
	for _, c := range cases {
		if got := market.PutIntrinsic(c.spot, c.strike); got != c.want {
			t.Errorf("PutIntrinsic(spot=%v, strike=%v) = %f, want %f", c.spot, c.strike, got, c.want)
		}
	}
}

func TestPutPriceAtExpiryEqualsIntrinsic(t *testing.T) {
	spot, strike := 90.0, 100.0
	iv := market.ImpliedVol(0.015)
	priceAtExpiry := market.PutPrice(spot, strike, iv, 0)
	intrinsic := market.PutIntrinsic(spot, strike)
	if math.Abs(priceAtExpiry-intrinsic) > 0.001 {
		t.Errorf("PutPrice at expiry = %f, intrinsic = %f, want equal", priceAtExpiry, intrinsic)
	}
}

func TestPutPriceITMExceedsIntrinsic(t *testing.T) {
	spot, strike := 90.0, 100.0
	iv := market.ImpliedVol(0.015)
	price := market.PutPrice(spot, strike, iv, 30)
	intrinsic := market.PutIntrinsic(spot, strike)
	if price <= intrinsic {
		t.Errorf("ITM put price %f <= intrinsic %f (expected time value > 0)", price, intrinsic)
	}
}

func TestPutPriceOTMCheaperThanITM(t *testing.T) {
	iv := market.ImpliedVol(0.015)
	otm := market.PutPrice(110, 100, iv, 30) // OTM: stock above strike
	itm := market.PutPrice(90, 100, iv, 30)  // ITM: stock below strike
	if otm >= itm {
		t.Errorf("OTM price %f >= ITM price %f, expected OTM < ITM", otm, itm)
	}
}

func TestPutPriceLongerExpiryMoreExpensive(t *testing.T) {
	spot, strike := 100.0, 100.0 // ATM
	iv := market.ImpliedVol(0.015)
	short := market.PutPrice(spot, strike, iv, 30)
	long := market.PutPrice(spot, strike, iv, 150)
	if long <= short {
		t.Errorf("longer expiry put price %f <= shorter %f, expected more time value", long, short)
	}
}
