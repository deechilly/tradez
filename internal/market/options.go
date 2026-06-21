package market

import "math"

// Game time scale: 1000 ticks (2 000 s ≈ 33 min of play) is treated as one
// "year" for Black-Scholes purposes. This gives reasonable premiums relative
// to the fast-moving price engine.
const ticksPerYear = 1000.0

// riskFreeRate is the annualised continuously-compounded risk-free rate.
const riskFreeRate = 0.05

// ImpliedVol converts a stock's per-tick volatility to an annualised implied
// volatility suitable for Black-Scholes.
func ImpliedVol(perTickVol float64) float64 {
	return perTickVol * math.Sqrt(ticksPerYear)
}

// PutPrice returns the Black-Scholes fair value of a European put option,
// priced per underlying share.
//
//   spot        – current stock price
//   strike      – option strike price
//   iv          – annualised implied volatility (from ImpliedVol)
//   ticksLeft   – ticks remaining until expiry
func PutPrice(spot, strike, iv float64, ticksLeft int) float64 {
	T := float64(ticksLeft) / ticksPerYear
	if T <= 0 || iv <= 0 || spot <= 0 {
		return math.Max(strike-spot, 0)
	}
	sqrtT := math.Sqrt(T)
	d1 := (math.Log(spot/strike) + (riskFreeRate+iv*iv/2)*T) / (iv * sqrtT)
	d2 := d1 - iv*sqrtT
	discount := math.Exp(-riskFreeRate * T)
	return strike*discount*normCDF(-d2) - spot*normCDF(-d1)
}

// PutIntrinsic returns the intrinsic (in-the-money) value of a put per share.
func PutIntrinsic(spot, strike float64) float64 {
	return math.Max(strike-spot, 0)
}

// normCDF is the standard normal cumulative distribution function, computed
// via the complementary error function provided by the math package.
func normCDF(x float64) float64 {
	return 0.5 * math.Erfc(-x/math.Sqrt2)
}
