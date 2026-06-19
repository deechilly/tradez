package market

import (
	"fmt"
	"math/rand"
	"time"
)

var industryData = map[Industry]struct {
	companies []struct{ name, symbol string }
	basePrice [2]float64
	beta      [2]float64
}{
	IndustryTech: {
		companies: []struct{ name, symbol string }{
			{"NovaSoft Systems", "NVSS"}, {"Apex Computing", "APXC"}, {"ByteForge Inc", "BYFG"},
			{"CloudPeak Technologies", "CPKT"}, {"DataStream Corp", "DSTR"}, {"Electron Labs", "ELBL"},
			{"FutureTech", "FRTK"}, {"GridMind AI", "GMAI"}, {"HyperCore", "HPRC"},
			{"IronByte Solutions", "IRBS"}, {"JetStream Dev", "JTSD"}, {"Kernel Systems", "KRNL"},
		},
		basePrice: [2]float64{50, 800},
		beta:      [2]float64{1.1, 2.2},
	},
	IndustryFinance: {
		companies: []struct{ name, symbol string }{
			{"Atlantic Capital", "ATLC"}, {"BlueStar Bank", "BLSB"}, {"Crown Financial", "CRNF"},
			{"Delta Trust", "DLTT"}, {"Empire Investments", "EMPI"}, {"Falcon Wealth", "FLCW"},
			{"Global Markets Inc", "GLMI"}, {"Harbor Securities", "HRBS"},
		},
		basePrice: [2]float64{20, 300},
		beta:      [2]float64{0.8, 1.5},
	},
	IndustryEnergy: {
		companies: []struct{ name, symbol string }{
			{"Apex Petroleum", "AXPT"}, {"Borealis Energy", "BORE"}, {"Carbon Free Power", "CFPW"},
			{"DeltaOil Corp", "DLOC"}, {"EcoFlare Systems", "ECFS"}, {"FusionGrid", "FSGR"},
		},
		basePrice: [2]float64{15, 200},
		beta:      [2]float64{0.9, 1.8},
	},
	IndustryHealthcare: {
		companies: []struct{ name, symbol string }{
			{"AlphaGen Pharma", "ALPG"}, {"BioNexus Labs", "BNXL"}, {"CurePoint Medical", "CPMD"},
			{"DeltaCell Biotech", "DCBT"}, {"EndoGen Research", "ENGR"}, {"FortisMed", "FTMD"},
			{"GeneSpark", "GNSP"}, {"HelixCore Bio", "HLXB"},
		},
		basePrice: [2]float64{30, 500},
		beta:      [2]float64{0.7, 1.6},
	},
	IndustryConsumer: {
		companies: []struct{ name, symbol string }{
			{"Apex Retail Group", "APRG"}, {"Bloom Brands", "BLMB"}, {"CoreStyle Inc", "CRST"},
			{"DayBreak Foods", "DYBF"}, {"EverGlow Consumer", "EVGC"}, {"FreshMart", "FRSM"},
		},
		basePrice: [2]float64{10, 150},
		beta:      [2]float64{0.5, 1.2},
	},
	IndustryIndustrial: {
		companies: []struct{ name, symbol string }{
			{"AeroForge Industries", "AFGI"}, {"BuildCore Corp", "BLDC"}, {"CastIron Works", "CSIW"},
			{"DriveShaft Inc", "DRSF"}, {"EdgeBuild Systems", "EDBS"},
		},
		basePrice: [2]float64{25, 250},
		beta:      [2]float64{0.8, 1.4},
	},
	IndustryCrypto: {
		companies: []struct{ name, symbol string }{
			{"AltChain Finance", "ALTF"}, {"BlockVault", "BLKV"}, {"CryptoNest", "CRYN"},
			{"DecentraEx", "DCEX"}, {"EtherBridge Corp", "ETBR"}, {"FluxCoin Group", "FLXG"},
		},
		basePrice: [2]float64{5, 300},
		beta:      [2]float64{2.0, 4.0},
	},
	IndustryReal: {
		companies: []struct{ name, symbol string }{
			{"Apex Properties", "APXP"}, {"BlueSky REIT", "BLSR"}, {"CityCore Realty", "CTCR"},
			{"DomainPrime Trust", "DMPT"}, {"EliteVista REIT", "ELVR"},
		},
		basePrice: [2]float64{10, 120},
		beta:      [2]float64{0.4, 1.0},
	},
	IndustryMaterials: {
		companies: []struct{ name, symbol string }{
			{"Alloy Prime Corp", "ALPC"}, {"BaseMetal Industries", "BMTI"}, {"CoreMine Ltd", "CRML"},
			{"DeltaSteel Works", "DSTW"},
		},
		basePrice: [2]float64{20, 180},
		beta:      [2]float64{0.9, 1.7},
	},
	IndustryUtilities: {
		companies: []struct{ name, symbol string }{
			{"AquaGrid Power", "AQGP"}, {"BrightWave Electric", "BRWE"}, {"CleanFlow Utilities", "CLFU"},
			{"DuraWatt Corp", "DRWT"},
		},
		basePrice: [2]float64{20, 100},
		beta:      [2]float64{0.3, 0.8},
	},
}

func GenerateMarket(rng *rand.Rand) []*Stock {
	stocks := make([]*Stock, 0, 50)
	now := time.Now()

	for industry, data := range industryData {
		used := map[int]bool{}
		count := 3 + rng.Intn(3)
		if count > len(data.companies) {
			count = len(data.companies)
		}
		for i := 0; i < count; i++ {
			idx := rng.Intn(len(data.companies))
			for used[idx] {
				idx = (idx + 1) % len(data.companies)
			}
			used[idx] = true
			c := data.companies[idx]

			lo, hi := data.basePrice[0], data.basePrice[1]
			price := lo + rng.Float64()*(hi-lo)
			price = roundTo(price, 2)

			betaLo, betaHi := data.beta[0], data.beta[1]
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
				Symbol:     c.symbol,
				Name:       c.name,
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
	step := -2 * time.Second
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
		_ = step
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
