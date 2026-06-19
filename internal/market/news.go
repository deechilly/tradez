package market

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

type NewsCategory string

const (
	NewsCatConflict   NewsCategory = "CONFLICT"
	NewsCatDisaster   NewsCategory = "DISASTER"
	NewsCatTech       NewsCategory = "TECH"
	NewsCatMA         NewsCategory = "M&A"
	NewsCatRegulation NewsCategory = "REGULATION"
	NewsCatEconomy    NewsCategory = "ECONOMY"
	NewsCatSpace      NewsCategory = "SPACE"
	NewsCatHealth     NewsCategory = "HEALTH"
	NewsCatEnergy     NewsCategory = "ENERGY"
	NewsCatCrypto     NewsCategory = "CRYPTO"
	NewsCatConsumer   NewsCategory = "CONSUMER"
	NewsCatEarnings   NewsCategory = "EARNINGS"
)

type NewsEvent struct {
	ID                 int
	Headline           string
	Category           NewsCategory
	AffectedIndustries []Industry
	AffectedSymbols    []string
	Sentiment          float64 // -1.0 (very bad) to +1.0 (very good)
	Magnitude          float64 // 0.1–1.0 strength of influence
	CreatedAt          time.Time
	ExpiresAt          time.Time
}

func (n *NewsEvent) IsActive() bool {
	return time.Now().Before(n.ExpiresAt)
}

func (n *NewsEvent) TimeAgo() string {
	d := time.Since(n.CreatedAt)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	default:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
}

func (n *NewsEvent) SentimentLabel() string {
	switch {
	case n.Sentiment > 0.4:
		return "▲▲"
	case n.Sentiment > 0.1:
		return "▲"
	case n.Sentiment < -0.4:
		return "▼▼"
	case n.Sentiment < -0.1:
		return "▼"
	default:
		return "◆"
	}
}

// MarketInfluence is the live effect a NewsEvent has on the market each tick.
type MarketInfluence struct {
	NewsID             int
	AffectedIndustries []Industry
	AffectedSymbols    []string
	Sentiment          float64
	Magnitude          float64
	CreatedAt          time.Time
	ExpiresAt          time.Time
}

func (mi *MarketInfluence) IsActive() bool {
	return time.Now().Before(mi.ExpiresAt)
}

// InfluenceStrength returns the current influence multiplier, decaying linearly over time.
func (mi *MarketInfluence) InfluenceStrength() float64 {
	total := mi.ExpiresAt.Sub(mi.CreatedAt).Seconds()
	remaining := time.Until(mi.ExpiresAt).Seconds()
	if total <= 0 || remaining <= 0 {
		return 0
	}
	decay := remaining / total
	return mi.Sentiment * mi.Magnitude * decay
}

// Targets returns true if this influence affects the given stock.
func (mi *MarketInfluence) Targets(s *Stock) bool {
	for _, sym := range mi.AffectedSymbols {
		if sym == s.Symbol {
			return true
		}
	}
	for _, ind := range mi.AffectedIndustries {
		if ind == s.Industry {
			return true
		}
	}
	return false
}

// --- news template system ---

type newsTemplate struct {
	headline    string // may contain {company}, {symbol}, {industry}
	category    NewsCategory
	industries  []Industry
	sentiment   float64
	magnitude   float64
	minDuration int // minutes
	maxDuration int
	companyNews bool // if true, picks a random stock and injects its name/symbol
}

