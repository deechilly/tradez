package market

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

type wordBank struct {
	prefixes  []string
	roots     []string
	suffixes  []string
	basePrice [2]float64
	beta      [2]float64
}

var industryBanks = map[Industry]wordBank{
	IndustryTech: {
		prefixes: []string{
			"Nova", "Apex", "Cloud", "Data", "Hyper", "Iron", "Jet", "Kernel",
			"Laser", "Micro", "Nano", "Orbit", "Pixel", "Quantum", "Rapid",
			"Sync", "Turbo", "Ultra", "Volt", "Zenith", "Vector", "Cyber",
			"Grid", "Neural", "Pulse", "Deep", "Edge", "Flash", "Swift", "Smart",
		},
		roots: []string{
			"Force", "Core", "Wave", "Link", "Path", "Mind", "Chip",
			"Hub", "Node", "Gate", "Logic", "Bridge", "Scale", "Stack",
			"Stream", "Mark", "Works", "Point", "Sight", "Shift",
		},
		suffixes: []string{
			"Systems", "Technologies", "Solutions", "Computing", "Networks",
			"Labs", "Software", "Corp", "Inc", "AI",
		},
		basePrice: [2]float64{50, 800},
		beta:      [2]float64{1.1, 2.2},
	},
	IndustryFinance: {
		prefixes: []string{
			"Atlantic", "Blue", "Crown", "Delta", "Empire", "Falcon", "Global",
			"Harbor", "Jade", "Knight", "Liberty", "Meridian", "Noble",
			"Pacific", "Royal", "Summit", "Titan", "Union", "Vanguard",
			"Crestwood", "Golden", "Highland", "Prestige", "Sterling", "Anchor",
		},
		roots: []string{
			"Capital", "Trust", "Wealth", "Markets", "Partners",
			"Investments", "Securities", "Advisors", "Bancorp",
			"Asset", "Equity", "Credit", "Commerce", "Reserve", "Ledger",
		},
		suffixes: []string{"Corp", "Inc", "Group", "Ltd", "Holdings"},
		basePrice: [2]float64{20, 300},
		beta:      [2]float64{0.8, 1.5},
	},
	IndustryEnergy: {
		prefixes: []string{
			"Apex", "Borealis", "Carbon", "Delta", "Eco", "Fusion", "Green",
			"Horizon", "Ion", "Kinetic", "Lumina", "Mesa", "Nova", "Orbit",
			"Peak", "Radiant", "Solar", "Terra", "Tidal", "Vortex",
			"Amber", "Blaze", "Crest", "Arctic", "Cascade",
		},
		roots: []string{
			"Oil", "Gas", "Power", "Energy", "Grid", "Fuel", "Force",
			"Flare", "Charge", "Stream", "Flow", "Field", "Core", "Works", "Dynamics",
		},
		suffixes: []string{"Corp", "Inc", "Systems", "Group", "Ltd"},
		basePrice: [2]float64{15, 200},
		beta:      [2]float64{0.9, 1.8},
	},
	IndustryHealthcare: {
		prefixes: []string{
			"Alpha", "Bio", "Cell", "Delta", "Endo", "Forte", "Gene", "Helix",
			"Immuno", "Kinase", "Locus", "Macro", "Nano", "Onco", "Proto",
			"Regen", "Soma", "Synth", "Vita", "Vivo", "Micro", "Neuro",
			"Amino", "Cura", "Epi",
		},
		roots: []string{
			"Gen", "Nexus", "Point", "Cure", "Core", "Spark", "Bridge",
			"Path", "Logic", "Scope", "Tech", "Mark", "Lab", "Quest",
			"Life", "Pharma", "Dynamics", "Solutions", "Works", "Genomics",
		},
		suffixes: []string{
			"Pharma", "Labs", "Medical", "Biotech", "Therapeutics",
			"Sciences", "Research", "Corp", "Inc",
		},
		basePrice: [2]float64{30, 500},
		beta:      [2]float64{0.7, 1.6},
	},
	IndustryConsumer: {
		prefixes: []string{
			"Apex", "Bloom", "Core", "Dawn", "Ever", "Fresh", "Green",
			"Haven", "Ivory", "Jade", "Kind", "Lux", "Maple", "Nest",
			"Oak", "Prime", "Quick", "Rise", "Sun", "True",
			"Urban", "Vivid", "Wave", "Zen", "Bright",
		},
		roots: []string{
			"Mart", "Brand", "Style", "Market", "Foods", "Living",
			"Home", "Goods", "Direct", "Choice", "Valley", "Source",
			"Basket", "Garden", "Kitchen", "Pantry", "Supply", "Table", "Craft", "Works",
		},
		suffixes: []string{"Group", "Corp", "Inc", "Brands", "Retail", "Co"},
		basePrice: [2]float64{10, 150},
		beta:      [2]float64{0.5, 1.2},
	},
	IndustryIndustrial: {
		prefixes: []string{
			"Aero", "Build", "Cast", "Drive", "Edge", "Forge", "Grid",
			"Heavy", "Iron", "Jet", "Kinetic", "Load", "Mech", "Nord",
			"Peak", "Quad", "Rig", "Steel", "Torque", "Ultra",
			"Vault", "Weld", "Axle", "Bridge", "Chain",
		},
		roots: []string{
			"Works", "Force", "Core", "Tech", "Craft", "Dynamics",
			"Parts", "Fab", "Plant", "Mill", "Lift", "Haul", "Frame", "Rig", "Build",
		},
		suffixes: []string{"Corp", "Inc", "Industries", "Systems", "Group", "Ltd"},
		basePrice: [2]float64{25, 250},
		beta:      [2]float64{0.8, 1.4},
	},
	IndustryCrypto: {
		prefixes: []string{
			"Alt", "Block", "Chain", "Defi", "Ether", "Flux", "Hash",
			"Ion", "Key", "Layer", "Meta", "Node", "Orbit", "Proto",
			"Quantum", "Relay", "Stake", "Token", "Vault", "Zero",
		},
		roots: []string{
			"Vault", "Net", "Coin", "Bridge", "Swap", "Link",
			"Pay", "Lock", "Ledger", "Guard", "Trust", "Hub", "Gate", "Relay", "Core",
		},
		suffixes: []string{"Finance", "Group", "Corp", "Protocol", "Labs", "Inc"},
		basePrice: [2]float64{5, 300},
		beta:      [2]float64{2.0, 4.0},
	},
	IndustryReal: {
		prefixes: []string{
			"Apex", "Blue", "City", "Domain", "Elite", "First", "Grand",
			"Harbor", "Imperial", "Jade", "Key", "Landmark", "Meridian",
			"Noble", "Oak", "Prestige", "Quest", "Regent", "Summit", "Tidal",
		},
		roots: []string{
			"Properties", "Realty", "Estates", "Ventures", "Development",
			"Assets", "Park", "Square", "Place", "Point",
			"Prime", "Ridge", "Land", "Haven", "Plaza",
		},
		suffixes: []string{"Corp", "Inc", "Group", "Ltd", "REIT"},
		basePrice: [2]float64{10, 120},
		beta:      [2]float64{0.4, 1.0},
	},
	IndustryMaterials: {
		prefixes: []string{
			"Alloy", "Base", "Crest", "Delta", "Element", "Forge", "Grade",
			"Heavy", "Iron", "Kilo", "Lode", "Metal", "North", "Ore",
			"Pure", "Quartz", "Ridge", "Steel", "Titan", "Ultra",
		},
		roots: []string{
			"Mine", "Works", "Industries", "Resources", "Minerals",
			"Stone", "Earth", "Processing", "Products", "Foundry",
			"Smelting", "Casting", "Extraction", "Alloys", "Solutions",
		},
		suffixes: []string{"Corp", "Ltd", "Inc", "Group"},
		basePrice: [2]float64{20, 180},
		beta:      [2]float64{0.9, 1.7},
	},
	IndustryUtilities: {
		prefixes: []string{
			"Aqua", "Bright", "Clean", "Dura", "Ever", "Green", "Hydro",
			"Ideal", "Kilo", "Metro", "Nova", "Open", "Pure", "Quick",
			"River", "Solar", "Terra", "Ultra", "Volt", "Watt",
		},
		roots: []string{
			"Grid", "Power", "Water", "Electric", "Energy", "Force",
			"Works", "Services", "Network", "Supply", "Source", "Steam",
			"Systems", "Current", "Charge",
		},
		suffixes: []string{"Corp", "Inc", "Group", "Ltd"},
		basePrice: [2]float64{20, 100},
		beta:      [2]float64{0.3, 0.8},
	},
}

