package market

import (
	"fmt"
	"math"
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

// InfluenceEffect describes the sentiment impact on a specific group of industries or symbols.
type InfluenceEffect struct {
	Industries      []Industry
	Symbols         []string // populated at runtime for companyNews events
	Sentiment       float64
	CompanyIndustry bool // if true, Industries is replaced with the picked company's industry at gen time
}

type NewsEvent struct {
	ID                 int
	Headline           string
	Category           NewsCategory
	Effects            []InfluenceEffect
	AffectedIndustries []Industry // union of all effects' industries (backward-compat display)
	AffectedSymbols    []string   // union of all effects' symbols
	Sentiment          float64    // strongest-magnitude effect's sentiment (for summary display)
	Magnitude          float64
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
	return SentimentLabel(n.Sentiment)
}

// SentimentLabel returns ▲▲ / ▲ / ◆ / ▼ / ▼▼ for a sentiment value.
func SentimentLabel(s float64) string {
	switch {
	case s > 0.4:
		return "▲▲"
	case s > 0.1:
		return "▲"
	case s < -0.4:
		return "▼▼"
	case s < -0.1:
		return "▼"
	default:
		return "◆"
	}
}

// SentimentFor returns the sentiment this event has for the given stock and whether the stock is targeted.
func (n *NewsEvent) SentimentFor(s *Stock) (float64, bool) {
	for _, eff := range n.Effects {
		for _, sym := range eff.Symbols {
			if sym == s.Symbol {
				return eff.Sentiment, true
			}
		}
		for _, ind := range eff.Industries {
			if ind == s.Industry {
				return eff.Sentiment, true
			}
		}
	}
	return 0, false
}

// MarketInfluence is the live per-tick effect a NewsEvent applies to the market.
type MarketInfluence struct {
	NewsID    int
	Effects   []InfluenceEffect
	Magnitude float64
	CreatedAt time.Time
	ExpiresAt time.Time
}

func (mi *MarketInfluence) IsActive() bool {
	return time.Now().Before(mi.ExpiresAt)
}

func (mi *MarketInfluence) decay() float64 {
	total := mi.ExpiresAt.Sub(mi.CreatedAt).Seconds()
	remaining := time.Until(mi.ExpiresAt).Seconds()
	if total <= 0 || remaining <= 0 {
		return 0
	}
	return remaining / total
}

// Targets returns true if the stock is covered by any effect.
func (mi *MarketInfluence) Targets(s *Stock) bool {
	for _, eff := range mi.Effects {
		for _, sym := range eff.Symbols {
			if sym == s.Symbol {
				return true
			}
		}
		for _, ind := range eff.Industries {
			if ind == s.Industry {
				return true
			}
		}
	}
	return false
}

// InfluenceStrengthFor returns the decayed directional influence for a specific stock.
func (mi *MarketInfluence) InfluenceStrengthFor(s *Stock) float64 {
	d := mi.decay()
	if d == 0 {
		return 0
	}
	for _, eff := range mi.Effects {
		for _, sym := range eff.Symbols {
			if sym == s.Symbol {
				return eff.Sentiment * mi.Magnitude * d
			}
		}
		for _, ind := range eff.Industries {
			if ind == s.Industry {
				return eff.Sentiment * mi.Magnitude * d
			}
		}
	}
	return 0
}

// PrimaryEffect returns the sentiment and absolute strength of the strongest effect, for display.
func (mi *MarketInfluence) PrimaryEffect() (sentiment float64, absStrength float64) {
	d := mi.decay()
	for _, eff := range mi.Effects {
		v := math.Abs(eff.Sentiment * mi.Magnitude * d)
		if v > absStrength {
			absStrength = v
			sentiment = eff.Sentiment
		}
	}
	return
}

// --- news template system ---

type newsTemplate struct {
	headline    string
	category    NewsCategory
	effects     []InfluenceEffect
	magnitude   float64
	minDuration int
	maxDuration int
	companyNews bool // effects[0].Symbols is filled with the picked company at runtime
}

var newsTemplates = []newsTemplate{

	// CONFLICT
	{
		headline: "Armed conflict erupts in major oil-producing region, supply routes disrupted",
		category: NewsCatConflict,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryEnergy, IndustryMaterials}, Sentiment: 0.5},
			{Industries: []Industry{IndustryIndustrial}, Sentiment: 0.25},
			{Industries: []Industry{IndustryConsumer, IndustryTech}, Sentiment: -0.25},
		},
		magnitude: 0.7, minDuration: 3, maxDuration: 8,
	},
	{
		headline: "Ceasefire declared — diplomatic talks resume in disputed territory",
		category: NewsCatConflict,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryFinance, IndustryConsumer}, Sentiment: 0.35},
			{Industries: []Industry{IndustryIndustrial}, Sentiment: 0.2},
			{Industries: []Industry{IndustryEnergy}, Sentiment: -0.2},
		},
		magnitude: 0.4, minDuration: 2, maxDuration: 5,
	},
	{
		headline: "Trade war escalates: new tariffs imposed on semiconductor imports",
		category: NewsCatConflict,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryTech, IndustryIndustrial}, Sentiment: -0.5},
			{Industries: []Industry{IndustryMaterials}, Sentiment: 0.2},
			{Industries: []Industry{IndustryFinance}, Sentiment: -0.2},
		},
		magnitude: 0.6, minDuration: 4, maxDuration: 9,
	},
	{
		headline: "Sanctions expanded on major exporting nation — energy markets brace",
		category: NewsCatConflict,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryEnergy, IndustryFinance}, Sentiment: -0.4},
			{Industries: []Industry{IndustryMaterials}, Sentiment: 0.2},
		},
		magnitude: 0.5, minDuration: 3, maxDuration: 7,
	},
	{
		headline: "Cybersecurity breach hits national infrastructure — defense spending surges",
		category: NewsCatConflict,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryTech}, Sentiment: 0.4},
			{Industries: []Industry{IndustryIndustrial}, Sentiment: 0.25},
			{Industries: []Industry{IndustryFinance}, Sentiment: -0.2},
		},
		magnitude: 0.5, minDuration: 2, maxDuration: 6,
	},

	// DISASTER
	{
		headline: "Category 5 hurricane makes landfall — insurance losses projected in billions",
		category: NewsCatDisaster,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryFinance, IndustryReal, IndustryEnergy}, Sentiment: -0.6},
			{Industries: []Industry{IndustryMaterials, IndustryIndustrial}, Sentiment: 0.35},
		},
		magnitude: 0.7, minDuration: 3, maxDuration: 8,
	},
	{
		headline: "Major earthquake damages supply chain hubs across Pacific Rim",
		category: NewsCatDisaster,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryIndustrial, IndustryConsumer, IndustryTech}, Sentiment: -0.5},
			{Industries: []Industry{IndustryMaterials}, Sentiment: 0.2},
		},
		magnitude: 0.6, minDuration: 2, maxDuration: 6,
	},
	{
		headline: "Severe drought declared across agricultural heartland — food prices spike",
		category: NewsCatDisaster,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryConsumer, IndustryMaterials}, Sentiment: -0.4},
			{Industries: []Industry{IndustryFinance}, Sentiment: -0.2},
			{Industries: []Industry{IndustryEnergy}, Sentiment: 0.1},
		},
		magnitude: 0.5, minDuration: 4, maxDuration: 10,
	},
	{
		headline: "Wildfire season forces closure of mining operations across three states",
		category: NewsCatDisaster,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryMaterials, IndustryEnergy}, Sentiment: -0.4},
			{Industries: []Industry{IndustryIndustrial}, Sentiment: 0.1},
		},
		magnitude: 0.4, minDuration: 2, maxDuration: 5,
	},

	// TECH
	{
		headline: "Breakthrough quantum computing milestone reached — encryption paradigm at risk",
		category: NewsCatTech,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryTech}, Sentiment: 0.6},
			{Industries: []Industry{IndustryFinance}, Sentiment: 0.4},
			{Industries: []Industry{IndustryMaterials}, Sentiment: -0.15},
		},
		magnitude: 0.6, minDuration: 3, maxDuration: 8,
	},
	{
		headline: "AI regulation bill passes committee — strict data governance rules incoming",
		category: NewsCatTech,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryTech}, Sentiment: -0.4},
			{Industries: []Industry{IndustryFinance}, Sentiment: -0.2},
			{Industries: []Industry{IndustryUtilities}, Sentiment: 0.1},
		},
		magnitude: 0.5, minDuration: 3, maxDuration: 7,
	},
	{
		headline: "Revolutionary solid-state battery tech claims 5x energy density breakthrough",
		category: NewsCatTech,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryTech}, Sentiment: 0.7},
			{Industries: []Industry{IndustryIndustrial}, Sentiment: 0.3},
			{Industries: []Industry{IndustryEnergy}, Sentiment: -0.35},
		},
		magnitude: 0.7, minDuration: 3, maxDuration: 9,
	},
	{
		headline: "New AI chip architecture promises 10x performance leap — cloud providers rally",
		category: NewsCatTech,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryTech}, Sentiment: 0.7},
			{Industries: []Industry{IndustryEnergy}, Sentiment: 0.2},
			{Industries: []Industry{IndustryIndustrial}, Sentiment: 0.15},
		},
		magnitude: 0.8, minDuration: 2, maxDuration: 6,
	},
	{
		headline: "Mass data breach exposes 200M user records — cybersecurity stocks surge",
		category: NewsCatTech,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryTech}, Sentiment: 0.4},
			{Industries: []Industry{IndustryFinance}, Sentiment: -0.3},
			{Industries: []Industry{IndustryConsumer}, Sentiment: -0.2},
		},
		magnitude: 0.5, minDuration: 2, maxDuration: 4,
	},
	{
		headline:    "{company} ({symbol}) unveils next-gen platform — analysts raise price targets",
		category:    NewsCatTech,
		effects:     []InfluenceEffect{{Sentiment: 0.6}, {Industries: []Industry{IndustryTech}, Sentiment: 0.1}},
		magnitude:   0.6, minDuration: 2, maxDuration: 5,
		companyNews: true,
	},

	// M&A
	{
		headline:    "{company} ({symbol}) in advanced acquisition talks — premium rumored at 35%",
		category:    NewsCatMA,
		effects:     []InfluenceEffect{{Sentiment: 0.8}},
		magnitude:   0.9, minDuration: 2, maxDuration: 5,
		companyNews: true,
	},
	{
		headline: "Regulators block mega-merger citing antitrust concerns — shares fall sharply",
		category: NewsCatMA,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryTech, IndustryFinance}, Sentiment: -0.5},
			{Industries: []Industry{IndustryConsumer}, Sentiment: 0.2},
		},
		magnitude: 0.6, minDuration: 2, maxDuration: 5,
	},
	{
		headline:    "{company} ({symbol}) completes $12B acquisition of rival — synergies expected",
		category:    NewsCatMA,
		effects:     []InfluenceEffect{{Sentiment: 0.5}, {CompanyIndustry: true, Sentiment: 0.1}},
		magnitude:   0.5, minDuration: 2, maxDuration: 6,
		companyNews: true,
	},
	{
		headline: "Private equity firm announces $8B leveraged buyout in {industry} sector",
		category: NewsCatMA,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryIndustrial, IndustryConsumer, IndustryReal}, Sentiment: 0.4},
			{Industries: []Industry{IndustryFinance}, Sentiment: 0.2},
		},
		magnitude: 0.4, minDuration: 2, maxDuration: 4,
	},

	// REGULATION
	{
		headline: "Central bank signals surprise rate cut — risk assets rally across the board",
		category: NewsCatRegulation,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryFinance, IndustryReal, IndustryTech}, Sentiment: 0.6},
			{Industries: []Industry{IndustryUtilities}, Sentiment: -0.2},
		},
		magnitude: 0.7, minDuration: 3, maxDuration: 7,
	},
	{
		headline: "Rate hike larger than expected — growth stocks under pressure",
		category: NewsCatRegulation,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryTech, IndustryCrypto, IndustryReal}, Sentiment: -0.6},
			{Industries: []Industry{IndustryFinance}, Sentiment: 0.35},
			{Industries: []Industry{IndustryUtilities}, Sentiment: 0.2},
		},
		magnitude: 0.7, minDuration: 3, maxDuration: 7,
	},
	{
		headline: "New environmental regulations mandate costly upgrades for energy producers",
		category: NewsCatRegulation,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryEnergy, IndustryMaterials, IndustryUtilities}, Sentiment: -0.4},
			{Industries: []Industry{IndustryTech}, Sentiment: 0.2},
			{Industries: []Industry{IndustryIndustrial}, Sentiment: 0.1},
		},
		magnitude: 0.5, minDuration: 4, maxDuration: 8,
	},
	{
		headline: "Banking stress tests reveal capital shortfalls — credit markets tighten",
		category: NewsCatRegulation,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryFinance, IndustryReal}, Sentiment: -0.5},
			{Industries: []Industry{IndustryConsumer}, Sentiment: -0.2},
			{Industries: []Industry{IndustryTech}, Sentiment: 0.1},
		},
		magnitude: 0.6, minDuration: 3, maxDuration: 6,
	},
	{
		headline: "Government announces major infrastructure stimulus package",
		category: NewsCatRegulation,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryIndustrial, IndustryMaterials}, Sentiment: 0.6},
			{Industries: []Industry{IndustryUtilities}, Sentiment: 0.4},
			{Industries: []Industry{IndustryFinance}, Sentiment: 0.2},
		},
		magnitude: 0.6, minDuration: 4, maxDuration: 9,
	},

	// ECONOMY
	{
		headline: "GDP growth beats expectations — consumer spending up 4.2% quarter-on-quarter",
		category: NewsCatEconomy,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryConsumer, IndustryFinance}, Sentiment: 0.5},
			{Industries: []Industry{IndustryReal}, Sentiment: 0.3},
			{Industries: []Industry{IndustryUtilities}, Sentiment: -0.1},
		},
		magnitude: 0.5, minDuration: 3, maxDuration: 6,
	},
	{
		headline: "Unemployment hits 50-year low — wage inflation concerns resurface",
		category: NewsCatEconomy,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryConsumer, IndustryFinance}, Sentiment: 0.3},
			{Industries: []Industry{IndustryReal}, Sentiment: 0.2},
			{Industries: []Industry{IndustryUtilities}, Sentiment: -0.1},
		},
		magnitude: 0.4, minDuration: 2, maxDuration: 5,
	},
	{
		headline: "Inflation prints hotter than expected — bond yields spike to multi-year highs",
		category: NewsCatEconomy,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryFinance, IndustryReal, IndustryUtilities}, Sentiment: -0.5},
			{Industries: []Industry{IndustryConsumer}, Sentiment: -0.3},
			{Industries: []Industry{IndustryEnergy, IndustryMaterials}, Sentiment: 0.3},
		},
		magnitude: 0.6, minDuration: 3, maxDuration: 7,
	},
	{
		headline: "Manufacturing PMI collapses to 12-year low — recession fears mount",
		category: NewsCatEconomy,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryIndustrial, IndustryMaterials, IndustryConsumer}, Sentiment: -0.6},
			{Industries: []Industry{IndustryFinance}, Sentiment: -0.3},
			{Industries: []Industry{IndustryUtilities}, Sentiment: 0.2},
		},
		magnitude: 0.7, minDuration: 3, maxDuration: 8,
	},
	{
		headline: "Trade surplus hits record high — exports surge on weak currency",
		category: NewsCatEconomy,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryIndustrial, IndustryMaterials}, Sentiment: 0.4},
			{Industries: []Industry{IndustryFinance}, Sentiment: 0.2},
		},
		magnitude: 0.4, minDuration: 2, maxDuration: 5,
	},

	// SPACE
	{
		headline: "Space agency announces crewed Mars mission — contracts awarded to aerospace firms",
		category: NewsCatSpace,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryTech, IndustryIndustrial}, Sentiment: 0.5},
			{Industries: []Industry{IndustryEnergy}, Sentiment: 0.1},
		},
		magnitude: 0.5, minDuration: 3, maxDuration: 7,
	},
	{
		headline: "Private space company achieves first orbital reusability milestone",
		category: NewsCatSpace,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryTech}, Sentiment: 0.6},
			{Industries: []Industry{IndustryIndustrial}, Sentiment: 0.3},
			{Industries: []Industry{IndustryMaterials}, Sentiment: -0.1},
		},
		magnitude: 0.5, minDuration: 2, maxDuration: 5,
	},
	{
		headline: "Asteroid mining company secures $3B in Series C — materials sector watches",
		category: NewsCatSpace,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryMaterials}, Sentiment: 0.5},
			{Industries: []Industry{IndustryTech}, Sentiment: 0.4},
			{Industries: []Industry{IndustryIndustrial}, Sentiment: 0.2},
		},
		magnitude: 0.4, minDuration: 2, maxDuration: 5,
	},
	{
		headline: "Satellite constellation launch fails — global communications disrupted",
		category: NewsCatSpace,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryTech}, Sentiment: -0.4},
			{Industries: []Industry{IndustryConsumer}, Sentiment: -0.2},
			{Industries: []Industry{IndustryIndustrial}, Sentiment: 0.1},
		},
		magnitude: 0.5, minDuration: 2, maxDuration: 4,
	},

	// HEALTH
	{
		headline: "Clinical trial shows 94% efficacy for new cancer immunotherapy",
		category: NewsCatHealth,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryHealthcare}, Sentiment: 0.7},
		},
		magnitude: 0.7, minDuration: 3, maxDuration: 7,
	},
	{
		headline: "Novel pathogen detected in three continents — WHO declares global health alert",
		category: NewsCatHealth,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryHealthcare}, Sentiment: 0.7},
			{Industries: []Industry{IndustryConsumer, IndustryReal}, Sentiment: -0.5},
			{Industries: []Industry{IndustryUtilities}, Sentiment: 0.1},
		},
		magnitude: 0.7, minDuration: 4, maxDuration: 10,
	},
	{
		headline:    "{company} ({symbol}) drug fails Phase III trial — FDA approval prospects dim",
		category:    NewsCatHealth,
		effects:     []InfluenceEffect{{Sentiment: -0.8}, {Industries: []Industry{IndustryHealthcare}, Sentiment: 0.2}},
		magnitude:   0.9, minDuration: 2, maxDuration: 5,
		companyNews: true,
	},
	{
		headline: "FDA fast-tracks approval for breakthrough gene therapy treatment",
		category: NewsCatHealth,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryHealthcare}, Sentiment: 0.6},
		},
		magnitude: 0.6, minDuration: 2, maxDuration: 6,
	},
	{
		headline: "Major pharmaceutical merger creates world's largest drug company",
		category: NewsCatHealth,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryHealthcare}, Sentiment: 0.5},
		},
		magnitude: 0.6, minDuration: 3, maxDuration: 6,
	},

	// ENERGY
	{
		headline: "OPEC+ agrees to surprise production cut of 1.5M barrels per day",
		category: NewsCatEnergy,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryEnergy}, Sentiment: 0.6},
			{Industries: []Industry{IndustryConsumer, IndustryIndustrial}, Sentiment: -0.3},
		},
		magnitude: 0.7, minDuration: 4, maxDuration: 9,
	},
	{
		headline: "Massive new oil reserve discovered offshore — production could begin in 2 years",
		category: NewsCatEnergy,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryEnergy}, Sentiment: -0.3},
			{Industries: []Industry{IndustryConsumer, IndustryIndustrial}, Sentiment: 0.2},
		},
		magnitude: 0.4, minDuration: 3, maxDuration: 7,
	},
	{
		headline: "Grid-scale fusion reactor achieves net energy gain — energy sector in turmoil",
		category: NewsCatEnergy,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryTech}, Sentiment: 0.8},
			{Industries: []Industry{IndustryUtilities}, Sentiment: 0.4},
			{Industries: []Industry{IndustryIndustrial}, Sentiment: 0.2},
			{Industries: []Industry{IndustryEnergy}, Sentiment: -0.5},
		},
		magnitude: 0.9, minDuration: 4, maxDuration: 10,
	},
	{
		headline: "Pipeline explosion shuts down 30% of regional natural gas supply",
		category: NewsCatEnergy,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryEnergy}, Sentiment: 0.4},
			{Industries: []Industry{IndustryUtilities, IndustryIndustrial}, Sentiment: -0.4},
			{Industries: []Industry{IndustryConsumer}, Sentiment: -0.2},
		},
		magnitude: 0.6, minDuration: 2, maxDuration: 5,
	},
	{
		headline: "Renewable energy surpasses fossil fuels in grid generation for first time",
		category: NewsCatEnergy,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryTech, IndustryUtilities}, Sentiment: 0.3},
			{Industries: []Industry{IndustryEnergy}, Sentiment: -0.3},
			{Industries: []Industry{IndustryIndustrial}, Sentiment: 0.1},
		},
		magnitude: 0.4, minDuration: 3, maxDuration: 6,
	},

	// CRYPTO
	{
		headline: "Major nation announces Bitcoin as legal tender — crypto markets surge",
		category: NewsCatCrypto,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryCrypto}, Sentiment: 0.8},
			{Industries: []Industry{IndustryFinance}, Sentiment: 0.4},
			{Industries: []Industry{IndustryTech}, Sentiment: 0.2},
		},
		magnitude: 0.9, minDuration: 3, maxDuration: 8,
	},
	{
		headline: "Crypto exchange collapses following $2B in unauthorized withdrawals",
		category: NewsCatCrypto,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryCrypto}, Sentiment: -0.8},
			{Industries: []Industry{IndustryFinance}, Sentiment: -0.4},
			{Industries: []Industry{IndustryTech}, Sentiment: -0.15},
		},
		magnitude: 0.9, minDuration: 3, maxDuration: 8,
	},
	{
		headline: "SEC approves first spot crypto ETF — institutional floodgates open",
		category: NewsCatCrypto,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryCrypto}, Sentiment: 0.7},
			{Industries: []Industry{IndustryFinance}, Sentiment: 0.5},
		},
		magnitude: 0.8, minDuration: 3, maxDuration: 7,
	},
	{
		headline: "G7 nations agree on sweeping crypto taxation framework",
		category: NewsCatCrypto,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryCrypto}, Sentiment: -0.5},
			{Industries: []Industry{IndustryFinance}, Sentiment: 0.2},
			{Industries: []Industry{IndustryTech}, Sentiment: -0.1},
		},
		magnitude: 0.6, minDuration: 3, maxDuration: 6,
	},
	{
		headline: "DeFi protocol hack drains $800M — smart contract audits under scrutiny",
		category: NewsCatCrypto,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryCrypto}, Sentiment: -0.6},
			{Industries: []Industry{IndustryTech}, Sentiment: -0.2},
			{Industries: []Industry{IndustryFinance}, Sentiment: 0.1},
		},
		magnitude: 0.7, minDuration: 2, maxDuration: 5,
	},

	// CONSUMER
	{
		headline: "Holiday retail sales surge 12% YoY — consumer confidence at 5-year high",
		category: NewsCatConsumer,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryConsumer}, Sentiment: 0.5},
			{Industries: []Industry{IndustryReal}, Sentiment: 0.3},
			{Industries: []Industry{IndustryFinance}, Sentiment: 0.2},
			{Industries: []Industry{IndustryIndustrial}, Sentiment: 0.1},
		},
		magnitude: 0.5, minDuration: 2, maxDuration: 5,
	},
	{
		headline: "Supply chain crisis deepens — shipping costs triple, shelves empty",
		category: NewsCatConsumer,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryConsumer, IndustryIndustrial}, Sentiment: -0.5},
			{Industries: []Industry{IndustryEnergy}, Sentiment: 0.2},
			{Industries: []Industry{IndustryMaterials}, Sentiment: -0.2},
		},
		magnitude: 0.6, minDuration: 3, maxDuration: 7,
	},
	{
		headline: "Consumer debt reaches all-time high — default rates beginning to rise",
		category: NewsCatConsumer,
		effects: []InfluenceEffect{
			{Industries: []Industry{IndustryConsumer, IndustryFinance}, Sentiment: -0.4},
			{Industries: []Industry{IndustryReal}, Sentiment: -0.2},
		},
		magnitude: 0.5, minDuration: 3, maxDuration: 6,
	},

	// EARNINGS (company-specific)
	{
		headline:    "{company} ({symbol}) reports blowout earnings — raises full-year guidance",
		category:    NewsCatEarnings,
		effects:     []InfluenceEffect{{Sentiment: 0.7}, {CompanyIndustry: true, Sentiment: 0.1}},
		magnitude:   0.7, minDuration: 2, maxDuration: 5,
		companyNews: true,
	},
	{
		headline:    "{company} ({symbol}) misses estimates badly — CEO steps down effective immediately",
		category:    NewsCatEarnings,
		effects:     []InfluenceEffect{{Sentiment: -0.7}, {CompanyIndustry: true, Sentiment: -0.1}},
		magnitude:   0.8, minDuration: 2, maxDuration: 5,
		companyNews: true,
	},
	{
		headline:    "{company} ({symbol}) announces $5B share buyback program",
		category:    NewsCatEarnings,
		effects:     []InfluenceEffect{{Sentiment: 0.5}},
		magnitude:   0.5, minDuration: 2, maxDuration: 4,
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
	headline := tmpl.headline

	// Deep-copy effects so template data is not mutated.
	effects := make([]InfluenceEffect, len(tmpl.effects))
	for i, eff := range tmpl.effects {
		cp := InfluenceEffect{
			Sentiment:       eff.Sentiment,
			CompanyIndustry: eff.CompanyIndustry,
		}
		if len(eff.Industries) > 0 {
			cp.Industries = make([]Industry, len(eff.Industries))
			copy(cp.Industries, eff.Industries)
		}
		if len(eff.Symbols) > 0 {
			cp.Symbols = make([]string, len(eff.Symbols))
			copy(cp.Symbols, eff.Symbols)
		}
		effects[i] = cp
	}

	if tmpl.companyNews && len(stocks) > 0 {
		pick := stocks[rng.Intn(len(stocks))]
		headline = strings.ReplaceAll(headline, "{company}", pick.Name)
		headline = strings.ReplaceAll(headline, "{symbol}", pick.Symbol)
		// effects[0] targets the company stock directly
		effects[0].Symbols = []string{pick.Symbol}
		// any effect flagged CompanyIndustry gets the company's actual industry
		for i := range effects {
			if effects[i].CompanyIndustry {
				effects[i].Industries = []Industry{pick.Industry}
			}
		}
	} else {
		// Replace {industry} placeholder if present
		if strings.Contains(headline, "{industry}") {
			for _, eff := range effects {
				if len(eff.Industries) > 0 {
					headline = strings.ReplaceAll(headline, "{industry}",
						string(eff.Industries[rng.Intn(len(eff.Industries))]))
					break
				}
			}
		}
	}

	// Jitter each effect's sentiment ±10% of its absolute value
	for i := range effects {
		jitter := (rng.Float64()*0.2 - 0.1) * math.Abs(effects[i].Sentiment)
		sent := effects[i].Sentiment + jitter
		if sent > 1.0 {
			sent = 1.0
		} else if sent < -1.0 {
			sent = -1.0
		}
		effects[i].Sentiment = roundTo(sent, 2)
	}

	magnitude := roundTo(tmpl.magnitude*(0.8+rng.Float64()*0.4), 2)
	duration := tmpl.minDuration + rng.Intn(tmpl.maxDuration-tmpl.minDuration+1)
	now := time.Now()
	expires := now.Add(time.Duration(duration) * time.Minute)

	// Build backward-compat flat fields and derive primary sentiment.
	var allIndustries []Industry
	var allSymbols []string
	primarySent := 0.0
	primaryAbs := 0.0
	for _, eff := range effects {
		allIndustries = append(allIndustries, eff.Industries...)
		allSymbols = append(allSymbols, eff.Symbols...)
		if a := math.Abs(eff.Sentiment); a > primaryAbs {
			primaryAbs = a
			primarySent = eff.Sentiment
		}
	}

	event := &NewsEvent{
		ID:                 nextID,
		Headline:           headline,
		Category:           tmpl.category,
		Effects:            effects,
		AffectedIndustries: allIndustries,
		AffectedSymbols:    allSymbols,
		Sentiment:          roundTo(primarySent, 2),
		Magnitude:          magnitude,
		CreatedAt:          now,
		ExpiresAt:          expires,
	}

	influence := &MarketInfluence{
		NewsID:    nextID,
		Effects:   effects,
		Magnitude: magnitude,
		CreatedAt: now,
		ExpiresAt: expires,
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
