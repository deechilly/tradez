package market

import "time"

type Industry string

const (
	IndustryTech        Industry = "Technology"
	IndustryFinance     Industry = "Finance"
	IndustryEnergy      Industry = "Energy"
	IndustryHealthcare  Industry = "Healthcare"
	IndustryConsumer    Industry = "Consumer"
	IndustryIndustrial  Industry = "Industrial"
	IndustryCrypto      Industry = "Crypto"
	IndustryReal        Industry = "Real Estate"
	IndustryMaterials   Industry = "Materials"
	IndustryUtilities   Industry = "Utilities"
)

type Stock struct {
	Symbol      string
	Name        string
	Industry    Industry
	Price       float64
	PrevClose   float64
	Open        float64
	High        float64
	Low         float64
	Volume      int64
	MarketCap   float64
	PERatio     float64
	EPS         float64
	DivYield    float64
	Beta        float64
	YTDGrowth   float64
	History     []PricePoint
	Volatility  float64
	Trend       float64
}

type PricePoint struct {
	Time  time.Time
	Price float64
}

func (s *Stock) Change() float64 {
	return s.Price - s.PrevClose
}

func (s *Stock) ChangePct() float64 {
	if s.PrevClose == 0 {
		return 0
	}
	return (s.Price - s.PrevClose) / s.PrevClose * 100
}