// skipForSymbol lists word that shouldn't contribute to ticker derivation.
var skipForSymbol = map[string]bool{
	"Corp": true, "Inc": true, "Ltd": true, "Group": true,
	"Holdings": true, "Co": true, "REIT": true, "AI": true,
	"Protocol": true, "Industries": true, "Technologies": true,
	"Solutions": true, "Computing": true, "Software": true,
	"Networks": true, "Therapeutics": true, "Sciences": true,
	"Research": true, "Finance": true, "Labs": true,
}

func deriveSymbol(name string, used map[string]bool, rng *rand.Rand) string {
	words := strings.Fields(name)
	var sig []string
	for _, w := range words {
		if !skipForSymbol[w] {
			sig = append(sig, strings.ToUpper(w))
		}
	}
	if len(sig) == 0 {
		sig = []string{strings.ToUpper(words[0])}
	}

	candidates := []string{}

	// First 4 letters of first significant word
	if len(sig[0]) >= 4 {
		candidates = append(candidates, sig[0][:4])
	}
	// First 3 of word1 + first of word2
	if len(sig) >= 2 && len(sig[0]) >= 3 {
		candidates = append(candidates, sig[0][:3]+string(sig[1][0]))
	}
	// First 2 of word1 + first 2 of word2
	if len(sig) >= 2 && len(sig[0]) >= 2 && len(sig[1]) >= 2 {
		candidates = append(candidates, sig[0][:2]+sig[1][:2])
	}
	// Initials padded with consonants from first word
	initials := ""
	for _, w := range sig {
		if len(initials) < 4 {
			initials += string(w[0])
		}
	}
	if len(initials) < 4 {
		for _, ch := range sig[0][1:] {
			if len(initials) >= 4 {
				break
			}
			if ch != 'A' && ch != 'E' && ch != 'I' && ch != 'O' && ch != 'U' {
				initials += string(ch)
			}
		}
		for len(initials) < 4 {
			initials += "X"
		}
	}
	candidates = append(candidates, initials[:4])

	trySymbol := func(s string) string {
		for len(s) < 4 {
			s += "X"
		}
		s = s[:4]
		if !used[s] {
			return s
		}
		return ""
	}

	for _, c := range candidates {
		if s := trySymbol(c); s != "" {
			return s
		}
	}

	// Collision resolution: vary last char of best candidate
	base := candidates[0]
	for len(base) < 3 {
		base += "X"
	}
	base = base[:3]
	for i := 0; i < 26; i++ {
		variant := base + string(rune('A'+i))
		if !used[variant] {
			return variant
		}
	}
	// Vary second char too
	for i := 0; i < 26; i++ {
		for j := 0; j < 26; j++ {
			variant := string(base[0]) + string(rune('A'+i)) + string(rune('A'+j)) + string(base[2])
			if !used[variant] {
				return variant
			}
		}
	}
	return ""
}