var newsTemplates = []newsTemplate{
	// CONFLICT
	{
		headline:   "Armed conflict erupts in major oil-producing region, supply routes disrupted",
		category:   NewsCatConflict,
		industries: []Industry{IndustryEnergy, IndustryMaterials},
		sentiment:  0.5, magnitude: 0.7, minDuration: 3, maxDuration: 8,
	},
	{
		headline:   "Ceasefire declared — diplomatic talks resume in disputed territory",
		category:   NewsCatConflict,
		industries: []Industry{IndustryEnergy, IndustryIndustrial},
		sentiment:  0.3, magnitude: 0.4, minDuration: 2, maxDuration: 5,
	},
	{
		headline:   "Trade war escalates: new tariffs imposed on semiconductor imports",
		category:   NewsCatConflict,
		industries: []Industry{IndustryTech, IndustryIndustrial},
		sentiment:  -0.5, magnitude: 0.6, minDuration: 4, maxDuration: 9,
	},
	{
		headline:   "Sanctions expanded on major exporting nation — energy markets brace",
		category:   NewsCatConflict,
		industries: []Industry{IndustryEnergy, IndustryFinance},
		sentiment:  -0.4, magnitude: 0.5, minDuration: 3, maxDuration: 7,
	},
	{
		headline:   "Cybersecurity breach hits national infrastructure — defense spending surges",
		category:   NewsCatConflict,
		industries: []Industry{IndustryTech, IndustryIndustrial},
		sentiment:  0.3, magnitude: 0.5, minDuration: 2, maxDuration: 6,
	},

	// DISASTER
	{
		headline:   "Category 5 hurricane makes landfall — insurance losses projected in billions",
		category:   NewsCatDisaster,
		industries: []Industry{IndustryFinance, IndustryReal, IndustryEnergy},
		sentiment:  -0.6, magnitude: 0.7, minDuration: 3, maxDuration: 8,
	},
	{
		headline:   "Major earthquake damages supply chain hubs across Pacific Rim",
		category:   NewsCatDisaster,
		industries: []Industry{IndustryIndustrial, IndustryConsumer, IndustryTech},
		sentiment:  -0.5, magnitude: 0.6, minDuration: 2, maxDuration: 6,
	},
	{
		headline:   "Severe drought declared across agricultural heartland — food prices spike",
		category:   NewsCatDisaster,
		industries: []Industry{IndustryConsumer, IndustryMaterials},
		sentiment:  -0.4, magnitude: 0.5, minDuration: 4, maxDuration: 10,
	},
	{
		headline:   "Wildfire season forces closure of mining operations across three states",
		category:   NewsCatDisaster,
		industries: []Industry{IndustryMaterials, IndustryEnergy},
		sentiment:  -0.4, magnitude: 0.4, minDuration: 2, maxDuration: 5,
	},

	// TECH
	{
		headline:   "Breakthrough quantum computing milestone reached — encryption paradigm at risk",
		category:   NewsCatTech,
		industries: []Industry{IndustryTech, IndustryFinance},
		sentiment:  0.6, magnitude: 0.6, minDuration: 3, maxDuration: 8,
	},
	{
		headline:   "AI regulation bill passes committee — strict data governance rules incoming",
		category:   NewsCatTech,
		industries: []Industry{IndustryTech},
		sentiment:  -0.4, magnitude: 0.5, minDuration: 3, maxDuration: 7,
	},
	{
		headline:   "Revolutionary solid-state battery tech claims 5x energy density breakthrough",
		category:   NewsCatTech,
		industries: []Industry{IndustryTech, IndustryEnergy},
		sentiment:  0.7, magnitude: 0.7, minDuration: 3, maxDuration: 9,
	},
	{
		headline:   "New AI chip architecture promises 10x performance leap — cloud providers rally",
		category:   NewsCatTech,
		industries: []Industry{IndustryTech},
		sentiment:  0.7, magnitude: 0.8, minDuration: 2, maxDuration: 6,
	},
	{
		headline:   "Mass data breach exposes 200M user records — cybersecurity stocks surge",
		category:   NewsCatTech,
		industries: []Industry{IndustryTech},
		sentiment:  0.4, magnitude: 0.5, minDuration: 2, maxDuration: 4,
		companyNews: false,
	},
	{
		headline:   "{company} ({symbol}) unveils next-gen platform — analysts raise price targets",
		category:   NewsCatTech,
		industries: []Industry{IndustryTech},
		sentiment:  0.6, magnitude: 0.6, minDuration: 2, maxDuration: 5,
		companyNews: true,
	},

	// M&A
	{
		headline:   "{company} ({symbol}) in advanced acquisition talks — premium rumored at 35%",
		category:   NewsCatMA,
		industries: nil,
		sentiment:  0.8, magnitude: 0.9, minDuration: 2, maxDuration: 5,
		companyNews: true,
	},
	{
		headline:   "Regulators block mega-merger citing antitrust concerns — shares fall sharply",
		category:   NewsCatMA,
		industries: []Industry{IndustryTech, IndustryFinance},
		sentiment:  -0.5, magnitude: 0.6, minDuration: 2, maxDuration: 5,
	},
	{
		headline:   "{company} ({symbol}) completes $12B acquisition of rival — synergies expected",
		category:   NewsCatMA,
		industries: nil,
		sentiment:  0.5, magnitude: 0.5, minDuration: 2, maxDuration: 6,
		companyNews: true,
	},
	{
		headline:   "Private equity firm announces $8B leveraged buyout in {industry} sector",
		category:   NewsCatMA,
		industries: []Industry{IndustryIndustrial, IndustryConsumer, IndustryReal},
		sentiment:  0.4, magnitude: 0.4, minDuration: 2, maxDuration: 4,
	},

	// REGULATION
	{
		headline:   "Central bank signals surprise rate cut — risk assets rally across the board",
		category:   NewsCatRegulation,
		industries: []Industry{IndustryFinance, IndustryReal, IndustryTech},
		sentiment:  0.6, magnitude: 0.7, minDuration: 3, maxDuration: 7,
	},
	{
		headline:   "Rate hike larger than expected — growth stocks under pressure",
		category:   NewsCatRegulation,
		industries: []Industry{IndustryTech, IndustryCrypto, IndustryReal},
		sentiment:  -0.6, magnitude: 0.7, minDuration: 3, maxDuration: 7,
	},
	{
		headline:   "New environmental regulations mandate costly upgrades for energy producers",
		category:   NewsCatRegulation,
		industries: []Industry{IndustryEnergy, IndustryMaterials, IndustryUtilities},
		sentiment:  -0.4, magnitude: 0.5, minDuration: 4, maxDuration: 8,
	},
	{
		headline:   "Banking stress tests reveal capital shortfalls — credit markets tighten",
		category:   NewsCatRegulation,
		industries: []Industry{IndustryFinance, IndustryReal},
		sentiment:  -0.5, magnitude: 0.6, minDuration: 3, maxDuration: 6,
	},
	{
		headline:   "Government announces major infrastructure stimulus package",
		category:   NewsCatRegulation,
		industries: []Industry{IndustryIndustrial, IndustryMaterials, IndustryUtilities},
		sentiment:  0.6, magnitude: 0.6, minDuration: 4, maxDuration: 9,
	},

	// ECONOMY
	{
		headline:   "GDP growth beats expectations — consumer spending up 4.2% quarter-on-quarter",
		category:   NewsCatEconomy,
		industries: []Industry{IndustryConsumer, IndustryFinance, IndustryReal},
		sentiment:  0.5, magnitude: 0.5, minDuration: 3, maxDuration: 6,
	},
	{
		headline:   "Unemployment hits 50-year low — wage inflation concerns resurface",
		category:   NewsCatEconomy,
		industries: []Industry{IndustryConsumer, IndustryFinance},
		sentiment:  0.3, magnitude: 0.4, minDuration: 2, maxDuration: 5,
	},
	{
		headline:   "Inflation prints hotter than expected — bond yields spike to multi-year highs",
		category:   NewsCatEconomy,
		industries: []Industry{IndustryFinance, IndustryReal, IndustryUtilities},
		sentiment:  -0.5, magnitude: 0.6, minDuration: 3, maxDuration: 7,
	},
	{
		headline:   "Manufacturing PMI collapses to 12-year low — recession fears mount",
		category:   NewsCatEconomy,
		industries: []Industry{IndustryIndustrial, IndustryMaterials, IndustryConsumer},
		sentiment:  -0.6, magnitude: 0.7, minDuration: 3, maxDuration: 8,
	},
	{
		headline:   "Trade surplus hits record high — exports surge on weak currency",
		category:   NewsCatEconomy,
		industries: []Industry{IndustryIndustrial, IndustryMaterials},
		sentiment:  0.4, magnitude: 0.4, minDuration: 2, maxDuration: 5,
	},

	// SPACE
	{
		headline:   "Space agency announces crewed Mars mission — contracts awarded to aerospace firms",
		category:   NewsCatSpace,
		industries: []Industry{IndustryTech, IndustryIndustrial},
		sentiment:  0.5, magnitude: 0.5, minDuration: 3, maxDuration: 7,
	},
	{
		headline:   "Private space company achieves first orbital reusability milestone",
		category:   NewsCatSpace,
		industries: []Industry{IndustryTech},
		sentiment:  0.6, magnitude: 0.5, minDuration: 2, maxDuration: 5,
	},
	{
		headline:   "Asteroid mining company secures $3B in Series C — materials sector watches",
		category:   NewsCatSpace,
		industries: []Industry{IndustryTech, IndustryMaterials},
		sentiment:  0.4, magnitude: 0.4, minDuration: 2, maxDuration: 5,
	},
	{
		headline:   "Satellite constellation launch fails — global communications disrupted",
		category:   NewsCatSpace,
		industries: []Industry{IndustryTech},
		sentiment:  -0.4, magnitude: 0.5, minDuration: 2, maxDuration: 4,
	},

	// HEALTH
	{
		headline:   "Clinical trial shows 94% efficacy for new cancer immunotherapy",
		category:   NewsCatHealth,
		industries: []Industry{IndustryHealthcare},
		sentiment:  0.7, magnitude: 0.7, minDuration: 3, maxDuration: 7,
	},
	{
		headline:   "Novel pathogen detected in three continents — WHO declares global health alert",
		category:   NewsCatHealth,
		industries: []Industry{IndustryHealthcare, IndustryConsumer, IndustryReal},
		sentiment:  0.5, magnitude: 0.7, minDuration: 4, maxDuration: 10,
	},
	{
		headline:   "{company} ({symbol}) drug fails Phase III trial — FDA approval prospects dim",
		category:   NewsCatHealth,
		industries: []Industry{IndustryHealthcare},
		sentiment:  -0.8, magnitude: 0.9, minDuration: 2, maxDuration: 5,
		companyNews: true,
	},
	{
		headline:   "FDA fast-tracks approval for breakthrough gene therapy treatment",
		category:   NewsCatHealth,
		industries: []Industry{IndustryHealthcare},
		sentiment:  0.6, magnitude: 0.6, minDuration: 2, maxDuration: 6,
	},
	{
		headline:   "Major pharmaceutical merger creates world's largest drug company",
		category:   NewsCatHealth,
		industries: []Industry{IndustryHealthcare},
		sentiment:  0.5, magnitude: 0.6, minDuration: 3, maxDuration: 6,
	},

	// ENERGY
	{
		headline:   "OPEC+ agrees to surprise production cut of 1.5M barrels per day",
		category:   NewsCatEnergy,
		industries: []Industry{IndustryEnergy},
		sentiment:  0.6, magnitude: 0.7, minDuration: 4, maxDuration: 9,
	},
	{
		headline:   "Massive new oil reserve discovered offshore — production could begin in 2 years",
		category:   NewsCatEnergy,
		industries: []Industry{IndustryEnergy},
		sentiment:  -0.3, magnitude: 0.4, minDuration: 3, maxDuration: 7,
	},
	{
		headline:   "Grid-scale fusion reactor achieves net energy gain — energy sector in turmoil",
		category:   NewsCatEnergy,
		industries: []Industry{IndustryEnergy, IndustryUtilities, IndustryTech},
		sentiment:  0.8, magnitude: 0.9, minDuration: 4, maxDuration: 10,
	},
	{
		headline:   "Pipeline explosion shuts down 30% of regional natural gas supply",
		category:   NewsCatEnergy,
		industries: []Industry{IndustryEnergy, IndustryUtilities},
		sentiment:  0.4, magnitude: 0.6, minDuration: 2, maxDuration: 5,
	},
	{
		headline:   "Renewable energy surpasses fossil fuels in grid generation for first time",
		category:   NewsCatEnergy,
		industries: []Industry{IndustryEnergy, IndustryUtilities},
		sentiment:  -0.3, magnitude: 0.4, minDuration: 3, maxDuration: 6,
	},

	// CRYPTO
	{
		headline:   "Major nation announces Bitcoin as legal tender — crypto markets surge",
		category:   NewsCatCrypto,
		industries: []Industry{IndustryCrypto, IndustryFinance},
		sentiment:  0.8, magnitude: 0.9, minDuration: 3, maxDuration: 8,
	},
	{
		headline:   "Crypto exchange collapses following $2B in unauthorized withdrawals",
		category:   NewsCatCrypto,
		industries: []Industry{IndustryCrypto, IndustryFinance},
		sentiment:  -0.8, magnitude: 0.9, minDuration: 3, maxDuration: 8,
	},
	{
		headline:   "SEC approves first spot crypto ETF — institutional floodgates open",
		category:   NewsCatCrypto,
		industries: []Industry{IndustryCrypto, IndustryFinance},
		sentiment:  0.7, magnitude: 0.8, minDuration: 3, maxDuration: 7,
	},
	{
		headline:   "G7 nations agree on sweeping crypto taxation framework",
		category:   NewsCatCrypto,
		industries: []Industry{IndustryCrypto},
		sentiment:  -0.5, magnitude: 0.6, minDuration: 3, maxDuration: 6,
	},
	{
		headline:   "DeFi protocol hack drains $800M — smart contract audits under scrutiny",
		category:   NewsCatCrypto,
		industries: []Industry{IndustryCrypto},
		sentiment:  -0.6, magnitude: 0.7, minDuration: 2, maxDuration: 5,
	},

	// CONSUMER
	{
		headline:   "Holiday retail sales surge 12% YoY — consumer confidence at 5-year high",
		category:   NewsCatConsumer,
		industries: []Industry{IndustryConsumer, IndustryReal},
		sentiment:  0.5, magnitude: 0.5, minDuration: 2, maxDuration: 5,
	},
	{
		headline:   "Supply chain crisis deepens — shipping costs triple, shelves empty",
		category:   NewsCatConsumer,
		industries: []Industry{IndustryConsumer, IndustryIndustrial},
		sentiment:  -0.5, magnitude: 0.6, minDuration: 3, maxDuration: 7,
	},
	{
		headline:   "Consumer debt reaches all-time high — default rates beginning to rise",
		category:   NewsCatConsumer,
		industries: []Industry{IndustryConsumer, IndustryFinance},
		sentiment:  -0.4, magnitude: 0.5, minDuration: 3, maxDuration: 6,
	},
	{
		headline:   "{company} ({symbol}) reports blowout earnings — raises full-year guidance",
		category:   NewsCatEarnings,
		industries: nil,
		sentiment:  0.7, magnitude: 0.7, minDuration: 2, maxDuration: 5,
		companyNews: true,
	},
	{
		headline:   "{company} ({symbol}) misses estimates badly — CEO steps down effective immediately",
		category:   NewsCatEarnings,
		industries: nil,
		sentiment:  -0.7, magnitude: 0.8, minDuration: 2, maxDuration: 5,
		companyNews: true,
	},
	{
		headline:   "{company} ({symbol}) announces $5B share buyback program",
		category:   NewsCatEarnings,
		industries: nil,
		sentiment:  0.5, magnitude: 0.5, minDuration: 2, maxDuration: 4,
		companyNews: true,
	},
}