func GenerateMarket(rng *rand.Rand) []*Stock {
	stocks := make([]*Stock, 0, 90)
	now := time.Now()

	usedPairs := make(map[string]bool)   // prefix|root uniqueness across all industries
	usedSymbols := make(map[string]bool) // ticker uniqueness

	for industry, bank := range industryBanks {
		count := 7 + rng.Intn(3) // 7–9 per industry → 70–90 total

		for i := 0; i < count; i++ {
			name, sym := "", ""
			ok := false
			for attempt := 0; attempt < 60; attempt++ {
				pre := bank.prefixes[rng.Intn(len(bank.prefixes))]
				root := bank.roots[rng.Intn(len(bank.roots))]
				suf := bank.suffixes[rng.Intn(len(bank.suffixes))]

				pairKey := pre + "|" + root
				if usedPairs[pairKey] {
					continue
				}

				candidate := pre + " " + root + " " + suf
				sym = deriveSymbol(candidate, usedSymbols, rng)
				if sym == "" {
					continue
				}

				usedPairs[pairKey] = true
				usedSymbols[sym] = true
				name = candidate
				ok = true
				break
			}
			if !ok {
				continue
			}

			lo, hi := bank.basePrice[0], bank.basePrice[1]
			price := lo + rng.Float64()*(hi-lo)
			price = roundTo(price, 2)

			betaLo, betaHi := bank.beta[0], bank.beta[1]
			beta := betaLo + rng.Float64()*(betaHi-betaLo)
			volatility := beta * (0.008 + rng.Float64()*0.012)
			trend := (rng.Float64()*2 - 1) * 0.0003

			eps := price / (8 + rng.Float64()*25)
			pe := price / eps
			divYield := 0.0
			if rng.Float64() < 0.5 {
				divYield = rng.Float64() * 5
			}

			marketCap := price * float64(int64(1e6)+rng.Int63n(int64(999e9)))
			volume := int64(100_000 + rng.Intn(50_000_000))
			ytd := (rng.Float64()*2 - 1) * 80

			prevClose := price * (1 + (rng.Float64()*2-1)*volatility*5)
			open := prevClose * (1 + (rng.Float64()*2-1)*volatility*2)

			history := generateHistory(rng, price, volatility, now, 200)

			stocks = append(stocks, &Stock{
				Symbol:     sym,
				Name:       name,
				Industry:   industry,
				Price:      price,
				PrevClose:  roundTo(prevClose, 2),
				Open:       roundTo(open, 2),
				High:       roundTo(price*1.02, 2),
				Low:        roundTo(price*0.98, 2),
				Volume:     volume,
				MarketCap:  marketCap,
				PERatio:    roundTo(pe, 1),
				EPS:        roundTo(eps, 2),
				DivYield:   roundTo(divYield, 2),
				Beta:       roundTo(beta, 2),
				YTDGrowth:  roundTo(ytd, 2),
				History:    history,
				Volatility: volatility,
				Trend:      trend,
			})
		}
	}

	return stocks
}

func generateHistory(rng *rand.Rand, currentPrice, volatility float64, now time.Time, points int) []PricePoint {
	history := make([]PricePoint, points)
	price := currentPrice * (1 + (rng.Float64()*2-1)*0.3)
	for i := points - 1; i >= 0; i-- {
		change := (rng.Float64()*2 - 1) * volatility * price
		price += change
		if price < 0.01 {
			price = 0.01
		}
		history[i] = PricePoint{
			Time:  now.Add(time.Duration(i-points+1) * 2 * time.Second),
			Price: roundTo(price, 2),
		}
	}
	history[points-1].Price = currentPrice
	return history
}

func roundTo(v float64, decimals int) float64 {
	factor := 1.0
	for i := 0; i < decimals; i++ {
		factor *= 10
	}
	return float64(int(v*factor+0.5)) / factor
}

func FormatMarketCap(cap float64) string {
	switch {
	case cap >= 1e12:
		return fmt.Sprintf("$%.2fT", cap/1e12)
	case cap >= 1e9:
		return fmt.Sprintf("$%.2fB", cap/1e9)
	case cap >= 1e6:
		return fmt.Sprintf("$%.2fM", cap/1e6)
	default:
		return fmt.Sprintf("$%.0f", cap)
	}
}