var industryNames = map[Industry]string{
	IndustryTech:       "Technology",
	IndustryFinance:    "Finance",
	IndustryEnergy:     "Energy",
	IndustryHealthcare: "Healthcare",
	IndustryConsumer:   "Consumer",
	IndustryIndustrial: "Industrial",
	IndustryCrypto:     "Crypto",
	IndustryReal:       "Real Estate",
	IndustryMaterials:  "Materials",
	IndustryUtilities:  "Utilities",
}

func GenerateNews(rng *rand.Rand, stocks []*Stock, nextID int) (*NewsEvent, *MarketInfluence) {
	tmpl := newsTemplates[rng.Intn(len(newsTemplates))]

	var affectedSymbols []string
	var affectedIndustries []Industry
	headline := tmpl.headline

	if tmpl.companyNews {
		// pick a random stock and inject it into the headline
		if len(stocks) > 0 {
			pick := stocks[rng.Intn(len(stocks))]
			headline = strings.ReplaceAll(headline, "{company}", pick.Name)
			headline = strings.ReplaceAll(headline, "{symbol}", pick.Symbol)
			affectedSymbols = []string{pick.Symbol}
			// also affect the whole industry at reduced magnitude
			affectedIndustries = []Industry{pick.Industry}
		}
	} else {
		affectedIndustries = tmpl.industries
		// replace {industry} placeholder if present
		if len(tmpl.industries) > 0 {
			headline = strings.ReplaceAll(headline, "{industry}", string(tmpl.industries[rng.Intn(len(tmpl.industries))]))
		}
	}

	// sentiment jitter ±20%
	jitter := (rng.Float64()*0.4 - 0.2) * tmpl.magnitude
	sentiment := tmpl.sentiment + jitter
	if sentiment > 1.0 {
		sentiment = 1.0
	}
	if sentiment < -1.0 {
		sentiment = -1.0
	}

	magnitude := tmpl.magnitude * (0.8 + rng.Float64()*0.4)

	duration := tmpl.minDuration + rng.Intn(tmpl.maxDuration-tmpl.minDuration+1)
	now := time.Now()
	expires := now.Add(time.Duration(duration) * time.Minute)

	event := &NewsEvent{
		ID:                 nextID,
		Headline:           headline,
		Category:           tmpl.category,
		AffectedIndustries: affectedIndustries,
		AffectedSymbols:    affectedSymbols,
		Sentiment:          roundTo(sentiment, 2),
		Magnitude:          roundTo(magnitude, 2),
		CreatedAt:          now,
		ExpiresAt:          expires,
	}

	influence := &MarketInfluence{
		NewsID:             nextID,
		AffectedIndustries: affectedIndustries,
		AffectedSymbols:    affectedSymbols,
		Sentiment:          sentiment,
		Magnitude:          magnitude,
		CreatedAt:          now,
		ExpiresAt:          expires,
	}

	return event, influence
}

func CategoryColor(cat NewsCategory) string {
	switch cat {
	case NewsCatConflict:
		return "#ff4757"
	case NewsCatDisaster:
		return "#ff6b35"
	case NewsCatTech:
		return "#1e90ff"
	case NewsCatMA:
		return "#a29bfe"
	case NewsCatRegulation:
		return "#fdcb6e"
	case NewsCatEconomy:
		return "#00cec9"
	case NewsCatSpace:
		return "#74b9ff"
	case NewsCatHealth:
		return "#00d26a"
	case NewsCatEnergy:
		return "#e17055"
	case NewsCatCrypto:
		return "#f9ca24"
	case NewsCatConsumer:
		return "#fd79a8"
	case NewsCatEarnings:
		return "#55efc4"
	default:
		return "#636e72"
	}
}
